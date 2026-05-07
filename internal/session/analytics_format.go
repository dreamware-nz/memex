package session

import (
	"fmt"
	"sort"
	"strings"
)

// emptyReportPlaceholder is returned by FormatReport when the report has no
// per-tool rows. Tests assert the substring "No session data" so do not
// reword without updating analytics_format_test.go.
const emptyReportPlaceholder = "No session data recorded."

// FormatReport renders r as a deterministic plain-text dashboard.
//
// Layout: a per-tool table sorted by total tokens descending, followed by a
// single summary line with byte totals and the savings ratio as a percentage.
func FormatReport(r *FullReport) string {
	if r == nil || len(r.Tools) == 0 {
		return emptyReportPlaceholder
	}

	tools := make([]ToolStat, len(r.Tools))
	copy(tools, r.Tools)
	sort.SliceStable(tools, func(i, j int) bool {
		ti := tools[i].TokensIn + tools[i].TokensOut
		tj := tools[j].TokensIn + tools[j].TokensOut
		if ti != tj {
			return ti > tj
		}
		return tools[i].ToolName < tools[j].ToolName
	})

	var b strings.Builder
	const (
		nameW   = 24
		intW    = 8
		tokenW  = 12
		header  = "tool"
		callsH  = "calls"
		tokInH  = "tokens_in"
		tokOutH = "tokens_out"
	)

	fmt.Fprintf(&b, "%-*s  %*s  %*s  %*s\n", nameW, header, intW, callsH, tokenW, tokInH, tokenW, tokOutH)
	fmt.Fprintf(&b, "%s\n", strings.Repeat("-", nameW+intW+tokenW*2+6))
	for _, t := range tools {
		name := t.ToolName
		if len(name) > nameW {
			name = name[:nameW-3] + "..."
		}
		fmt.Fprintf(&b, "%-*s  %*d  %*d  %*d\n",
			nameW, name,
			intW, t.CallCount,
			tokenW, t.TokensIn,
			tokenW, t.TokensOut,
		)
	}

	pct := r.SavingsRatio * 100.0
	fmt.Fprintf(&b,
		"\nprocessed=%d B  returned=%d B  savings=%.2f%%\n",
		r.TotalBytesProcessed, r.TotalBytesReturned, pct,
	)

	return b.String()
}
