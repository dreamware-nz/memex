package claudecode

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dreamware-nz/memex/internal/adapters"
	"github.com/dreamware-nz/memex/internal/skills"
)

// Adapter implements adapters.Adapter for Claude Code via a synthetic
// "dreamware" local marketplace. Claude Code refuses to load a plugin
// keyed @local because the marketplace must be in known_marketplaces.json
// and have a marketplace.json catalog — so we author a minimal local
// marketplace and register it through the same files Claude Code's
// `claude plugin marketplace add` + `claude plugin install` commands write.
type Adapter struct{}

func New() *Adapter { return &Adapter{} }

var _ adapters.Adapter = (*Adapter)(nil)

const (
	marketplaceID = "dreamware"
	pluginName    = "memex"
	pluginKey     = pluginName + "@" + marketplaceID
	pluginVersion = "0.0.1"
)

// SettingsPath returns ~/.claude/settings.json.
func (a *Adapter) SettingsPath() string {
	return filepath.Join(claudeRoot(), "settings.json")
}

// SkillsPath returns the marketplace-side skills root. `claude plugin
// install` (or our equivalent file copy) propagates this tree into the
// per-version cache directory that Claude Code actually loads at runtime.
func (a *Adapter) SkillsPath() string {
	return filepath.Join(a.marketplaceRoot(), "skills")
}

// marketplaceRoot is the on-disk source-of-truth for the dreamware
// marketplace. Claude Code's "directory" marketplace source points at it.
func (a *Adapter) marketplaceRoot() string {
	return filepath.Join(claudeRoot(), "plugins", "marketplaces", marketplaceID)
}

// cacheRoot is the per-version cache directory Claude Code reads at
// runtime. installed_plugins.json's installPath must equal this path.
func (a *Adapter) cacheRoot() string {
	return filepath.Join(claudeRoot(), "plugins", "cache", marketplaceID, pluginName, pluginVersion)
}

// pluginManifestPath: ~/.claude/plugins/installed_plugins.json (which
// plugins are installed, where, and at what version).
func (a *Adapter) pluginManifestPath() string {
	return filepath.Join(claudeRoot(), "plugins", "installed_plugins.json")
}

// knownMarketplacesPath: ~/.claude/plugins/known_marketplaces.json
// (registry of marketplaces Claude Code knows about).
func (a *Adapter) knownMarketplacesPath() string {
	return filepath.Join(claudeRoot(), "plugins", "known_marketplaces.json")
}

func (a *Adapter) marketplaceJSONPath() string {
	return filepath.Join(a.marketplaceRoot(), ".claude-plugin", "marketplace.json")
}

func (a *Adapter) pluginJSONPath() string {
	return filepath.Join(a.marketplaceRoot(), ".claude-plugin", "plugin.json")
}

// mcpJSONPath: <marketplaceRoot>/.mcp.json
//
// Claude Code reads MCP server declarations from this file at plugin load
// time, not from `mcpServers` inside `.claude-plugin/plugin.json`. The
// distinction matters: a manifest with `mcpServers` in the wrong file passes
// every other validate check but the agent never sees the registered tools.
func (a *Adapter) mcpJSONPath() string {
	return filepath.Join(a.marketplaceRoot(), ".mcp.json")
}

func claudeRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude"
	}
	return filepath.Join(home, ".claude")
}

