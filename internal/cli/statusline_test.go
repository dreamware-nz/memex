package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runStatusline(t *testing.T, dataDir string) string {
	t.Helper()
	t.Setenv("MEMEX_DATA_DIR", dataDir)
	cmd := NewStatuslineCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return buf.String()
}

func TestStatuslineCmd_NoStatsFile(t *testing.T) {
	out := runStatusline(t, t.TempDir())
	if out != "" {
		t.Fatalf("expected empty stdout, got %q", out)
	}
}

func TestStatuslineCmd_RendersFromStatsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memex", "session", "stats.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `{"schemaVersion":1,"dollars_saved_session":0.42,"dollars_saved_lifetime":3.14,"pct_efficient":87}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write stats: %v", err)
	}

	out := runStatusline(t, dir)
	for _, want := range []string{
		"$0.42 saved this session",
		"$3.14 saved across sessions",
		"87% efficient",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("expected trailing newline, got %q", out)
	}
}

func TestStatuslineCmd_GarbageJSONExitsCleanly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memex", "session", "stats.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	out := runStatusline(t, dir)
	if out != "" {
		t.Fatalf("expected empty stdout on bad JSON, got %q", out)
	}
}
