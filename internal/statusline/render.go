package statusline

import (
	"fmt"
	"strings"
)

// Render builds the one-line status summary.  It is a pure function: same
// input always yields the same output, no I/O, no globals.  A nil payload
// yields the empty string so callers can pipe directly to stdout.
func Render(p *StatsPayload) string {
	if p == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("memex  ●  ")
	b.WriteString(fmtUSD(p.DollarsSavedSession))
	b.WriteString(" saved this session")
	if p.DollarsSavedLifetime > 0 {
		b.WriteString("  ·  ")
		b.WriteString(fmtUSD(p.DollarsSavedLifetime))
		b.WriteString(" saved across sessions")
	}
	b.WriteString("  ·  ")
	fmt.Fprintf(&b, "%d%% efficient", p.PctEfficient)
	return b.String()
}

func fmtUSD(f float64) string {
	return fmt.Sprintf("$%.2f", f)
}
