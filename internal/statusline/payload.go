// Package statusline parses the persisted stats payload and renders the
// one-line summary printed by `ctx statusline` for Claude Code's status bar.
package statusline

import "encoding/json"

// StatsPayload mirrors the stats JSON file written by the session-analytics
// layer.  Optional fields use pointer-free zero values; absence of
// dollars_saved_lifetime is indistinguishable from 0, which matches the
// TS reference implementation's "skip lifetime block when zero" rule.
type StatsPayload struct {
	SchemaVersion        int     `json:"schemaVersion"`
	DollarsSavedSession  float64 `json:"dollars_saved_session"`
	DollarsSavedLifetime float64 `json:"dollars_saved_lifetime"`
	PctEfficient         int     `json:"pct_efficient"`
	UptimeMs             int64   `json:"uptime_ms"`
}

// ParsePayload decodes a stats JSON blob.  Empty input yields (nil, nil) so
// callers can treat "no file" and "empty file" identically.
func ParsePayload(data []byte) (*StatsPayload, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var p StatsPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