// Install:
//  1. merge memex hook entries into settings.json (idempotent)
//  2. ensure enabledPlugins[pluginKey] = true
//  3. write marketplace catalog + plugin.json + skills under marketplaceRoot
//  4. mirror marketplaceRoot into cacheRoot (Claude Code loads from cache)
//  5. register marketplace in known_marketplaces.json
//  6. register plugin in installed_plugins.json
//
// On any failure after hooks are written, the partial state is rolled
// back so the user is not left with hooks pointing at a half-install.
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
	enabled := mapValue(settings, "enabledPlugins")
	enabled[pluginKey] = true
	settings["enabledPlugins"] = enabled
	if err := adapters.WriteSettings(a.SettingsPath(), settings); err != nil {
		return err
	}

	if err := a.writeMarketplaceTree(binaryPath); err != nil {
		_ = a.uninstallHooksAndToggle(binaryPath)
		return err
	}
	if err := skills.Write(a.SkillsPath()); err != nil {
		_ = os.RemoveAll(a.marketplaceRoot())
		_ = a.uninstallHooksAndToggle(binaryPath)
		return err
	}
	if err := mirrorTree(a.marketplaceRoot(), a.cacheRoot()); err != nil {
		_ = os.RemoveAll(a.cacheRoot())
		_ = os.RemoveAll(a.marketplaceRoot())
		_ = a.uninstallHooksAndToggle(binaryPath)
		return err
	}
	if err := a.registerMarketplace(); err != nil {
		_ = os.RemoveAll(a.cacheRoot())
		_ = os.RemoveAll(a.marketplaceRoot())
		_ = a.uninstallHooksAndToggle(binaryPath)
		return err
	}
	if err := a.registerPlugin(); err != nil {
		_ = a.unregisterMarketplace()
		_ = os.RemoveAll(a.cacheRoot())
		_ = os.RemoveAll(a.marketplaceRoot())
		_ = a.uninstallHooksAndToggle(binaryPath)
		return err
	}
	return nil
}

// Uninstall reverses Install. Best-effort throughout: a failure in one
// step does not abort the others. Returns the first error encountered
// after attempting every cleanup, so the caller still surfaces problems.
//
// User-authored files inside marketplaceRoot/skills/ survive — only the
// embedded skill subset is pruned (skills.Remove preserves outsiders).
// cacheRoot is always nuked because it's a derived copy we own outright.
func (a *Adapter) Uninstall(binaryPath string) error {
	var firstErr error
	keep := func(err error) {
		if firstErr == nil && err != nil {
			firstErr = err
		}
	}
	keep(a.uninstallHooksAndToggle(binaryPath))
	keep(a.unregisterPlugin())
	keep(a.unregisterMarketplace())
	_ = os.RemoveAll(a.cacheRoot())
	_ = skills.Remove(a.SkillsPath())
	_ = os.Remove(a.mcpJSONPath())
	_ = os.Remove(a.pluginJSONPath())
	_ = os.Remove(a.marketplaceJSONPath())
	_ = os.Remove(filepath.Dir(a.pluginJSONPath())) // .claude-plugin/ if empty
	_ = os.Remove(a.marketplaceRoot())              // marketplaceRoot/ if empty
	return firstErr
}

func (a *Adapter) uninstallHooksAndToggle(binaryPath string) error {
	settings, err := adapters.ReadSettings(a.SettingsPath())
	if err != nil {
		return err
	}
	if hooks, ok := settings["hooks"].(map[string]any); ok {
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
	}
	if enabled, ok := settings["enabledPlugins"].(map[string]any); ok {
		delete(enabled, pluginKey)
		if len(enabled) == 0 {
			delete(settings, "enabledPlugins")
		} else {
			settings["enabledPlugins"] = enabled
		}
	}
	return adapters.WriteSettings(a.SettingsPath(), settings)
}

// Validate reports the health of every install component. A non-running
// Claude Code can still verify everything is wired correctly.
func (a *Adapter) Validate(binaryPath string) []adapters.DiagnosticResult {
	results := make([]adapters.DiagnosticResult, 0, 7)
	settings, err := adapters.ReadSettings(a.SettingsPath())
	if err != nil {
		return append(results, adapters.DiagnosticResult{
			Check: "settings.json", Status: "fail",
			Message: "could not read settings.json: " + err.Error(),
			Fix:     "memex setup install",
		})
	}
	hooks, _ := settings["hooks"].(map[string]any)
	for _, ht := range []string{"PreToolUse", "SessionStart"} {
		r := adapters.DiagnosticResult{Check: ht + " hook"}
		if hookTypePresent(hooks, ht, binaryPath) {
			r.Status, r.Message = "pass", ht+" hook configured"
		} else {
			r.Status, r.Message, r.Fix = "fail", "no memex-managed "+ht+" hook found", "memex setup install"
		}
		results = append(results, r)
	}
	results = append(results, enabledPluginDiagnostic(settings))
	results = append(results, a.knownMarketplaceDiagnostic())
	results = append(results, a.pluginManifestDiagnostic())
	results = append(results, a.pluginJSONDiagnostic(binaryPath))
	results = append(results, skillsDiagnostic(a.SkillsPath()))
	return results
}

