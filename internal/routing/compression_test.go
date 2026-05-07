package routing

import (
	"strings"
	"testing"
)

func TestOutputCompressionDoc(t *testing.T) {
	got := OutputCompressionDoc()
	if got == "" {
		t.Fatal("OutputCompressionDoc() returned empty string")
	}
	for _, want := range []string{"caveman", "Fragments OK"} {
		if !strings.Contains(got, want) {
			t.Errorf("OutputCompressionDoc() missing substring %q", want)
		}
	}
}
