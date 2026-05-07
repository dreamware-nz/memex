package tools

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"
)

// DoctorTool implements memex_doctor: runs in-process checks (Go runtime,
// SQLite FTS5 support, KB/analytics files, claude-code settings) and returns
// a plain-text [OK]/[FAIL]/[WARN] report. All checks are pure Go — no
// shell-out to node, npm, or any external runtime.
type DoctorTool struct {
	// Version is the memex binary version reported in the OK line. Wired in
	// by NewMCPCmd from the same string passed to NewVersionCmd.
	Version string
	// KBPath / AnalyticsPath / SettingsPath are resolved by the caller so
	// tests can substitute paths without monkey-patching.
	KBPath        string
	AnalyticsPath string
	SettingsPath  string
}

func (t *DoctorTool) Name() string { return "memex_doctor" }

func (t *DoctorTool) Description() string {
	return "Run memex diagnostics. Returns a plain-text report with [OK]/[FAIL]/[WARN] lines for runtime, SQLite FTS5, KB, analytics, and host settings checks."
}

func (t *DoctorTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (t *DoctorTool) Call(_ json.RawMessage) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "[OK] memex %s (%s/%s)\n", versionOrUnknown(t.Version), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "[OK] go runtime %s\n", runtime.Version())

	if err := probeFTS5(); err != nil {
		fmt.Fprintf(&b, "[FAIL] sqlite FTS5: %v\n", err)
	} else {
		b.WriteString("[OK] sqlite FTS5 available\n")
	}

	writeFileCheck(&b, "kb", t.KBPath)
	writeFileCheck(&b, "analytics", t.AnalyticsPath)
	writeSettingsCheck(&b, t.SettingsPath)

	return b.String(), nil
}

func versionOrUnknown(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}

func probeFTS5() error {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`CREATE VIRTUAL TABLE _fts5_probe USING fts5(x)`)
	return err
}

func writeFileCheck(b *strings.Builder, label, path string) {
	if path == "" {
		fmt.Fprintf(b, "[WARN] %s path: not configured\n", label)
		return
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Fprintf(b, "[WARN] %s: not yet created (%s)\n", label, path)
		return
	}
	if err != nil {
		fmt.Fprintf(b, "[FAIL] %s: %v\n", label, err)
		return
	}
	fmt.Fprintf(b, "[OK] %s: %s (%d bytes)\n", label, path, info.Size())
}

func writeSettingsCheck(b *strings.Builder, path string) {
	if path == "" {
		fmt.Fprintf(b, "[WARN] host settings: not configured\n")
		return
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Fprintf(b, "[WARN] host settings: %s missing — run `memex setup install`\n", path)
		return
	}
	if err != nil {
		fmt.Fprintf(b, "[FAIL] host settings: %v\n", err)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(b, "[FAIL] host settings read %s: %v\n", path, err)
		return
	}
	if !strings.Contains(string(data), "memex") {
		fmt.Fprintf(b, "[WARN] host settings: no memex hooks in %s — run `memex setup install`\n", path)
		return
	}
	fmt.Fprintf(b, "[OK] host settings: %s (%d bytes, memex hooks present)\n", filepath.Base(path), info.Size())
}
