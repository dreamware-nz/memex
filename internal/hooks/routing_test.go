package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withIsolatedTmpdir overrides $TMPDIR so guidanceOnce markers don't leak
// across tests in the same process. Restores the previous value on cleanup.
func withIsolatedTmpdir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev, hadPrev := os.LookupEnv("TMPDIR")
	if err := os.Setenv("TMPDIR", dir); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() {
		if hadPrev {
			_ = os.Setenv("TMPDIR", prev)
		} else {
			_ = os.Unsetenv("TMPDIR")
		}
	})
	return dir
}

func TestCanonicalNameAliases(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"gemini run_shell", "run_shell_command", "Bash"},
		{"opencode fetch", "fetch", "WebFetch"},
		{"codex shell", "shell", "Bash"},
		{"codex container", "container.exec", "Bash"},
		{"cursor mcp_web", "mcp_web_fetch", "WebFetch"},
		{"vscode terminal", "run_in_terminal", "Bash"},
		{"kiro fs_read", "fs_read", "Read"},
		{"unknown passthrough", "SomeRandomTool", "SomeRandomTool"},
		{"canonical itself", "Bash", "Bash"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canonicalName(tc.in); got != tc.want {
				t.Errorf("canonicalName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestGuidanceOnceFiresOnce(t *testing.T) {
	withIsolatedTmpdir(t)
	first := guidanceOnce("read", ReadGuidance, "sid-A")
	if first.Action != ActionContext {
		t.Fatalf("first call: want ActionContext, got %v", first.Action)
	}
	if !strings.Contains(first.AdditionalCtx, "memex_execute_file") {
		t.Errorf("missing memex_execute_file in: %q", first.AdditionalCtx)
	}
	second := guidanceOnce("read", ReadGuidance, "sid-A")
	if second.Action != ActionPassthrough {
		t.Errorf("second call: want ActionPassthrough, got %v", second.Action)
	}
}

func TestGuidanceOnceSeparateSessions(t *testing.T) {
	withIsolatedTmpdir(t)
	if d := guidanceOnce("read", ReadGuidance, "sid-A"); d.Action != ActionContext {
		t.Fatalf("sid-A first: want context, got %v", d.Action)
	}
	if d := guidanceOnce("read", ReadGuidance, "sid-B"); d.Action != ActionContext {
		t.Errorf("sid-B should fire independently, got %v", d.Action)
	}
}

func TestGuidanceOnceSeparateTypes(t *testing.T) {
	withIsolatedTmpdir(t)
	if d := guidanceOnce("read", ReadGuidance, "sid-X"); d.Action != ActionContext {
		t.Fatalf("read first: want context, got %v", d.Action)
	}
	if d := guidanceOnce("grep", GrepGuidance, "sid-X"); d.Action != ActionContext {
		t.Errorf("grep should be independent of read, got %v", d.Action)
	}
}

func TestGuidanceOnceUnwritableTmpdirDegrades(t *testing.T) {
	dir := withIsolatedTmpdir(t)
	// Replace tmpdir with a regular file so MkdirAll fails.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("rm: %v", err)
	}
	if err := os.WriteFile(dir, []byte("blocker"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	d := guidanceOnce("read", ReadGuidance, "sid-Z")
	if d.Action != ActionPassthrough {
		t.Errorf("unwritable tmpdir should yield ActionPassthrough, got %v", d.Action)
	}
}

func TestRouteWebFetchDeny(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("WebFetch", map[string]any{"url": "https://example.com/x"}, "sid")
	if d.Action != ActionDeny {
		t.Fatalf("want ActionDeny, got %v", d.Action)
	}
	for _, tok := range []string{"memex_fetch_and_index", "memex_search", "https://example.com/x"} {
		if !strings.Contains(d.Reason, tok) {
			t.Errorf("reason missing %q: %s", tok, d.Reason)
		}
	}
}

func TestRouteWebFetchAliasDeny(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("web_fetch", map[string]any{"url": "https://x"}, "sid")
	if d.Action != ActionDeny {
		t.Errorf("alias web_fetch should deny, got %v", d.Action)
	}
}

func TestRouteUnknownPassesThrough(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("SomeRandomTool", map[string]any{}, "sid")
	if d.Action != ActionPassthrough {
		t.Errorf("unknown tool should passthrough, got %v", d.Action)
	}
}

func TestRouteReadFirstFiresContextSecondPasses(t *testing.T) {
	withIsolatedTmpdir(t)
	first := Route("Read", map[string]any{"file_path": "/x"}, "sid-1")
	if first.Action != ActionContext {
		t.Fatalf("first Read: want context, got %v", first.Action)
	}
	if !strings.Contains(first.AdditionalCtx, "memex_execute_file") {
		t.Errorf("missing memex_execute_file in %q", first.AdditionalCtx)
	}
	second := Route("Read", map[string]any{"file_path": "/y"}, "sid-1")
	if second.Action != ActionPassthrough {
		t.Errorf("second Read: want passthrough, got %v", second.Action)
	}
}

func TestRouteReadAliasFires(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("read_file", map[string]any{"file_path": "/x"}, "sid-alias")
	if d.Action != ActionContext {
		t.Errorf("alias read_file should fire context, got %v", d.Action)
	}
}

func TestRouteGrepIndependentSlot(t *testing.T) {
	withIsolatedTmpdir(t)
	if d := Route("Read", map[string]any{}, "sid-Z"); d.Action != ActionContext {
		t.Fatalf("seed read failed: %v", d.Action)
	}
	d := Route("Grep", map[string]any{"pattern": "x"}, "sid-Z")
	if d.Action != ActionContext {
		t.Errorf("Grep should fire independent slot, got %v", d.Action)
	}
	if !strings.Contains(d.AdditionalCtx, "memex_execute") {
		t.Errorf("Grep guidance missing memex_execute: %q", d.AdditionalCtx)
	}
}

func TestRouteBashCurlBlocked(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("Bash", map[string]any{"command": "curl https://example.com"}, "sid")
	if d.Action != ActionModify {
		t.Fatalf("bare curl: want modify, got %v", d.Action)
	}
	cmd, _ := d.UpdatedInput["command"].(string)
	if !strings.Contains(cmd, "memex_execute") {
		t.Errorf("modified command missing memex_execute: %q", cmd)
	}
}

func TestRouteBashSilentCurlAllowed(t *testing.T) {
	withIsolatedTmpdir(t)
	// Burn the bash advisory slot first.
	_ = Route("Bash", map[string]any{"command": "ls"}, "sid")
	d := Route("Bash", map[string]any{"command": "curl -s https://example.com -o /tmp/page.html"}, "sid")
	if d.Action != ActionPassthrough {
		t.Errorf("silent curl with -o file should pass, got %v (%+v)", d.Action, d)
	}
}

func TestRouteBashCurlVerboseBlocked(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("Bash", map[string]any{"command": "curl -v https://example.com -o /tmp/x"}, "sid")
	if d.Action != ActionModify {
		t.Errorf("curl -v should be blocked, got %v", d.Action)
	}
}

func TestRouteBashCurlStdoutAliasBlocked(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("Bash", map[string]any{"command": "curl https://example.com -o -"}, "sid")
	if d.Action != ActionModify {
		t.Errorf("curl -o - should be blocked, got %v", d.Action)
	}
}

func TestRouteBashQuotedCurlAllowed(t *testing.T) {
	withIsolatedTmpdir(t)
	_ = Route("Bash", map[string]any{"command": "ls"}, "sid")
	d := Route("Bash", map[string]any{"command": `gh issue edit 1 --body "see curl example"`}, "sid")
	if d.Action != ActionPassthrough {
		t.Errorf("quoted curl in body should pass, got %v (%+v)", d.Action, d)
	}
}

func TestRouteBashInlineFetchBlocked(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("Bash", map[string]any{"command": `node -e 'fetch("https://x")'`}, "sid")
	if d.Action != ActionModify {
		t.Errorf("inline node fetch should be blocked, got %v", d.Action)
	}
}

func TestRouteBashInlineRequestsBlocked(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("Bash", map[string]any{"command": `python3 -c "requests.get('https://x')"`}, "sid")
	if d.Action != ActionModify {
		t.Errorf("inline python requests should be blocked, got %v", d.Action)
	}
}

func TestRouteBashGradleBlocked(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("Bash", map[string]any{"command": "./gradlew build"}, "sid")
	if d.Action != ActionModify {
		t.Fatalf("./gradlew build should be blocked, got %v", d.Action)
	}
	cmd, _ := d.UpdatedInput["command"].(string)
	if !strings.Contains(cmd, "memex_execute") {
		t.Errorf("modified gradle command missing memex_execute: %q", cmd)
	}
}

func TestRouteBashGradleSubstringAllowed(t *testing.T) {
	withIsolatedTmpdir(t)
	_ = Route("Bash", map[string]any{"command": "ls"}, "sid")
	d := Route("Bash", map[string]any{"command": "echo gradle-wrapper-config"}, "sid")
	if d.Action != ActionPassthrough {
		t.Errorf("substring gradle-wrapper-config should pass, got %v (%+v)", d.Action, d)
	}
}

func TestRouteBashFirstAdvisoryThenSilent(t *testing.T) {
	withIsolatedTmpdir(t)
	first := Route("Bash", map[string]any{"command": "ls"}, "sid-bash")
	if first.Action != ActionContext {
		t.Fatalf("first Bash: want context, got %v", first.Action)
	}
	second := Route("Bash", map[string]any{"command": "pwd"}, "sid-bash")
	if second.Action != ActionPassthrough {
		t.Errorf("second Bash: want passthrough, got %v", second.Action)
	}
}

func TestRouteAgentAppendsRoutingBlock(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("Agent", map[string]any{
		"prompt":        "do thing",
		"subagent_type": "general-purpose",
	}, "sid")
	if d.Action != ActionModify {
		t.Fatalf("Agent should ActionModify, got %v", d.Action)
	}
	prompt, _ := d.UpdatedInput["prompt"].(string)
	if !strings.HasPrefix(prompt, "do thing") {
		t.Errorf("prompt should start with original: %q", prompt)
	}
	if !strings.Contains(prompt, "memex_search") {
		t.Errorf("appended block missing memex_search: %q", prompt)
	}
	if d.UpdatedInput["subagent_type"] != "general-purpose" {
		t.Errorf("non-Bash subagent should be preserved, got %v", d.UpdatedInput["subagent_type"])
	}
}

func TestRouteAgentRewritesBashSubagent(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("Agent", map[string]any{
		"prompt":        "do thing",
		"subagent_type": "Bash",
	}, "sid")
	if d.UpdatedInput["subagent_type"] != "general-purpose" {
		t.Errorf("Bash subagent should be rewritten, got %v", d.UpdatedInput["subagent_type"])
	}
}

func TestRouteAgentTaskAlias(t *testing.T) {
	withIsolatedTmpdir(t)
	d := Route("Task", map[string]any{"prompt": "x"}, "sid")
	if d.Action != ActionModify {
		t.Errorf("Task should route as Agent, got %v", d.Action)
	}
}

func TestFormatClaudeCodeShapes(t *testing.T) {
	cases := []struct {
		name string
		d    Decision
		want string // substring expected in JSON
	}{
		{"passthrough", Decision{Action: ActionPassthrough}, "{}"},
		{"deny", Decision{Action: ActionDeny, Reason: "no"}, `"permissionDecision":"deny"`},
		{"ask", Decision{Action: ActionAsk}, `"permissionDecision":"ask"`},
		{"modify", Decision{Action: ActionModify, UpdatedInput: map[string]any{"command": "x"}}, `"modifiedToolInput"`},
		{"context", Decision{Action: ActionContext, AdditionalCtx: "hi"}, `"additionalContext":"hi"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := formatClaudeCode(tc.d)
			b, err := jsonMarshal(resp)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if !strings.Contains(b, tc.want) {
				t.Errorf("want %q in %s", tc.want, b)
			}
		})
	}
}

// jsonMarshal returns the JSON encoding as a string for substring matching.
func jsonMarshal(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

func TestSessionMarkerSeparation(t *testing.T) {
	dir := withIsolatedTmpdir(t)
	_ = guidanceOnce("read", ReadGuidance, "sid-A")
	_ = guidanceOnce("read", ReadGuidance, "sid-B")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	var have []string
	for _, e := range entries {
		have = append(have, e.Name())
		// Each session dir should have exactly one marker.
		sub, err := os.ReadDir(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("subdir: %v", err)
		}
		if len(sub) != 1 {
			t.Errorf("session %s: want 1 marker, got %d", e.Name(), len(sub))
		}
	}
	if len(have) != 2 {
		t.Errorf("want 2 session dirs, got %v", have)
	}
}
