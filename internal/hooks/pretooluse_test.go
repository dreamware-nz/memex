package hooks

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestHandlePreToolUseAllow(t *testing.T) {
	in := strings.NewReader(`{"tool_name":"Bash","tool_input":{}}`)
	var out bytes.Buffer
	if err := HandlePreToolUse(in, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := strings.TrimSpace(out.String())
	if got != "{}" {
		t.Fatalf("want %q, got %q", "{}", got)
	}
}

func TestHandlePreToolUseDeny(t *testing.T) {
	// Routing logic lands later; here we exercise the response wire format
	// directly to lock in the deny shape agents expect.
	resp := PreToolUseResponse{Decision: DecisionDeny, Reason: "blocked"}
	b, err := json.Marshal(&resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"decision":"deny"`) {
		t.Fatalf("want decision:deny in %s", b)
	}
}

func TestHandlePreToolUseMalformed(t *testing.T) {
	in := strings.NewReader("not-json")
	var out bytes.Buffer
	if err := HandlePreToolUse(in, &out); err == nil {
		t.Fatal("want error, got nil")
	}
}

func TestPreToolUseTransformRoundTrip(t *testing.T) {
	want := PreToolUseResponse{UpdatedInput: map[string]any{"cmd": "echo hi"}}
	b, err := json.Marshal(&want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"updatedInput"`) {
		t.Fatalf("want updatedInput key in %s", b)
	}
	var got PreToolUseResponse
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.UpdatedInput["cmd"] != "echo hi" {
		t.Fatalf("want cmd=echo hi, got %v", got.UpdatedInput["cmd"])
	}
}

// runHandle runs HandlePreToolUse against a JSON event and returns the
// stringified response for substring assertions.
func runHandle(t *testing.T, ev string) string {
	t.Helper()
	withIsolatedTmpdir(t)
	in := strings.NewReader(ev)
	var out bytes.Buffer
	if err := HandlePreToolUse(in, &out); err != nil {
		t.Fatalf("handle: %v", err)
	}
	return strings.TrimSpace(out.String())
}

func TestHandleWebFetchDeniesWithMemexReason(t *testing.T) {
	got := runHandle(t, `{"tool_name":"WebFetch","tool_input":{"url":"https://example.com"},"session_id":"sid"}`)
	for _, tok := range []string{`"permissionDecision":"deny"`, "memex_fetch_and_index", "https://example.com"} {
		if !strings.Contains(got, tok) {
			t.Errorf("missing %q in %s", tok, got)
		}
	}
}

func TestHandleDangerousBashModifies(t *testing.T) {
	got := runHandle(t, `{"tool_name":"Bash","tool_input":{"command":"curl https://example.com"},"session_id":"sid"}`)
	for _, tok := range []string{`"modifiedToolInput"`, "memex_execute"} {
		if !strings.Contains(got, tok) {
			t.Errorf("missing %q in %s", tok, got)
		}
	}
}

func TestHandleFirstReadAddsContextSecondPasses(t *testing.T) {
	withIsolatedTmpdir(t)
	first := bytes.Buffer{}
	if err := HandlePreToolUse(strings.NewReader(
		`{"tool_name":"Read","tool_input":{"file_path":"/x"},"session_id":"sid-A"}`), &first); err != nil {
		t.Fatalf("first: %v", err)
	}
	if !strings.Contains(first.String(), `"additionalContext"`) {
		t.Errorf("first Read missing additionalContext: %s", first.String())
	}
	if !strings.Contains(first.String(), "memex_execute_file") {
		t.Errorf("first Read missing memex_execute_file: %s", first.String())
	}

	second := bytes.Buffer{}
	if err := HandlePreToolUse(strings.NewReader(
		`{"tool_name":"Read","tool_input":{"file_path":"/y"},"session_id":"sid-A"}`), &second); err != nil {
		t.Fatalf("second: %v", err)
	}
	if strings.TrimSpace(second.String()) != "{}" {
		t.Errorf("second Read should be empty, got: %s", second.String())
	}
}