func enabledPluginDiagnostic(settings map[string]any) adapters.DiagnosticResult {
	r := adapters.DiagnosticResult{Check: "enabledPlugins"}
	enabled, _ := settings["enabledPlugins"].(map[string]any)
	if v, ok := enabled[pluginKey].(bool); ok && v {
		r.Status, r.Message = "pass", pluginKey+" enabled"
		return r
	}
	r.Status, r.Message, r.Fix = "fail", pluginKey+" not enabled", "memex setup install"
	return r
}

func (a *Adapter) knownMarketplaceDiagnostic() adapters.DiagnosticResult {
	r := adapters.DiagnosticResult{Check: "marketplace"}
	data, err := adapters.ReadSettings(a.knownMarketplacesPath())
	if err != nil {
		r.Status, r.Message, r.Fix = "fail", "could not read known_marketplaces.json: "+err.Error(), "memex setup install"
		return r
	}
	entry, _ := data[marketplaceID].(map[string]any)
	if entry == nil {
		r.Status, r.Message, r.Fix = "fail", "marketplace "+marketplaceID+" not registered", "memex setup install"
		return r
	}
	r.Status, r.Message = "pass", marketplaceID+" marketplace registered"
	return r
}

func (a *Adapter) pluginManifestDiagnostic() adapters.DiagnosticResult {
	r := adapters.DiagnosticResult{Check: "plugin manifest"}
	data, err := adapters.ReadSettings(a.pluginManifestPath())
	if err != nil {
		r.Status, r.Message, r.Fix = "fail", "could not read installed_plugins.json: "+err.Error(), "memex setup install"
		return r
	}
	plugins, _ := data["plugins"].(map[string]any)
	entries, _ := plugins[pluginKey].([]any)
	if len(entries) == 0 {
		r.Status, r.Message, r.Fix = "fail", "no "+pluginKey+" entry", "memex setup install"
		return r
	}
	first, _ := entries[0].(map[string]any)
	if ip, _ := first["installPath"].(string); ip != a.cacheRoot() {
		r.Status, r.Message, r.Fix = "fail", "installPath mismatch: "+ip, "memex setup install"
		return r
	}
	r.Status, r.Message = "pass", pluginKey+" registered"
	return r
}

func (a *Adapter) pluginJSONDiagnostic(binaryPath string) adapters.DiagnosticResult {
	r := adapters.DiagnosticResult{Check: ".mcp.json"}
	data, err := adapters.ReadSettings(a.mcpJSONPath())
	if err != nil {
		r.Status, r.Message, r.Fix = "fail", "could not read .mcp.json: "+err.Error(), "memex setup install"
		return r
	}
	mcp, _ := data["mcpServers"].(map[string]any)
	entry, _ := mcp[pluginName].(map[string]any)
	if entry == nil {
		r.Status, r.Message, r.Fix = "fail", ".mcp.json missing mcpServers."+pluginName, "memex setup install"
		return r
	}
	if cmd, _ := entry["command"].(string); cmd != binaryPath {
		r.Status, r.Message, r.Fix = "fail", ".mcp.json command does not match running binary", "memex setup install"
		return r
	}
	r.Status, r.Message = "pass", ".mcp.json declares mcpServers."+pluginName
	return r
}

func skillsDiagnostic(skillsRoot string) adapters.DiagnosticResult {
	r := adapters.DiagnosticResult{Check: "skills"}
	for _, rel := range []string{
		filepath.Join("memex", "SKILL.md"),
		filepath.Join("memex-doctor", "SKILL.md"),
		filepath.Join("memex-ops", "SKILL.md"),
	} {
		info, err := os.Stat(filepath.Join(skillsRoot, rel))
		if err != nil || info.Size() == 0 {
			r.Status, r.Message, r.Fix = "fail", "missing or empty: "+rel, "memex setup install"
			return r
		}
	}
	r.Status, r.Message = "pass", "skill bundle present"
	return r
}

