package cli

import (
	"bytes"
	"testing"
)

func TestVersionCmd_PrintsVersion(t *testing.T) {
	cmd := NewVersionCmd("9.9.9")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got, want := buf.String(), "9.9.9\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
