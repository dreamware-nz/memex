package session

import (
	"testing"
)

func TestBuildSnapshot(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"AggregatesToolCounts", testAggregatesToolCounts},
		{"ExtractsReadAndWritePaths", testExtractsReadAndWritePaths},
		{"EmptyDBYieldsZeroSnapshot", testEmptyDBYieldsZeroSnapshot},
		{"DeduplicatesPaths", testDeduplicatesPaths},
		{"CountsErrors", testCountsErrors},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func testAggregatesToolCounts(t *testing.T) {
	d := openTestDB(t)

	for _, in := range []HookInput{
		{ToolName: "Read", ToolInput: map[string]any{"file_path": "/a"}},
		{ToolName: "Read", ToolInput: map[string]any{"file_path": "/b"}},
		{ToolName: "Write", ToolInput: map[string]any{"file_path": "/c"}},
	} {
		if err := PersistToolCall(d, in); err != nil {
			t.Fatalf("PersistToolCall: %v", err)
		}
	}

	snap, err := BuildSnapshot(d, "")
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if got := snap.ToolCounts["Read"]; got != 2 {
		t.Fatalf("ToolCounts[Read] = %d, want 2", got)
	}
	if got := snap.ToolCounts["Write"]; got != 1 {
		t.Fatalf("ToolCounts[Write] = %d, want 1", got)
	}
}

func testExtractsReadAndWritePaths(t *testing.T) {
	d := openTestDB(t)

	for _, in := range []HookInput{
		{ToolName: "Read", ToolInput: map[string]any{"file_path": "/r1"}},
		{ToolName: "Write", ToolInput: map[string]any{"file_path": "/w1"}},
		{ToolName: "Edit", ToolInput: map[string]any{"file_path": "/w2"}},
	} {
		if err := PersistToolCall(d, in); err != nil {
			t.Fatalf("PersistToolCall: %v", err)
		}
	}

	snap, err := BuildSnapshot(d, "")
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if !contains(snap.FilesRead, "/r1") {
		t.Fatalf("FilesRead = %v, want it to contain /r1", snap.FilesRead)
	}
	if !contains(snap.FilesWritten, "/w1") || !contains(snap.FilesWritten, "/w2") {
		t.Fatalf("FilesWritten = %v, want /w1 and /w2", snap.FilesWritten)
	}
}

func testEmptyDBYieldsZeroSnapshot(t *testing.T) {
	d := openTestDB(t)
	snap, err := BuildSnapshot(d, "")
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if snap == nil {
		t.Fatalf("snap = nil, want non-nil")
	}
	if len(snap.ToolCounts) != 0 || len(snap.FilesRead) != 0 || len(snap.FilesWritten) != 0 ||
		snap.ErrorCount != 0 || snap.FirstEventTS != 0 || snap.LastEventTS != 0 {
		t.Fatalf("snap = %+v, want zero", snap)
	}
}

func testDeduplicatesPaths(t *testing.T) {
	d := openTestDB(t)

	for i := 0; i < 3; i++ {
		if err := PersistToolCall(d, HookInput{
			ToolName: "Read", ToolInput: map[string]any{"file_path": "/same"},
		}); err != nil {
			t.Fatalf("PersistToolCall: %v", err)
		}
	}
	snap, err := BuildSnapshot(d, "")
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	count := 0
	for _, p := range snap.FilesRead {
		if p == "/same" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("FilesRead occurrences of /same = %d, want 1 (got %v)", count, snap.FilesRead)
	}
}

func testCountsErrors(t *testing.T) {
	d := openTestDB(t)

	if err := PersistToolCall(d, HookInput{
		ToolName: "Bash", ToolOutput: &ToolOutput{IsError: true},
	}); err != nil {
		t.Fatalf("PersistToolCall: %v", err)
	}
	if err := PersistToolCall(d, HookInput{
		ToolName: "Bash", ToolOutput: &ToolOutput{IsError: false},
	}); err != nil {
		t.Fatalf("PersistToolCall: %v", err)
	}
	snap, err := BuildSnapshot(d, "")
	if err != nil {
		t.Fatalf("BuildSnapshot: %v", err)
	}
	if snap.ErrorCount != 1 {
		t.Fatalf("ErrorCount = %d, want 1", snap.ErrorCount)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
