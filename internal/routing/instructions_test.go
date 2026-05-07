package routing

import (
	"strings"
	"testing"
)

func TestRoutingInstructions(t *testing.T) {
	got := RoutingInstructions()
	if got == "" {
		t.Fatal("RoutingInstructions() returned empty string")
	}
	for _, want := range []string{"memex_batch_execute", "BLOCKED", "memex stats"} {
		if !strings.Contains(got, want) {
			t.Errorf("RoutingInstructions() missing substring %q", want)
		}
	}
}
