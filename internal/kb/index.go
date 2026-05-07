package kb

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	sourceKindMarkdown  = "markdown"
	sourceKindPlainText = "plaintext"
)

// IndexResult describes the rows persisted by an indexing call.
type IndexResult struct {
	SourceID    int64
	TotalChunks int
	CodeChunks  int
	Label       string
}

func (s *Store) Index(label, content string) error {
	_, err := s.indexInto(label, content, sourceKindMarkdown)
	return err
}

// IndexMarkdown is the IndexResult-returning sibling of Index. Used by tools
// that want to render the TS-parity pointer text (chunk counts, source hint)
// after indexing markdown content.
func (s *Store) IndexMarkdown(label, content string) (IndexResult, error) {
	return s.indexInto(label, content, sourceKindMarkdown)
}

// IndexPlainText stores arbitrary line/paragraph-chunked content under source.
// Used by the intent-search path on execution output: it returns the new
// SourceID so callers can scope GetDistinctiveTerms to this run.
func (s *Store) IndexPlainText(content, source string) (IndexResult, error) {
	return s.indexInto(source, content, sourceKindPlainText)
}

func (s *Store) indexInto(label, content, kind string) (IndexResult, error) {
	if label == "" {
		return IndexResult{}, errors.New("kb: index requires non-empty label")
	}
	if content == "" {
		return IndexResult{}, errors.New("kb: index requires non-empty content")
	}

	var sections []Section
	switch kind {
	case sourceKindPlainText:
		sections = chunkPlainText(content, defaultChunkBytes)
	default:
		sections = chunkMarkdown(content, defaultChunkBytes)
	}
	if len(sections) == 0 {
		return IndexResult{}, errors.New("kb: index produced no sections")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return IndexResult{}, fmt.Errorf("kb: begin index tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM sources WHERE label = ?`, label); err != nil {
		return IndexResult{}, fmt.Errorf("kb: delete prior source %q: %w", label, err)
	}

	now := time.Now().Unix()
	res, err := tx.Exec(
		`INSERT INTO sources (label, kind, created_at) VALUES (?, ?, ?)`,
		label, kind, now,
	)
	if err != nil {
		return IndexResult{}, fmt.Errorf("kb: insert source %q: %w", label, err)
	}
	sourceID, err := res.LastInsertId()
	if err != nil {
		return IndexResult{}, fmt.Errorf("kb: source id: %w", err)
	}

	stmt, err := tx.Prepare(
		`INSERT INTO sections (source_id, heading, body, byte_size, created_at) VALUES (?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return IndexResult{}, fmt.Errorf("kb: prepare section insert: %w", err)
	}
	defer stmt.Close()

	codeChunks := 0
	for _, sec := range sections {
		if _, err := stmt.Exec(sourceID, sec.Heading, sec.Body, len(sec.Body), now); err != nil {
			return IndexResult{}, fmt.Errorf("kb: insert section: %w", err)
		}
		if looksLikeCode(sec.Body) {
			codeChunks++
		}
	}

	if err := tx.Commit(); err != nil {
		return IndexResult{}, fmt.Errorf("kb: commit index tx: %w", err)
	}
	return IndexResult{
		SourceID:    sourceID,
		TotalChunks: len(sections),
		CodeChunks:  codeChunks,
		Label:       label,
	}, nil
}

// looksLikeCode flags chunks that resemble code so callers can render a
// "(<n> with code)" hint. Heuristic: a code fence anywhere, OR ≥ 30% of
// non-empty lines start with whitespace-indent or a punctuation-heavy token.
func looksLikeCode(body string) bool {
	if strings.Contains(body, "```") || strings.Contains(body, "~~~") {
		return true
	}
	lines := strings.Split(body, "\n")
	nonEmpty, indented := 0, 0
	for _, l := range lines {
		t := strings.TrimRight(l, " \t")
		if t == "" {
			continue
		}
		nonEmpty++
		if strings.HasPrefix(l, "    ") || strings.HasPrefix(l, "\t") {
			indented++
			continue
		}
		if strings.ContainsAny(l, "{};=()") {
			indented++
		}
	}
	if nonEmpty == 0 {
		return false
	}
	return indented*10 >= nonEmpty*3
}

func (s *Store) Delete(label string) error {
	if label == "" {
		return errors.New("kb: delete requires non-empty label")
	}
	if _, err := s.db.Exec(`DELETE FROM sources WHERE label = ?`, label); err != nil {
		return fmt.Errorf("kb: delete source %q: %w", label, err)
	}
	return nil
}
