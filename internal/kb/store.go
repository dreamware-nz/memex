package kb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db   *sql.DB
	path string
}

func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("kb: mkdir %s: %w", dir, err)
		}
	}

	dsn := path + "?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("kb: open %s: %w", path, err)
	}

	var hasFTS5 int
	row := db.QueryRow(
		`SELECT COUNT(1) FROM pragma_compile_options WHERE compile_options='ENABLE_FTS5'`,
	)
	if err := row.Scan(&hasFTS5); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("kb: probe compile options: %w", err)
	}
	if hasFTS5 == 0 {
		_ = db.Close()
		return nil, fmt.Errorf("kb: SQLite driver lacks FTS5 support")
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db, path: path}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
