package vscodecopilot

import "strings"

// Hook subcommand names — must match the verbs accepted by the
// `memex hook ...` CLI dispatcher. VS Code Copilot reuses the Claude
// Code hook protocol verbatim (PreToolUse/PostToolUse/PreCompact/
// SessionStart) so the same dispatcher handles both platforms.
const (
	hookCmdPreToolUse   = "pretooluse"
	hookCmdPostToolUse  = "posttooluse"
	hookCmdPreCompact   = "precompact"
	hookCmdSessionStart = "sessionstart"
)

// preToolUseMatchers lists the tool-name patterns ctx intercepts. Each
// matcher becomes its own entry under hooks.json's PreToolUse array so
// users can disable individual matchers without touching the others.
var preToolUseMatchers = []string{
	"Bash",
	"WebFetch",
	"Read",
	"Grep",
	"mcp__plugin_memex_*",
}

func hookCommand(binaryPath, verb string) string {
	return binaryPath + " hook " + verb
}

func hookEntry(binaryPath, verb, matcher string) map[string]any {
	return map[string]any{
		"matcher": matcher,
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": hookCommand(binaryPath, verb),
			},
		},
	}
}

// hookBlock returns the full hooks map ctx wants installed under the
// "hooks" key of the VS Code Copilot hooks.json. Shape mirrors
// configs/vscode-copilot/hooks.json and src/adapters/vscode-copilot/
// so the TS and Go installs produce identical hooks.json files.
func hookBlock(binaryPath string) map[string]any {
	pre := make([]any, 0, len(preToolUseMatchers))
	for _, m := range preToolUseMatchers {
		pre = append(pre, hookEntry(binaryPath, hookCmdPreToolUse, m))
	}
	return map[string]any{
		"PreToolUse":   pre,
		"PostToolUse":  []any{hookEntry(binaryPath, hookCmdPostToolUse, "")},
		"PreCompact":   []any{hookEntry(binaryPath, hookCmdPreCompact, "")},
		"SessionStart": []any{hookEntry(binaryPath, hookCmdSessionStart, "")},
	}
}

// isMemexHookEntry reports whether cmd belongs to ctx. A command is treated
// as memex-managed when it references binaryPath or contains the canonical
// "memex hook " marker — the latter catches stale entries written by a
// previous install whose binary has since moved.
func isMemexHookEntry(cmd, binaryPath string) bool {
	if cmd == "" {
		return false
	}
	if binaryPath != "" && strings.Contains(cmd, binaryPath) {
		return true
	}
	return strings.Contains(cmd, "memex hook ")
}
