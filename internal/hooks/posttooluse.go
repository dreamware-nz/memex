package hooks

import (
	"encoding/json"
	"fmt"
	"io"
)

// HandlePostToolUse reads a single PostToolUse JSON event from r, records it
// to the session event stream, and writes an empty `{}` response to w.
//
// Returning a non-nil error signals an internal hook failure — callers wiring
// this to os.Stdin/os.Stdout should exit 1 so agents treat the call as a hard
// hook crash.
func HandlePostToolUse(r io.Reader, w io.Writer) error {
	var ev PostToolUseEvent
	if err := json.NewDecoder(r).Decode(&ev); err != nil {
		return fmt.Errorf("decode posttooluse event: %w", err)
	}
	// TODO(session-tracking): persist event
	_ = ev
	resp := PostToolUseResponse{}
	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		return fmt.Errorf("encode posttooluse response: %w", err)
	}
	return nil
}
