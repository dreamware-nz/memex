package codex

import "strings"

// Hook subcommand names — must match the verbs accepted by the
// `memex hook ...` CLI dispatcher.
const (
	hookCmdPreToolUse   = "pretooluse"
	hookCmdPostToolUse  = "posttooluse"
	hookCmdPreCompact   = "precompact"
	hookCmdSessionStart = "sessionstart"
)

// preToolUseMatchers mirrors Claude Code so a user switching builds sees
// the same intercept set on both platforms.
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

// hookBlock returns the Codex CLI hooks map ctx wants installed. Shape
// matches Claude Code's settings.json so a user switching between hosts
// gets the same intercepts and so the merge/uninstall code can be shared.
//
// TODO(upstream): updatedInput not yet supported in Codex CLI
// (openai/codex#18491). The PreToolUse handler may allow or deny but
// MUST NOT emit updatedInput until upstream resolves the issue.
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

// isMemexHookEntry reports whether cmd belongs to ctx. Mirrors the Claude
// Code adapter's check so stale entries from a moved binary are still
// detected via the canonical "memex hook " marker.
func isMemexHookEntry(cmd, binaryPath string) bool {
	if cmd == "" {
		return false
	}
	if binaryPath != "" && strings.Contains(cmd, binaryPath) {
		return true
	}
	return strings.Contains(cmd, "memex hook ")
}
