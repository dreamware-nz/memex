package session

import (
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestDB(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"TestOpen_CreatesParentDirsAndFile", testOpenCreatesParentDirsAndFile},
		{"TestOpen_IsIdempotent", testOpenIsIdempotent},
		{"TestOpen_SetsPragmas", testOpenSetsPragmas},
		{"TestClose_InvalidatesUnderlyingHandle", testCloseInvalidatesUnderlyingHandle},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func testOpenCreatesParentDirsAndFile(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "deep", "session.sqlite")

	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected db file at %s: %v", dbPath, err)
	}
}

func testOpenIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "session.sqlite")

	d1, err := Open(dbPath)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := d1.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	d2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	t.Cleanup(func() { _ = d2.Close() })

	var rows int
	if err := d2.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=1`).Scan(&rows); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if rows != 1 {
		t.Fatalf("schema_migrations rows for v1 = %d, want 1", rows)
	}
}

func testOpenSetsPragmas(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "session.sqlite")

	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	var journal string
	if err := d.db.QueryRow(`PRAGMA journal_mode`).Scan(&journal); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if journal != "wal" {
		t.Fatalf("journal_mode = %q, want %q", journal, "wal")
	}

	var busy int
	if err := d.db.QueryRow(`PRAGMA busy_timeout`).Scan(&busy); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if busy != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", busy)
	}
}

func testCloseInvalidatesUnderlyingHandle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "session.sqlite")

	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	handle := d.db
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if d.db != nil {
		t.Fatalf("internal db field not nilled out after Close")
	}

	if _, err := handle.Exec(`SELECT 1`); err == nil {
		t.Fatalf("expected error on closed handle, got nil")
	}
}
