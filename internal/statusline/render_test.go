package statusline

import (
	"strings"
	"testing"
)

func TestRender_FullPayload(t *testing.T) {
	out := Render(&StatsPayload{
		DollarsSavedSession:  0.42,
		DollarsSavedLifetime: 3.14,
		PctEfficient:         87,
	})
	for _, want := range []string{
		"$0.42 saved this session",
		"$3.14 saved across sessions",
		"87% efficient",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestRender_OmitsLifetimeWhenZero(t *testing.T) {
	out := Render(&StatsPayload{
		DollarsSavedSession:  1.00,
		DollarsSavedLifetime: 0,
		PctEfficient:         50,
	})
	if strings.Contains(out, "saved across sessions") {
		t.Fatalf("expected lifetime block omitted, got %q", out)
	}
	if !strings.Contains(out, "saved this session") {
		t.Fatalf("missing session segment in %q", out)
	}
	if !strings.Contains(out, "50% efficient") {
		t.Fatalf("missing efficiency segment in %q", out)
	}
}

func TestRender_Deterministic(t *testing.T) {
	p := &StatsPayload{DollarsSavedSession: 0.1, DollarsSavedLifetime: 0.2, PctEfficient: 5}
	if Render(p) != Render(p) {
		t.Fatal("Render is not pure")
	}
}

func TestRender_NilPayload(t *testing.T) {
	if Render(nil) != "" {
		t.Fatal("nil payload should yield empty string")
	}
}

func TestParsePayload_EmptyInput(t *testing.T) {
	p, err := ParsePayload(nil)
	if err != nil || p != nil {
		t.Fatalf("expected (nil, nil), got (%v, %v)", p, err)
	}
}

func TestParsePayload_Valid(t *testing.T) {
	p, err := ParsePayload([]byte(`{"dollars_saved_session":1.5,"pct_efficient":42}`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if p.DollarsSavedSession != 1.5 || p.PctEfficient != 42 {
		t.Fatalf("decoded wrong: %+v", p)
	}
}
