package cli

import (
	"bytes"
	"testing"
)

func TestUpgradeCmd_PrintsInstallCommand(t *testing.T) {
	cmd := NewUpgradeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got, want := buf.String(), "go install github.com/dreamware-nz/memex/cmd/memex@latest\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
