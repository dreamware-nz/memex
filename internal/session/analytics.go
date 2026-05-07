package session

import (
	"database/sql"
	"fmt"
)

// ComputeReport aggregates the `events` table into a FullReport.
//
// Returns (nil, nil) when `events` is empty so callers can distinguish
// "no session activity" from a real error.
func ComputeReport(db *DB) (*FullReport, error) {
	if db == nil || db.db == nil {
		return nil, fmt.Errorf("session: ComputeReport: nil DB handle")
	}

	var rowCount int
	if err := db.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&rowCount); err != nil {
		return nil, fmt.Errorf("session: ComputeReport count: %w", err)
	}
	if rowCount == 0 {
		return nil, nil
	}

	const aggSQL = `
SELECT
    payload_json ->> '$.tool_name'                                  AS tool_name,
    COUNT(*)                                                        AS call_count,
    COALESCE(SUM(CAST(payload_json ->> '$.tokens_in'  AS INTEGER)), 0) AS tokens_in,
    COALESCE(SUM(CAST(payload_json ->> '$.tokens_out' AS INTEGER)), 0) AS tokens_out,
    COALESCE(SUM(CAST(payload_json ->> '$.bytes_in'   AS INTEGER)), 0) AS bytes_in,
    COALESCE(SUM(CAST(payload_json ->> '$.bytes_out'  AS INTEGER)), 0) AS bytes_out
FROM events
WHERE payload_json ->> '$.tool_name' IS NOT NULL
GROUP BY tool_name
ORDER BY (tokens_in + tokens_out) DESC, tool_name ASC
`
	rows, err := db.db.Query(aggSQL)
	if err != nil {
		return nil, fmt.Errorf("session: ComputeReport aggregate: %w", err)
	}
	defer rows.Close()

	report := &FullReport{}
	for rows.Next() {
		var s ToolStat
		if err := rows.Scan(
			&s.ToolName,
			&s.CallCount,
			&s.TokensIn,
			&s.TokensOut,
			&s.BytesIn,
			&s.BytesOut,
		); err != nil {
			return nil, fmt.Errorf("session: ComputeReport scan: %w", err)
		}
		report.Tools = append(report.Tools, s)
		report.TotalBytesProcessed += s.BytesIn
		report.TotalBytesReturned += s.BytesOut
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("session: ComputeReport rows: %w", err)
	}

	report.SavingsRatio = computeSavingsRatio(report.TotalBytesProcessed, report.TotalBytesReturned)

	uptime, err := computeUptimeMs(db.db)
	if err != nil {
		return nil, err
	}
	report.UptimeMs = uptime

	return report, nil
}

// computeSavingsRatio returns (processed - returned) / processed clamped to
// [0, 1] and rounded to four decimal places.
func computeSavingsRatio(processed, returned int64) float64 {
	if processed <= 0 || returned >= processed {
		return 0.0
	}
	r := float64(processed-returned) / float64(processed)
	if r < 0 {
		return 0.0
	}
	if r > 1 {
		return 1.0
	}
	// Round to 4 decimal places — keeps test fixtures stable across runs.
	return float64(int64(r*10000+0.5)) / 10000.0
}

// computeUptimeMs returns MAX(ts) - MIN(ts) over the events table.
// Returns 0 when no rows are present (caller already short-circuits the
// empty-DB case before calling, but this stays defensive against races).
func computeUptimeMs(db *sql.DB) (int64, error) {
	var minTS, maxTS sql.NullInt64
	if err := db.QueryRow(`SELECT MIN(ts), MAX(ts) FROM events`).Scan(&minTS, &maxTS); err != nil {
		return 0, fmt.Errorf("session: ComputeReport uptime: %w", err)
	}
	if !minTS.Valid || !maxTS.Valid {
		return 0, nil
	}
	d := maxTS.Int64 - minTS.Int64
	if d < 0 {
		return 0, nil
	}
	return d, nil
}
