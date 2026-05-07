package kb

import (
	"database/sql"
	"fmt"
	"time"
)

var migrations = []string{
	// v1: sources, sections, sections_fts (FTS5 contentless), schema_migrations.
	`
CREATE TABLE IF NOT EXISTS sources (
    id         INTEGER PRIMARY KEY,
    label      TEXT    NOT NULL UNIQUE,
    kind       TEXT    NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sections (
    id         INTEGER PRIMARY KEY,
    source_id  INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    heading    TEXT,
    body       TEXT    NOT NULL,
    byte_size  INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS sections_fts USING fts5(
    heading,
    body,
    content='sections',
    content_rowid='id'
);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);
`,
	// v2: triggers to keep sections_fts in sync with sections.
	`
CREATE TRIGGER IF NOT EXISTS sections_ai AFTER INSERT ON sections BEGIN
    INSERT INTO sections_fts(rowid, heading, body) VALUES (new.id, new.heading, new.body);
END;

CREATE TRIGGER IF NOT EXISTS sections_ad AFTER DELETE ON sections BEGIN
    INSERT INTO sections_fts(sections_fts, rowid, heading, body) VALUES ('delete', old.id, old.heading, old.body);
END;

CREATE TRIGGER IF NOT EXISTS sections_au AFTER UPDATE ON sections BEGIN
    INSERT INTO sections_fts(sections_fts, rowid, heading, body) VALUES ('delete', old.id, old.heading, old.body);
    INSERT INTO sections_fts(rowid, heading, body) VALUES (new.id, new.heading, new.body);
END;
`,
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
        version    INTEGER PRIMARY KEY,
        applied_at INTEGER NOT NULL
    )`); err != nil {
		return fmt.Errorf("kb: bootstrap schema_migrations: %w", err)
	}

	var current int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return fmt.Errorf("kb: read schema_migrations: %w", err)
	}

	for i, ddl := range migrations {
		version := i + 1
		if version <= current {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("kb: begin migration v%d: %w", version, err)
		}
		if _, err := tx.Exec(ddl); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("kb: apply migration v%d: %w", version, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)`,
			version, time.Now().Unix(),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("kb: record migration v%d: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("kb: commit migration v%d: %w", version, err)
		}
	}
	return nil
}
