package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PARITY-CHECKED 2026-05-07: response shape preserves the original simpler
// format — `## <label>` per command, single-call indexing under
// `batch:<labels>`, appended `## <query>` search blocks, 80 KB output cap.
// context-mode/src/server.ts has migrated to a richer "Indexed Sections"
// inventory + "Searchable terms for follow-up" footer; that richer shape is
// deferred (tracked in design.md Open Questions).

const batchOutputCap = 80 * 1024

// BatchTool implements the memex_batch_execute MCP tool: it runs a list of
// shell commands sequentially, indexes the combined output into the KB, and
// appends inline search results so a research agent can complete a round in
// one tool call.
type BatchTool struct {
	Executor Executor
	Indexer  Indexer
	Searcher Searcher
}

func (t *BatchTool) Name() string { return "memex_batch_execute" }

func (t *BatchTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"commands":{"type":"array","minItems":1,"items":{"type":"object","properties":{"label":{"type":"string"},"command":{"type":"string"}},"required":["label","command"]}},"queries":{"type":"array","minItems":1,"items":{"type":"string"}}},"required":["commands","queries"]}`)
}

type batchCommand struct {
	Label   string `json:"label"`
	Command string `json:"command"`
}

type batchArgs struct {
	Commands []batchCommand `json:"commands"`
	Queries  []string       `json:"queries"`
}

func (t *BatchTool) Call(raw json.RawMessage) (string, error) {
	var a batchArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", &RPCError{Code: -32602, Message: fmt.Sprintf("invalid arguments: %v", err)}
		}
	}
	if len(a.Commands) == 0 {
		return "", &RPCError{Code: -32602, Message: "commands is required and must contain at least one entry"}
	}
	if len(a.Queries) == 0 {
		return "", &RPCError{Code: -32602, Message: "queries is required and must contain at least one string"}
	}

	var combined strings.Builder
	labels := make([]string, 0, len(a.Commands))
	for _, c := range a.Commands {
		labels = append(labels, c.Label)
		stdout, stderr, _, err := t.Executor.Execute("shell", c.Command)
		if err != nil {
			return "", fmt.Errorf("execute %s: %w", c.Label, err)
		}
		fmt.Fprintf(&combined, "## %s\n%s\n", c.Label, stdout)
		if stderr != "" {
			fmt.Fprintf(&combined, "[stderr]\n%s\n", stderr)
		}
	}

	indexLabel := "batch:" + strings.Join(labels, ",")
	if err := t.Indexer.Index(indexLabel, combined.String()); err != nil {
		return "", fmt.Errorf("index batch output: %w", err)
	}

	var out strings.Builder
	out.WriteString(combined.String())
	out.WriteString("\n---\n")
	for _, q := range a.Queries {
		fmt.Fprintf(&out, "## %s\n", q)
		hits := t.Searcher.Search([]string{q})
		if len(hits) == 0 {
			out.WriteString("No results.\n")
			continue
		}
		for _, h := range hits {
			fmt.Fprintf(&out, "### %s\n%s\n", h.Heading, h.Snippet)
		}
	}

	return capOutput(out.String(), batchOutputCap), nil
}

func capOutput(s string, max int) string {
	if len(s) <= max {
		return s
	}
	const marker = "\n(output cap reached)"
	cut := max - len(marker)
	if cut < 0 {
		cut = 0
	}
	return s[:cut] + marker
}
