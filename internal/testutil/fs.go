package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// SetSkillConfigHome creates a temporary directory for skill config and sets XDG_CONFIG_HOME.
// Returns the temporary directory path. Caller (typically via t.Cleanup) must clean up.
func SetSkillConfigHome(t *testing.T) string {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "gwx", "skills")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill config dir: %v", err)
	}
	// Set environment variable so skill.LoadAll() reads from tmpDir
	oldHome := os.Getenv("GWX_SKILL_HOME")
	os.Setenv("GWX_SKILL_HOME", skillDir)
	t.Cleanup(func() {
		if oldHome != "" {
			os.Setenv("GWX_SKILL_HOME", oldHome)
		} else {
			os.Unsetenv("GWX_SKILL_HOME")
		}
	})
	return skillDir
}

// TempSkillFile creates a temporary YAML skill file with the given content.
// Returns the file path. Cleanup is automatic via t.TempDir.
func TempSkillFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-skill.yaml")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp skill file: %v", err)
	}
	return filePath
}
