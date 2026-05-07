package tools

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/kb"
)

type stubSearcher struct {
	results []kb.SearchResult
	calls   [][]string
}

func (s *stubSearcher) Search(queries []string) []kb.SearchResult {
	s.calls = append(s.calls, append([]string(nil), queries...))
	return s.results
}

func TestSearchTool_Schema(t *testing.T) {
	tool := &SearchTool{}
	var schema map[string]any
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing or wrong shape: %v", schema)
	}
	queries, ok := props["queries"].(map[string]any)
	if !ok {
		t.Fatalf("properties.queries missing: %v", props)
	}
	if queries["type"] != "array" {
		t.Errorf("properties.queries.type = %v, want array", queries["type"])
	}
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("required missing or wrong shape: %v", schema)
	}
	found := false
	for _, r := range required {
		if r == "queries" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("required does not contain queries: %v", required)
	}
}

func TestSearchTool_Call_NoQueries(t *testing.T) {
	tool := &SearchTool{Searcher: &stubSearcher{}}

	cases := []struct {
		name string
		args json.RawMessage
	}{
		{"empty object", json.RawMessage(`{}`)},
		{"empty array", json.RawMessage(`{"queries":[]}`)},
		{"nil args", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tool.Call(tc.args)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			var rpcErr *RPCError
			if !errors.As(err, &rpcErr) {
				t.Fatalf("error is not *RPCError: %T %v", err, err)
			}
			if rpcErr.Code != -32602 {
				t.Errorf("code = %d, want -32602", rpcErr.Code)
			}
		})
	}
}

func TestSearchTool_Call_OneQuery(t *testing.T) {
	stub := &stubSearcher{results: []kb.SearchResult{{
		Heading: "Result Title",
		Snippet: "matching snippet here",
		Source:  "doc.md",
	}}}
	tool := &SearchTool{Searcher: stub}

	out, err := tool.Call(json.RawMessage(`{"queries":["myquery"]}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !strings.Contains(out, "## myquery") {
		t.Errorf("output missing `## myquery` heading: %q", out)
	}
	if !strings.Contains(out, "Result Title") {
		t.Errorf("output missing result title: %q", out)
	}
	if !strings.Contains(out, "matching snippet here") {
		t.Errorf("output missing snippet: %q", out)
	}
	if len(stub.calls) != 1 || len(stub.calls[0]) != 1 || stub.calls[0][0] != "myquery" {
		t.Errorf("stub invoked with %v, want [[myquery]]", stub.calls)
	}
}

func TestSearchTool_Call_NoResults(t *testing.T) {
	tool := &SearchTool{Searcher: &stubSearcher{results: nil}}

	out, err := tool.Call(json.RawMessage(`{"queries":["nothing"]}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !strings.Contains(out, "No results") {
		t.Errorf("output missing `No results` marker: %q", out)
	}
	if !strings.Contains(out, "## nothing") {
		t.Errorf("output missing query heading: %q", out)
	}
}

func TestSearchTool_Call_MultiQuery(t *testing.T) {
	tool := &SearchTool{Searcher: &stubSearcher{results: nil}}
	out, err := tool.Call(json.RawMessage(`{"queries":["alpha","beta"]}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	a := strings.Index(out, "## alpha")
	b := strings.Index(out, "## beta")
	if a < 0 || b < 0 {
		t.Fatalf("missing headings: %q", out)
	}
	if a >= b {
		t.Errorf("expected ## alpha before ## beta, got positions %d/%d in %q", a, b, out)
	}
}
