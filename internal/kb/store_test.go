package kb

import (
	"database/sql"
	"path/filepath"
	"sort"
	"testing"

	_ "modernc.org/sqlite"
)

func TestStore(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"TestOpen_CreatesParentDirs", testOpenCreatesParentDirs},
		{"TestOpen_AppliesV1Migration", testOpenAppliesV1Migration},
		{"TestOpen_IsIdempotent", testOpenIsIdempotent},
		{"TestOpen_SetsPragmas", testOpenSetsPragmas},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func testOpenCreatesParentDirs(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "deep", "kb.sqlite")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if _, err := openCheck(dbPath); err != nil {
		t.Fatalf("expected db file at %s: %v", dbPath, err)
	}
}

func testOpenAppliesV1Migration(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "kb.sqlite")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	want := []string{"schema_migrations", "sections", "sections_fts", "sources"}
	got := tableNames(t, s.db)
	if !equalStrings(got, want) {
		t.Fatalf("tables = %v, want superset of %v", got, want)
	}

	var version int
	if err := s.db.QueryRow(`SELECT version FROM schema_migrations WHERE version=1`).Scan(&version); err != nil {
		t.Fatalf("schema_migrations row missing: %v", err)
	}
	if version != 1 {
		t.Fatalf("version = %d, want 1", version)
	}
}

func testOpenIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "kb.sqlite")

	s1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	s2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })

	var rows int
	if err := s2.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=1`).Scan(&rows); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if rows != 1 {
		t.Fatalf("schema_migrations rows for v1 = %d, want 1", rows)
	}
}

func testOpenSetsPragmas(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "kb.sqlite")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	var journal string
	if err := s.db.QueryRow(`PRAGMA journal_mode`).Scan(&journal); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if journal != "wal" {
		t.Fatalf("journal_mode = %q, want %q", journal, "wal")
	}

	var busy int
	if err := s.db.QueryRow(`PRAGMA busy_timeout`).Scan(&busy); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if busy != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", busy)
	}
}

// 4.2 — guards against silent driver swap (no build tags; uses default driver).
func TestFTS5SmokeTest(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "fts5_smoke.sqlite"))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE VIRTUAL TABLE t USING fts5(x)`); err != nil {
		t.Fatalf("FTS5 unavailable: %v", err)
	}
}

func tableNames(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type IN ('table','view') AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		out = append(out, n)
	}
	return out
}

func equalStrings(have, want []string) bool {
	set := map[string]bool{}
	for _, s := range have {
		set[s] = true
	}
	missing := []string{}
	for _, w := range want {
		if !set[w] {
			missing = append(missing, w)
		}
	}
	sort.Strings(missing)
	return len(missing) == 0
}

func openCheck(path string) (bool, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return false, err
	}
	defer db.Close()
	return true, db.Ping()
}
