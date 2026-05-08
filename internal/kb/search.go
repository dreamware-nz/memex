package kb

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type SearchResult struct {
	Heading string
	Body    string
	Snippet string
	Source  string
	Rank    float64
}

const snippetTokens = 64

type byRank []SearchResult

func (b byRank) Len() int           { return len(b) }
func (b byRank) Less(i, j int) bool { return b[i].Rank < b[j].Rank }
func (b byRank) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

type rankedHit struct {
	id     int64
	result SearchResult
}

func (s *Store) Search(queries []string, limit int) ([]SearchResult, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("kb: search on nil store")
	}
	if limit <= 0 {
		return []SearchResult{}, nil
	}

	best := map[int64]rankedHit{}
	for _, q := range queries {
		if q == "" {
			continue
		}
		hits, err := queryOne(s.db, q, limit)
		if err != nil {
			return nil, err
		}
		for _, h := range hits {
			cur, ok := best[h.id]
			if !ok || h.result.Rank < cur.result.Rank {
				best[h.id] = h
			}
		}
	}

	merged := make([]SearchResult, 0, len(best))
	for _, h := range best {
		merged = append(merged, h.result)
	}
	sort.Sort(byRank(merged))
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged, nil
}

// SearchWithFallback runs a three-stage search chain: porter (BM25) → trigram
// → fuzzy. It returns the first non-empty stage's results. When source is
// non-empty, every stage scopes its match to sections whose source label
// equals source.
func (s *Store) SearchWithFallback(query string, limit int, source string) ([]SearchResult, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("kb: search on nil store")
	}
	if limit <= 0 {
		return []SearchResult{}, nil
	}
	q := strings.TrimSpace(query)
	if q == "" {
		return []SearchResult{}, nil
	}

	if hits, err := porterStage(s.db, q, limit, source); err != nil {
		return nil, err
	} else if len(hits) > 0 {
		return hits, nil
	}

	if hits, err := trigramStage(s.db, q, limit, source); err != nil {
		return nil, err
	} else if len(hits) > 0 {
		return hits, nil
	}

	if hits, err := fuzzyStage(s.db, q, limit, source); err != nil {
		return nil, err
	} else if len(hits) > 0 {
		return hits, nil
	}

	return []SearchResult{}, nil
}

func porterStage(db *sql.DB, query string, limit int, source string) ([]SearchResult, error) {
	hits, err := scopedFTSQuery(db, sanitizeFTS5Query(query), limit, source)
	if err != nil {
		return nil, err
	}
	return rankedHitsToResults(hits, limit), nil
}

// trigramStage breaks the query into 3-character shingles and OR-matches them
// in FTS5. With the default tokenizer this still recovers near-misses by
// exposing partial tokens as sub-terms.
func trigramStage(db *sql.DB, query string, limit int, source string) ([]SearchResult, error) {
	tokens := extractWordTokens(query)
	var grams []string
	seen := map[string]bool{}
	for _, t := range tokens {
		if len(t) < 3 {
			if !seen[t] && len(t) > 0 {
				seen[t] = true
				grams = append(grams, t)
			}
			continue
		}
		for i := 0; i+3 <= len(t); i++ {
			g := t[i : i+3]
			if !seen[g] {
				seen[g] = true
				grams = append(grams, g)
			}
		}
	}
	if len(grams) == 0 {
		return nil, nil
	}
	var quoted []string
	for _, g := range grams {
		quoted = append(quoted, `"`+strings.ReplaceAll(g, `"`, `""`)+`"`)
	}
	expr := strings.Join(quoted, " OR ")
	hits, err := scopedFTSQuery(db, expr, limit, source)
	if err != nil {
		return nil, err
	}
	return rankedHitsToResults(hits, limit), nil
}

