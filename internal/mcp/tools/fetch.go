package tools

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// PARITY-CHECKED 2026-05-07: Content-Type-driven HTML→Markdown / plain-text
// passthrough matches context-mode/src/server.ts §memex_fetch_and_index.
// Confirmation text uses the simpler "Fetched and indexed <label> (<bytes>
// bytes Markdown)" wording vs the TS "Cached: ... — N sections, indexed
// <ago>..." rich shape; cache-aware response is deferred and tracked in
// design.md Open Questions.

// Fetcher is the HTTP-GET dependency FetchTool needs. The concrete
// implementation lives in internal/fetch and applies timeout, redirect, and
// size caps; tests inject a stub.
type Fetcher interface {
	Get(url string) (body []byte, contentType string, err error)
}

// Converter turns an HTML document into Markdown. The concrete
// implementation lives in internal/htmltomd; tests inject a stub.
type Converter interface {
	Convert(html string) (string, error)
}

// FetchTool implements the memex_fetch_and_index MCP tool: GET a URL, convert
// HTML to Markdown when applicable, index the result, and return a concise
// confirmation.
type FetchTool struct {
	Fetcher   Fetcher
	Converter Converter
	Indexer   Indexer
}

func (t *FetchTool) Name() string { return "memex_fetch_and_index" }

func (t *FetchTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"url":{"type":"string"},"label":{"type":"string"}},"required":["url"]}`)
}

type fetchArgs struct {
	URL   string `json:"url"`
	Label string `json:"label"`
}

func (t *FetchTool) Call(raw json.RawMessage) (string, error) {
	var a fetchArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a); err != nil {
			return "", &RPCError{Code: -32602, Message: fmt.Sprintf("invalid arguments: %v", err)}
		}
	}
	if a.URL == "" {
		return "", &RPCError{Code: -32602, Message: "url is required"}
	}
	parsed, err := url.Parse(a.URL)
	if err != nil || parsed.Host == "" {
		return "", &RPCError{Code: -32602, Message: fmt.Sprintf("invalid url: %q", a.URL)}
	}

	body, contentType, err := t.Fetcher.Get(a.URL)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", a.URL, err)
	}

	var content string
	if strings.Contains(contentType, "text/html") {
		md, err := t.Converter.Convert(string(body))
		if err != nil {
			return "", fmt.Errorf("convert html: %w", err)
		}
		content = md
	} else {
		content = string(body)
	}

	label := a.Label
	if label == "" {
		label = parsed.Host + parsed.Path
	}

	if err := t.Indexer.Index(label, content); err != nil {
		return "", err
	}
	return fmt.Sprintf("Fetched and indexed %s (%d bytes Markdown)", label, len(content)), nil
}
