package kb

import (
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"TestSearch_ReturnsRankedResults", testSearchReturnsRankedResults},
		{"TestSearch_Dedup", testSearchDedup},
		{"TestSearch_SnippetPopulated", testSearchSnippetPopulated},
		{"TestSearch_EmptyStore", testSearchEmptyStore},
		{"TestSearchWithFallback_PorterStage", testFallbackPorterStage},
		{"TestSearchWithFallback_FuzzyStage", testFallbackFuzzyStage},
		{"TestSearchWithFallback_SourceScoping", testFallbackSourceScoping},
		{"TestSearchWithFallback_NoMatches", testFallbackNoMatches},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func testFallbackPorterStage(t *testing.T) {
	s := newStore(t)
	if _, err := s.IndexPlainText("hello world from the kb", "src-a"); err != nil {
		t.Fatalf("IndexPlainText: %v", err)
	}
	results, err := s.SearchWithFallback("hello world", 5, "")
	if err != nil {
		t.Fatalf("SearchWithFallback: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("results = 0, want >= 1 from porter stage")
	}
	if results[0].Source != "src-a" {
		t.Fatalf("Source = %q, want src-a", results[0].Source)
	}
}

func testFallbackFuzzyStage(t *testing.T) {
	s := newStore(t)
	// Single token without any near-match for porter or trigram tokens that
	// would re-match. Then a query that shares no FTS tokens but contains a
	// substring match.
	if _, err := s.IndexPlainText("application encountered an unexpected fault", "src-fuzzy"); err != nil {
		t.Fatalf("IndexPlainText: %v", err)
	}
	results, err := s.SearchWithFallback("faul", 5, "")
	if err != nil {
		t.Fatalf("SearchWithFallback: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("results = 0, want >= 1 from fallback stages")
	}
	if results[0].Source != "src-fuzzy" {
		t.Fatalf("Source = %q, want src-fuzzy", results[0].Source)
	}
}

func testFallbackSourceScoping(t *testing.T) {
	s := newStore(t)
	if _, err := s.IndexPlainText("token zqxlepton matches here", "execute:shell"); err != nil {
		t.Fatalf("IndexPlainText shell: %v", err)
	}
	if _, err := s.IndexPlainText("token zqxlepton elsewhere too", "execute:python"); err != nil {
		t.Fatalf("IndexPlainText python: %v", err)
	}
	results, err := s.SearchWithFallback("zqxlepton", 5, "execute:shell")
	if err != nil {
		t.Fatalf("SearchWithFallback: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("results = 0, want >= 1")
	}
	for _, r := range results {
		if r.Source != "execute:shell" {
			t.Fatalf("Source = %q, want execute:shell", r.Source)
		}
	}
}

func testFallbackNoMatches(t *testing.T) {
	s := newStore(t)
	if _, err := s.IndexPlainText("ordinary content with nothing peculiar", "src"); err != nil {
		t.Fatalf("IndexPlainText: %v", err)
	}
	results, err := s.SearchWithFallback("zzzzqqqqyyyy", 5, "")
	if err != nil {
		t.Fatalf("SearchWithFallback: %v", err)
	}
	if results == nil {
		t.Fatalf("results = nil, want non-nil empty slice")
	}
	if len(results) != 0 {
		t.Fatalf("results len = %d, want 0", len(results))
	}
}

func testSearchReturnsRankedResults(t *testing.T) {
	s := newStore(t)

	if err := s.Index("alpha", "# Alpha\n\nzqxlepton orbital dynamics\n"); err != nil {
		t.Fatalf("Index alpha: %v", err)
	}
	if err := s.Index("beta", "# Beta\n\nunrelated material about turtles\n"); err != nil {
		t.Fatalf("Index beta: %v", err)
	}

	results, err := s.Search([]string{"zqxlepton"}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) < 1 {
		t.Fatalf("results = %d, want >= 1", len(results))
	}
	if results[0].Source != "alpha" {
		t.Fatalf("Source = %q, want %q", results[0].Source, "alpha")
	}
}

func testSearchDedup(t *testing.T) {
	s := newStore(t)

	const doc = "# Heading\n\nfoobar zqxlepton appears together\n"
	if err := s.Index("dup-doc", doc); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := s.Search([]string{"foobar", "zqxlepton"}, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	count := 0
	for _, r := range results {
		if r.Source == "dup-doc" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("dup-doc appearances = %d, want 1 (results=%d)", count, len(results))
	}
}

func testSearchSnippetPopulated(t *testing.T) {
	s := newStore(t)

	if err := s.Index("snip", "# H\n\nthe magic word is yzwflarp inside this body\n"); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := s.Search([]string{"yzwflarp"}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) < 1 {
		t.Fatalf("results = %d, want >= 1", len(results))
	}
	if strings.TrimSpace(results[0].Snippet) == "" {
		t.Fatalf("Snippet empty, want non-empty")
	}
}

func testSearchEmptyStore(t *testing.T) {
	s := newStore(t)

	results, err := s.Search([]string{"anything"}, 5)
	if err != nil {
		t.Fatalf("Search empty: %v", err)
	}
	if results == nil {
		t.Fatalf("results = nil, want non-nil empty slice")
	}
	if len(results) != 0 {
		t.Fatalf("results len = %d, want 0", len(results))
	}
}
