package session

import (
	"strings"
	"testing"
)

func TestRenderSnapshot(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"NonEmptySnapshotRendersAllSections", testRenderNonEmpty},
		{"EmptySnapshotRendersPlaceholder", testRenderEmpty},
		{"NilSnapshotRendersPlaceholder", testRenderNil},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func testRenderNonEmpty(t *testing.T) {
	snap := &Snapshot{
		ToolCounts:   map[string]int{"Read": 2, "Write": 1},
		FilesRead:    []string{"/a", "/b"},
		FilesWritten: []string{"/c"},
		ErrorCount:   3,
		FirstEventTS: 1000,
		LastEventTS:  2000,
	}
	out := RenderSnapshot(snap)
	for _, want := range []string{
		"Read", "| 2 |",
		"Write", "| 1 |",
		"/a", "/b", "/c",
		"Errors: 3",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("RenderSnapshot output missing %q\n---\n%s", want, out)
		}
	}
}

func testRenderEmpty(t *testing.T) {
	out := RenderSnapshot(&Snapshot{ToolCounts: map[string]int{}})
	if out == "" {
		t.Fatalf("RenderSnapshot returned empty string")
	}
	if !strings.Contains(strings.ToLower(out), "no events") {
		t.Fatalf("empty snapshot output missing no-events indicator: %q", out)
	}
}

func testRenderNil(t *testing.T) {
	out := RenderSnapshot(nil)
	if out == "" {
		t.Fatalf("RenderSnapshot(nil) returned empty string")
	}
	if !strings.Contains(strings.ToLower(out), "no events") {
		t.Fatalf("nil snapshot output missing no-events indicator: %q", out)
	}
}
