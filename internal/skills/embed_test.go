package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var topLevelBundles = []string{
	"memex",
	"memex-doctor",
	"memex-insight",
	"memex-ops",
	"memex-purge",
	"memex-stats",
	"memex-upgrade",
	"diagnose",
	"grill-me",
	"grill-with-docs",
	"improve-codebase-architecture",
	"tdd",
}

func TestEmbeddedFiles_NonEmpty(t *testing.T) {
	files := EmbeddedFiles()
	if len(files) == 0 {
		t.Fatalf("EmbeddedFiles() returned empty slice")
	}
}

func TestEmbeddedFiles_AllTopLevelBundlesPresent(t *testing.T) {
	files := EmbeddedFiles()
	for _, want := range topLevelBundles {
		prefix := want + "/"
		found := false
		for _, f := range files {
			if strings.HasPrefix(f, prefix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no embedded file under %q", prefix)
		}
	}
}

func TestEmbeddedFiles_AtLeast12SkillFiles(t *testing.T) {
	files := EmbeddedFiles()
	count := 0
	for _, f := range files {
		if strings.HasSuffix(f, "/SKILL.md") {
			count++
		}
	}
	if count < 12 {
		t.Errorf("only %d SKILL.md files embedded, want >= 12", count)
	}
}

func TestEmbeddedFiles_NoHiddenOrUnderscored(t *testing.T) {
	for _, f := range EmbeddedFiles() {
		for _, comp := range strings.Split(f, "/") {
			if comp == "" {
				continue
			}
			if strings.HasPrefix(comp, ".") || strings.HasPrefix(comp, "_") {
				t.Errorf("embedded path %q has hidden/underscored component %q", f, comp)
			}
		}
	}
}

func TestWrite_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	for _, rel := range EmbeddedFiles() {
		dest := filepath.Join(dir, rel)
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Errorf("read %s: %v", dest, err)
			continue
		}
		want, err := Assets.ReadFile(filepath.Join(embedRoot, rel))
		if err != nil {
			t.Fatalf("read embedded %s: %v", rel, err)
		}
		if string(got) != string(want) {
			t.Errorf("byte mismatch for %s", rel)
		}
	}
}

func TestWrite_Idempotent(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	sample := filepath.Join(dir, "memex", "SKILL.md")
	stat1, err := os.Stat(sample)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if err := Write(dir); err != nil {
		t.Fatalf("second Write: %v", err)
	}
	stat2, err := os.Stat(sample)
	if err != nil {
		t.Fatalf("stat after second write: %v", err)
	}
	if !stat1.ModTime().Equal(stat2.ModTime()) {
		t.Errorf("idempotent re-write touched mtime: %v -> %v", stat1.ModTime(), stat2.ModTime())
	}
}

func TestWrite_DriftOverwrite(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	target := filepath.Join(dir, "memex", "SKILL.md")
	if err := os.WriteFile(target, []byte("CORRUPTED"), 0o644); err != nil {
		t.Fatalf("corrupt: %v", err)
	}
	if err := Write(dir); err != nil {
		t.Fatalf("re-Write: %v", err)
	}
	got, _ := os.ReadFile(target)
	want, _ := Assets.ReadFile(filepath.Join(embedRoot, "memex", "SKILL.md"))
	if string(got) != string(want) {
		t.Errorf("drift not restored")
	}
}

func TestWrite_PreservesUserSiblings(t *testing.T) {
	dir := t.TempDir()
	userPath := filepath.Join(dir, "my-skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(userPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	userContent := []byte("# user skill\n")
	if err := os.WriteFile(userPath, userContent, 0o644); err != nil {
		t.Fatalf("write user file: %v", err)
	}
	if err := Write(dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("read user file: %v", err)
	}
	if string(got) != string(userContent) {
		t.Errorf("user file mutated: %q", got)
	}
}

func TestRemove_OnlyEmbedded(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	userPath := filepath.Join(dir, "my-skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(userPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userPath, []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Remove(dir); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	for _, rel := range EmbeddedFiles() {
		if _, err := os.Stat(filepath.Join(dir, rel)); !os.IsNotExist(err) {
			t.Errorf("embedded %s still present after Remove (err=%v)", rel, err)
		}
	}
	if _, err := os.Stat(userPath); err != nil {
		t.Errorf("user file removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "memex")); !os.IsNotExist(err) {
		t.Errorf("empty subdir memex/ still present")
	}
}

func TestRemove_MissingTargetIsNoOp(t *testing.T) {
	if err := Remove(filepath.Join(t.TempDir(), "definitely-missing")); err != nil {
		t.Errorf("Remove on missing dir: %v", err)
	}
}
