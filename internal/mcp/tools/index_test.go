package tools

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/kb"
	"github.com/dreamware-nz/memex/internal/mcp"
)

type stubIndexer struct {
	err   error
	calls []struct{ Label, Content string }
}

func (s *stubIndexer) Index(label, content string) error {
	s.calls = append(s.calls, struct{ Label, Content string }{label, content})
	return s.err
}

type stubMarkdownIndexer struct {
	err   error
	calls []struct{ Label, Content string }
}

func (s *stubMarkdownIndexer) IndexMarkdown(label, content string) (kb.IndexResult, error) {
	s.calls = append(s.calls, struct{ Label, Content string }{label, content})
	if s.err != nil {
		return kb.IndexResult{}, s.err
	}
	return kb.IndexResult{
		SourceID:    1,
		TotalChunks: 3,
		CodeChunks:  1,
		Label:       label,
	}, nil
}

func TestIndexTool_Schema(t *testing.T) {
	tool := &IndexTool{}
	var schema map[string]any
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("required missing or wrong shape: %v", schema)
	}
	want := map[string]bool{"label": false, "content": false}
	for _, r := range required {
		if s, ok := r.(string); ok {
			if _, exists := want[s]; exists {
				want[s] = true
			}
		}
	}
	for k, found := range want {
		if !found {
			t.Errorf("required missing %q: %v", k, required)
		}
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing or wrong shape: %v", schema)
	}
	for _, k := range []string{"label", "content"} {
		field, ok := props[k].(map[string]any)
		if !ok {
			t.Fatalf("properties.%s missing: %v", k, props)
		}
		if field["type"] != "string" {
			t.Errorf("properties.%s.type = %v, want string", k, field["type"])
		}
	}
}

func TestIndexTool_ImplementsToolInterface(t *testing.T) {
	var _ mcp.Tool = (*IndexTool)(nil)
}

func TestIndexTool_Call_MissingArgs(t *testing.T) {
	cases := []struct {
		name string
		args json.RawMessage
	}{
		{"missing content", json.RawMessage(`{"label":"foo"}`)},
		{"missing label", json.RawMessage(`{"content":"bar"}`)},
		{"both empty", json.RawMessage(`{"label":"","content":""}`)},
		{"nil args", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tool := &IndexTool{Indexer: &stubMarkdownIndexer{}}
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

func TestIndexTool_Call_Success(t *testing.T) {
	stub := &stubMarkdownIndexer{}
	tool := &IndexTool{Indexer: stub}

	out, err := tool.Call(json.RawMessage(`{"label":"docs","content":"hello world"}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !strings.Contains(out, "Indexed") {
		t.Errorf("output missing `Indexed` marker: %q", out)
	}
	if !strings.Contains(out, "docs") {
		t.Errorf("output missing label: %q", out)
	}
	if !strings.Contains(out, "memex_search") {
		t.Errorf("output missing memex_search hint: %q", out)
	}
	if !strings.Contains(out, "sections") {
		t.Errorf("output missing 'sections' word: %q", out)
	}
	if len(stub.calls) != 1 {
		t.Fatalf("indexer invoked %d times, want 1", len(stub.calls))
	}
	if stub.calls[0].Label != "docs" || stub.calls[0].Content != "hello world" {
		t.Errorf("indexer got (%q, %q), want (docs, hello world)", stub.calls[0].Label, stub.calls[0].Content)
	}
}

func TestIndexTool_Call_IndexerError(t *testing.T) {
	stub := &stubMarkdownIndexer{err: errors.New("disk full")}
	tool := &IndexTool{Indexer: stub}

	_, err := tool.Call(json.RawMessage(`{"label":"docs","content":"hi"}`))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Errorf("error %q does not contain underlying message", err.Error())
	}
}
