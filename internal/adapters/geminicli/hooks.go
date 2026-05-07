package geminicli

import "strings"

// Hook subcommand names — must match the verbs accepted by the
// `memex hook ...` CLI dispatcher.
const (
	hookCmdPreToolUse   = "pretooluse"
	hookCmdPostToolUse  = "posttooluse"
	hookCmdPreCompact   = "precompact"
	hookCmdSessionStart = "sessionstart"
)

// GeminiHookNames maps memex hook verbs to Gemini CLI's settings.json
// top-level keys. Gemini renames PreToolUse → BeforeTool, PostToolUse →
// AfterTool, PreCompact → PreCompress; SessionStart is unchanged.
var GeminiHookNames = map[string]string{
	hookCmdPreToolUse:   "BeforeTool",
	hookCmdPostToolUse:  "AfterTool",
	hookCmdPreCompact:   "PreCompress",
	hookCmdSessionStart: "SessionStart",
}

// preToolUseMatchers lists the tool-name patterns ctx intercepts. Each
// matcher becomes its own entry under the BeforeTool array so users can
// disable individual matchers without touching the others.
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

// hookBlock returns the hook entries ctx wants installed at the top level
// of ~/.gemini/settings.json. Each value is the array that lives under
// the corresponding GeminiHookNames key.
func hookBlock(binaryPath string) map[string]any {
	pre := make([]any, 0, len(preToolUseMatchers))
	for _, m := range preToolUseMatchers {
		pre = append(pre, hookEntry(binaryPath, hookCmdPreToolUse, m))
	}
	return map[string]any{
		GeminiHookNames[hookCmdPreToolUse]:   pre,
		GeminiHookNames[hookCmdPostToolUse]:  []any{hookEntry(binaryPath, hookCmdPostToolUse, "")},
		GeminiHookNames[hookCmdPreCompact]:   []any{hookEntry(binaryPath, hookCmdPreCompact, "")},
		GeminiHookNames[hookCmdSessionStart]: []any{hookEntry(binaryPath, hookCmdSessionStart, "")},
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
