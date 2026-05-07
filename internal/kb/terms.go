package kb

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const (
	distinctiveTermsCap = 40
	minTermLength       = 3
)

var stopWords = map[string]bool{
	"the": true, "and": true, "for": true, "are": true, "but": true, "not": true,
	"you": true, "all": true, "any": true, "can": true, "had": true, "her": true,
	"was": true, "one": true, "our": true, "out": true, "his": true, "has": true,
	"him": true, "she": true, "they": true, "them": true, "with": true, "from": true,
	"this": true, "that": true, "what": true, "have": true, "will": true, "your": true,
	"been": true, "were": true, "there": true, "their": true, "would": true, "could": true,
	"should": true, "about": true, "which": true, "when": true, "than": true, "then": true,
	"into": true, "more": true, "some": true, "such": true, "also": true, "only": true,
	"over": true, "very": true, "just": true, "like": true, "each": true, "much": true,
	"those": true, "these": true, "where": true, "while": true, "after": true, "before": true,
	"because": true, "through": true,
}

// extractWordTokens splits a string into lowercased word tokens on any
// non-letter/non-digit boundary. It does not filter by length or stop-words —
// callers do that themselves.
func extractWordTokens(s string) []string {
	if s == "" {
		return nil
	}
	var (
		out []string
		cur strings.Builder
	)
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, strings.ToLower(cur.String()))
			cur.Reset()
		}
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}

// GetDistinctiveTerms returns up to 40 single-token terms ranked by
// per-source distinctiveness. Tokens shorter than 3 characters and built-in
// stop-words are excluded. An unknown sourceID returns an empty slice.
func (s *Store) GetDistinctiveTerms(sourceID int64) ([]string, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("kb: terms on nil store")
	}

	rows, err := s.db.Query(`SELECT body FROM sections WHERE source_id = ?`, sourceID)
	if err != nil {
		return nil, fmt.Errorf("kb: source bodies: %w", err)
	}
	defer rows.Close()

	sourceCounts := map[string]int{}
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return nil, fmt.Errorf("kb: scan body: %w", err)
		}
		for _, tok := range extractWordTokens(body) {
			if len(tok) < minTermLength {
				continue
			}
			if stopWords[tok] {
				continue
			}
			sourceCounts[tok]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kb: source rows: %w", err)
	}
	if len(sourceCounts) == 0 {
		return []string{}, nil
	}

	globalCounts, err := globalTokenCounts(s, sourceCounts)
	if err != nil {
		return nil, err
	}

	type scored struct {
		term  string
		score float64
		count int
	}
	ranked := make([]scored, 0, len(sourceCounts))
	for tok, n := range sourceCounts {
		g := globalCounts[tok]
		if g < n {
			g = n
		}
		// Score favours tokens that appear often in this source but are rare
		// elsewhere. The +1 keeps the denominator non-zero.
		score := float64(n) / float64(1+g-n)
		ranked = append(ranked, scored{term: tok, score: score, count: n})
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			if ranked[i].count == ranked[j].count {
				return ranked[i].term < ranked[j].term
			}
			return ranked[i].count > ranked[j].count
		}
		return ranked[i].score > ranked[j].score
	})

	if len(ranked) > distinctiveTermsCap {
		ranked = ranked[:distinctiveTermsCap]
	}
	out := make([]string, 0, len(ranked))
	for _, r := range ranked {
		out = append(out, r.term)
	}
	return out, nil
}

// globalTokenCounts computes how often each candidate token appears across the
// entire sections table, for distinctiveness scoring. It scans bodies once
// and tokenises in Go to avoid relying on the FTS5 dictionary tables.
func globalTokenCounts(s *Store, candidates map[string]int) (map[string]int, error) {
	rows, err := s.db.Query(`SELECT body FROM sections`)
	if err != nil {
		return nil, fmt.Errorf("kb: global bodies: %w", err)
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return nil, fmt.Errorf("kb: scan global body: %w", err)
		}
		for _, tok := range extractWordTokens(body) {
			if _, want := candidates[tok]; !want {
				continue
			}
			out[tok]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kb: global rows: %w", err)
	}
	return out, nil
}
