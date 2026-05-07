package session

import (
	"strings"
	"testing"
)

func TestFormatReport(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"NonEmptyReportContainsToolAndTokens", testFormatNonEmpty},
		{"EmptyReportReturnsPlaceholder", testFormatEmpty},
		{"NilReportReturnsPlaceholder", testFormatNil},
		{"ToolsSortedByTotalTokensDesc", testFormatSortedByTokens},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func testFormatNonEmpty(t *testing.T) {
	r := &FullReport{
		Tools: []ToolStat{
			{ToolName: "Read", CallCount: 3, TokensIn: 1234, TokensOut: 567, BytesIn: 9000, BytesOut: 1500},
		},
		TotalBytesProcessed: 9000,
		TotalBytesReturned:  1500,
		SavingsRatio:        0.8333,
		UptimeMs:            12345,
	}

	out := FormatReport(r)
	if !strings.Contains(out, "Read") {
		t.Fatalf("output missing tool name 'Read':\n%s", out)
	}
	if !strings.Contains(out, "1234") {
		t.Fatalf("output missing tokens_in count 1234:\n%s", out)
	}
	if !strings.Contains(out, "83.33%") {
		t.Fatalf("output missing savings percentage 83.33%%:\n%s", out)
	}
	if !strings.Contains(out, "9000") {
		t.Fatalf("output missing total bytes processed 9000:\n%s", out)
	}
}

func testFormatEmpty(t *testing.T) {
	r := &FullReport{}
	out := FormatReport(r)
	if !strings.Contains(out, "No session data") {
		t.Fatalf("output missing 'No session data' placeholder: %q", out)
	}
}

func testFormatNil(t *testing.T) {
	out := FormatReport(nil)
	if !strings.Contains(out, "No session data") {
		t.Fatalf("nil report should yield placeholder, got: %q", out)
	}
}

func testFormatSortedByTokens(t *testing.T) {
	r := &FullReport{
		Tools: []ToolStat{
			{ToolName: "Bash", CallCount: 1, TokensIn: 10, TokensOut: 10},
			{ToolName: "Read", CallCount: 1, TokensIn: 500, TokensOut: 500},
			{ToolName: "Write", CallCount: 1, TokensIn: 100, TokensOut: 100},
		},
	}
	out := FormatReport(r)
	readIdx := strings.Index(out, "Read")
	writeIdx := strings.Index(out, "Write")
	bashIdx := strings.Index(out, "Bash")
	if !(readIdx < writeIdx && writeIdx < bashIdx) {
		t.Fatalf("expected Read < Write < Bash by row order, got idx Read=%d Write=%d Bash=%d in:\n%s",
			readIdx, writeIdx, bashIdx, out)
	}
}
