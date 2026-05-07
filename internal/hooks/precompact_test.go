package hooks

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestHandlePreCompactEmptyResponse(t *testing.T) {
	in := strings.NewReader(`{"session_id":"abc","trigger":"context_limit"}`)
	var out bytes.Buffer
	if err := HandlePreCompact(in, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := strings.TrimSpace(out.String())
	if got != "{}" {
		t.Fatalf("want %q, got %q", "{}", got)
	}
}

func TestHandlePreCompactMalformed(t *testing.T) {
	in := strings.NewReader("not-json")
	var out bytes.Buffer
	if err := HandlePreCompact(in, &out); err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestPreCompactEventDecodeFields(t *testing.T) {
	raw := `{"session_id":"s1","trigger":"context_limit","context_window_usage":"0.92"}`
	var ev PreCompactEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.SessionID != "s1" {
		t.Fatalf("want session_id=s1, got %q", ev.SessionID)
	}
	if ev.Trigger != "context_limit" {
		t.Fatalf("want trigger=context_limit, got %q", ev.Trigger)
	}
	if ev.ContextWindowUsage != "0.92" {
		t.Fatalf("want context_window_usage=0.92, got %q", ev.ContextWindowUsage)
	}
}
