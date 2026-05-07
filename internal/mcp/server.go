package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

const (
	protocolVersion = "2024-11-05"
	serverName      = "memex"
	serverVersion   = "0.0.0-dev"
)

// Tool is the minimal contract every MCP tool implementation satisfies.
type Tool interface {
	Name() string
	Schema() json.RawMessage
	Call(args json.RawMessage) (string, error)
}

// Describer is an optional interface tools can implement to expose a
// human-readable description in tools/list output.
type Describer interface {
	Description() string
}

type Server struct {
	tools map[string]Tool
}

func New() *Server {
	return &Server{tools: make(map[string]Tool)}
}

func (s *Server) Register(t Tool) {
	s.tools[t.Name()] = t
}

type toolEntry struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

func (s *Server) toolList() []toolEntry {
	out := make([]toolEntry, 0, len(s.tools))
	for _, t := range s.tools {
		entry := toolEntry{Name: t.Name(), InputSchema: t.Schema()}
		if d, ok := t.(Describer); ok {
			entry.Description = d.Description()
		}
		out = append(out, entry)
	}
	return out
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type initializeResult struct {
	ProtocolVersion string            `json:"protocolVersion"`
	ServerInfo      map[string]string `json:"serverInfo"`
	Capabilities    map[string]any    `json:"capabilities"`
}

func (s *Server) handleInitialize(id json.RawMessage) response {
	return response{
		JSONRPC: "2.0",
		ID:      ensureID(id),
		Result: initializeResult{
			ProtocolVersion: protocolVersion,
			ServerInfo:      map[string]string{"name": serverName, "version": serverVersion},
			Capabilities: map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
		},
	}
}

func (s *Server) handleToolsList(id json.RawMessage) response {
	return response{
		JSONRPC: "2.0",
		ID:      ensureID(id),
		Result:  map[string]any{"tools": s.toolList()},
	}
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolsCallResult struct {
	Content []contentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

func (s *Server) handleToolsCall(id json.RawMessage, params json.RawMessage) (resp response) {
	var p toolsCallParams
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return errorResponse(id, -32602, fmt.Sprintf("invalid params: %v", err))
		}
	}
	t, ok := s.tools[p.Name]
	if !ok {
		return errorResponse(id, -32602, fmt.Sprintf("unknown tool: %s", p.Name))
	}
	defer func() {
		if r := recover(); r != nil {
			resp = response{
				JSONRPC: "2.0",
				ID:      ensureID(id),
				Result: toolsCallResult{
					Content: []contentItem{{Type: "text", Text: fmt.Sprintf("tool %q panicked: %v", p.Name, r)}},
					IsError: true,
				},
			}
		}
	}()
	text, err := t.Call(p.Arguments)
	if err != nil {
		return response{
			JSONRPC: "2.0",
			ID:      ensureID(id),
			Result: toolsCallResult{
				Content: []contentItem{{Type: "text", Text: err.Error()}},
				IsError: true,
			},
		}
	}
	return response{
		JSONRPC: "2.0",
		ID:      ensureID(id),
		Result: toolsCallResult{
			Content: []contentItem{{Type: "text", Text: text}},
		},
	}
}

func errorResponse(id json.RawMessage, code int, msg string) response {
	return response{
		JSONRPC: "2.0",
		ID:      ensureID(id),
		Error:   &rpcError{Code: code, Message: msg},
	}
}

func ensureID(id json.RawMessage) json.RawMessage {
	if len(id) == 0 {
		return json.RawMessage("null")
	}
	return id
}

// Serve reads newline-delimited JSON-RPC 2.0 requests from r and writes
// newline-delimited responses to w. It returns when r is exhausted or a
// scan/encode error occurs. Malformed input is reported as a -32700 error
// and does not abort the loop.
func (s *Server) Serve(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	enc := json.NewEncoder(w)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			if encErr := enc.Encode(errorResponse(nil, -32700, "parse error")); encErr != nil {
				return encErr
			}
			continue
		}
		isNotification := len(req.ID) == 0
		var resp response
		switch req.Method {
		case "initialize":
			resp = s.handleInitialize(req.ID)
		case "tools/list":
			resp = s.handleToolsList(req.ID)
		case "tools/call":
			resp = s.handleToolsCall(req.ID, req.Params)
		case "notifications/initialized", "notifications/cancelled", "notifications/progress":
			continue
		default:
			if isNotification {
				continue
			}
			resp = errorResponse(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
		}
		if isNotification {
			continue
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
	return scanner.Err()
}
