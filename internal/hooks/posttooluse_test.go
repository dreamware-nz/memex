package hooks

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestHandlePostToolUseEmptyResponse(t *testing.T) {
	in := strings.NewReader(`{"tool_name":"Bash","tool_output":"hello"}`)
	var out bytes.Buffer
	if err := HandlePostToolUse(in, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := strings.TrimSpace(out.String())
	if got != "{}" {
		t.Fatalf("want %q, got %q", "{}", got)
	}
}

func TestPostToolUseEventDecodePreservesOutput(t *testing.T) {
	raw := `{"tool_name":"Bash","tool_output":"line1\nline2","session_id":"s1"}`
	var ev PostToolUseEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.ToolOutput != "line1\nline2" {
		t.Fatalf("want tool_output preserved, got %q", ev.ToolOutput)
	}
	if ev.ToolName != "Bash" {
		t.Fatalf("want tool_name=Bash, got %q", ev.ToolName)
	}
	if ev.SessionID != "s1" {
		t.Fatalf("want session_id=s1, got %q", ev.SessionID)
	}
}

func TestHandlePostToolUseMalformed(t *testing.T) {
	in := strings.NewReader("not-json")
	var out bytes.Buffer
	if err := HandlePostToolUse(in, &out); err == nil {
		t.Fatal("want error, got nil")
	}
}
