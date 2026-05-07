package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// HandlePreCompact reads a single PreCompact JSON event from r, attempts to
// flush any buffered session events to durable storage, and writes an empty
// `{}` response to w. The hook is advisory — agents fire it asynchronously
// and do not block on the response — so the handler always responds quickly
// and never propagates flush errors.
//
// Returning a non-nil error signals an internal hook failure (decode or
// encode). Callers wiring this to os.Stdin/os.Stdout should exit 1 so agents
// treat the call as a hard hook crash.
func HandlePreCompact(r io.Reader, w io.Writer) error {
	var ev PreCompactEvent
	if err := json.NewDecoder(r).Decode(&ev); err != nil {
		return fmt.Errorf("decode precompact event: %w", err)
	}
	if os.Getenv("CTX_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "precompact: session=%s trigger=%s usage=%s\n",
			ev.SessionID, ev.Trigger, ev.ContextWindowUsage)
	}
	// TODO(session-tracking): flush session events to durable store
	resp := PreCompactResponse{}
	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		return fmt.Errorf("encode precompact response: %w", err)
	}
	return nil
}
