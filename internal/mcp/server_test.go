package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func runOnce(t *testing.T, s *Server, req string) map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := s.Serve(strings.NewReader(req+"\n"), &out); err != nil {
		t.Fatalf("serve: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response %q: %v", out.String(), err)
	}
	return resp
}

func TestServe_Initialize(t *testing.T) {
	s := New()
	resp := runOnce(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"0.0.1"}}}`)
	if resp["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", resp["jsonrpc"])
	}
	if id, _ := resp["id"].(float64); id != 1 {
		t.Errorf("id = %v, want 1", resp["id"])
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %v", resp)
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("protocolVersion = %v, want 2024-11-05", result["protocolVersion"])
	}
	info, _ := result["serverInfo"].(map[string]any)
	if info["name"] != "memex" {
		t.Errorf("serverInfo.name = %v, want memex", info["name"])
	}
	caps, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("capabilities missing or wrong type: %v", result)
	}
	if _, ok := caps["tools"].(map[string]any); !ok {
		t.Errorf("capabilities.tools missing — clients won't query tools/list: %v", caps)
	}
}

func TestServe_NotificationsInitialized_NoResponse(t *testing.T) {
	s := New()
	var out bytes.Buffer
	if err := s.Serve(strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n"), &out); err != nil {
		t.Fatalf("serve: %v", err)
	}
	if got := out.String(); got != "" {
		t.Errorf("notification produced response, want silence: %q", got)
	}
}

func TestServe_UnknownNotification_NoResponse(t *testing.T) {
	s := New()
	var out bytes.Buffer
	if err := s.Serve(strings.NewReader(`{"jsonrpc":"2.0","method":"some/unknown/notif"}`+"\n"), &out); err != nil {
		t.Fatalf("serve: %v", err)
	}
	if got := out.String(); got != "" {
		t.Errorf("unknown notification produced response, want silence: %q", got)
	}
}

func TestServe_ToolsList_Empty(t *testing.T) {
	s := New()
	resp := runOnce(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %v", resp)
	}
	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools not an array: %v", result)
	}
	if len(tools) != 0 {
		t.Errorf("expected empty tools, got %v", tools)
	}
}

func TestServe_ToolsCall_Unknown(t *testing.T) {
	s := New()
	resp := runOnce(t, s, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"nope","arguments":{}}}`)
	if _, hasResult := resp["result"]; hasResult {
		t.Errorf("error response should not include result: %v", resp)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error: %v", resp)
	}
	if int(errObj["code"].(float64)) != -32602 {
		t.Errorf("error.code = %v, want -32602", errObj["code"])
	}
}

func TestServe_UnknownMethod(t *testing.T) {
	s := New()
	resp := runOnce(t, s, `{"jsonrpc":"2.0","id":4,"method":"frobnicate","params":{}}`)
	if _, hasResult := resp["result"]; hasResult {
		t.Errorf("error response should not include result: %v", resp)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error: %v", resp)
	}
	if int(errObj["code"].(float64)) != -32601 {
		t.Errorf("error.code = %v, want -32601", errObj["code"])
	}
}

type stubTool struct{}

func (stubTool) Name() string            { return "memex_stub" }
func (stubTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object","properties":{}}`) }
func (stubTool) Call(_ json.RawMessage) (string, error) {
	return "stub-ok", nil
}

type panicTool struct{}

func (panicTool) Name() string            { return "memex_panic" }
func (panicTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (panicTool) Call(_ json.RawMessage) (string, error) {
	var p *struct{ X int }
	return "", fmt.Errorf("unreachable: %d", p.X)
}

func TestRegister_PanickingToolReturnsErrorButServerStaysAlive(t *testing.T) {
	s := New()
	s.Register(panicTool{})
	s.Register(stubTool{})

	var out bytes.Buffer
	in := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"memex_panic","arguments":{}}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"memex_stub","arguments":{}}}` + "\n"
	if err := s.Serve(strings.NewReader(in), &out); err != nil {
		t.Fatalf("serve: %v", err)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 responses, got %d: %q", len(lines), out.String())
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("line 1 unmarshal: %v", err)
	}
	res, _ := first["result"].(map[string]any)
	if res == nil {
		t.Fatalf("panic call missing result envelope (would close connection): %v", first)
	}
	if isErr, _ := res["isError"].(bool); !isErr {
		t.Errorf("panic should set isError=true: %v", res)
	}
	var second map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("line 2 unmarshal: %v", err)
	}
	if second["id"].(float64) != 2 {
		t.Errorf("server did not survive panic to handle next call: %v", second)
	}
}

func TestRegister_And_Call(t *testing.T) {
	s := New()
	s.Register(stubTool{})

	listResp := runOnce(t, s, `{"jsonrpc":"2.0","id":10,"method":"tools/list","params":{}}`)
	tools := listResp["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(tools))
	}
	first := tools[0].(map[string]any)
	if first["name"] != "memex_stub" {
		t.Errorf("name = %v, want memex_stub", first["name"])
	}
	if _, ok := first["inputSchema"].(map[string]any); !ok {
		t.Errorf("inputSchema missing or wrong shape: %v", first)
	}

	callResp := runOnce(t, s, `{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"memex_stub","arguments":{}}}`)
	result, ok := callResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %v", callResp)
	}
	contents, ok := result["content"].([]any)
	if !ok || len(contents) == 0 {
		t.Fatalf("content missing/empty: %v", result)
	}
	c0 := contents[0].(map[string]any)
	if c0["type"] != "text" {
		t.Errorf("type = %v, want text", c0["type"])
	}
	if c0["text"] != "stub-ok" {
		t.Errorf("text = %v, want stub-ok", c0["text"])
	}
	if isErr, _ := result["isError"].(bool); isErr {
		t.Errorf("isError = true, want false/absent")
	}
}

func TestServe_MalformedJSON(t *testing.T) {
	s := New()
	var out bytes.Buffer
	if err := s.Serve(strings.NewReader("not-json\n{\"jsonrpc\":\"2.0\",\"id\":99,\"method\":\"initialize\",\"params\":{}}\n"), &out); err != nil {
		t.Fatalf("serve: %v", err)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 response lines, got %d: %q", len(lines), out.String())
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("line 1 not JSON: %v", err)
	}
	errObj, ok := first["error"].(map[string]any)
	if !ok {
		t.Fatalf("line 1 missing error: %v", first)
	}
	if int(errObj["code"].(float64)) != -32700 {
		t.Errorf("line 1 error.code = %v, want -32700", errObj["code"])
	}
	var second map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("line 2 not JSON: %v", err)
	}
	if _, ok := second["result"]; !ok {
		t.Errorf("line 2 missing result; loop did not continue: %v", second)
	}
}
