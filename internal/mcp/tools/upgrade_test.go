package tools

import (
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/mcp"
)

func TestUpgradeTool_ImplementsToolInterface(t *testing.T) {
	var _ mcp.Tool = (*UpgradeTool)(nil)
}

func TestUpgradeTool_Name(t *testing.T) {
	if got := (&UpgradeTool{}).Name(); got != "memex_upgrade" {
		t.Errorf("Name() = %q, want memex_upgrade", got)
	}
}

func TestUpgradeTool_Call_ReturnsConfiguredCommand(t *testing.T) {
	tool := &UpgradeTool{Command: "go install github.com/dreamware-nz/memex/cmd/memex@latest"}
	out, err := tool.Call(nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !strings.Contains(out, "go install github.com/dreamware-nz/memex/cmd/memex@latest") {
		t.Errorf("output missing install command: %q", out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("output should be newline-terminated: %q", out)
	}
}
