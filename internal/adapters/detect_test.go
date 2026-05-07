package adapters

import (
	"testing"
)

// allPlatformEnvVars is the union of every env var that platformTable
// uses for detection, plus MEMEX_PLATFORM. Tests clear all of
// them up-front so the host process's real environment can't leak in.
var allPlatformEnvVars = []string{
	"MEMEX_PLATFORM",
	"CLAUDE_PROJECT_DIR", "CLAUDE_SESSION_ID",
	"ANTIGRAVITY_CLI_ALIAS",
	"CURSOR_TRACE_ID", "CURSOR_CLI",
	"KILO_PID",
	"OPENCODE", "OPENCODE_PID",
	"ZED_SESSION_ID", "ZED_TERM",
	"CODEX_THREAD_ID", "CODEX_CI",
	"GEMINI_PROJECT_DIR", "GEMINI_CLI",
	"VSCODE_PID", "VSCODE_CWD",
	"IDEA_INITIAL_DIRECTORY",
	"QWEN_PROJECT_DIR",
	"PI_PROJECT_DIR",
}

func clearPlatformEnv(t *testing.T) {
	t.Helper()
	for _, v := range allPlatformEnvVars {
		t.Setenv(v, "")
	}
}

func TestDetectPlatform_OverrideWins(t *testing.T) {
	clearPlatformEnv(t)
	t.Setenv("MEMEX_PLATFORM", "codex")

	sig := DetectPlatform()
	if sig.Platform != "codex" {
		t.Errorf("Platform = %q, want %q", sig.Platform, "codex")
	}
	if sig.Confidence != "high" {
		t.Errorf("Confidence = %q, want %q", sig.Confidence, "high")
	}
}

func TestDetectPlatform_ClaudeProjectDir(t *testing.T) {
	clearPlatformEnv(t)
	t.Setenv("CLAUDE_PROJECT_DIR", "/some/project")

	sig := DetectPlatform()
	if sig.Platform != "claude-code" {
		t.Errorf("Platform = %q, want %q", sig.Platform, "claude-code")
	}
	if sig.Confidence != "high" {
		t.Errorf("Confidence = %q, want %q", sig.Confidence, "high")
	}
}

func TestDetectPlatform_FallbackLow(t *testing.T) {
	clearPlatformEnv(t)
	// Point HOME at an empty temp dir so no platform config dirs exist.
	t.Setenv("HOME", t.TempDir())

	sig := DetectPlatform()
	if sig.Platform != "claude-code" {
		t.Errorf("Platform = %q, want %q", sig.Platform, "claude-code")
	}
	if sig.Confidence != "low" {
		t.Errorf("Confidence = %q, want %q", sig.Confidence, "low")
	}
	if sig.Reason != "fallback" {
		t.Errorf("Reason = %q, want %q", sig.Reason, "fallback")
	}
}

func TestDetectPlatform_CursorBeatsVSCode(t *testing.T) {
	clearPlatformEnv(t)
	// Cursor inherits VSCODE_PID — a naive check would mis-classify it as
	// vscode-copilot. The platformTable lists cursor first to prevent that.
	t.Setenv("CURSOR_TRACE_ID", "trace-abc")
	t.Setenv("VSCODE_PID", "12345")

	sig := DetectPlatform()
	if sig.Platform != "cursor" {
		t.Errorf("Platform = %q, want %q (vscode-copilot would be a regression)", sig.Platform, "cursor")
	}
	if sig.Confidence != "high" {
		t.Errorf("Confidence = %q, want %q", sig.Confidence, "high")
	}
}

func TestGetSessionDirSegments(t *testing.T) {
	cases := []struct {
		platform PlatformID
		want     []string
	}{
		{"codex", []string{".codex"}},
		{"claude-code", []string{".claude"}},
		{"kilo", []string{".config", "kilo"}},
		{"unknown-platform", nil},
	}
	for _, tc := range cases {
		got := GetSessionDirSegments(tc.platform)
		if !equalStringSlices(got, tc.want) {
			t.Errorf("GetSessionDirSegments(%q) = %v, want %v", tc.platform, got, tc.want)
		}
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
