package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// recapStub is the placeholder prepended to the routing block when
// `--continue` is in effect. Replaced by a real prior-session summary once
// session-tracking lands.
const recapStub = "[prior session resumed]\n\n"

// platformGeminiCLI matches the value adapter-detect writes to
// MEMEX_PLATFORM when the host agent is the Gemini CLI.
const platformGeminiCLI = "gemini-cli"

// HandleSessionStart reads a single SessionStart JSON event from r, builds the
// routing-instructions system prompt (optionally prefixed with a
// prior-session recap when continueSession is true), and writes a
// platform-shaped response to w.
//
// The response field name is selected from MEMEX_PLATFORM: Gemini CLI
// expects `hookSpecificOutput.system_context`, every other platform (and the
// unset default) expects `systemPrompt`.
//
// Returning a non-nil error signals an internal hook failure — callers wiring
// this to os.Stdin/os.Stdout should exit 1 so agents treat the call as a hard
// hook crash.
func HandleSessionStart(r io.Reader, w io.Writer, continueSession bool) error {
	var ev SessionStartEvent
	if err := json.NewDecoder(r).Decode(&ev); err != nil {
		return fmt.Errorf("decode sessionstart event: %w", err)
	}

	prompt := RoutingBlock()
	if continueSession {
		prompt = recapStub + prompt
	}

	resp := SessionStartResponse{}
	if os.Getenv("MEMEX_PLATFORM") == platformGeminiCLI {
		resp.HookSpecificOutput = &SessionStartOutput{SystemContext: prompt}
	} else {
		resp.SystemPrompt = prompt
	}

	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		return fmt.Errorf("encode sessionstart response: %w", err)
	}
	return nil
}
