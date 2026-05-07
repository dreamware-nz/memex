package analytics

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// seedAnalyticsDB writes the schema and given rows to a fresh SQLite file at
// path. It exists so reader_test stays the single source of truth for the
// shape `Reader.All` expects to find on disk.
func seedAnalyticsDB(t *testing.T, path string, rows []ToolRow) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("seed open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE tool_stats (
		tool         TEXT    PRIMARY KEY,
		calls        INTEGER NOT NULL DEFAULT 0,
		tokens_saved INTEGER NOT NULL DEFAULT 0,
		usd_saved    REAL    NOT NULL DEFAULT 0
	)`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	for _, r := range rows {
		if _, err := db.Exec(
			`INSERT INTO tool_stats(tool, calls, tokens_saved, usd_saved) VALUES (?, ?, ?, ?)`,
			r.Tool, r.Calls, r.TokensSaved, r.USDSaved,
		); err != nil {
			t.Fatalf("seed insert %s: %v", r.Tool, err)
		}
	}
}

func TestReader_All_OrdersByUSDDesc(t *testing.T) {
	path := filepath.Join(t.TempDir(), "analytics.db")
	seedAnalyticsDB(t, path, []ToolRow{
		{Tool: "low", Calls: 2, TokensSaved: 100, USDSaved: 0.05},
		{Tool: "high", Calls: 7, TokensSaved: 999, USDSaved: 0.42},
	})

	r, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer r.Close()

	got, err := r.All()
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Tool != "high" || got[1].Tool != "low" {
		t.Fatalf("order = [%s, %s], want [high, low]", got[0].Tool, got[1].Tool)
	}
	if got[0].Calls != 7 || got[0].TokensSaved != 999 || got[0].USDSaved != 0.42 {
		t.Fatalf("high row = %+v", got[0])
	}
}

func TestOpen_MissingFile(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "does-not-exist.db"))
	if err == nil {
		t.Fatal("expected error opening missing file in read-only mode")
	}
}
