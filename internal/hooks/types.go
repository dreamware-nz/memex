package hooks

// Legacy decision strings for the cross-platform `decision` field. An empty
// string means "allow"; "deny" blocks the tool call. New code SHOULD prefer
// PreToolUseHookOutput.PermissionDecision via HookSpecificOutput, but the
// legacy field stays so older agents (and integration tests) keep working.
const (
	DecisionAllow string = ""
	DecisionDeny  string = "deny"
)

// PreToolUseEvent is the JSON envelope agents send on stdin before a tool call.
// All fields are optional so unknown agents and future fields parse cleanly.
type PreToolUseEvent struct {
	SessionID string         `json:"session_id,omitempty"`
	Cwd       string         `json:"cwd,omitempty"`
	ToolName  string         `json:"tool_name,omitempty"`
	ToolInput map[string]any `json:"tool_input,omitempty"`
}

// PreToolUseHookOutput is the Claude Code-shaped response payload nested under
// `hookSpecificOutput`. Exactly one of PermissionDecision, ModifiedToolInput,
// or AdditionalContext is populated for any single decision.
type PreToolUseHookOutput struct {
	HookEventName            string         `json:"hookEventName,omitempty"`
	PermissionDecision       string         `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string         `json:"permissionDecisionReason,omitempty"`
	ModifiedToolInput        map[string]any `json:"modifiedToolInput,omitempty"`
	AdditionalContext        string         `json:"additionalContext,omitempty"`
}

// PreToolUseResponse is the JSON the hook writes to stdout to convey its
// decision. An all-zero value marshals to `{}` which agents read as "allow".
// HookSpecificOutput is the Claude Code-preferred channel; the top-level
// fields stay for cross-platform / legacy use.
type PreToolUseResponse struct {
	Decision           string                `json:"decision,omitempty"`
	Reason             string                `json:"reason,omitempty"`
	UpdatedInput       map[string]any        `json:"updatedInput,omitempty"`
	HookSpecificOutput *PreToolUseHookOutput `json:"hookSpecificOutput,omitempty"`
}

// PostToolUseEvent is the JSON envelope agents send on stdin after a tool call
// completes. All fields are optional so unknown agents and future fields parse
// cleanly.
type PostToolUseEvent struct {
	SessionID     string         `json:"session_id,omitempty"`
	CWD           string         `json:"cwd,omitempty"`
	HookEventName string         `json:"hook_event_name,omitempty"`
	ToolName      string         `json:"tool_name,omitempty"`
	ToolInput     map[string]any `json:"tool_input,omitempty"`
	ToolOutput    string         `json:"tool_output,omitempty"`
}

// PostToolUseResponse is the JSON the hook writes to stdout. The handler does
// not modify tool output yet, so a zero value marshals to `{}`.
type PostToolUseResponse struct{}

// PreCompactEvent is the JSON envelope agents send on stdin before they
// compact (or compress, in Gemini CLI vocabulary) the conversation context
// window. The hook is advisory — agents do not block on the response — so all
// fields are optional and unknown fields parse cleanly.
type PreCompactEvent struct {
	SessionID          string `json:"session_id,omitempty"`
	Trigger            string `json:"trigger,omitempty"`
	ContextWindowUsage string `json:"context_window_usage,omitempty"`
}

// PreCompactResponse is the JSON the hook writes to stdout. The handler is
// advisory and has no side-effects to communicate, so a zero value marshals
// to `{}`.
type PreCompactResponse struct{}

// SessionStartEvent is the JSON envelope agents send on stdin when a session
// begins. All fields are optional so unknown agents and future fields parse
// cleanly.
type SessionStartEvent struct {
	SessionID      string `json:"session_id,omitempty"`
	TranscriptPath string `json:"transcript_path,omitempty"`
	CWD            string `json:"cwd,omitempty"`
	Model          string `json:"model,omitempty"`
}

// SessionStartOutput carries the Gemini-CLI-shaped system context payload.
type SessionStartOutput struct {
	SystemContext string `json:"system_context,omitempty"`
}

// SessionStartResponse is the JSON the hook writes to stdout. Exactly one of
// SystemPrompt (Claude Code) or HookSpecificOutput (Gemini CLI) is populated.
type SessionStartResponse struct {
	SystemPrompt       string              `json:"systemPrompt,omitempty"`
	HookSpecificOutput *SessionStartOutput `json:"hookSpecificOutput,omitempty"`
}
