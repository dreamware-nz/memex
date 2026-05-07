package tools

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/kb"
	"github.com/dreamware-nz/memex/internal/mcp"
)

type stubFileExecutor struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
	calls    []struct{ Path, Language string }
}

func (s *stubFileExecutor) ExecuteFile(path, language string) (string, string, int, error) {
	s.calls = append(s.calls, struct{ Path, Language string }{path, language})
	return s.stdout, s.stderr, s.exitCode, s.err
}

func TestExecuteFileTool_ImplementsToolInterface(t *testing.T) {
	var _ mcp.Tool = (*ExecuteFileTool)(nil)
}

func TestExecuteFileTool_Schema(t *testing.T) {
	tool := &ExecuteFileTool{}
	var schema map[string]any
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("required missing or wrong shape: %v", schema)
	}
	gotRequired := map[string]bool{}
	for _, r := range required {
		if s, ok := r.(string); ok {
			gotRequired[s] = true
		}
	}
	if !gotRequired["path"] {
		t.Errorf("required missing 'path': %v", required)
	}
	if gotRequired["language"] || gotRequired["code"] || gotRequired["intent"] {
		t.Errorf("required must contain only 'path': %v", required)
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing or wrong shape: %v", schema)
	}
	for _, k := range []string{"path", "language", "code", "intent"} {
		field, ok := props[k].(map[string]any)
		if !ok {
			t.Fatalf("properties.%s missing: %v", k, props)
		}
		if field["type"] != "string" {
			t.Errorf("properties.%s.type = %v, want string", k, field["type"])
		}
	}
}

func TestExecuteFileTool_MissingPath(t *testing.T) {
	tool := &ExecuteFileTool{Executor: &stubFileExecutor{}, Store: newStubStore()}
	_, err := tool.Call(json.RawMessage(`{}`))
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

func TestExecuteFileTool_WithCodeArg(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.py")
	exec := &stubFileExecutor{stdout: "ok", exitCode: 0}
	tool := &ExecuteFileTool{Executor: exec, Store: newStubStore()}

	args, _ := json.Marshal(map[string]string{
		"path":     path,
		"language": "python",
		"code":     "print('hi')",
	})
	if _, err := tool.Call(args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(contents) != "print('hi')" {
		t.Errorf("file contents = %q, want %q", string(contents), "print('hi')")
	}
	if len(exec.calls) != 1 {
		t.Fatalf("executor called %d times, want 1", len(exec.calls))
	}
	if exec.calls[0].Path != path || exec.calls[0].Language != "python" {
		t.Errorf("executor call = %+v, want path=%q lang=python", exec.calls[0], path)
	}
}

func TestExecuteFileTool_LanguageInferred(t *testing.T) {
	cases := []struct {
		path     string
		wantLang string
	}{
		{"script.py", "python"},
		{"script.js", "javascript"},
		{"script.sh", "shell"},
		{"weird.unknown", "shell"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			exec := &stubFileExecutor{stdout: "ok"}
			tool := &ExecuteFileTool{Executor: exec, Store: newStubStore()}
			args, _ := json.Marshal(map[string]string{"path": tc.path})
			if _, err := tool.Call(args); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(exec.calls) != 1 {
				t.Fatalf("executor called %d times, want 1", len(exec.calls))
			}
			if exec.calls[0].Language != tc.wantLang {
				t.Errorf("language = %q, want %q", exec.calls[0].Language, tc.wantLang)
			}
		})
	}
}

func TestExecuteFileTool_IntentMatchesPreviewResponse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.py")
	stdout := strings.Repeat("noise\n", 1200) + "timeout occurred\n"
	store := newStubStore()
	store.results["timeout"] = []kb.SearchResult{
		{Heading: "timeout block", Body: "timeout occurred"},
	}
	store.terms[1] = []string{"timeout", "noise"}
	exec := &stubFileExecutor{stdout: stdout, exitCode: 0}
	tool := &ExecuteFileTool{Executor: exec, Store: store}

	args, _ := json.Marshal(map[string]string{"path": path, "intent": "timeout"})
	out, err := tool.Call(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "1 sections matched \"timeout\"") {
		t.Errorf("missing matched header: %q", out)
	}
	if len(store.calls) != 1 || store.calls[0].Source != "file:"+path {
		t.Errorf("expected source 'file:%s', got %+v", path, store.calls)
	}
}

func TestExecuteFileTool_NoIntentLargeOutputAutoIndexes(t *testing.T) {
	stdout := strings.Repeat("xxxxxxxxxx", 12000)
	store := newStubStore()
	exec := &stubFileExecutor{stdout: stdout, exitCode: 0}
	tool := &ExecuteFileTool{Executor: exec, Store: store}

	args, _ := json.Marshal(map[string]string{"path": "/tmp/big.py"})
	out, err := tool.Call(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Indexed") {
		t.Errorf("output missing 'Indexed' marker")
	}
	if !strings.Contains(out, "memex_search(queries") {
		t.Errorf("missing memex_search hint: %q", out[:200])
	}
	if len(store.calls) != 1 {
		t.Fatalf("indexer called %d times, want 1", len(store.calls))
	}
	if !strings.HasPrefix(store.calls[0].Source, "file:") {
		t.Errorf("indexer source = %q, want file: prefix", store.calls[0].Source)
	}
}

func TestExecuteFileTool_NoIntentMediumOutputInline(t *testing.T) {
	stdout := strings.Repeat("medium line\n", 4000)
	store := newStubStore()
	exec := &stubFileExecutor{stdout: stdout, exitCode: 0}
	tool := &ExecuteFileTool{Executor: exec, Store: store}

	args, _ := json.Marshal(map[string]string{"path": "/tmp/med.py"})
	out, err := tool.Call(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != stdout {
		t.Errorf("medium output should be inline; got len=%d want len=%d", len(out), len(stdout))
	}
	if len(store.calls) != 0 {
		t.Errorf("medium output should not auto-index, got %d calls", len(store.calls))
	}
}

func TestExecuteFileTool_NonZeroExitLargeOutputAutoSearch(t *testing.T) {
	stdout := strings.Repeat("trace_long_filler_line_here_xxxx\n", 4000) + "Exception: boom\n"
	store := newStubStore()
	store.results["errors failures exceptions"] = []kb.SearchResult{
		{Heading: "exception block", Body: "Exception: boom"},
	}
	store.terms[1] = []string{"exception", "trace"}
	exec := &stubFileExecutor{stdout: stdout, exitCode: 9}
	tool := &ExecuteFileTool{Executor: exec, Store: store}

	args, _ := json.Marshal(map[string]string{"path": "/tmp/err.py"})
	_, err := tool.Call(args)
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
	if len(store.calls) != 1 || !strings.HasSuffix(store.calls[0].Source, ":error") {
		t.Errorf("expected source suffix ':error', got %+v", store.calls)
	}
}
