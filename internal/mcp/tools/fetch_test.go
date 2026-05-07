package tools

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/mcp"
)

type stubFetcher struct {
	body        []byte
	contentType string
	err         error
	gotURL      string
}

func (s *stubFetcher) Get(u string) ([]byte, string, error) {
	s.gotURL = u
	return s.body, s.contentType, s.err
}

type stubConverter struct {
	out    string
	err    error
	called int
	gotIn  string
}

func (s *stubConverter) Convert(html string) (string, error) {
	s.called++
	s.gotIn = html
	return s.out, s.err
}

func TestFetchTool_ImplementsToolInterface(t *testing.T) {
	var _ mcp.Tool = (*FetchTool)(nil)
}

func TestFetchTool_Schema(t *testing.T) {
	tool := &FetchTool{}
	var schema map[string]any
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("required missing or wrong shape: %v", schema)
	}
	hasURL, hasLabel := false, false
	for _, r := range required {
		s, _ := r.(string)
		if s == "url" {
			hasURL = true
		}
		if s == "label" {
			hasLabel = true
		}
	}
	if !hasURL {
		t.Errorf("required must contain url: %v", required)
	}
	if hasLabel {
		t.Errorf("required must NOT contain label: %v", required)
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing: %v", schema)
	}
	for _, k := range []string{"url", "label"} {
		field, ok := props[k].(map[string]any)
		if !ok {
			t.Fatalf("properties.%s missing: %v", k, props)
		}
		if field["type"] != "string" {
			t.Errorf("properties.%s.type = %v, want string", k, field["type"])
		}
	}
}

func TestFetchTool_MissingURL(t *testing.T) {
	cases := []struct {
		name string
		args json.RawMessage
	}{
		{"empty url", json.RawMessage(`{"url":""}`)},
		{"no fields", json.RawMessage(`{}`)},
		{"nil args", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tool := &FetchTool{Fetcher: &stubFetcher{}, Converter: &stubConverter{}, Indexer: &stubIndexer{}}
			_, err := tool.Call(tc.args)
			if err == nil {
				t.Fatal("expected error, got nil")
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

func TestFetchTool_FetchError(t *testing.T) {
	fetcher := &stubFetcher{err: errors.New("network down")}
	tool := &FetchTool{Fetcher: fetcher, Converter: &stubConverter{}, Indexer: &stubIndexer{}}
	_, err := tool.Call(json.RawMessage(`{"url":"https://example.com"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "network down") {
		t.Errorf("error missing fetch cause: %v", err)
	}
	var rpcErr *RPCError
	if errors.As(err, &rpcErr) {
		t.Errorf("fetch error must NOT be an RPCError (it surfaces as isError, not protocol error): %v", rpcErr)
	}
}

func TestFetchTool_HTMLConverted(t *testing.T) {
	fetcher := &stubFetcher{body: []byte("<h1>Hello</h1>"), contentType: "text/html; charset=utf-8"}
	conv := &stubConverter{out: "# Hello"}
	idx := &stubIndexer{}
	tool := &FetchTool{Fetcher: fetcher, Converter: conv, Indexer: idx}

	out, err := tool.Call(json.RawMessage(`{"url":"https://example.com/page","label":"page"}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if conv.called != 1 {
		t.Errorf("converter called %d times, want 1", conv.called)
	}
	if conv.gotIn != "<h1>Hello</h1>" {
		t.Errorf("converter input = %q, want raw HTML", conv.gotIn)
	}
	if len(idx.calls) != 1 || idx.calls[0].Content != "# Hello" {
		t.Fatalf("indexer calls = %+v, want one call with markdown", idx.calls)
	}
	if !strings.Contains(out, "Fetched and indexed page") {
		t.Errorf("confirmation missing label: %q", out)
	}
}

func TestFetchTool_PlainTextPassthrough(t *testing.T) {
	fetcher := &stubFetcher{body: []byte("just text"), contentType: "text/plain"}
	conv := &stubConverter{}
	idx := &stubIndexer{}
	tool := &FetchTool{Fetcher: fetcher, Converter: conv, Indexer: idx}

	if _, err := tool.Call(json.RawMessage(`{"url":"https://example.com/raw","label":"raw"}`)); err != nil {
		t.Fatalf("call: %v", err)
	}
	if conv.called != 0 {
		t.Errorf("converter must not be called for text/plain, got %d", conv.called)
	}
	if len(idx.calls) != 1 || idx.calls[0].Content != "just text" {
		t.Errorf("indexer received %+v, want raw text", idx.calls)
	}
}

func TestFetchTool_LabelDefault(t *testing.T) {
	fetcher := &stubFetcher{body: []byte("hi"), contentType: "text/plain"}
	idx := &stubIndexer{}
	tool := &FetchTool{Fetcher: fetcher, Converter: &stubConverter{}, Indexer: idx}

	if _, err := tool.Call(json.RawMessage(`{"url":"https://example.com/docs"}`)); err != nil {
		t.Fatalf("call: %v", err)
	}
	if len(idx.calls) != 1 {
		t.Fatalf("indexer calls = %d, want 1", len(idx.calls))
	}
	if got := idx.calls[0].Label; got != "example.com/docs" {
		t.Errorf("default label = %q, want %q", got, "example.com/docs")
	}
}
