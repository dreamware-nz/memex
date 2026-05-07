package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/dreamware-nz/memex/internal/analytics"
)

// StatsReader is the analytics-reader dependency StatsTool needs. The CLI's
// analytics.Reader satisfies it; tests provide a fake.
type StatsReader interface {
	All() ([]analytics.ToolRow, error)
	Close() error
}

// StatsOpener resolves the analytics database file at call time so the path
// can be re-checked for existence between calls (the file may not exist
// until the first SessionStart event lands).
type StatsOpener func(path string) (StatsReader, error)

// StatsTool implements memex_stats: reads the per-tool savings table from
// the analytics SQLite file and returns a tab-aligned text report. Read-only
// — never mutates the analytics DB.
type StatsTool struct {
	// AnalyticsPath is the absolute path to the analytics SQLite file. If
	// the file is missing, Call returns "No stats yet" rather than an
	// error: that's the normal first-session state.
	AnalyticsPath string
	// Open opens the analytics DB. Defaults to wrapping analytics.Open via
	// NewMCPCmd. Tests inject a fake.
	Open StatsOpener
}

func (t *StatsTool) Name() string { return "memex_stats" }

func (t *StatsTool) Description() string {
	return "Show per-tool token and USD savings for the current session. Read-only."
}

func (t *StatsTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (t *StatsTool) Call(_ json.RawMessage) (string, error) {
	if t.AnalyticsPath == "" {
		return "No stats yet — analytics path not configured.\n", nil
	}
	if _, err := os.Stat(t.AnalyticsPath); os.IsNotExist(err) {
		return "No stats yet — run a session first.\n", nil
	} else if err != nil {
		return "", fmt.Errorf("stat analytics db: %w", err)
	}
	if t.Open == nil {
		return "", fmt.Errorf("memex_stats: no analytics opener wired")
	}
	r, err := t.Open(t.AnalyticsPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	rows, err := r.All()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	tw := tabwriter.NewWriter(&sb, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "Tool\tCalls\tTokens Saved\tUSD Saved")
	var totalCalls, totalTokens int64
	var totalUSD float64
	for _, row := range rows {
		fmt.Fprintf(tw, "%s\t%d\t%d\t%.4f\n",
			row.Tool, row.Calls, row.TokensSaved, row.USDSaved)
		totalCalls += row.Calls
		totalTokens += row.TokensSaved
		totalUSD += row.USDSaved
	}
	fmt.Fprintln(tw, strings.Repeat("-", 8)+"\t"+
		strings.Repeat("-", 5)+"\t"+
		strings.Repeat("-", 12)+"\t"+
		strings.Repeat("-", 9))
	fmt.Fprintf(tw, "TOTAL\t%d\t%d\t%.4f\n", totalCalls, totalTokens, totalUSD)
	if err := tw.Flush(); err != nil {
		return "", err
	}
	return sb.String(), nil
}
