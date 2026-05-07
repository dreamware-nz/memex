package tools

import "encoding/json"

// UpgradeTool implements memex_upgrade. It does not execute the upgrade —
// it returns the canonical `go install` command for the agent (or user) to
// run. This mirrors `memex upgrade` and avoids elevating MCP from a tool
// surface to a self-modifying installer.
type UpgradeTool struct {
	// Command is the install command emitted in the response. NewMCPCmd
	// wires the canonical "go install github.com/dreamware-nz/memex/cmd/memex@latest"
	// string so MCP and CLI stay in lockstep.
	Command string
}

func (t *UpgradeTool) Name() string { return "memex_upgrade" }

func (t *UpgradeTool) Description() string {
	return "Return the shell command to install the latest memex from source. The agent or user should execute the command separately."
}

func (t *UpgradeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (t *UpgradeTool) Call(_ json.RawMessage) (string, error) {
	return t.Command + "\n", nil
}
