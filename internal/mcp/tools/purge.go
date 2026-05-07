package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PurgeTool implements memex_purge: deletes the KB SQLite file and the
// session analytics SQLite file. Destructive and irreversible — requires
// `confirm: true` in the args. Without confirm it returns a warning listing
// what *would* be deleted.
type PurgeTool struct {
	// Paths are the absolute file paths to remove. Wired in by NewMCPCmd
	// from the same defaultKBPath / defaultAnalyticsPath helpers used by
	// the CLI so MCP and CLI agree on what gets purged.
	Paths []string
}

func (t *PurgeTool) Name() string { return "memex_purge" }

func (t *PurgeTool) Description() string {
	return "Permanently delete the memex knowledge base and session analytics SQLite files. Requires confirm=true. Irreversible."
}

func (t *PurgeTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"confirm":{"type":"boolean"}}}`)
}

type purgeArgs struct {
	Confirm bool `json:"confirm"`
}

func (t *PurgeTool) Call(raw json.RawMessage) (string, error) {
	var a purgeArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", &RPCError{Code: -32602, Message: fmt.Sprintf("invalid arguments: %v", err)}
		}
	}
	if !a.Confirm {
		var b strings.Builder
		b.WriteString("memex_purge requires confirm=true. The following would be deleted:\n")
		for _, p := range t.Paths {
			fmt.Fprintf(&b, "  - %s\n", p)
		}
		return b.String(), nil
	}

	var deleted, missing []string
	for _, p := range t.Paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); os.IsNotExist(err) {
			missing = append(missing, p)
			continue
		} else if err != nil {
			return "", fmt.Errorf("stat %s: %w", p, err)
		}
		if err := os.Remove(p); err != nil {
			return "", fmt.Errorf("remove %s: %w", p, err)
		}
		deleted = append(deleted, p)
	}

	var b strings.Builder
	if len(deleted) == 0 && len(missing) == len(t.Paths) {
		b.WriteString("Nothing to purge — all paths already absent.\n")
		return b.String(), nil
	}
	if len(deleted) > 0 {
		b.WriteString("Deleted:\n")
		for _, p := range deleted {
			fmt.Fprintf(&b, "  - %s\n", p)
		}
	}
	if len(missing) > 0 {
		b.WriteString("Already absent:\n")
		for _, p := range missing {
			fmt.Fprintf(&b, "  - %s\n", p)
		}
	}
	return b.String(), nil
}
