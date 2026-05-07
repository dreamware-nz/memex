package session

// ToolOutput mirrors the optional `tool_output` field of the Claude Code
// PostToolUse hook payload.
type ToolOutput struct {
	IsError bool `json:"is_error"`
}

// HookInput mirrors the JSON shape Claude Code writes to a PostToolUse hook's
// stdin. Field names match the wire format via snake_case JSON tags so
// `json.Unmarshal` round-trips without a translation layer.
type HookInput struct {
	ToolName     string         `json:"tool_name"`
	ToolInput    map[string]any `json:"tool_input"`
	ToolResponse string         `json:"tool_response"`
	ToolOutput   *ToolOutput    `json:"tool_output,omitempty"`
}
