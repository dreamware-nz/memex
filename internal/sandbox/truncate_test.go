package sandbox

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/kb"
)

func newKBStore(t *testing.T) *kb.Store {
	t.Helper()
	s, err := kb.Open(filepath.Join(t.TempDir(), "kb.sqlite"))
	if err != nil {
		t.Fatalf("kb.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestTruncate_UnderThreshold(t *testing.T) {
	res, err := Run(context.Background(), RunOptions{
		Command: []string{"sh", "-c", "for i in $(seq 1 10); do echo line$i; done"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Truncated {
		t.Fatalf("Truncated = true, want false")
	}
	if res.IndexedAs != "" {
		t.Fatalf("IndexedAs = %q, want empty", res.IndexedAs)
	}
	if !strings.Contains(res.Stdout, "line1\n") || !strings.Contains(res.Stdout, "line10\n") {
		t.Fatalf("Stdout missing expected lines: %q", res.Stdout)
	}
}

func TestTruncate_OverLineThreshold(t *testing.T) {
	store := newKBStore(t)
	res, err := Run(context.Background(), RunOptions{
		Command: []string{"sh", "-c", "for i in $(seq 1 250); do echo line$i; done"},
		Store:   store,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Truncated {
		t.Fatalf("Truncated = false, want true")
	}
	if res.IndexedAs == "" {
		t.Fatalf("IndexedAs empty, want non-empty")
	}
	if !strings.HasPrefix(res.IndexedAs, "sandbox/") {
		t.Fatalf("IndexedAs = %q, want sandbox/* prefix", res.IndexedAs)
	}
	if !strings.Contains(res.Stdout, "truncated") {
		t.Fatalf("Stdout = %q, want summary text", res.Stdout)
	}
	if !strings.Contains(res.Stdout, res.IndexedAs) {
		t.Fatalf("Stdout %q does not contain label %q", res.Stdout, res.IndexedAs)
	}
}

func TestTruncate_OverByteThreshold(t *testing.T) {
	store := newKBStore(t)
	// Emit ~40 KiB of output across few lines (under line cap), exceeding byte cap.
	res, err := Run(context.Background(), RunOptions{
		Command:  []string{"sh", "-c", "head -c 40960 /dev/zero | tr '\\0' 'a'; echo"},
		MaxBytes: 64 * 1024,
		Store:    store,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Truncated {
		t.Fatalf("Truncated = false, want true")
	}
}

func TestTruncate_NilStore(t *testing.T) {
	res, err := Run(context.Background(), RunOptions{
		Command: []string{"sh", "-c", "for i in $(seq 1 250); do echo line$i; done"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.Truncated {
		t.Fatalf("Truncated = false, want true")
	}
	if res.IndexedAs != "" {
		t.Fatalf("IndexedAs = %q, want empty", res.IndexedAs)
	}
	if res.Stdout == "" {
		t.Fatalf("Stdout empty, want non-empty summary")
	}
	if !strings.Contains(res.Stdout, "truncated") {
		t.Fatalf("Stdout = %q, want summary text", res.Stdout)
	}
}