// fuzzyStage falls back to LIKE substring matching across each query token.
// Results are ranked by the count of matched tokens (more = better).
func fuzzyStage(db *sql.DB, query string, limit int, source string) ([]SearchResult, error) {
	tokens := extractWordTokens(query)
	var keep []string
	for _, t := range tokens {
		if len(t) >= 3 {
			keep = append(keep, t)
		}
	}
	if len(keep) == 0 {
		return nil, nil
	}

	var (
		clauses []string
		args    []interface{}
	)
	for _, t := range keep {
		clauses = append(clauses, "LOWER(s.body) LIKE ?")
		args = append(args, "%"+strings.ToLower(t)+"%")
	}
	where := "(" + strings.Join(clauses, " OR ") + ")"
	if source != "" {
		where += " AND sr.label = ?"
		args = append(args, source)
	}
	args = append(args, limit)

	stmt := `
SELECT s.id, s.heading, s.body, sr.label
FROM sections s
JOIN sources sr ON sr.id = s.source_id
WHERE ` + where + `
LIMIT ?`

	rows, err := db.Query(stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("kb: fuzzy stage: %w", err)
	}
	defer rows.Close()

	var hits []rankedHit
	for rows.Next() {
		var (
			id      int64
			heading sql.NullString
			body    string
			label   string
		)
		if err := rows.Scan(&id, &heading, &body, &label); err != nil {
			return nil, fmt.Errorf("kb: fuzzy scan: %w", err)
		}
		lc := strings.ToLower(body)
		matches := 0
		for _, t := range keep {
			if strings.Contains(lc, strings.ToLower(t)) {
				matches++
			}
		}
		hits = append(hits, rankedHit{
			id: id,
			result: SearchResult{
				Heading: heading.String,
				Body:    body,
				Snippet: snippetFor(body, keep),
				Source:  label,
				Rank:    float64(matches),
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kb: fuzzy rows: %w", err)
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].result.Rank > hits[j].result.Rank })
	return rankedHitsToResults(hits, limit), nil
}

func scopedFTSQuery(db *sql.DB, matchExpr string, limit int, source string) ([]rankedHit, error) {
	stmt := `
SELECT s.id,
       s.heading,
       s.body,
       sr.label,
       snippet(sections_fts, -1, '<b>', '</b>', '…', ?),
       sections_fts.rank
FROM sections_fts
JOIN sections s  ON s.id = sections_fts.rowid
JOIN sources  sr ON sr.id = s.source_id
WHERE sections_fts MATCH ?`
	args := []interface{}{snippetTokens, matchExpr}
	if source != "" {
		stmt += " AND sr.label = ?"
		args = append(args, source)
	}
	stmt += `
ORDER BY sections_fts.rank
LIMIT ?`
	args = append(args, limit)

	rows, err := db.Query(stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("kb: fts query %q: %w", matchExpr, err)
	}
	defer rows.Close()

	var out []rankedHit
	for rows.Next() {
		var (
			id      int64
			heading sql.NullString
			body    string
			source  string
			snippet string
			rank    float64
		)
		if err := rows.Scan(&id, &heading, &body, &source, &snippet, &rank); err != nil {
			return nil, fmt.Errorf("kb: fts scan: %w", err)
		}
		out = append(out, rankedHit{
			id: id,
			result: SearchResult{
				Heading: heading.String,
				Body:    body,
				Snippet: snippet,
				Source:  source,
				Rank:    -rank,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kb: fts rows: %w", err)
	}
	return out, nil
}

func rankedHitsToResults(hits []rankedHit, limit int) []SearchResult {
	if len(hits) == 0 {
		return []SearchResult{}
	}
	out := make([]SearchResult, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.result)
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// snippetFor produces a short window of the body around the first matched
// token, mirroring the FTS5 snippet shape for non-FTS stages.
func snippetFor(body string, tokens []string) string {
	lc := strings.ToLower(body)
	for _, t := range tokens {
		idx := strings.Index(lc, strings.ToLower(t))
		if idx < 0 {
			continue
		}
		start := idx - 40
		if start < 0 {
			start = 0
		}
		end := idx + len(t) + 80
		if end > len(body) {
			end = len(body)
		}
		out := body[start:end]
		if start > 0 {
			out = "…" + out
		}
		if end < len(body) {
			out = out + "…"
		}
		return out
	}
	if len(body) > 160 {
		return body[:160] + "…"
	}
	return body
}

func queryOne(db *sql.DB, query string, limit int) ([]rankedHit, error) {
	const sqlStmt = `
SELECT s.id,
       s.heading,
       s.body,
       sr.label,
       snippet(sections_fts, -1, '<b>', '</b>', '…', ?),
       sections_fts.rank
FROM sections_fts
JOIN sections s  ON s.id = sections_fts.rowid
JOIN sources  sr ON sr.id = s.source_id
WHERE sections_fts MATCH ?
ORDER BY sections_fts.rank
LIMIT ?`

	rows, err := db.Query(sqlStmt, snippetTokens, sanitizeFTS5Query(query), limit)
	if err != nil {
		return nil, fmt.Errorf("kb: search query %q: %w", query, err)
	}
	defer rows.Close()

	var out []rankedHit
	for rows.Next() {
		var (
			id      int64
			heading sql.NullString
			body    string
			source  string
			snippet string
			rank    float64
		)
		if err := rows.Scan(&id, &heading, &body, &source, &snippet, &rank); err != nil {
			return nil, fmt.Errorf("kb: search scan: %w", err)
		}
		out = append(out, rankedHit{
			id: id,
			result: SearchResult{
				Heading: heading.String,
				Body:    body,
				Snippet: snippet,
				Source:  source,
				Rank:    -rank,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kb: search rows: %w", err)
	}
	return out, nil
}
