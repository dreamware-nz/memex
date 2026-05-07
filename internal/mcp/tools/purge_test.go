package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/mcp"
)

func TestPurgeTool_ImplementsToolInterface(t *testing.T) {
	var _ mcp.Tool = (*PurgeTool)(nil)
}

func TestPurgeTool_Name(t *testing.T) {
	if got := (&PurgeTool{}).Name(); got != "memex_purge" {
		t.Errorf("Name() = %q, want memex_purge", got)
	}
}

func TestPurgeTool_Call_NoConfirmDoesNotDelete(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "kb.sqlite")
	b := filepath.Join(dir, "analytics.db")
	if err := os.WriteFile(a, []byte("kb"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &PurgeTool{Paths: []string{a, b}}
	out, err := tool.Call(nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !strings.Contains(out, "requires confirm=true") {
		t.Errorf("dry-run output missing confirm warning: %q", out)
	}
	if _, err := os.Stat(a); err != nil {
		t.Errorf("file %s should still exist after dry-run, got %v", a, err)
	}
	if _, err := os.Stat(b); err != nil {
		t.Errorf("file %s should still exist after dry-run, got %v", b, err)
	}
}

func TestPurgeTool_Call_ConfirmTrueDeletes(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "kb.sqlite")
	b := filepath.Join(dir, "analytics.db")
	if err := os.WriteFile(a, []byte("kb"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &PurgeTool{Paths: []string{a, b}}
	out, err := tool.Call(json.RawMessage(`{"confirm":true}`))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !strings.Contains(out, "Deleted:") {
		t.Errorf("confirmed-purge output missing 'Deleted:' header: %q", out)
	}
	for _, p := range []string{a, b} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("file %s should be gone, got err=%v", p, err)
		}
	}
}

func TestPurgeTool_Call_AllAbsent(t *testing.T) {
	dir := t.TempDir()
	tool := &PurgeTool{Paths: []string{
		filepath.Join(dir, "missing-a"),
		filepath.Join(dir, "missing-b"),
	}}
	out, err := tool.Call(json.RawMessage(`{"confirm":true}`))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !strings.Contains(out, "Nothing to purge") {
		t.Errorf("expected 'Nothing to purge' message, got: %q", out)
	}
}

func TestPurgeTool_Call_PartiallyMissing(t *testing.T) {
	dir := t.TempDir()
	present := filepath.Join(dir, "present")
	missing := filepath.Join(dir, "missing")
	if err := os.WriteFile(present, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := &PurgeTool{Paths: []string{present, missing}}
	out, err := tool.Call(json.RawMessage(`{"confirm":true}`))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !strings.Contains(out, "Deleted:") || !strings.Contains(out, present) {
		t.Errorf("expected Deleted block listing %s, got: %q", present, out)
	}
	if !strings.Contains(out, "Already absent:") || !strings.Contains(out, missing) {
		t.Errorf("expected Already-absent block listing %s, got: %q", missing, out)
	}
}

func TestPurgeTool_Call_BadJSON(t *testing.T) {
	tool := &PurgeTool{Paths: nil}
	if _, err := tool.Call(json.RawMessage(`{not json`)); err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}
