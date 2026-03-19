package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// setConfigHome sets up a temp config dir that works on all platforms.
// On macOS os.UserConfigDir() uses $HOME/Library/Application Support.
func setConfigHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if runtime.GOOS == "darwin" {
		// Create the expected structure: tmp/Library/Application Support
		home := tmp
		t.Setenv("HOME", home)
		return filepath.Join(home, "Library", "Application Support", appName)
	}
	t.Setenv("XDG_CONFIG_HOME", tmp)
	return filepath.Join(tmp, appName)
}

func TestWorkflowConfigGetMissing(t *testing.T) {
	setConfigHome(t)

	val, err := GetWorkflowConfig("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "" {
		t.Fatalf("expected empty string, got %q", val)
	}
}

func TestWorkflowConfigSetAndGet(t *testing.T) {
	setConfigHome(t)

	if err := SetWorkflowConfig("test-matrix.sheet-id", "abc123"); err != nil {
		t.Fatalf("SetWorkflowConfig failed: %v", err)
	}

	val, err := GetWorkflowConfig("test-matrix.sheet-id")
	if err != nil {
		t.Fatalf("GetWorkflowConfig failed: %v", err)
	}
	if val != "abc123" {
		t.Fatalf("expected abc123, got %q", val)
	}
}

func TestWorkflowConfigAtomicWrite(t *testing.T) {
	configDir := setConfigHome(t)

	if err := SetWorkflowConfig("key1", "val1"); err != nil {
		t.Fatalf("first set failed: %v", err)
	}
	if err := SetWorkflowConfig("key2", "val2"); err != nil {
		t.Fatalf("second set failed: %v", err)
	}

	// Both should exist
	v1, _ := GetWorkflowConfig("key1")
	v2, _ := GetWorkflowConfig("key2")
	if v1 != "val1" || v2 != "val2" {
		t.Fatalf("expected val1/val2, got %q/%q", v1, v2)
	}

	// No temp file should remain
	entries, _ := os.ReadDir(configDir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("temp file not cleaned up: %s", e.Name())
		}
	}
}

func TestWorkflowConfigAutoCreateDir(t *testing.T) {
	configDir := setConfigHome(t)

	// Dir doesn't exist yet
	if err := SetWorkflowConfig("auto.key", "auto.val"); err != nil {
		t.Fatalf("auto-create dir failed: %v", err)
	}

	// File should exist
	path := filepath.Join(configDir, workflowConfigFile)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
}

func TestWorkflowConfigGetAll(t *testing.T) {
	setConfigHome(t)

	_ = SetWorkflowConfig("a", "1")
	_ = SetWorkflowConfig("b", "2")

	all, err := GetAllWorkflowConfig()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(all) != 2 || all["a"] != "1" || all["b"] != "2" {
		t.Fatalf("unexpected: %v", all)
	}
}

func TestWorkflowConfigMalformedJSON(t *testing.T) {
	configDir := setConfigHome(t)

	// Write malformed JSON
	os.MkdirAll(configDir, 0700)
	os.WriteFile(filepath.Join(configDir, workflowConfigFile), []byte("{bad"), 0600)

	// Should return empty map, no error
	val, err := GetWorkflowConfig("key")
	if err != nil {
		t.Fatalf("unexpected error on malformed JSON: %v", err)
	}
	if val != "" {
		t.Fatalf("expected empty, got %q", val)
	}
}
