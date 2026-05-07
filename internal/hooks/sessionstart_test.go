package hooks

import (
	"bytes"
	"strings"
	"testing"
)

func TestHandleSessionStartDefaultPlatform(t *testing.T) {
	t.Setenv("MEMEX_PLATFORM", "")
	in := strings.NewReader(`{"session_id":"s1","cwd":"/tmp"}`)
	var out bytes.Buffer
	if err := HandleSessionStart(in, &out, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, `"systemPrompt"`) {
		t.Fatalf("want systemPrompt key in %s", got)
	}
	if strings.Contains(got, `"hookSpecificOutput"`) {
		t.Fatalf("did not want hookSpecificOutput in %s", got)
	}
	if strings.Contains(got, recapStub) {
		t.Fatalf("did not want recap stub without --continue in %s", got)
	}
}

func TestHandleSessionStartContinueRecap(t *testing.T) {
	t.Setenv("MEMEX_PLATFORM", "")
	in := strings.NewReader(`{"session_id":"s1"}`)
	var out bytes.Buffer
	if err := HandleSessionStart(in, &out, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "[prior session resumed]") {
		t.Fatalf("want recap stub in %s", got)
	}
}

func TestHandleSessionStartGeminiPlatform(t *testing.T) {
	t.Setenv("MEMEX_PLATFORM", "gemini-cli")
	in := strings.NewReader(`{"session_id":"s1"}`)
	var out bytes.Buffer
	if err := HandleSessionStart(in, &out, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, `"hookSpecificOutput"`) {
		t.Fatalf("want hookSpecificOutput key in %s", got)
	}
	if !strings.Contains(got, `"system_context"`) {
		t.Fatalf("want system_context key in %s", got)
	}
	if strings.Contains(got, `"systemPrompt"`) {
		t.Fatalf("did not want systemPrompt in %s", got)
	}
}

func TestHandleSessionStartMalformed(t *testing.T) {
	in := strings.NewReader("not-json")
	var out bytes.Buffer
	if err := HandleSessionStart(in, &out, false); err == nil {
		t.Fatal("want error, got nil")
	}
}
