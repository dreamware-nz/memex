package tools

import (
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/kb"
)

func TestRouteOutput_IntentTriggersIntentSearch(t *testing.T) {
	store := newStubStore()
	store.results["the intent"] = []kb.SearchResult{
		{Heading: "section A", Body: "matched line"},
	}
	store.terms[1] = []string{"intent", "matched", "section"}

	output := strings.Repeat("body line that is longer than thirty bytes\n", 200) // ~8.6KB
	if len(output) <= IntentSearchThreshold {
		t.Fatalf("test setup: output (%d) must exceed IntentSearchThreshold (%d)", len(output), IntentSearchThreshold)
	}

	text, indexed, err := routeOutput(output, "the intent", "execute:shell", store)
	if err != nil {
		t.Fatalf("routeOutput: %v", err)
	}
	if !indexed {
		t.Errorf("indexed = false, want true")
	}
	if !strings.Contains(text, "1 sections matched \"the intent\"") {
		t.Errorf("missing matched header: %q", text[:200])
	}
	if !strings.Contains(text, "Searchable terms:") {
		t.Errorf("missing distinctive terms line: %q", text)
	}
}

func TestRouteOutput_IntentBelowThresholdReturnsRaw(t *testing.T) {
	store := newStubStore()
	output := strings.Repeat("a", 1000) // < 5KB

	text, indexed, err := routeOutput(output, "anything", "execute:shell", store)
	if err != nil {
		t.Fatalf("routeOutput: %v", err)
	}
	if indexed {
		t.Errorf("indexed = true, want false (output below intent threshold)")
	}
	if text != output {
		t.Errorf("text = %q, want raw output", text[:100])
	}
	if len(store.calls) != 0 {
		t.Errorf("indexer called %d times, want 0", len(store.calls))
	}
}

func TestRouteOutput_IntentBoundaryAt5KB(t *testing.T) {
	store := newStubStore()
	store.terms[1] = []string{"foo", "bar"}

	exactly5K := strings.Repeat("a", IntentSearchThreshold)
	text, indexed, err := routeOutput(exactly5K, "intent", "src", store)
	if err != nil {
		t.Fatalf("routeOutput: %v", err)
	}
	if indexed {
		t.Errorf("at exactly threshold should NOT trigger; want raw")
	}
	if text != exactly5K {
		t.Errorf("text mismatch at boundary")
	}

	store2 := newStubStore()
	store2.terms[1] = []string{"foo", "bar"}
	overBy1 := strings.Repeat("a", IntentSearchThreshold+1)
	_, indexed2, err := routeOutput(overBy1, "intent", "src", store2)
	if err != nil {
		t.Fatalf("routeOutput over: %v", err)
	}
	if !indexed2 {
		t.Errorf("over threshold should trigger intent path")
	}
}

func TestRouteOutput_NoIntentLargeTriggersAutoIndex(t *testing.T) {
	store := newStubStore()
	output := strings.Repeat("z", LargeOutputThreshold+10)

	text, indexed, err := routeOutput(output, "", "execute:python", store)
	if err != nil {
		t.Fatalf("routeOutput: %v", err)
	}
	if !indexed {
		t.Errorf("indexed = false, want true (over large threshold)")
	}
	if !strings.Contains(text, "Indexed") {
		t.Errorf("missing 'Indexed' marker: %q", text[:200])
	}
	if !strings.Contains(text, "execute:python") {
		t.Errorf("missing source label: %q", text[:200])
	}
	if !strings.Contains(text, "memex_search(queries") {
		t.Errorf("missing memex_search hint: %q", text[:200])
	}
}

func TestRouteOutput_NoIntentMediumOutputInline(t *testing.T) {
	store := newStubStore()
	output := strings.Repeat("a", 50_000) // between 5KB and 100KB

	text, indexed, err := routeOutput(output, "", "execute:shell", store)
	if err != nil {
		t.Fatalf("routeOutput: %v", err)
	}
	if indexed {
		t.Errorf("indexed = true, want false for medium-no-intent path")
	}
	if text != output {
		t.Errorf("text != output for medium-no-intent path")
	}
}
