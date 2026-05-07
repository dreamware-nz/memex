package session

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestPersistToolCall(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"PersistsKnownToolName", testPersistsKnownToolName},
		{"PreservesToolOutputIsError", testPreservesToolOutputIsError},
		{"AcceptsUnknownToolName", testAcceptsUnknownToolName},
		{"PreservesToolResponse", testPreservesToolResponse},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func openTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "session.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func testPersistsKnownToolName(t *testing.T) {
	d := openTestDB(t)
	before := time.Now().UnixMilli()

	input := HookInput{
		ToolName:     "Read",
		ToolInput:    map[string]any{"file_path": "/tmp/foo"},
		ToolResponse: "hello",
	}
	if err := PersistToolCall(d, input); err != nil {
		t.Fatalf("PersistToolCall: %v", err)
	}

	after := time.Now().UnixMilli()

	var (
		count   int
		kind    string
		ts      int64
		payload string
	)
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&count); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 1 {
		t.Fatalf("events row count = %d, want 1", count)
	}
	if err := d.db.QueryRow(
		`SELECT kind, ts, payload_json FROM events LIMIT 1`,
	).Scan(&kind, &ts, &payload); err != nil {
		t.Fatalf("select event: %v", err)
	}
	if kind != "tool_call" {
		t.Fatalf("kind = %q, want %q", kind, "tool_call")
	}
	if ts < before-1000 || ts > after+1000 {
		t.Fatalf("ts = %d, want within 1s of [%d,%d]", ts, before, after)
	}

	var got HookInput
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("unmarshal payload: %v (raw=%s)", err, payload)
	}
	if got.ToolName != "Read" {
		t.Fatalf("payload tool_name = %q, want %q", got.ToolName, "Read")
	}
}

func testPreservesToolOutputIsError(t *testing.T) {
	d := openTestDB(t)

	input := HookInput{
		ToolName:   "Bash",
		ToolOutput: &ToolOutput{IsError: true},
	}
	if err := PersistToolCall(d, input); err != nil {
		t.Fatalf("PersistToolCall: %v", err)
	}

	var payload string
	if err := d.db.QueryRow(`SELECT payload_json FROM events LIMIT 1`).Scan(&payload); err != nil {
		t.Fatalf("select payload: %v", err)
	}

	var got HookInput
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("unmarshal payload: %v (raw=%s)", err, payload)
	}
	if got.ToolOutput == nil {
		t.Fatalf("tool_output missing from payload: %s", payload)
	}
	if !got.ToolOutput.IsError {
		t.Fatalf("tool_output.is_error = false, want true (raw=%s)", payload)
	}
}

func testAcceptsUnknownToolName(t *testing.T) {
	d := openTestDB(t)

	input := HookInput{ToolName: "DefinitelyNotAToolWeKnow"}
	if err := PersistToolCall(d, input); err != nil {
		t.Fatalf("PersistToolCall with unknown tool name: %v", err)
	}

	var kind string
	if err := d.db.QueryRow(`SELECT kind FROM events LIMIT 1`).Scan(&kind); err != nil {
		t.Fatalf("select kind: %v", err)
	}
	if kind != "tool_call" {
		t.Fatalf("kind = %q, want %q", kind, "tool_call")
	}
}

func testPreservesToolResponse(t *testing.T) {
	d := openTestDB(t)

	const body = "line one\nline two\n"
	if err := PersistToolCall(d, HookInput{ToolName: "Read", ToolResponse: body}); err != nil {
		t.Fatalf("PersistToolCall: %v", err)
	}

	var payload string
	if err := d.db.QueryRow(`SELECT payload_json FROM events LIMIT 1`).Scan(&payload); err != nil {
		t.Fatalf("select payload: %v", err)
	}

	var got HookInput
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("unmarshal payload: %v (raw=%s)", err, payload)
	}
	if got.ToolResponse != body {
		t.Fatalf("tool_response = %q, want %q", got.ToolResponse, body)
	}
}
