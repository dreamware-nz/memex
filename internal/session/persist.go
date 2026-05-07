package session

import "time"

// kindToolCall is the stable `events.kind` discriminator for hook-captured
// tool I/O. The specific tool name lives inside `payload_json.tool_name`.
const kindToolCall = "tool_call"

// PersistToolCall inserts one `events` row capturing a single Claude Code
// PostToolUse hook firing. The full HookInput is preserved verbatim in
// `payload_json` so downstream analytics and snapshot consumers can recover
// any field without schema migrations.
func PersistToolCall(db *DB, input HookInput) error {
	return insertEvent(db, kindToolCall, time.Now().UnixMilli(), input)
}
