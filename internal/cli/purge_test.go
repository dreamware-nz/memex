package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// purgePaths returns the (kb, analytics) paths inside a fresh temp data dir
// and creates the files so each scenario can assert on deletion.
func purgePaths(t *testing.T) (dir, kb, analytics string) {
	t.Helper()
	dir = t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("CTX_DB", "")
	t.Setenv("MEMEX_DATA_DIR", "")

	kb = filepath.Join(dir, "memex", "kb.sqlite")
	analytics = filepath.Join(dir, "memex", "session", "analytics.db")
	return
}

func seedFiles(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
}

func runPurge(t *testing.T, stdin string, args ...string) (stdout, stderr string) {
	t.Helper()
	cmd := NewPurgeCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return out.String(), errBuf.String()
}

func TestPurgeCmd_YesDeletesFiles(t *testing.T) {
	_, kb, analytics := purgePaths(t)
	seedFiles(t, kb, analytics)

	stdout, _ := runPurge(t, "", "--yes")

	if _, err := os.Stat(kb); !os.IsNotExist(err) {
		t.Fatalf("kb not removed: err=%v", err)
	}
	if _, err := os.Stat(analytics); !os.IsNotExist(err) {
		t.Fatalf("analytics not removed: err=%v", err)
	}
	if !strings.Contains(stdout, "Purged") {
		t.Fatalf("stdout missing 'Purged':\n%s", stdout)
	}
}

func TestPurgeCmd_YesNoFiles(t *testing.T) {
	purgePaths(t)

	stdout, _ := runPurge(t, "", "--yes")

	if !strings.Contains(stdout, "Nothing to purge") {
		t.Fatalf("stdout missing 'Nothing to purge':\n%s", stdout)
	}
}

func TestPurgeCmd_AbortsWithoutYes(t *testing.T) {
	_, kb, analytics := purgePaths(t)
	seedFiles(t, kb, analytics)

	_, stderr := runPurge(t, "n\n")

	if _, err := os.Stat(kb); err != nil {
		t.Fatalf("kb should still exist: %v", err)
	}
	if _, err := os.Stat(analytics); err != nil {
		t.Fatalf("analytics should still exist: %v", err)
	}
	if !strings.Contains(stderr, "Aborted") {
		t.Fatalf("stderr missing 'Aborted':\n%s", stderr)
	}
}

func TestPurgeCmd_AcceptsYesViaStdin(t *testing.T) {
	_, kb, analytics := purgePaths(t)
	seedFiles(t, kb, analytics)

	runPurge(t, "y\n")

	if _, err := os.Stat(kb); !os.IsNotExist(err) {
		t.Fatalf("kb not removed: err=%v", err)
	}
	if _, err := os.Stat(analytics); !os.IsNotExist(err) {
		t.Fatalf("analytics not removed: err=%v", err)
	}
}
