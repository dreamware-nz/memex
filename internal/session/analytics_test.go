package session

import (
	"testing"
)

func TestComputeReport(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"AggregatesPerToolStats", testAggregatesPerToolStats},
		{"EmptyDatabaseReturnsNilNil", testEmptyDatabaseReturnsNilNil},
		{"SavingsRatioPositive", testSavingsRatioPositive},
		{"SavingsRatioClampedWhenInverted", testSavingsRatioClampedWhenInverted},
		{"UptimeMsFromMinMaxTs", testUptimeMsFromMinMaxTs},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

// insertAnalyticsEvent writes a row directly so tests can control the exact
// payload shape (tokens_in, tokens_out, bytes_in, bytes_out) — fields the
// real HookInput type does not yet model.
func insertAnalyticsEvent(t *testing.T, d *DB, ts int64, payload string) {
	t.Helper()
	if _, err := d.db.Exec(
		`INSERT INTO events (kind, ts, payload_json) VALUES (?, ?, ?)`,
		"tool_call", ts, payload,
	); err != nil {
		t.Fatalf("insert analytics event: %v", err)
	}
}

func testAggregatesPerToolStats(t *testing.T) {
	d := openTestDB(t)

	insertAnalyticsEvent(t, d, 1000, `{"tool_name":"Read","tokens_in":100,"tokens_out":50,"bytes_in":4000,"bytes_out":1000}`)
	insertAnalyticsEvent(t, d, 1500, `{"tool_name":"Read","tokens_in":200,"tokens_out":80,"bytes_in":8000,"bytes_out":2000}`)
	insertAnalyticsEvent(t, d, 2000, `{"tool_name":"Bash","tokens_in":30,"tokens_out":10,"bytes_in":500,"bytes_out":100}`)

	r, err := ComputeReport(d)
	if err != nil {
		t.Fatalf("ComputeReport: %v", err)
	}
	if r == nil {
		t.Fatalf("expected non-nil report for populated DB")
	}
	if len(r.Tools) != 2 {
		t.Fatalf("len(Tools) = %d, want 2", len(r.Tools))
	}

	byName := map[string]ToolStat{}
	for _, s := range r.Tools {
		byName[s.ToolName] = s
	}

	read, ok := byName["Read"]
	if !ok {
		t.Fatalf("missing Read stat: %#v", r.Tools)
	}
	if read.CallCount != 2 {
		t.Fatalf("Read.CallCount = %d, want 2", read.CallCount)
	}
	if read.TokensIn != 300 {
		t.Fatalf("Read.TokensIn = %d, want 300", read.TokensIn)
	}
	if read.TokensOut != 130 {
		t.Fatalf("Read.TokensOut = %d, want 130", read.TokensOut)
	}
	if read.BytesIn != 12000 {
		t.Fatalf("Read.BytesIn = %d, want 12000", read.BytesIn)
	}
	if read.BytesOut != 3000 {
		t.Fatalf("Read.BytesOut = %d, want 3000", read.BytesOut)
	}

	bash, ok := byName["Bash"]
	if !ok {
		t.Fatalf("missing Bash stat: %#v", r.Tools)
	}
	if bash.CallCount != 1 {
		t.Fatalf("Bash.CallCount = %d, want 1", bash.CallCount)
	}
	if bash.TokensIn != 30 || bash.TokensOut != 10 {
		t.Fatalf("Bash tokens = (%d,%d), want (30,10)", bash.TokensIn, bash.TokensOut)
	}

	if r.TotalBytesProcessed != 12500 {
		t.Fatalf("TotalBytesProcessed = %d, want 12500", r.TotalBytesProcessed)
	}
	if r.TotalBytesReturned != 3100 {
		t.Fatalf("TotalBytesReturned = %d, want 3100", r.TotalBytesReturned)
	}
}

func testEmptyDatabaseReturnsNilNil(t *testing.T) {
	d := openTestDB(t)

	r, err := ComputeReport(d)
	if err != nil {
		t.Fatalf("ComputeReport: %v", err)
	}
	if r != nil {
		t.Fatalf("expected nil report for empty DB, got %#v", r)
	}
}

func testSavingsRatioPositive(t *testing.T) {
	d := openTestDB(t)

	insertAnalyticsEvent(t, d, 1000, `{"tool_name":"Read","bytes_in":1000,"bytes_out":250}`)

	r, err := ComputeReport(d)
	if err != nil || r == nil {
		t.Fatalf("ComputeReport: r=%v err=%v", r, err)
	}
	// (1000 - 250) / 1000 = 0.75
	if r.SavingsRatio != 0.75 {
		t.Fatalf("SavingsRatio = %v, want 0.75", r.SavingsRatio)
	}
}

func testSavingsRatioClampedWhenInverted(t *testing.T) {
	d := openTestDB(t)

	// bytes_out > bytes_in — would yield a negative ratio without clamping.
	insertAnalyticsEvent(t, d, 1000, `{"tool_name":"Read","bytes_in":100,"bytes_out":500}`)

	r, err := ComputeReport(d)
	if err != nil || r == nil {
		t.Fatalf("ComputeReport: r=%v err=%v", r, err)
	}
	if r.SavingsRatio != 0.0 {
		t.Fatalf("SavingsRatio = %v, want 0.0 (clamped)", r.SavingsRatio)
	}
}

func testUptimeMsFromMinMaxTs(t *testing.T) {
	d := openTestDB(t)

	insertAnalyticsEvent(t, d, 1000, `{"tool_name":"Read","bytes_in":10,"bytes_out":1}`)
	insertAnalyticsEvent(t, d, 4500, `{"tool_name":"Read","bytes_in":10,"bytes_out":1}`)

	r, err := ComputeReport(d)
	if err != nil || r == nil {
		t.Fatalf("ComputeReport: r=%v err=%v", r, err)
	}
	if r.UptimeMs != 3500 {
		t.Fatalf("UptimeMs = %d, want 3500", r.UptimeMs)
	}
}
