package cursor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dreamware-nz/memex/internal/skills"
)

func TestSkillsPath_Deterministic(t *testing.T) {
	home := setupHome(t)
	a := New()
	got := a.SkillsPath()
	want := filepath.Join(home, ".cursor", "plugins", "memex", "skills")
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
		info, err := os.Stat(filepath.Join(a.SkillsPath(), rel))
		if err != nil || info.Size() == 0 {
			t.Errorf("missing/empty %s: %v", rel, err)
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
			t.Errorf("embedded %s still present", rel)
		}
	}
	if _, err := os.Stat(userPath); err != nil {
		t.Errorf("user file removed: %v", err)
	}
}

func TestValidate_SkillsCheck(t *testing.T) {
	setupHome(t)
	a := New()
	if err := a.Install(testBinary); err != nil {
		t.Fatalf("Install: %v", err)
	}
	results := a.Validate(testBinary)
	found := false
	for _, r := range results {
		if r.Check == "skills" {
			found = true
			if r.Status != "pass" {
				t.Errorf("skills = %q, want pass", r.Status)
			}
		}
	}
	if !found {
		t.Errorf("no 'skills' diagnostic")
	}
}

func TestInstall_RollsBackHooksOnSkillsFailure(t *testing.T) {
	home := setupHome(t)
	blocker := filepath.Join(home, ".cursor", "plugins", "memex")
	if err := os.MkdirAll(filepath.Dir(blocker), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(blocker, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	a := New()
	if err := a.Install(testBinary); err == nil {
		t.Fatalf("Install: expected error, got nil")
	}
	parsed := readSettingsForTest(t, a.SettingsPath())
	hooks, _ := parsed["hooks"].(map[string]any)
	if hookTypePresent(hooks, "PreToolUse", testBinary) {
		t.Errorf("hooks not rolled back")
	}
}
