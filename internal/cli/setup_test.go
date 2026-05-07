package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/skills"
)

// runSetup executes a setup subcommand with given args against an
// isolated $HOME and returns stdout/stderr (combined) and the RunE
// error.
func runSetup(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	root.SetArgs(append([]string{"setup"}, args...))
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	err := root.Execute()
	return buf.String(), err
}

func setIsolatedHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// Avoid platform-detection misfires from outer env.
	for _, ev := range []string{
		"MEMEX_PLATFORM",
		"CLAUDE_PROJECT_DIR", "CLAUDE_SESSION_ID",
		"CURSOR_TRACE_ID", "CURSOR_CLI",
		"CODEX_THREAD_ID", "CODEX_CI",
		"GEMINI_PROJECT_DIR", "GEMINI_CLI",
		"VSCODE_PID", "VSCODE_CWD",
	} {
		t.Setenv(ev, "")
	}
	return dir
}

func TestSetup_InstallValidateUninstallRoundTrip_ClaudeCode(t *testing.T) {
	home := setIsolatedHome(t)

	out, err := runSetup(t, "install", "--platform", "claude-code")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	for _, want := range []string{"install hooks", "install plugin-manifest", "install skills"} {
		if !strings.Contains(out, want) {
			t.Errorf("install stdout missing %q\nfull:\n%s", want, out)
		}
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Errorf("settings.json missing: %v", err)
	}
	pluginManifest := filepath.Join(home, ".claude", "plugins", "installed_plugins.json")
	if _, err := os.Stat(pluginManifest); err != nil {
		t.Errorf("plugin manifest missing: %v", err)
	}
	skillsRoot := filepath.Join(home, ".claude", "plugins", "marketplaces", "dreamware", "skills")
	for _, rel := range skills.EmbeddedFiles() {
		if _, err := os.Stat(filepath.Join(skillsRoot, rel)); err != nil {
			t.Errorf("skill %s missing: %v", rel, err)
			break // one is enough to flag the failure
		}
	}

	out, err = runSetup(t, "validate", "--platform", "claude-code")
	if err != nil {
		t.Fatalf("validate (post-install): %v\n%s", err, out)
	}
	if !strings.Contains(out, "pass skills:") {
		t.Errorf("validate stdout missing 'pass skills:'\n%s", out)
	}

	out, err = runSetup(t, "uninstall", "--platform", "claude-code")
	if err != nil {
		t.Fatalf("uninstall: %v\n%s", err, out)
	}
	for _, want := range []string{"uninstall hooks", "uninstall plugin-manifest", "uninstall skills"} {
		if !strings.Contains(out, want) {
			t.Errorf("uninstall stdout missing %q\nfull:\n%s", want, out)
		}
	}
	for _, rel := range skills.EmbeddedFiles() {
		if _, err := os.Stat(filepath.Join(skillsRoot, rel)); !os.IsNotExist(err) {
			t.Errorf("embedded skill %s remained after uninstall (err=%v)", rel, err)
			break
		}
	}
}

func TestSetup_UnknownPlatformReturnsError(t *testing.T) {
	setIsolatedHome(t)
	out, err := runSetup(t, "install", "--platform", "no-such-platform")
	if err == nil {
		t.Fatalf("install: expected error for unknown platform, got nil\n%s", out)
	}
	msg := err.Error()
	for _, want := range []string{"unsupported platform", "claude-code", "cursor"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q missing %q", msg, want)
		}
	}
}

func TestSetup_ValidateExitCodes(t *testing.T) {
	setIsolatedHome(t)
	// Pre-install case: validate must error out (RunE returns non-nil).
	out, err := runSetup(t, "validate", "--platform", "claude-code")
	if err == nil {
		t.Fatalf("validate (no install): expected non-nil error\n%s", out)
	}

	// Post-install case: must succeed.
	if _, err := runSetup(t, "install", "--platform", "claude-code"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, err := runSetup(t, "validate", "--platform", "claude-code"); err != nil {
		t.Fatalf("validate (post-install): %v", err)
	}
}
