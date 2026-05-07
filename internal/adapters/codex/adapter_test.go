package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/adapters"
)

const testBinary = "/usr/local/bin/memex"

func setupHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

func readSettingsForTest(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func TestInstall_WritesToCodexHooksJSON(t *testing.T) {
	home := setupHome(t)
	a := New()
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("Install: %v", err)
	}
	want := filepath.Join(home, ".codex", "hooks.json")
	if got := a.SettingsPath(); got != want {
		t.Errorf("SettingsPath = %q, want %q", got, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("hooks.json not written: %v", err)
	}
	settings := readSettingsForTest(t, want)
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("settings.hooks missing or wrong type: %T", settings["hooks"])
	}
	for _, key := range []string{"PreToolUse", "PostToolUse", "PreCompact", "SessionStart"} {
		entries, ok := hooks[key].([]any)
		if !ok || len(entries) == 0 {
			t.Errorf("hooks[%q] missing or empty: %v", key, hooks[key])
		}
	}
	pre := hooks["PreToolUse"].([]any)
	if len(pre) != len(preToolUseMatchers) {
		t.Errorf("PreToolUse entries = %d, want %d", len(pre), len(preToolUseMatchers))
	}
}

func TestInstall_CreatesMissingCodexDir(t *testing.T) {
	home := setupHome(t)
	if err := os.RemoveAll(filepath.Join(home, ".codex")); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	a := New()
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".codex")); err != nil {
		t.Fatalf(".codex dir not created: %v", err)
	}
}

func TestInstall_Idempotent(t *testing.T) {
	setupHome(t)
	a := New()
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("first Install: %v", err)
	}
	first, _ := json.Marshal(readSettingsForTest(t, a.SettingsPath()))
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("second Install: %v", err)
	}
	second, _ := json.Marshal(readSettingsForTest(t, a.SettingsPath()))
	if string(first) != string(second) {
		t.Errorf("settings changed on reinstall\nfirst:  %s\nsecond: %s", first, second)
	}
}

func TestUninstall_PreservesUserHookInSameMatcherEntry(t *testing.T) {
	setupHome(t)
	a := New()
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("Install: %v", err)
	}

	const userCmd = "/usr/local/bin/user-hook --custom"
	settings := readSettingsForTest(t, a.SettingsPath())
	hooks := settings["hooks"].(map[string]any)
	pre := hooks["PreToolUse"].([]any)
	for _, e := range pre {
		m := e.(map[string]any)
		if m["matcher"] == "Bash" {
			inner := m["hooks"].([]any)
			inner = append(inner, map[string]any{
				"type":    "command",
				"command": userCmd,
			})
			m["hooks"] = inner
		}
	}
	if err := adapters.WriteSettings(a.SettingsPath(), settings); err != nil {
		t.Fatalf("WriteSettings: %v", err)
	}

	if err := a.Uninstall(testBinary); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	settings = readSettingsForTest(t, a.SettingsPath())
	hooksAfter, _ := settings["hooks"].(map[string]any)
	preAfter, _ := hooksAfter["PreToolUse"].([]any)

	foundUser, foundCtx := false, false
	for _, e := range preAfter {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		if m["matcher"] != "Bash" {
			continue
		}
		inner, _ := m["hooks"].([]any)
		for _, h := range inner {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			cmd, _ := hm["command"].(string)
			if cmd == userCmd {
				foundUser = true
			}
			if isMemexHookEntry(cmd, testBinary) {
				foundCtx = true
			}
		}
	}
	if !foundUser {
		t.Errorf("user hook removed by uninstall — expected to survive")
	}
	if foundCtx {
		t.Errorf("memex hook still present after uninstall")
	}
}

func TestValidate_FailWhenPreToolUseAbsent(t *testing.T) {
	setupHome(t)
	a := New()
	results := a.Validate(testBinary)
	if len(results) == 0 {
		t.Fatalf("no diagnostic results")
	}
	foundPre := false
	for _, r := range results {
		if strings.HasPrefix(r.Check, "PreToolUse") {
			foundPre = true
			if r.Status != "fail" {
				t.Errorf("PreToolUse = %q, want fail (no hooks.json yet)", r.Status)
			}
		}
	}
	if !foundPre {
		t.Errorf("no PreToolUse diagnostic emitted: %+v", results)
	}
}

func TestValidate_PassWhenInstalled(t *testing.T) {
	setupHome(t)
	a := New()
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("Install: %v", err)
	}
	results := a.Validate(testBinary)
	for _, r := range results {
		if r.Status != "pass" {
			t.Errorf("%s = %q, want pass: %s", r.Check, r.Status, r.Message)
		}
	}
}
