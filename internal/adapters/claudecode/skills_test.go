package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dreamware-nz/memex/internal/skills"
)

func TestSkillsPath_Deterministic(t *testing.T) {
	home := setupHome(t)
	a := New()
	got := a.SkillsPath()
	want := filepath.Join(home, ".claude", "plugins", "marketplaces", "dreamware", "skills")
	if got != want {
		t.Errorf("SkillsPath() = %q, want %q", got, want)
	}
	if got != a.SkillsPath() {
		t.Errorf("SkillsPath() not deterministic across calls")
	}
}

func TestInstall_WritesSkillsTree(t *testing.T) {
	setupHome(t)
	a := New()
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("Install: %v", err)
	}
	for _, rel := range skills.EmbeddedFiles() {
		p := filepath.Join(a.SkillsPath(), rel)
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("missing %s: %v", rel, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("zero-size %s", rel)
		}
	}
}

func TestUninstall_RemovesEmbeddedSkillsKeepsUserFiles(t *testing.T) {
	setupHome(t)
	a := New()
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("Install: %v", err)
	}
	userPath := filepath.Join(a.SkillsPath(), "my-user-skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(userPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(userPath, []byte("# user\n"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	if err := a.Uninstall(testBinary); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	for _, rel := range skills.EmbeddedFiles() {
		if _, err := os.Stat(filepath.Join(a.SkillsPath(), rel)); !os.IsNotExist(err) {
			t.Errorf("embedded %s still present (err=%v)", rel, err)
		}
	}
	if _, err := os.Stat(userPath); err != nil {
		t.Errorf("user file removed: %v", err)
	}
}

func TestInstall_RestoresAfterDrift(t *testing.T) {
	setupHome(t)
	a := New()
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("Install: %v", err)
	}
	target := filepath.Join(a.SkillsPath(), "memex", "SKILL.md")
	if err := os.WriteFile(target, []byte("CORRUPTED"), 0o644); err != nil {
		t.Fatalf("corrupt: %v", err)
	}
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("re-Install: %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) == "CORRUPTED" {
		t.Errorf("skill drift not restored")
	}
}

func TestValidate_SkillsCheck(t *testing.T) {
	setupHome(t)
	a := New()
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("Install: %v", err)
	}
	results := a.Validate(testBinary)
	var skillsResult *struct {
		status, msg string
	}
	for _, r := range results {
		if r.Check == "skills" {
			skillsResult = &struct{ status, msg string }{r.Status, r.Message}
		}
	}
	if skillsResult == nil {
		t.Fatalf("no 'skills' diagnostic in Validate output")
	}
	if skillsResult.status != "pass" {
		t.Errorf("skills status = %q, want pass (msg=%q)", skillsResult.status, skillsResult.msg)
	}

	if err := os.RemoveAll(a.SkillsPath()); err != nil {
		t.Fatalf("rm skills: %v", err)
	}
	results = a.Validate(testBinary)
	for _, r := range results {
		if r.Check == "skills" {
			if r.Status != "fail" {
				t.Errorf("skills after rm = %q, want fail", r.Status)
			}
			if r.Fix != "memex setup install" {
				t.Errorf("skills Fix = %q, want %q", r.Fix, "memex setup install")
			}
		}
	}
}

func TestInstall_RollsBackHooksOnSkillsFailure(t *testing.T) {
	home := setupHome(t)
	// Make skills.Write fail by pre-creating a *file* at the path that
	// skills.Write needs to be a directory. os.MkdirAll on a path whose
	// ancestor is a non-directory returns ENOTDIR.
	a := New()
	blocker := a.SkillsPath()
	if err := os.MkdirAll(filepath.Dir(blocker), 0o755); err != nil {
		t.Fatalf("mkdir parents: %v", err)
	}
	if err := os.WriteFile(blocker, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	_ = home

	err := a.Install(testBinary)
	if err == nil {
		t.Fatalf("Install: expected error from skills.Write, got nil")
	}
	if !strings.Contains(err.Error(), "not a directory") &&
		!strings.Contains(err.Error(), "file exists") &&
		!strings.Contains(err.Error(), "ENOTDIR") {
		t.Logf("Install err (non-fatal log): %v", err)
	}
	// Hook delta should have been rolled back: settings.json must not
	// reference a memex hook for this binary.
	settings := readSettingsForTest(t, a.SettingsPath())
	hooks, _ := settings["hooks"].(map[string]any)
	if hookTypePresent(hooks, "PreToolUse", testBinary) {
		t.Errorf("hooks not rolled back after skills.Write failure")
	}
}
