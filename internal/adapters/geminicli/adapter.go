package geminicli

import (
	"os"
	"path/filepath"

	"github.com/dreamware-nz/memex/internal/adapters"
	"github.com/dreamware-nz/memex/internal/skills"
)

// Adapter implements adapters.Adapter for Gemini CLI.
type Adapter struct{}

// New returns a Gemini CLI adapter.
func New() *Adapter { return &Adapter{} }

// compile-time interface assertion
var _ adapters.Adapter = (*Adapter)(nil)

// SettingsPath returns ~/.gemini/settings.json. If the home dir cannot be
// resolved, a relative path is returned so callers see the failure when
// they try to read or write the file rather than as a panic here.
func (a *Adapter) SettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".gemini", "settings.json")
	}
	return filepath.Join(home, ".gemini", "settings.json")
}

// SkillsPath returns ~/.gemini/plugins/memex/skills, the
// install-anchored skills root for the Gemini CLI adapter.
func (a *Adapter) SkillsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".gemini", "plugins", "memex", "skills")
	}
	return filepath.Join(home, ".gemini", "plugins", "memex", "skills")
}

// Install merges memex hook entries into ~/.gemini/settings.json without
// duplicating existing entries. Hook keys live at the top level (no
// "hooks" wrapper) using Gemini's names.
func (a *Adapter) Install(binaryPath string) error {
	settings, err := adapters.ReadSettings(a.SettingsPath())
	if err != nil {
		return err
	}

	for hookType, blockVal := range hookBlock(binaryPath) {
		blockEntries, _ := blockVal.([]any)
		existing := sliceValue(settings, hookType)
		for _, ne := range blockEntries {
			entry, ok := ne.(map[string]any)
			if !ok {
				continue
			}
			matcher, _ := entry["matcher"].(string)
			if !memexEntryPresent(existing, binaryPath, matcher) {
				existing = append(existing, entry)
			}
		}
		settings[hookType] = existing
	}

	if err := adapters.WriteSettings(a.SettingsPath(), settings); err != nil {
		return err
	}
	if err := skills.Write(a.SkillsPath()); err != nil {
		_ = a.Uninstall(binaryPath)
		return err
	}
	return nil
}

// Uninstall strips memex-managed inner hook commands from every top-level
// hook key, prunes entries whose hooks array becomes empty, and deletes
// the hook key entirely when no entries remain. User hooks co-located in
// the same matcher entry are preserved.
func (a *Adapter) Uninstall(binaryPath string) error {
	settings, err := adapters.ReadSettings(a.SettingsPath())
	if err != nil {
		return err
	}
	for _, hookType := range GeminiHookNames {
		entries, ok := settings[hookType].([]any)
		if !ok {
			continue
		}
		pruned := make([]any, 0, len(entries))
		for _, e := range entries {
			m, ok := e.(map[string]any)
			if !ok {
				pruned = append(pruned, e)
				continue
			}
			inner, ok := m["hooks"].([]any)
			if !ok {
				pruned = append(pruned, m)
				continue
			}
			kept := make([]any, 0, len(inner))
			for _, h := range inner {
				hm, ok := h.(map[string]any)
				if !ok {
					kept = append(kept, h)
					continue
				}
				cmd, _ := hm["command"].(string)
				if !isMemexHookEntry(cmd, binaryPath) {
					kept = append(kept, h)
				}
			}
			if len(kept) > 0 {
				m["hooks"] = kept
				pruned = append(pruned, m)
			}
		}
		if len(pruned) == 0 {
			delete(settings, hookType)
		} else {
			settings[hookType] = pruned
		}
	}
	if err := adapters.WriteSettings(a.SettingsPath(), settings); err != nil {
		return err
	}
	_ = skills.Remove(a.SkillsPath())
	return nil
}

// Validate checks that BeforeTool and SessionStart hook keys reference
// ctx — the two hook types ctx cannot function without on Gemini CLI.
func (a *Adapter) Validate(binaryPath string) []adapters.DiagnosticResult {
	results := make([]adapters.DiagnosticResult, 0, 2)
	settings, err := adapters.ReadSettings(a.SettingsPath())
	if err != nil {
		results = append(results, adapters.DiagnosticResult{
			Check:   "settings.json",
			Status:  "fail",
			Message: "could not read settings.json: " + err.Error(),
			Fix:     "memex setup install",
		})
		return results
	}
	for _, ht := range []string{
		GeminiHookNames[hookCmdPreToolUse],
		GeminiHookNames[hookCmdSessionStart],
	} {
		r := adapters.DiagnosticResult{Check: ht + " hook"}
		if hookKeyPresent(settings, ht, binaryPath) {
			r.Status = "pass"
			r.Message = ht + " hook configured"
		} else {
			r.Status = "fail"
			r.Message = "no memex-managed " + ht + " hook found"
			r.Fix = "memex setup install"
		}
		results = append(results, r)
	}
	results = append(results, skillsDiagnostic(a.SkillsPath()))
	return results
}

// skillsDiagnostic checks that a representative subset of the embedded
// skill bundle is present and non-empty under skillsRoot.
func skillsDiagnostic(skillsRoot string) adapters.DiagnosticResult {
	r := adapters.DiagnosticResult{Check: "skills"}
	for _, rel := range []string{
		filepath.Join("memex", "SKILL.md"),
		filepath.Join("memex-doctor", "SKILL.md"),
		filepath.Join("memex-ops", "SKILL.md"),
	} {
		info, err := os.Stat(filepath.Join(skillsRoot, rel))
		if err != nil || info.Size() == 0 {
			r.Status = "fail"
			r.Message = "missing or empty: " + rel
			r.Fix = "memex setup install"
			return r
		}
	}
	r.Status = "pass"
	r.Message = "skill bundle present"
	return r
}

// ── helpers ──────────────────────────────────────────────────

// sliceValue returns m[key] as []any, or nil when missing or of the
// wrong type. The caller appends to it and writes the result back.
func sliceValue(m map[string]any, key string) []any {
	if v, ok := m[key].([]any); ok {
		return v
	}
	return nil
}

// memexEntryPresent reports whether entries already contains a memex-managed
// hook entry with the given matcher.
func memexEntryPresent(entries []any, binaryPath, matcher string) bool {
	for _, e := range entries {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		em, _ := m["matcher"].(string)
		if em != matcher {
			continue
		}
		inner, _ := m["hooks"].([]any)
		for _, h := range inner {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			cmd, _ := hm["command"].(string)
			if isMemexHookEntry(cmd, binaryPath) {
				return true
			}
		}
	}
	return false
}

// hookKeyPresent reports whether settings[hookType] has any inner hook
// command attributable to ctx.
func hookKeyPresent(settings map[string]any, hookType, binaryPath string) bool {
	entries, ok := settings[hookType].([]any)
	if !ok {
		return false
	}
	for _, e := range entries {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		inner, _ := m["hooks"].([]any)
		for _, h := range inner {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			cmd, _ := hm["command"].(string)
			if isMemexHookEntry(cmd, binaryPath) {
				return true
			}
		}
	}
	return false
}
