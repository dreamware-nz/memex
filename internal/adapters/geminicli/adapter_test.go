package geminicli

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/dreamware-nz/memex/internal/adapters"
)

const testBinary = "/usr/local/bin/memex"

// setupHome points HOME at a fresh temp dir so tests cannot read or
// clobber the developer's real ~/.gemini.
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

func TestInstall_OnEmptySettings_AddsAllHookKeys(t *testing.T) {
	setupHome(t)
	a := New()
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("Install: %v", err)
	}
	settings := readSettingsForTest(t, a.SettingsPath())
	for _, key := range []string{"BeforeTool", "AfterTool", "PreCompress", "SessionStart"} {
		entries, ok := settings[key].([]any)
		if !ok || len(entries) == 0 {
			t.Errorf("settings[%q] missing or empty: %v", key, settings[key])
		}
	}
	if _, ok := settings["PreToolUse"]; ok {
		t.Errorf("settings should not contain Claude Code key PreToolUse")
	}
	pre := settings["BeforeTool"].([]any)
	if len(pre) != len(preToolUseMatchers) {
		t.Errorf("BeforeTool entries = %d, want %d", len(pre), len(preToolUseMatchers))
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
	pre := settings["BeforeTool"].([]any)
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
	preAfter, _ := settings["BeforeTool"].([]any)

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

func TestValidate_FailWhenBeforeToolAbsent(t *testing.T) {
	setupHome(t)
	a := New()
	results := a.Validate(testBinary)
	if len(results) == 0 {
		t.Fatalf("no diagnostic results")
	}
	for _, r := range results {
		if r.Status != "fail" {
			t.Errorf("%s = %q, want fail (no settings.json yet)", r.Check, r.Status)
		}
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
