package tools

import (
	"encoding/json"
	"fmt"
)

// Executor runs a snippet of code in a language-specific sandbox and returns
// the captured streams plus exit status.
type Executor interface {
	Execute(language, code string) (stdout, stderr string, exitCode int, err error)
}

// ExecuteTool implements the memex_execute MCP tool: runs code in a sandbox and
// routes the captured output through the intent / auto-index / inline policy
// matched against context-mode/src/server.ts.
type ExecuteTool struct {
	Executor Executor
	// Store handles plain-text indexing, fallback search, and distinctive-term
	// extraction for the intent-driven and auto-index paths.
	Store IntentIndexer
}

func (t *ExecuteTool) Name() string { return "memex_execute" }

func (t *ExecuteTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"language":{"type":"string"},"code":{"type":"string"},"intent":{"type":"string","description":"Optional. When set, output above the intent-search threshold is indexed into the KB and matched section previews are returned in place of raw output."}},"required":["language","code"]}`)
}

type executeArgs struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Intent   string `json:"intent,omitempty"`
}

func (t *ExecuteTool) Call(raw json.RawMessage) (string, error) {
	var a executeArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", &RPCError{Code: -32602, Message: fmt.Sprintf("invalid arguments: %v", err)}
		}
	}
	if a.Language == "" {
		return "", &RPCError{Code: -32602, Message: "language is required"}
	}
	if a.Code == "" {
		return "", &RPCError{Code: -32602, Message: "code is required"}
	}

	stdout, stderr, exitCode, err := t.Executor.Execute(a.Language, a.Code)
	if err != nil {
		return "", fmt.Errorf("execute: %w", err)
	}

	output := stdout
	if stderr != "" {
		output = output + "\n[stderr]\n" + stderr
	}
	if output == "" {
		output = "(no output)"
	}

	sourceTag := "execute:" + a.Language
	if exitCode != 0 {
		sourceTag = sourceTag + ":error"
	}

	// Error path: if no intent and combined output is large, run the
	// auto-search-for-errors branch using a literal canned query.
	if exitCode != 0 && a.Intent == "" && len(output) > LargeOutputThreshold {
		text, err := intentSearch(output, "errors failures exceptions", sourceTag, t.Store, intentSearchMaxResults)
		if err != nil {
			return "", err
		}
		return "", &execNonZero{output: text, exitCode: exitCode}
	}

	text, indexed, err := routeOutput(output, a.Intent, sourceTag, t.Store)
	if err != nil {
		return "", err
	}

	if exitCode != 0 && !indexed {
		return "", &execNonZero{output: text, exitCode: exitCode}
	}
	if exitCode != 0 && indexed {
		// Indexed error response surfaces text but still flags isError.
		return "", &execNonZero{output: text, exitCode: exitCode}
	}
	return text, nil
}

// execNonZero signals a non-zero exit while carrying the captured output as
// the error message so the MCP server can surface both `isError: true` and
// the program's text in `content[0].text`.
type execNonZero struct {
	output   string
	exitCode int
}

func (e *execNonZero) Error() string { return e.output }
