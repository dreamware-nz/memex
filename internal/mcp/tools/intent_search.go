package tools

import (
	"fmt"
	"strings"

	"github.com/dreamware-nz/memex/internal/kb"
)

// IntentIndexer is the dependency the intent-search and auto-index paths
// need: indexing plain-text output, fallback search with optional source
// scoping, and distinctive-term extraction. *kb.Store satisfies it.
type IntentIndexer interface {
	IndexPlainText(content, source string) (kb.IndexResult, error)
	SearchWithFallback(query string, limit int, source string) ([]kb.SearchResult, error)
	GetDistinctiveTerms(sourceID int64) ([]string, error)
}

const intentSearchMaxResults = 5

// intentSearch indexes output under source, runs SearchWithFallback for the
// intent against that source, and renders the matched-preview / no-match
// response strings verbatim from context-mode/src/server.ts §intentSearch.
func intentSearch(output, intent, source string, store IntentIndexer, maxResults int) (string, error) {
	if maxResults <= 0 {
		maxResults = intentSearchMaxResults
	}
	totalLines := strings.Count(output, "\n") + 1
	totalBytes := len(output)

	indexed, err := store.IndexPlainText(output, source)
	if err != nil {
		return "", fmt.Errorf("intent index: %w", err)
	}
	results, err := store.SearchWithFallback(intent, maxResults, source)
	if err != nil {
		return "", fmt.Errorf("intent search: %w", err)
	}
	terms, err := store.GetDistinctiveTerms(indexed.SourceID)
	if err != nil {
		return "", fmt.Errorf("intent terms: %w", err)
	}

	var b strings.Builder

	if len(results) == 0 {
		fmt.Fprintf(&b, "Indexed %d sections from %q into knowledge base.\n", indexed.TotalChunks, source)
		fmt.Fprintf(&b, "No sections matched intent %q in %d-line output (%.1fKB).", intent, totalLines, float64(totalBytes)/1024.0)
		if len(terms) > 0 {
			b.WriteString("\n\n")
			fmt.Fprintf(&b, "Searchable terms: %s", strings.Join(terms, ", "))
		}
		b.WriteString("\n\n")
		b.WriteString("Use memex_search(queries: [...]) to explore the indexed content.")
		return b.String(), nil
	}

	fmt.Fprintf(&b, "Indexed %d sections from %q into knowledge base.\n", indexed.TotalChunks, source)
	fmt.Fprintf(&b, "%d sections matched %q (%d lines, %.1fKB):\n\n", len(results), intent, totalLines, float64(totalBytes)/1024.0)

	for _, r := range results {
		preview := strings.SplitN(r.Body, "\n", 2)[0]
		if len(preview) > 120 {
			preview = preview[:120]
		}
		title := r.Heading
		if title == "" {
			title = "(section)"
		}
		fmt.Fprintf(&b, "  - %s: %s\n", title, preview)
	}

	if len(terms) > 0 {
		b.WriteString("\n")
		fmt.Fprintf(&b, "Searchable terms: %s", strings.Join(terms, ", "))
	}
	b.WriteString("\n\n")
	b.WriteString("Use memex_search(queries: [...]) to retrieve full content of any section.")
	return b.String(), nil
}

// indexStdoutPointer mirrors TS indexStdout(): pointer-style summary used by
// the auto-index branch when output exceeds the large-output threshold and
// no intent was supplied.
func indexStdoutPointer(output, source string, store IntentIndexer) (string, error) {
	indexed, err := store.IndexPlainText(output, source)
	if err != nil {
		return "", fmt.Errorf("auto-index: %w", err)
	}
	return fmt.Sprintf(
		"Indexed %d sections (%d with code) from: %s\nUse memex_search(queries: [\"...\"]) to query this content. Use source: %q to scope results.",
		indexed.TotalChunks, indexed.CodeChunks, indexed.Label, indexed.Label,
	), nil
}
