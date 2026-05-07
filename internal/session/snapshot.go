package session

import (
	"fmt"
	"sort"
	"strings"
)

// RenderSnapshot produces a deterministic Markdown recap of a Snapshot
// suitable for injection into a Claude context window. Output contains no
// timestamps so callers can do exact string assertions in tests.
func RenderSnapshot(s *Snapshot) string {
	if s == nil || isEmptySnapshot(s) {
		return "## Session Snapshot\n\nNo events recorded.\n"
	}

	var b strings.Builder
	b.WriteString("## Session Snapshot\n\n")

	if len(s.ToolCounts) > 0 {
		b.WriteString("### Tool calls\n\n")
		b.WriteString("| Tool | Count |\n")
		b.WriteString("| --- | --- |\n")
		names := make([]string, 0, len(s.ToolCounts))
		for k := range s.ToolCounts {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(&b, "| %s | %d |\n", name, s.ToolCounts[name])
		}
		b.WriteString("\n")
	}

	if len(s.FilesRead) > 0 {
		b.WriteString("### Files read\n\n")
		for _, p := range s.FilesRead {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		b.WriteString("\n")
	}

	if len(s.FilesWritten) > 0 {
		b.WriteString("### Files written\n\n")
		for _, p := range s.FilesWritten {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "### Errors: %d\n", s.ErrorCount)
	return b.String()
}

func isEmptySnapshot(s *Snapshot) bool {
	return len(s.ToolCounts) == 0 &&
		len(s.FilesRead) == 0 &&
		len(s.FilesWritten) == 0 &&
		s.ErrorCount == 0 &&
		s.FirstEventTS == 0 &&
		s.LastEventTS == 0
}
