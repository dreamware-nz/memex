package hooks

import (
	"encoding/json"
	"fmt"
	"io"
)

// HandlePreToolUse reads a single PreToolUse JSON event from r, runs it
// through the router, and writes the formatted response to w.
//
// A panic inside Route or formatClaudeCode is recovered and degrades to an
// empty `{}` response so a hook crash never blocks a tool call. Decode errors
// (malformed JSON on stdin) DO propagate — callers should exit non-zero so the
// host distinguishes "hook crashed" from "hook said allow".
func HandlePreToolUse(r io.Reader, w io.Writer) (err error) {
	var ev PreToolUseEvent
	if decodeErr := json.NewDecoder(r).Decode(&ev); decodeErr != nil {
		return fmt.Errorf("decode pretooluse event: %w", decodeErr)
	}

	defer func() {
		if rec := recover(); rec != nil {
			// Routing must never block a tool call. Emit `{}` and report success.
			_ = json.NewEncoder(w).Encode(&PreToolUseResponse{})
			err = nil
		}
	}()

	decision := Route(ev.ToolName, ev.ToolInput, ev.SessionID)
	resp := formatClaudeCode(decision)
	if encErr := json.NewEncoder(w).Encode(&resp); encErr != nil {
		return fmt.Errorf("encode pretooluse response: %w", encErr)
	}
	return nil
}
