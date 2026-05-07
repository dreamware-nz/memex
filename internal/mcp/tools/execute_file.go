package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileExecutor runs the script at path under the given language in the
// sandbox and returns the captured streams plus exit status.
type FileExecutor interface {
	ExecuteFile(path, language string) (stdout, stderr string, exitCode int, err error)
}

// ExecuteFileTool implements the memex_execute_file MCP tool: optionally
// writes inline `code` to `path`, infers language from extension when
// absent, runs the file in the sandbox, and routes captured output through
// the intent / auto-index / inline policy.
type ExecuteFileTool struct {
	Executor FileExecutor
	Store    IntentIndexer
}

func (t *ExecuteFileTool) Name() string { return "memex_execute_file" }

func (t *ExecuteFileTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"language":{"type":"string"},"code":{"type":"string"},"intent":{"type":"string","description":"Optional. When set, output above the intent-search threshold is indexed into the KB and matched section previews are returned in place of raw output."}},"required":["path"]}`)
}

type executeFileArgs struct {
	Path     string `json:"path"`
	Language string `json:"language"`
	Code     string `json:"code"`
	Intent   string `json:"intent,omitempty"`
}

// extToLang maps file extensions to executor language keys. Matches the
// inference order documented in design.md.
var extToLang = map[string]string{
	".py": "python",
	".js": "javascript",
	".sh": "shell",
}

func (t *ExecuteFileTool) Call(raw json.RawMessage) (string, error) {
	var a executeFileArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", &RPCError{Code: -32602, Message: fmt.Sprintf("invalid arguments: %v", err)}
		}
	}
	if a.Path == "" {
		return "", &RPCError{Code: -32602, Message: "path is required"}
	}

	if a.Code != "" {
		if err := os.WriteFile(a.Path, []byte(a.Code), 0o644); err != nil {
			return "", fmt.Errorf("write code to %s: %w", a.Path, err)
		}
	}

	resolvedPath, err := filepath.Abs(a.Path)
	if err != nil {
		resolvedPath = a.Path
	}

	language := a.Language
	if language == "" {
		ext := strings.ToLower(filepath.Ext(a.Path))
		if l, ok := extToLang[ext]; ok {
			language = l
		} else {
			language = "shell"
		}
	}

	stdout, stderr, exitCode, err := t.Executor.ExecuteFile(a.Path, language)
	if err != nil {
		return "", fmt.Errorf("execute file: %w", err)
	}

	output := stdout
	if stderr != "" {
		output = output + "\n[stderr]\n" + stderr
	}
	if output == "" {
		output = "(no output)"
	}

	sourceTag := "file:" + resolvedPath
	if exitCode != 0 {
		sourceTag = sourceTag + ":error"
	}

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

	if exitCode != 0 {
		_ = indexed
		return "", &execNonZero{output: text, exitCode: exitCode}
	}
	return text, nil
}
