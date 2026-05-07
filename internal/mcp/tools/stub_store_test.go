package tools

import "github.com/dreamware-nz/memex/internal/kb"

// stubStore implements IntentIndexer for tests. It records IndexPlainText
// calls and lets tests pre-load search results and distinctive terms.
type stubStore struct {
	indexErr  error
	searchErr error
	termsErr  error

	results map[string][]kb.SearchResult
	terms   map[int64][]string

	nextID int64
	calls  []struct {
		Source  string
		Content string
	}
	searches []struct {
		Query  string
		Source string
	}
}

func newStubStore() *stubStore {
	return &stubStore{
		results: map[string][]kb.SearchResult{},
		terms:   map[int64][]string{},
	}
}

func (s *stubStore) IndexPlainText(content, source string) (kb.IndexResult, error) {
	if s.indexErr != nil {
		return kb.IndexResult{}, s.indexErr
	}
	s.calls = append(s.calls, struct {
		Source  string
		Content string
	}{source, content})
	s.nextID++
	chunks := 1 + len(content)/2048
	return kb.IndexResult{
		SourceID:    s.nextID,
		TotalChunks: chunks,
		CodeChunks:  0,
		Label:       source,
	}, nil
}

func (s *stubStore) SearchWithFallback(query string, limit int, source string) ([]kb.SearchResult, error) {
	if s.searchErr != nil {
		return nil, s.searchErr
	}
	s.searches = append(s.searches, struct {
		Query  string
		Source string
	}{query, source})
	if r, ok := s.results[query]; ok {
		if limit > 0 && len(r) > limit {
			return r[:limit], nil
		}
		return r, nil
	}
	return []kb.SearchResult{}, nil
}

func (s *stubStore) GetDistinctiveTerms(sourceID int64) ([]string, error) {
	if s.termsErr != nil {
		return nil, s.termsErr
	}
	if t, ok := s.terms[sourceID]; ok {
		return t, nil
	}
	return []string{}, nil
}
