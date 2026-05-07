package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/analytics"
	"github.com/dreamware-nz/memex/internal/mcp"
)

func TestStatsTool_ImplementsToolInterface(t *testing.T) {
	var _ mcp.Tool = (*StatsTool)(nil)
}

func TestStatsTool_Name(t *testing.T) {
	if got := (&StatsTool{}).Name(); got != "memex_stats" {
		t.Errorf("Name() = %q, want memex_stats", got)
	}
}

type fakeStatsReader struct {
	rows []analytics.ToolRow
	err  error
}

func (f *fakeStatsReader) All() ([]analytics.ToolRow, error) { return f.rows, f.err }
func (f *fakeStatsReader) Close() error                      { return nil }

func TestStatsTool_Call_NoFile(t *testing.T) {
	dir := t.TempDir()
	tool := &StatsTool{
		AnalyticsPath: filepath.Join(dir, "missing.db"),
		Open: func(string) (StatsReader, error) {
			t.Fatal("Open should not be called when file is missing")
			return nil, nil
		},
	}
	out, err := tool.Call(nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !strings.Contains(out, "No stats yet") {
		t.Errorf("expected 'No stats yet', got: %q", out)
	}
}

func TestStatsTool_Call_EmptyPath(t *testing.T) {
	tool := &StatsTool{}
	out, err := tool.Call(nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !strings.Contains(out, "No stats yet") {
		t.Errorf("expected 'No stats yet' for empty path, got: %q", out)
	}
}

func TestStatsTool_Call_WithRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "analytics.db")
	if err := writeStubFile(path); err != nil {
		t.Fatal(err)
	}

	rows := []analytics.ToolRow{
		{Tool: "memex_search", Calls: 4, TokensSaved: 12000, USDSaved: 0.42},
		{Tool: "memex_index", Calls: 2, TokensSaved: 3000, USDSaved: 0.10},
	}
	tool := &StatsTool{
		AnalyticsPath: path,
		Open: func(p string) (StatsReader, error) {
			if p != path {
				t.Errorf("Open got %q, want %q", p, path)
			}
			return &fakeStatsReader{rows: rows}, nil
		},
	}
	out, err := tool.Call(nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	for _, want := range []string{
		"Tool", "Calls", "Tokens Saved", "USD Saved",
		"memex_search", "memex_index",
		"TOTAL", "15000", "0.5200",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--full--\n%s", want, out)
		}
	}
}

func TestStatsTool_Call_OpenError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "analytics.db")
	if err := writeStubFile(path); err != nil {
		t.Fatal(err)
	}
	tool := &StatsTool{
		AnalyticsPath: path,
		Open: func(string) (StatsReader, error) {
			return nil, errors.New("open boom")
		},
	}
	if _, err := tool.Call(nil); err == nil || !strings.Contains(err.Error(), "open boom") {
		t.Errorf("expected open error to surface, got %v", err)
	}
}

func TestStatsTool_Call_AllError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "analytics.db")
	if err := writeStubFile(path); err != nil {
		t.Fatal(err)
	}
	tool := &StatsTool{
		AnalyticsPath: path,
		Open: func(string) (StatsReader, error) {
			return &fakeStatsReader{err: errors.New("query boom")}, nil
		},
	}
	if _, err := tool.Call(nil); err == nil || !strings.Contains(err.Error(), "query boom") {
		t.Errorf("expected query error to surface, got %v", err)
	}
}

func writeStubFile(path string) error {
	return os.WriteFile(path, []byte("stub"), 0o644)
}
