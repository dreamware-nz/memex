// Package cursor is the Cursor platform adapter — patcher for
// ~/.cursor/extensions/memex/hooks.json. Cursor is a VS Code fork
// that uses ~/.cursor/ as its config root and CURSOR_TRACE_ID / CURSOR_CLI
// as its detection env vars. The hook block shape is identical to VS Code
// Copilot (same wire protocol) so the same `memex hook` dispatcher serves
// both platforms.
package cursor
