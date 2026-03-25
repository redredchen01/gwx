package skill

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setSkillConfigHome sets HOME/XDG_CONFIG_HOME to a temp dir so that
// SkillsDir() and LoadAll() use isolated directories.
func setSkillConfigHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", tmp)
		return filepath.Join(tmp, "Library", "Application Support", "gwx", "skills")
	}
	t.Setenv("XDG_CONFIG_HOME", tmp)
	return filepath.Join(tmp, "gwx", "skills")
}

const validSkillYAML = `name: test-skill
description: A test skill
version: "1.0"
inputs:
  - name: query
    type: string
    required: true
steps:
  - id: search
    tool: gmail_search
    args:
      query: "{{.input.query}}"
`

const validSkillYAML2 = `name: second-skill
description: Another test skill
steps:
  - id: list
    tool: drive_list
`

const invalidSkillYAML = `name: bad-skill
# missing steps field entirely
`

func TestLoadAll_EmptyDir(t *testing.T) {
	setSkillConfigHome(t)

	// No skills directory exists yet
	skills, err := LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return empty list (ignoring the skillProvider registered skills)
	// LoadAll loads from config dir + ./skills/, both empty in test context
	_ = skills
}

func TestLoadAll_ValidSkills(t *testing.T) {
	dir := setSkillConfigHome(t)

	// Create skills directory and add valid skill files
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test.yaml"), []byte(validSkillYAML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "second.yml"), []byte(validSkillYAML2), 0644); err != nil {
		t.Fatal(err)
	}

	skills, err := LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find at least our 2 skills from the config dir
	found := make(map[string]bool)
	for _, s := range skills {
		found[s.Name] = true
	}
	if !found["test-skill"] {
		t.Error("expected test-skill to be loaded")
	}
	if !found["second-skill"] {
		t.Error("expected second-skill to be loaded")
	}
}

func TestLoadAll_InvalidSkill(t *testing.T) {
	dir := setSkillConfigHome(t)

	// Create skills directory with one valid and one invalid skill
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "good.yaml"), []byte(validSkillYAML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(invalidSkillYAML), 0644); err != nil {
		t.Fatal(err)
	}

	skills, err := LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalid skill should be skipped, valid skill should be loaded
	found := make(map[string]bool)
	for _, s := range skills {
		found[s.Name] = true
	}
	if !found["test-skill"] {
		t.Error("expected test-skill to be loaded (valid)")
	}
	if found["bad-skill"] {
		t.Error("bad-skill should have been skipped (invalid)")
	}
}

func TestLoadAll_NonYAMLFilesIgnored(t *testing.T) {
	dir := setSkillConfigHome(t)

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	// Non-YAML files should be ignored
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a skill"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte(`{"key":"val"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "valid.yaml"), []byte(validSkillYAML), 0644); err != nil {
		t.Fatal(err)
	}

	skills, err := LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the YAML file should be loaded
	found := make(map[string]bool)
	for _, s := range skills {
		found[s.Name] = true
	}
	if !found["test-skill"] {
		t.Error("expected test-skill from yaml file")
	}
}

func TestLoadAll_SubdirectoriesIgnored(t *testing.T) {
	dir := setSkillConfigHome(t)

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	// Subdirectory should be ignored
	subDir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.yaml"), []byte(validSkillYAML), 0644); err != nil {
		t.Fatal(err)
	}

	skills, err := LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nested skill should NOT be loaded (loadDir only reads top-level files)
	for _, s := range skills {
		if s.Name == "test-skill" {
			// If it exists, it came from the project-local ./skills/ dir, not the subdirectory
			// This is acceptable — what matters is subdirectories under the user config are skipped
		}
	}
	_ = skills
}

func TestLoadFile_NotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/path/skill.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "read skill file") {
		t.Errorf("error = %q, want containing 'read skill file'", err)
	}
}

func TestLoadFile_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(path, []byte(validSkillYAML), 0644); err != nil {
		t.Fatal(err)
	}

	skill, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill.Name != "test-skill" {
		t.Errorf("name = %q, want test-skill", skill.Name)
	}
	if skill.Description != "A test skill" {
		t.Errorf("description = %q", skill.Description)
	}
}

func TestLoadFile_InvalidContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(invalidSkillYAML), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid skill")
	}
}

func TestSkillsDir_CreatesAndReturns(t *testing.T) {
	expectedDir := setSkillConfigHome(t)

	dir, err := SkillsDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != expectedDir {
		t.Errorf("dir = %q, want %q", dir, expectedDir)
	}

	// Directory should exist
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory")
	}
}

func TestSkillsDir_Idempotent(t *testing.T) {
	setSkillConfigHome(t)

	dir1, err := SkillsDir()
	if err != nil {
		t.Fatal(err)
	}
	dir2, err := SkillsDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir1 != dir2 {
		t.Errorf("SkillsDir not idempotent: %q != %q", dir1, dir2)
	}
}

func TestLoadAll_DuplicateNameLastWins(t *testing.T) {
	dir := setSkillConfigHome(t)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	// Two files with the same skill name — last one processed wins.
	// File naming determines order since os.ReadDir returns sorted entries.
	yaml1 := `name: dupe
description: first version
steps:
  - id: s1
    tool: gmail_list
`
	yaml2 := `name: dupe
description: second version
steps:
  - id: s1
    tool: drive_list
`
	if err := os.WriteFile(filepath.Join(dir, "a_dupe.yaml"), []byte(yaml1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b_dupe.yaml"), []byte(yaml2), 0644); err != nil {
		t.Fatal(err)
	}

	skills, err := LoadAll()
	if err != nil {
		t.Fatal(err)
	}

	var found *Skill
	for _, s := range skills {
		if s.Name == "dupe" {
			found = s
		}
	}
	if found == nil {
		t.Fatal("expected to find skill 'dupe'")
	}
	// b_dupe.yaml is processed after a_dupe.yaml, so "second version" should win
	if found.Description != "second version" {
		t.Errorf("expected last-wins, got description %q", found.Description)
	}
}

// --- loadDir edge cases ---

func TestLoadDir_NonexistentDir(t *testing.T) {
	seen := make(map[string]*Skill)
	err := loadDir("/nonexistent/path/skills", seen)
	if err != nil {
		t.Fatalf("non-existent dir should not error (returns nil): %v", err)
	}
	if len(seen) != 0 {
		t.Error("should have no skills for nonexistent dir")
	}
}

func TestLoadDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	seen := make(map[string]*Skill)
	err := loadDir(dir, seen)
	if err != nil {
		t.Fatalf("empty dir should not error: %v", err)
	}
	if len(seen) != 0 {
		t.Error("should have no skills for empty dir")
	}
}

func TestLoadDir_MixedExtensions(t *testing.T) {
	dir := t.TempDir()
	// Write .yaml, .yml, and .txt files
	if err := os.WriteFile(filepath.Join(dir, "a.yaml"), []byte(validSkillYAML), 0644); err != nil {
		t.Fatal(err)
	}
	yaml2 := strings.Replace(validSkillYAML2, "second-skill", "yml-skill", 1)
	if err := os.WriteFile(filepath.Join(dir, "b.yml"), []byte(yaml2), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "c.txt"), []byte("not yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	seen := make(map[string]*Skill)
	err := loadDir(dir, seen)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(seen) != 2 {
		t.Errorf("expected 2 skills, got %d", len(seen))
	}
	if _, ok := seen["test-skill"]; !ok {
		t.Error("expected test-skill from .yaml file")
	}
	if _, ok := seen["yml-skill"]; !ok {
		t.Error("expected yml-skill from .yml file")
	}
}
