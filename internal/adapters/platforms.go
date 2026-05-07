package adapters

// PlatformID identifies a host agent. Aliased to string so new platforms
// can be added without changing this module's signature.
type PlatformID = string

// DetectionSignal is the result of a platform detection attempt.
// Confidence is one of "high", "medium", "low".
type DetectionSignal struct {
	Platform   PlatformID
	Confidence string
	Reason     string
}

// platformEntry is one row in the detection table.
//
// EnvVars: priority-ordered env vars that mark this platform (high confidence).
// SessionDir: home-relative path segments to the platform's config dir.
// SkipConfigDetect: when true, SessionDir is reported by GetSessionDirSegments
// but is NOT used for the config-dir fallback tier (e.g. antigravity inherits
// ~/.gemini/ from gemini-cli, so detecting on the dir would be a false hit).
type platformEntry struct {
	ID               PlatformID
	EnvVars          []string
	SessionDir       []string
	SkipConfigDetect bool
}

// platformTable mirrors src/adapters/detect.ts PLATFORM_ENV_VARS ordering:
// forks listed BEFORE the parent agent so collision detection works
// (cursor/antigravity before vscode-copilot, kilo before opencode).
var platformTable = []platformEntry{
	{ID: "claude-code", EnvVars: []string{"CLAUDE_PROJECT_DIR", "CLAUDE_SESSION_ID"}, SessionDir: []string{".claude"}},
	{ID: "antigravity", EnvVars: []string{"ANTIGRAVITY_CLI_ALIAS"}, SessionDir: []string{".gemini"}, SkipConfigDetect: true},
	{ID: "cursor", EnvVars: []string{"CURSOR_TRACE_ID", "CURSOR_CLI"}, SessionDir: []string{".cursor"}},
	{ID: "kilo", EnvVars: []string{"KILO_PID"}, SessionDir: []string{".config", "kilo"}},
	{ID: "opencode", EnvVars: []string{"OPENCODE", "OPENCODE_PID"}, SessionDir: []string{".config", "opencode"}},
	{ID: "zed", EnvVars: []string{"ZED_SESSION_ID", "ZED_TERM"}, SessionDir: []string{".config", "zed"}},
	{ID: "codex", EnvVars: []string{"CODEX_THREAD_ID", "CODEX_CI"}, SessionDir: []string{".codex"}},
	{ID: "gemini-cli", EnvVars: []string{"GEMINI_PROJECT_DIR", "GEMINI_CLI"}, SessionDir: []string{".gemini"}},
	{ID: "vscode-copilot", EnvVars: []string{"VSCODE_PID", "VSCODE_CWD"}, SessionDir: []string{".vscode"}},
	{ID: "jetbrains-copilot", EnvVars: []string{"IDEA_INITIAL_DIRECTORY"}, SessionDir: []string{".config", "JetBrains"}},
	{ID: "qwen-code", EnvVars: []string{"QWEN_PROJECT_DIR"}, SessionDir: []string{".qwen"}},
	{ID: "pi", EnvVars: []string{"PI_PROJECT_DIR"}, SessionDir: []string{".pi"}},
	{ID: "kiro", EnvVars: nil, SessionDir: []string{".kiro"}},
}

// GetSessionDirSegments returns the home-relative path segments where the
// given platform stores its config. Returns nil for unknown platforms.
func GetSessionDirSegments(platform PlatformID) []string {
	for _, p := range platformTable {
		if p.ID == platform {
			return p.SessionDir
		}
	}
	return nil
}

// isValidPlatform reports whether the given id appears in platformTable.
func isValidPlatform(id string) bool {
	for _, p := range platformTable {
		if string(p.ID) == id {
			return true
		}
	}
	return false
}
