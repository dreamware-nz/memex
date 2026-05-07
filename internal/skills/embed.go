// Package skills bundles the memex skill markdown tree into the Go
// binary via embed.FS and exposes Write/Remove helpers used by the host
// adapters during `memex setup install` / `memex setup uninstall`.
//
// The skill tree is a static snapshot of memex/skills/. The snapshot
// commit and date are recorded in assets/UPSTREAM-CREDITS.md.
package skills

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

//go:embed all:assets
var Assets embed.FS

// embedRoot is the prefix that fs.WalkDir reports for the embed root. We
// strip it so callers see clean relative paths like "memex/SKILL.md".
const embedRoot = "assets"

// EmbeddedFiles returns every embedded file path relative to assets/, in
// stable lexical order. Directories are omitted.
func EmbeddedFiles() []string {
	var out []string
	_ = fs.WalkDir(Assets, embedRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(embedRoot, p)
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out
}

// Write places every embedded skill file under targetDir, preserving the
// embedded directory structure. Files whose contents already match the
// embedded copy byte-for-byte are skipped (idempotent re-runs do not bump
// mtime). Files at the destination that are not part of the embed are left
// untouched, so user-authored skills colocated under the same root survive.
func Write(targetDir string) error {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(Assets, embedRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(embedRoot, p)
		dest := filepath.Join(targetDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		want, err := Assets.ReadFile(p)
		if err != nil {
			return err
		}
		if existing, err := os.ReadFile(dest); err == nil && bytes.Equal(existing, want) {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dest, want, 0o644)
	})
}

// Remove deletes every embedded file path from targetDir, then prunes any
// subdirectory under targetDir that became empty as a result. Files at the
// destination that are not part of the embed are preserved. A missing
// targetDir is a no-op (returns nil).
func Remove(targetDir string) error {
	if _, err := os.Stat(targetDir); errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	files := EmbeddedFiles()
	for _, rel := range files {
		dest := filepath.Join(targetDir, rel)
		if err := os.Remove(dest); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return pruneEmptyDirs(targetDir)
}

// pruneEmptyDirs walks targetDir bottom-up and removes any directory that
// is empty after the file removals. targetDir itself is preserved even if
// empty so callers can re-Write into it without recreating.
func pruneEmptyDirs(targetDir string) error {
	var dirs []string
	_ = filepath.WalkDir(targetDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && p != targetDir {
			dirs = append(dirs, p)
		}
		return nil
	})
	sort.Sort(sort.Reverse(sort.StringSlice(dirs)))
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		if len(entries) == 0 {
			_ = os.Remove(d)
		}
	}
	return nil
}
