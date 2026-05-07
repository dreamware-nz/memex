package kb

import (
	"strings"
	"testing"
)

func TestDistinctiveTerms(t *testing.T) {
	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"TestDistinctiveTerms_ReturnsTokens", testDistinctiveTermsReturnsTokens},
		{"TestDistinctiveTerms_BoundedAt40", testDistinctiveTermsBounded},
		{"TestDistinctiveTerms_StopWordsFiltered", testDistinctiveTermsStopWords},
		{"TestDistinctiveTerms_UnknownSource", testDistinctiveTermsUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}

func testDistinctiveTermsReturnsTokens(t *testing.T) {
	s := newStore(t)
	res, err := s.IndexPlainText("alpha bravo charlie alpha bravo alpha", "label-1")
	if err != nil {
		t.Fatalf("IndexPlainText: %v", err)
	}
	terms, err := s.GetDistinctiveTerms(res.SourceID)
	if err != nil {
		t.Fatalf("GetDistinctiveTerms: %v", err)
	}
	if len(terms) == 0 {
		t.Fatalf("terms = 0, want > 0")
	}
	want := map[string]bool{"alpha": true, "bravo": true, "charlie": true}
	found := false
	for _, term := range terms {
		if want[term] {
			found = true
		}
	}
	if !found {
		t.Fatalf("terms = %v, expected at least one of alpha/bravo/charlie", terms)
	}
}

func testDistinctiveTermsBounded(t *testing.T) {
	s := newStore(t)
	var b strings.Builder
	for i := 0; i < 200; i++ {
		b.WriteString("token")
		// 4-letter alphabetic suffix to keep tokens >= 3 chars and unique-ish
		b.WriteString(string(rune('a' + (i%26))))
		b.WriteString(string(rune('a' + ((i/26)%26))))
		b.WriteString(string(rune('a' + ((i/676)%26))))
		b.WriteString(" ")
	}
	res, err := s.IndexPlainText(b.String(), "many-tokens")
	if err != nil {
		t.Fatalf("IndexPlainText: %v", err)
	}
	terms, err := s.GetDistinctiveTerms(res.SourceID)
	if err != nil {
		t.Fatalf("GetDistinctiveTerms: %v", err)
	}
	if len(terms) > 40 {
		t.Fatalf("len(terms) = %d, want <= 40", len(terms))
	}
}

func testDistinctiveTermsStopWords(t *testing.T) {
	s := newStore(t)
	res, err := s.IndexPlainText("the cat and the dog are with us in the room", "stops")
	if err != nil {
		t.Fatalf("IndexPlainText: %v", err)
	}
	terms, err := s.GetDistinctiveTerms(res.SourceID)
	if err != nil {
		t.Fatalf("GetDistinctiveTerms: %v", err)
	}
	for _, term := range terms {
		if stopWords[term] {
			t.Fatalf("stop-word %q in terms %v", term, terms)
		}
		if len(term) < 3 {
			t.Fatalf("short term %q in terms %v", term, terms)
		}
	}
}

func testDistinctiveTermsUnknown(t *testing.T) {
	s := newStore(t)
	terms, err := s.GetDistinctiveTerms(99999)
	if err != nil {
		t.Fatalf("GetDistinctiveTerms unknown: %v", err)
	}
	if terms == nil {
		t.Fatalf("terms = nil, want non-nil empty slice")
	}
	if len(terms) != 0 {
		t.Fatalf("len(terms) = %d, want 0", len(terms))
	}
}
