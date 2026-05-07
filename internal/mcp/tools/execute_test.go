package tools

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/kb"
	"github.com/dreamware-nz/memex/internal/mcp"
)

type stubExecutor struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
	calls    []struct{ Language, Code string }
}

func (s *stubExecutor) Execute(language, code string) (string, string, int, error) {
	s.calls = append(s.calls, struct{ Language, Code string }{language, code})
	return s.stdout, s.stderr, s.exitCode, s.err
}

func TestExecuteTool_ImplementsToolInterface(t *testing.T) {
	var _ mcp.Tool = (*ExecuteTool)(nil)
}

func TestExecuteTool_Schema(t *testing.T) {
	tool := &ExecuteTool{}
	var schema map[string]any
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("required missing or wrong shape: %v", schema)
	}
	want := map[string]bool{"language": false, "code": false}
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
	for _, r := range required {
		if r == "intent" {
			t.Errorf("intent must NOT be in required: %v", required)
		}
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing or wrong shape: %v", schema)
	}
	for _, k := range []string{"language", "code", "intent"} {
		field, ok := props[k].(map[string]any)
		if !ok {
			t.Fatalf("properties.%s missing: %v", k, props)
		}
		if field["type"] != "string" {
			t.Errorf("properties.%s.type = %v, want string", k, field["type"])
		}
	}
}

