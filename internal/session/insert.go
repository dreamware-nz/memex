package session

import (
	"encoding/json"
	"fmt"
)

// insertEvent marshals payload to JSON and inserts a single row into `events`.
// Each call is its own auto-commit statement; WAL mode keeps concurrent
// readers safe.
func insertEvent(db *DB, kind string, ts int64, payload any) (err error) {
	if db == nil || db.db == nil {
		return fmt.Errorf("session: insertEvent: nil DB handle")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("session: marshal %s payload: %w", kind, err)
	}

	tx, err := db.db.Begin()
	if err != nil {
		return fmt.Errorf("session: begin %s insert: %w", kind, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(
		`INSERT INTO events (kind, ts, payload_json) VALUES (?, ?, ?)`,
		kind, ts, string(body),
	); err != nil {
		return fmt.Errorf("session: insert %s event: %w", kind, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("session: commit %s insert: %w", kind, err)
	}
	return nil
}
