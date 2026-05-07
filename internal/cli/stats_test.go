package cli

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func runStats(t *testing.T, dataDir string) string {
	t.Helper()
	t.Setenv("MEMEX_DATA_DIR", dataDir)
	cmd := NewStatsCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return buf.String()
}

func TestStatsCmd_NoAnalyticsFile(t *testing.T) {
	out := runStats(t, t.TempDir())
	if !strings.Contains(out, "No stats yet") {
		t.Fatalf("output missing 'No stats yet':\n%s", out)
	}
}

func TestStatsCmd_RendersRowsAndTotal(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "memex", "session", "analytics.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open seed: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE tool_stats (
		tool         TEXT    PRIMARY KEY,
		calls        INTEGER NOT NULL DEFAULT 0,
		tokens_saved INTEGER NOT NULL DEFAULT 0,
		usd_saved    REAL    NOT NULL DEFAULT 0
	)`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO tool_stats(tool, calls, tokens_saved, usd_saved) VALUES (?, ?, ?, ?)`,
		"memex_search", int64(3), int64(1234), 0.1700,
	); err != nil {
		t.Fatalf("seed insert: %v", err)
	}
	_ = db.Close()

	out := runStats(t, dir)
	if !strings.Contains(out, "memex_search") {
		t.Fatalf("output missing tool name:\n%s", out)
	}
	if !strings.Contains(out, "TOTAL") {
		t.Fatalf("output missing TOTAL row:\n%s", out)
	}
	if !strings.Contains(out, "0.1700") {
		t.Fatalf("output missing USD value:\n%s", out)
	}
}
