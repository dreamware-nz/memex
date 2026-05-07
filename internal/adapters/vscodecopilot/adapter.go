package vscodecopilot

import (
	"os"
	"path/filepath"

	"github.com/dreamware-nz/memex/internal/adapters"
	"github.com/dreamware-nz/memex/internal/skills"
)

// Adapter implements adapters.Adapter for VS Code Copilot.
type Adapter struct{}

// New returns a VS Code Copilot adapter.
func New() *Adapter { return &Adapter{} }

// compile-time interface assertion
var _ adapters.Adapter = (*Adapter)(nil)

// SettingsPath returns ~/.vscode/extensions/memex/hooks.json. If
// the home dir cannot be resolved, a relative path is returned so callers
// see the failure when they try to read or write the file rather than as
// a panic here.
func (a *Adapter) SettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".vscode", "extensions", "memex", "hooks.json")
	}
	return filepath.Join(home, ".vscode", "extensions", "memex", "hooks.json")
}

// SkillsPath returns ~/.vscode/plugins/memex/skills, the
// install-anchored skills root for the VS Code Copilot adapter. We
// deliberately use ~/.vscode/plugins/ rather than ~/.vscode/extensions/
// — extensions/ is owned by the VS Code marketplace installer.
func (a *Adapter) SkillsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".vscode", "plugins", "memex", "skills")
	}
	return filepath.Join(home, ".vscode", "plugins", "memex", "skills")
}

// Install merges memex hook entries into hooks.json without duplicating
// existing entries. Creates the extension directory via WriteSettings'
// MkdirAll if absent. Hook entries live under the "hooks" wrapper key,
// matching the Claude Code shape.
func (a *Adapter) Install(binaryPath string) error {
	settings, err := adapters.ReadSettings(a.SettingsPath())
	if err != nil {
		return err
	}
	hooks := mapValue(settings, "hooks")

	for hookType, blockVal := range hookBlock(binaryPath) {
		blockEntries, _ := blockVal.([]any)
		existing := sliceValue(hooks, hookType)
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
		hooks[hookType] = existing
	}
	settings["hooks"] = hooks

	if err := adapters.WriteSettings(a.SettingsPath(), settings); err != nil {
		return err
	}
	if err := skills.Write(a.SkillsPath()); err != nil {
		_ = a.Uninstall(binaryPath)
		return err
	}
	return nil
}

// Uninstall strips memex-managed inner hook commands from every matcher
// entry, prunes entries whose hooks array becomes empty, and deletes the
// hook key entirely when no entries remain. User hooks co-located in the
// same matcher entry are preserved.
func (a *Adapter) Uninstall(binaryPath string) error {
	settings, err := adapters.ReadSettings(a.SettingsPath())
	if err != nil {
		return err
	}
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		if err := adapters.WriteSettings(a.SettingsPath(), settings); err != nil {
			return err
		}
		_ = skills.Remove(a.SkillsPath())
		return nil
	}
	for hookType, raw := range hooks {
		entries, ok := raw.([]any)
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
			delete(hooks, hookType)
		} else {
			hooks[hookType] = pruned
		}
	}
	if len(hooks) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooks
	}
	if err := adapters.WriteSettings(a.SettingsPath(), settings); err != nil {
		return err
	}
	_ = skills.Remove(a.SkillsPath())
	return nil
}

// Validate checks that PreToolUse and SessionStart hook keys reference
// ctx — the two hook types ctx cannot function without on VS Code
// Copilot.
func (a *Adapter) Validate(binaryPath string) []adapters.DiagnosticResult {
	results := make([]adapters.DiagnosticResult, 0, 2)
	settings, err := adapters.ReadSettings(a.SettingsPath())
	if err != nil {
		results = append(results, adapters.DiagnosticResult{
			Check:   "hooks.json",
			Status:  "fail",
			Message: "could not read hooks.json: " + err.Error(),
			Fix:     "memex setup install",
		})
		return results
	}
	hooks, _ := settings["hooks"].(map[string]any)
	for _, ht := range []string{"PreToolUse", "SessionStart"} {
		r := adapters.DiagnosticResult{Check: ht + " hook"}
		if hookTypePresent(hooks, ht, binaryPath) {
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

// mapValue returns m[key] as a map[string]any, creating and storing one
// when missing or of the wrong type.
func mapValue(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	out := map[string]any{}
	m[key] = out
	return out
}

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

// hookTypePresent reports whether hooks[hookType] has any inner hook
// command attributable to ctx.
func hookTypePresent(hooks map[string]any, hookType, binaryPath string) bool {
	if hooks == nil {
		return false
	}
	entries, ok := hooks[hookType].([]any)
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
