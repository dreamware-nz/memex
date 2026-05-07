package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/mcp"
)

func TestDoctorTool_ImplementsToolInterface(t *testing.T) {
	var _ mcp.Tool = (*DoctorTool)(nil)
}

func TestDoctorTool_NameAndSchema(t *testing.T) {
	tool := &DoctorTool{}
	if got := tool.Name(); got != "memex_doctor" {
		t.Errorf("Name() = %q, want memex_doctor", got)
	}
	var schema map[string]any
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("schema type = %v, want object", schema["type"])
	}
}

func TestDoctorTool_Call_AllPresent(t *testing.T) {
	dir := t.TempDir()
	kbPath := filepath.Join(dir, "kb.sqlite")
	analyticsPath := filepath.Join(dir, "analytics.db")
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(kbPath, []byte("kb"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(analyticsPath, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"command":"memex hook pretooluse"}]}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	tool := &DoctorTool{
		Version:       "1.2.3",
		KBPath:        kbPath,
		AnalyticsPath: analyticsPath,
		SettingsPath:  settingsPath,
	}
	out, err := tool.Call(nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	for _, want := range []string{
		"[OK] memex 1.2.3",
		"[OK] go runtime",
		"[OK] sqlite FTS5 available",
		"[OK] kb:",
		"[OK] analytics:",
		"[OK] host settings:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--full output--\n%s", want, out)
		}
	}
}

func TestDoctorTool_Call_MissingFiles(t *testing.T) {
	dir := t.TempDir()
	tool := &DoctorTool{
		Version:       "0.0.0-dev",
		KBPath:        filepath.Join(dir, "missing.sqlite"),
		AnalyticsPath: filepath.Join(dir, "missing.db"),
		SettingsPath:  filepath.Join(dir, "missing-settings.json"),
	}
	out, err := tool.Call(nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	for _, want := range []string{
		"[WARN] kb: not yet created",
		"[WARN] analytics: not yet created",
		"[WARN] host settings:",
		"memex setup install",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--full output--\n%s", want, out)
		}
	}
}

func TestDoctorTool_Call_SettingsMissingMemexHook(t *testing.T) {
	dir := t.TempDir()
	settings := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settings, []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"command":"some-other-tool"}]}]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	tool := &DoctorTool{SettingsPath: settings}
	out, err := tool.Call(nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if !strings.Contains(out, "no memex hooks") {
		t.Errorf("expected 'no memex hooks' WARN, got: %s", out)
	}
}

func TestDoctorTool_Call_EmptyPathsAreWarnings(t *testing.T) {
	tool := &DoctorTool{}
	out, err := tool.Call(nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	for _, want := range []string{
		"[WARN] kb path: not configured",
		"[WARN] analytics path: not configured",
		"[WARN] host settings: not configured",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--full output--\n%s", want, out)
		}
	}
}
