// Package vscodecopilot is the VS Code Copilot platform adapter —
// patcher for ~/.vscode/extensions/memex/hooks.json. Hook entries
// live under a top-level "hooks" key using the Claude Code hook protocol
// (PreToolUse/PostToolUse/PreCompact/SessionStart) so the same `memex hook`
// dispatcher serves both platforms.
package vscodecopilot
