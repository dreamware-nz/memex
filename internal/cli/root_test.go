package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCmd_HelpListsVersion(t *testing.T) {
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute --help: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Usage:") {
		t.Errorf("help output missing %q:\n%s", "Usage:", out)
	}
	if !strings.Contains(out, "version") {
		t.Errorf("help output missing %q:\n%s", "version", out)
	}
}

func TestRootCmd_UnknownCommandErrors(t *testing.T) {
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"bogus"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error message missing %q: %v", "bogus", err)
	}
}
