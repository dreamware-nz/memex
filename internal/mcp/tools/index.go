package tools

import (
	"encoding/json"
	"fmt"

	"github.com/dreamware-nz/memex/internal/kb"
)

// PARITY-CHECKED 2026-05-07: response text now matches context-mode/src/server.ts
// §memex_index — pointer-style with chunk counts and memex_search hint.

// Indexer is the kb-index dependency IndexTool needs.
type Indexer interface {
	Index(label, content string) error
}

// MarkdownIndexer extends Indexer with the IndexResult-returning shape used
// by IndexTool to render the parity pointer text.
type MarkdownIndexer interface {
	IndexMarkdown(label, content string) (kb.IndexResult, error)
}

// IndexTool implements the memex_index MCP tool: it adds a labelled blob of
// content to the knowledge base for later retrieval via memex_search.
type IndexTool struct {
	Indexer MarkdownIndexer
}

func (t *IndexTool) Name() string { return "memex_index" }

func (t *IndexTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"label":{"type":"string"},"content":{"type":"string"}},"required":["label","content"]}`)
}

type indexArgs struct {
	Label   string `json:"label"`
	Content string `json:"content"`
}

func (t *IndexTool) Call(raw json.RawMessage) (string, error) {
	var a indexArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", &RPCError{Code: -32602, Message: fmt.Sprintf("invalid arguments: %v", err)}
		}
	}
	if a.Label == "" {
		return "", &RPCError{Code: -32602, Message: "label is required"}
	}
	if a.Content == "" {
		return "", &RPCError{Code: -32602, Message: "content is required"}
	}
	res, err := t.Indexer.IndexMarkdown(a.Label, a.Content)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"Indexed %d sections (%d with code) from: %s\nUse memex_search(queries: [\"...\"]) to query this content. Use source: %q to scope results.",
		res.TotalChunks, res.CodeChunks, res.Label, res.Label,
	), nil
}
