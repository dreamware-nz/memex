// Package tools houses concrete mcp.Tool implementations registered by the
// `memex mcp` command.
package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dreamware-nz/memex/internal/kb"
)

// PARITY-CHECKED 2026-05-07: response shape (one `## <query>` block per query,
// `No results.` on empty, multi-query concatenation order) matches
// context-mode/src/server.ts §memex_search. The TS schema also exposes a
// `source` filter and a `limit` knob; both are deferred for now and tracked
// in design.md Open Questions. Adding the `source` filter is a future
// follow-up — Searcher would gain a per-query source argument.

// Searcher is the kb-search dependency SearchTool needs. The concrete
// backing store decides how many hits to return per query.
type Searcher interface {
	Search(queries []string) []kb.SearchResult
}

// RPCError lets a tool surface a JSON-RPC error code from Call rather than a
// generic string. The MCP server may pattern-match this type to emit a
// proper error envelope.
type RPCError struct {
	Code    int
	Message string
}

func (e *RPCError) Error() string { return e.Message }

// SearchTool implements the memex_search MCP tool: a batch full-text search
// over the knowledge base, grouped by query in the response.
type SearchTool struct {
	Searcher Searcher
}

func (s *SearchTool) Name() string { return "memex_search" }

func (s *SearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"queries":{"type":"array","items":{"type":"string"},"minItems":1}},"required":["queries"]}`)
}

type searchArgs struct {
	Queries []string `json:"queries"`
}

func (s *SearchTool) Call(raw json.RawMessage) (string, error) {
	var a searchArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", &RPCError{Code: -32602, Message: fmt.Sprintf("invalid arguments: %v", err)}
		}
	}
	if len(a.Queries) == 0 {
		return "", &RPCError{Code: -32602, Message: "queries is required and must contain at least one string"}
	}

	var b strings.Builder
	for i, q := range a.Queries {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "## %s\n", q)
		hits := s.Searcher.Search([]string{q})
		if len(hits) == 0 {
			b.WriteString("No results.\n")
			continue
		}
		for _, h := range hits {
			fmt.Fprintf(&b, "### %s\n%s\n", h.Heading, h.Snippet)
		}
	}
	return b.String(), nil
}
