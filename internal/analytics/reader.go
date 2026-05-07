// Package analytics provides a read-only view over the session analytics
// SQLite database that the session-analytics writer maintains. The Reader is
// intentionally minimal: it powers `memex stats` and similar inspection
// commands without ever mutating the underlying file.
package analytics

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// ToolRow is one aggregated per-tool savings row from the analytics database.
type ToolRow struct {
	Tool        string
	Calls       int64
	TokensSaved int64
	USDSaved    float64
}

// Reader holds an open handle to the analytics database.
type Reader struct {
	db *sql.DB
}

// Open opens the analytics SQLite database at path in read-only mode.
//
// Callers are responsible for verifying the file exists before calling Open;
// the read-only DSN will fail on a missing file rather than create an empty
// database, which keeps the "no stats yet" branch in `memex stats` correct.
func Open(path string) (*Reader, error) {
	dsn := "file:" + path + "?mode=ro&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("analytics: open %s: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("analytics: ping %s: %w", path, err)
	}
	return &Reader{db: db}, nil
}

// Close releases the underlying database handle.
func (r *Reader) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	err := r.db.Close()
	r.db = nil
	return err
}

// All returns every tool_stats row ordered by USD saved descending.
func (r *Reader) All() ([]ToolRow, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("analytics: All: nil reader")
	}
	const q = `SELECT tool, calls, tokens_saved, usd_saved
		FROM tool_stats
		ORDER BY usd_saved DESC, tool ASC`
	rows, err := r.db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("analytics: query tool_stats: %w", err)
	}
	defer rows.Close()

	var out []ToolRow
	for rows.Next() {
		var t ToolRow
		if err := rows.Scan(&t.Tool, &t.Calls, &t.TokensSaved, &t.USDSaved); err != nil {
			return nil, fmt.Errorf("analytics: scan tool_stats: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("analytics: iterate tool_stats: %w", err)
	}
	return out, nil
}
