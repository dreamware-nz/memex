package session

import (
	"encoding/json"
	"fmt"
	"sort"
)

// BuildSnapshot aggregates `events` rows for the given sessionID into a typed
// Snapshot. When sessionID is empty, all rows are considered. Returns a
// non-nil *Snapshot even when the events table is empty.
func BuildSnapshot(db *DB, sessionID string) (*Snapshot, error) {
	if db == nil || db.db == nil {
		return nil, fmt.Errorf("session: BuildSnapshot: nil DB handle")
	}

	rows, err := db.db.Query(
		`SELECT kind, ts, payload_json FROM events ORDER BY ts ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("session: query events: %w", err)
	}
	defer rows.Close()

	snap := &Snapshot{ToolCounts: map[string]int{}}
	readSet := map[string]struct{}{}
	writeSet := map[string]struct{}{}

	for rows.Next() {
		var (
			kind    string
			ts      int64
			payload string
		)
		if err := rows.Scan(&kind, &ts, &payload); err != nil {
			return nil, fmt.Errorf("session: scan event: %w", err)
		}

		var body map[string]any
		if err := json.Unmarshal([]byte(payload), &body); err != nil {
			// Malformed payload — skip, do not fail the whole build.
			continue
		}

		if sessionID != "" {
			if sid, ok := body["session_id"].(string); !ok || sid != sessionID {
				continue
			}
		}

		if snap.FirstEventTS == 0 || ts < snap.FirstEventTS {
			snap.FirstEventTS = ts
		}
		if ts > snap.LastEventTS {
			snap.LastEventTS = ts
		}

		if kind != kindToolCall {
			continue
		}

		toolName, _ := body["tool_name"].(string)
		if toolName != "" {
			snap.ToolCounts[toolName]++
		}

		if filePath := extractFilePath(body); filePath != "" {
			switch toolName {
			case "Read":
				readSet[filePath] = struct{}{}
			case "Write", "Edit":
				writeSet[filePath] = struct{}{}
			}
		}

		if isErrorEvent(body) {
			snap.ErrorCount++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("session: iterate events: %w", err)
	}

	snap.FilesRead = sortedKeys(readSet)
	snap.FilesWritten = sortedKeys(writeSet)
	return snap, nil
}

// extractFilePath pulls `tool_input.file_path` from a payload body.
func extractFilePath(body map[string]any) string {
	input, ok := body["tool_input"].(map[string]any)
	if !ok {
		return ""
	}
	p, _ := input["file_path"].(string)
	return p
}

// isErrorEvent returns true when `tool_output.is_error` is truthy.
func isErrorEvent(body map[string]any) bool {
	out, ok := body["tool_output"].(map[string]any)
	if !ok {
		return false
	}
	v, _ := out["is_error"].(bool)
	return v
}

func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