// ── marketplace + plugin file authors ─────────────────────────

func (a *Adapter) writeMarketplaceTree(binaryPath string) error {
	plugin := map[string]any{
		"name":        pluginName,
		"version":     pluginVersion,
		"description": "Go port: MCP server + skills for context window management",
		"author":      map[string]any{"name": "Dreamware"},
		"skills":      "./skills/",
	}
	if err := adapters.WriteSettings(a.pluginJSONPath(), plugin); err != nil {
		return err
	}
	if err := os.Chmod(a.pluginJSONPath(), 0o644); err != nil {
		return err
	}
	mcp := map[string]any{
		"mcpServers": map[string]any{
			pluginName: map[string]any{
				"command": binaryPath,
				"args":    []any{"mcp"},
			},
		},
	}
	if err := adapters.WriteSettings(a.mcpJSONPath(), mcp); err != nil {
		return err
	}
	if err := os.Chmod(a.mcpJSONPath(), 0o644); err != nil {
		return err
	}
	market := map[string]any{
		"name":        marketplaceID,
		"description": "Dreamware local plugin marketplace",
		"owner":       map[string]any{"name": "Dreamware"},
		"plugins": []any{
			map[string]any{
				"name":        pluginName,
				"version":     pluginVersion,
				"description": "Go port: MCP server + skills for context window management",
				"author":      map[string]any{"name": "Dreamware"},
				"category":    "development",
				"source":      "./",
			},
		},
	}
	if err := adapters.WriteSettings(a.marketplaceJSONPath(), market); err != nil {
		return err
	}
	return os.Chmod(a.marketplaceJSONPath(), 0o644)
}

func (a *Adapter) registerMarketplace() error {
	data, err := adapters.ReadSettings(a.knownMarketplacesPath())
	if err != nil {
		return err
	}
	data[marketplaceID] = map[string]any{
		"source": map[string]any{
			"source": "directory",
			"path":   a.marketplaceRoot(),
		},
		"installLocation": a.marketplaceRoot(),
		"lastUpdated":     time.Now().UTC().Format(time.RFC3339),
	}
	return adapters.WriteSettings(a.knownMarketplacesPath(), data)
}

func (a *Adapter) unregisterMarketplace() error {
	data, err := adapters.ReadSettings(a.knownMarketplacesPath())
	if err != nil {
		return err
	}
	delete(data, marketplaceID)
	return adapters.WriteSettings(a.knownMarketplacesPath(), data)
}

func (a *Adapter) registerPlugin() error {
	path := a.pluginManifestPath()
	data, err := adapters.ReadSettings(path)
	if err != nil {
		return err
	}
	if _, ok := data["version"]; !ok {
		data["version"] = float64(2)
	}
	plugins := mapValue(data, "plugins")
	now := time.Now().UTC().Format(time.RFC3339)
	plugins[pluginKey] = []any{
		map[string]any{
			"scope":       "user",
			"installPath": a.cacheRoot(),
			"version":     pluginVersion,
			"installedAt": now,
			"lastUpdated": now,
		},
	}
	return adapters.WriteSettings(path, data)
}

func (a *Adapter) unregisterPlugin() error {
	path := a.pluginManifestPath()
	data, err := adapters.ReadSettings(path)
	if err != nil {
		return err
	}
	if plugins, ok := data["plugins"].(map[string]any); ok {
		delete(plugins, pluginKey)
		if len(plugins) == 0 {
			delete(data, "plugins")
		} else {
			data["plugins"] = plugins
		}
	}
	return adapters.WriteSettings(path, data)
}

// mirrorTree copies src into dst (replacing dst). Used to seed cacheRoot
// from marketplaceRoot — this is what `claude plugin install` does
// internally. Symlinks are followed, file modes are preserved.
func mirrorTree(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(p, target, info.Mode().Perm())
	})
}

func copyFile(src, dst string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

// ── helpers ──────────────────────────────────────────────────

func mapValue(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	out := map[string]any{}
	m[key] = out
	return out
}

func sliceValue(m map[string]any, key string) []any {
	if v, ok := m[key].([]any); ok {
		return v
	}
	return nil
}

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
