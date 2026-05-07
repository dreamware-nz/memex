package cli

import (
	"bytes"
	"strings"
	"testing"
)

func runDoctor(t *testing.T) string {
	t.Helper()
	cmd := NewDoctorCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return buf.String()
}

func TestDoctorCmd_NonEmptyAndContainsGoVersion(t *testing.T) {
	out := runDoctor(t)
	if strings.TrimSpace(out) == "" {
		t.Fatal("doctor output is empty")
	}
	if !strings.Contains(out, "go version") {
		t.Fatalf("doctor output missing 'go version':\n%s", out)
	}
}

func TestDoctorCmd_SingleLineCommand(t *testing.T) {
	out := runDoctor(t)
	// Exactly one trailing newline; command itself must be a single line.
	trimmed := strings.TrimRight(out, "\n")
	if strings.Contains(trimmed, "\n") {
		t.Fatalf("doctor command contains embedded newline:\n%q", trimmed)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("doctor output missing trailing newline: %q", out)
	}
}
