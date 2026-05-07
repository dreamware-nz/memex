package tools

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/kb"
	"github.com/dreamware-nz/memex/internal/mcp"
)

func TestBatchTool_ImplementsToolInterface(t *testing.T) {
	var _ mcp.Tool = (*BatchTool)(nil)
}

func TestBatchTool_Schema(t *testing.T) {
	tool := &BatchTool{}
	var schema map[string]any
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("required missing or wrong shape: %v", schema)
	}
	want := map[string]bool{"commands": false, "queries": false}
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
}

func TestBatchTool_EmptyCommands(t *testing.T) {
	tool := &BatchTool{
		Executor: &stubExecutor{},
		Indexer:  &stubIndexer{},
		Searcher: &stubSearcher{},
	}
	_, err := tool.Call(json.RawMessage(`{"commands":[],"queries":["q"]}`))
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
}

func TestBatchTool_EmptyQueries(t *testing.T) {
	tool := &BatchTool{
		Executor: &stubExecutor{},
		Indexer:  &stubIndexer{},
		Searcher: &stubSearcher{},
	}
	_, err := tool.Call(json.RawMessage(`{"commands":[{"label":"x","command":"true"}],"queries":[]}`))
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
}

func TestBatchTool_SingleCommand(t *testing.T) {
	exec := &stubExecutor{stdout: "hello"}
	tool := &BatchTool{
		Executor: exec,
		Indexer:  &stubIndexer{},
		Searcher: &stubSearcher{},
	}
	out, err := tool.Call(json.RawMessage(`{"commands":[{"label":"greet","command":"echo hello"}],"queries":["greet"]}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !strings.Contains(out, "## greet") {
		t.Errorf("output missing `## greet` heading: %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("output missing command stdout: %q", out)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("executor invoked %d times, want 1", len(exec.calls))
	}
	if exec.calls[0].Language != "shell" {
		t.Errorf("executor called with language %q, want shell", exec.calls[0].Language)
	}
	if exec.calls[0].Code != "echo hello" {
		t.Errorf("executor called with code %q, want echo hello", exec.calls[0].Code)
	}
}

func TestBatchTool_IndexCalled(t *testing.T) {
	exec := &stubExecutor{stdout: "out-a"}
	idx := &stubIndexer{}
	tool := &BatchTool{
		Executor: exec,
		Indexer:  idx,
		Searcher: &stubSearcher{},
	}
	_, err := tool.Call(json.RawMessage(`{"commands":[{"label":"a","command":"true"},{"label":"b","command":"true"}],"queries":["q"]}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if len(idx.calls) != 1 {
		t.Fatalf("indexer invoked %d times, want 1", len(idx.calls))
	}
	if idx.calls[0].Label != "batch:a,b" {
		t.Errorf("indexer label = %q, want batch:a,b", idx.calls[0].Label)
	}
	if !strings.Contains(idx.calls[0].Content, "## a") || !strings.Contains(idx.calls[0].Content, "## b") {
		t.Errorf("indexer content missing command headings: %q", idx.calls[0].Content)
	}
	if !strings.Contains(idx.calls[0].Content, "out-a") {
		t.Errorf("indexer content missing command stdout: %q", idx.calls[0].Content)
	}
}

func TestBatchTool_QueryResultsAppended(t *testing.T) {
	exec := &stubExecutor{stdout: "ran"}
	srch := &stubSearcher{results: []kb.SearchResult{{
		Heading: "Section Title",
		Snippet: "snippet body",
	}}}
	tool := &BatchTool{
		Executor: exec,
		Indexer:  &stubIndexer{},
		Searcher: srch,
	}
	out, err := tool.Call(json.RawMessage(`{"commands":[{"label":"job","command":"true"}],"queries":["myquery"]}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	cmdIdx := strings.Index(out, "## job")
	qIdx := strings.Index(out, "## myquery")
	if cmdIdx < 0 || qIdx < 0 {
		t.Fatalf("missing headings in output: %q", out)
	}
	if cmdIdx >= qIdx {
		t.Errorf("expected command block before query heading; got positions %d/%d", cmdIdx, qIdx)
	}
	if !strings.Contains(out, "Section Title") {
		t.Errorf("output missing search hit heading: %q", out)
	}
}

func TestBatchTool_OutputCap(t *testing.T) {
	exec := &stubExecutor{stdout: strings.Repeat("x", 90*1024)}
	tool := &BatchTool{
		Executor: exec,
		Indexer:  &stubIndexer{},
		Searcher: &stubSearcher{},
	}
	out, err := tool.Call(json.RawMessage(`{"commands":[{"label":"big","command":"true"}],"queries":["q"]}`))
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if len(out) > batchOutputCap {
		t.Errorf("output length = %d, want <= %d", len(out), batchOutputCap)
	}
	if !strings.Contains(out, "(output cap reached)") {
		t.Errorf("output missing cap marker: %q", truncate(out, 80))
	}
}
