package session

import (
	"database/sql"
	"fmt"
	"time"
)

var migrations = []string{
	// v1: events + schema_migrations.
	`
CREATE TABLE IF NOT EXISTS events (
    id           INTEGER PRIMARY KEY,
    kind         TEXT    NOT NULL,
    ts           INTEGER NOT NULL,
    payload_json TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);
`,
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
        version    INTEGER PRIMARY KEY,
        applied_at INTEGER NOT NULL
    )`); err != nil {
		return fmt.Errorf("session: bootstrap schema_migrations: %w", err)
	}

	var current int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("session: read schema_migrations: %w", err)
	}

	for i, ddl := range migrations {
		version := i + 1
		if version <= current {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("session: begin migration v%d: %w", version, err)
		}
		if _, err := tx.Exec(ddl); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("session: apply migration v%d: %w", version, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)`,
			version, time.Now().Unix(),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("session: record migration v%d: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("session: commit migration v%d: %w", version, err)
		}
	}
	return nil
}