func TestExecuteTool_Call_MissingArgs(t *testing.T) {
	cases := []struct {
		name string
		args json.RawMessage
	}{
		{"missing code", json.RawMessage(`{"language":"shell"}`)},
		{"missing language", json.RawMessage(`{"code":"echo hi"}`)},
		{"both empty", json.RawMessage(`{"language":"","code":""}`)},
		{"nil args", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tool := &ExecuteTool{Executor: &stubExecutor{}, Store: newStubStore()}
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

func TestExecuteTool_SmallOutput(t *testing.T) {
	stdout := strings.Repeat("a", 100)
	exec := &stubExecutor{stdout: stdout, exitCode: 0}
	store := newStubStore()
	tool := &ExecuteTool{Executor: exec, Store: store}

	out, err := tool.Call(json.RawMessage(`{"language":"shell","code":"true"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != stdout {
		t.Errorf("output = %q, want full stdout", out)
	}
	if len(store.calls) != 0 {
		t.Errorf("indexer called %d times, want 0", len(store.calls))
	}
}

func TestExecuteTool_IntentMatchesPreviewResponse(t *testing.T) {
	stdout := strings.Repeat("filler line\n", 600) + "failing tests inside\n"
	store := newStubStore()
	store.results["failing tests"] = []kb.SearchResult{
		{Heading: "section A", Body: "failing tests inside\nmore", Source: "execute:shell"},
	}
	store.terms[1] = []string{"failing", "tests", "filler"}
	exec := &stubExecutor{stdout: stdout, exitCode: 0}
	tool := &ExecuteTool{Executor: exec, Store: store}

	out, err := tool.Call(json.RawMessage(`{"language":"shell","code":"x","intent":"failing tests"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "1 sections matched \"failing tests\"") {
		t.Errorf("missing matched header: %q", out)
	}
	if !strings.Contains(out, "Searchable terms:") {
		t.Errorf("missing distinctive terms line: %q", out)
	}
	if !strings.Contains(out, "Use memex_search(queries:") {
		t.Errorf("missing memex_search hint: %q", out)
	}
	if len(store.calls) != 1 || store.calls[0].Source != "execute:shell" {
		t.Errorf("expected 1 call to execute:shell, got %+v", store.calls)
	}
}

func TestExecuteTool_IntentNoMatchesResponse(t *testing.T) {
	stdout := strings.Repeat("uneventful line\n", 600)
	store := newStubStore()
	store.terms[1] = []string{"uneventful", "line"}
	exec := &stubExecutor{stdout: stdout, exitCode: 0}
	tool := &ExecuteTool{Executor: exec, Store: store}

	out, err := tool.Call(json.RawMessage(`{"language":"shell","code":"x","intent":"meltdown"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Indexed") {
		t.Errorf("missing 'Indexed' marker: %q", out)
	}
	if !strings.Contains(out, "No sections matched intent \"meltdown\"") {
		t.Errorf("missing no-match line: %q", out)
	}
	if !strings.Contains(out, "Searchable terms:") {
		t.Errorf("missing distinctive terms line: %q", out)
	}
}

func TestExecuteTool_IntentBelowThresholdReturnsRaw(t *testing.T) {
	stdout := "small\n"
	store := newStubStore()
	exec := &stubExecutor{stdout: stdout, exitCode: 0}
	tool := &ExecuteTool{Executor: exec, Store: store}

	out, err := tool.Call(json.RawMessage(`{"language":"shell","code":"x","intent":"anything"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != stdout {
		t.Errorf("output = %q, want raw %q", out, stdout)
	}
	if len(store.calls) != 0 {
		t.Errorf("indexer called %d times, want 0", len(store.calls))
	}
}

func TestExecuteTool_NoIntentLargeOutputAutoIndexes(t *testing.T) {
	stdout := strings.Repeat("xxxxxxxxxx", 12000) // ~120KB
	store := newStubStore()
	exec := &stubExecutor{stdout: stdout, exitCode: 0}
	tool := &ExecuteTool{Executor: exec, Store: store}

	out, err := tool.Call(json.RawMessage(`{"language":"python","code":"x"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Indexed") {
		t.Errorf("missing 'Indexed' marker: %q", out[:200])
	}
	if !strings.Contains(out, "execute:python") {
		t.Errorf("missing source label: %q", out[:200])
	}
	if !strings.Contains(out, "memex_search(queries") {
		t.Errorf("missing memex_search hint: %q", out[:200])
	}
	if len(store.calls) != 1 {
		t.Fatalf("indexer called %d times, want 1", len(store.calls))
	}
}

func TestExecuteTool_NoIntentMediumOutputInline(t *testing.T) {
	stdout := strings.Repeat("medium line\n", 4000) // ~50KB
	store := newStubStore()
	exec := &stubExecutor{stdout: stdout, exitCode: 0}
	tool := &ExecuteTool{Executor: exec, Store: store}

	out, err := tool.Call(json.RawMessage(`{"language":"shell","code":"x"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != stdout {
		t.Errorf("medium output should be inline; len(out)=%d, len(stdout)=%d", len(out), len(stdout))
	}
	if len(store.calls) != 0 {
		t.Errorf("medium output should not auto-index, got %d calls", len(store.calls))
	}
}

func TestExecuteTool_NonZeroExit(t *testing.T) {
	stdout := "boom\n"
	exec := &stubExecutor{stdout: stdout, exitCode: 1}
	tool := &ExecuteTool{Executor: exec, Store: newStubStore()}

	_, err := tool.Call(json.RawMessage(`{"language":"shell","code":"false"}`))
	if err == nil {
		t.Fatalf("expected error to signal non-zero exit, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error message = %q, want it to contain stdout", err.Error())
	}
}

func TestExecuteTool_NonZeroExitLargeOutputAutoSearch(t *testing.T) {
	stdout := strings.Repeat("trace line\n", 12000) + "Exception: boom\n"
	store := newStubStore()
	store.results["errors failures exceptions"] = []kb.SearchResult{
		{Heading: "exception block", Body: "Exception: boom", Source: "execute:shell:error"},
	}
	store.terms[1] = []string{"exception", "boom", "trace"}
	exec := &stubExecutor{stdout: stdout, exitCode: 7}
	tool := &ExecuteTool{Executor: exec, Store: store}

	_, err := tool.Call(json.RawMessage(`{"language":"shell","code":"x"}`))
	if err == nil {
		t.Fatalf("expected non-zero-exit error, got nil")
	}
	var nz *execNonZero
	if !errors.As(err, &nz) {
		t.Fatalf("error is not *execNonZero: %T", err)
	}
	if !strings.Contains(nz.output, "errors failures exceptions") {
		t.Errorf("missing canned-query marker: %q", nz.output)
	}
	if len(store.calls) != 1 || store.calls[0].Source != "execute:shell:error" {
		t.Errorf("expected 1 call to execute:shell:error, got %+v", store.calls)
	}
}

func TestExecuteTool_StderrAppended(t *testing.T) {
	exec := &stubExecutor{stdout: "ok\n", stderr: "warn!\n", exitCode: 0}
	tool := &ExecuteTool{Executor: exec, Store: newStubStore()}

	out, err := tool.Call(json.RawMessage(`{"language":"shell","code":"echo ok"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[stderr]") {
		t.Errorf("output missing [stderr] marker: %q", out)
	}
	if !strings.Contains(out, "warn!") {
		t.Errorf("output missing stderr content: %q", out)
	}
}

func TestExecuteTool_ExecutorError(t *testing.T) {
	exec := &stubExecutor{err: errors.New("sandbox unavailable")}
	tool := &ExecuteTool{Executor: exec, Store: newStubStore()}

	_, err := tool.Call(json.RawMessage(`{"language":"shell","code":"echo"}`))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "sandbox unavailable") {
		t.Errorf("error = %q, want underlying sandbox error", err.Error())
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
