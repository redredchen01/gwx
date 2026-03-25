package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setTestConfigHome sets up an isolated config directory for testing.
// Returns the expected gwx config directory path.
func setTestConfigHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", tmp)
		return filepath.Join(tmp, "Library", "Application Support", appName)
	}
	t.Setenv("XDG_CONFIG_HOME", tmp)
	return filepath.Join(tmp, appName)
}

// --- Dir tests ---

func TestDir_ReturnsPath(t *testing.T) {
	expectedDir := setTestConfigHome(t)

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if dir != expectedDir {
		t.Errorf("Dir() = %q, want %q", dir, expectedDir)
	}
}

func TestDir_ContainsAppName(t *testing.T) {
	setTestConfigHome(t)

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if !strings.HasSuffix(dir, appName) {
		t.Errorf("Dir() = %q, should end with %q", dir, appName)
	}
}

func TestDir_Idempotent(t *testing.T) {
	setTestConfigHome(t)

	dir1, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	dir2, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if dir1 != dir2 {
		t.Errorf("Dir() not idempotent: %q != %q", dir1, dir2)
	}
}

// --- EnsureDir tests ---

func TestEnsureDir_CreatesDir(t *testing.T) {
	expectedDir := setTestConfigHome(t)

	dir, err := EnsureDir()
	if err != nil {
		t.Fatalf("EnsureDir() error: %v", err)
	}
	if dir != expectedDir {
		t.Errorf("EnsureDir() = %q, want %q", dir, expectedDir)
	}

	// Verify directory was actually created
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory")
	}
}

func TestEnsureDir_Permissions(t *testing.T) {
	setTestConfigHome(t)

	dir, err := EnsureDir()
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0700 {
		t.Errorf("directory permissions = %o, want 0700", perm)
	}
}

func TestEnsureDir_Idempotent(t *testing.T) {
	setTestConfigHome(t)

	dir1, err := EnsureDir()
	if err != nil {
		t.Fatal(err)
	}
	dir2, err := EnsureDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir1 != dir2 {
		t.Errorf("EnsureDir() not idempotent: %q != %q", dir1, dir2)
	}
}

func TestEnsureDir_CanWriteFile(t *testing.T) {
	setTestConfigHome(t)

	dir, err := EnsureDir()
	if err != nil {
		t.Fatal(err)
	}

	// Should be able to create a file in the directory
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("cannot write to config dir: %v", err)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "test" {
		t.Errorf("read back %q, want 'test'", data)
	}
}

// --- Load tests (using loadFrom for isolation) ---

func TestLoad_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), preferencesFile)
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	prefs, err := loadFrom(path)
	if err != nil {
		t.Fatalf("loadFrom error: %v", err)
	}
	if len(prefs) != 0 {
		t.Errorf("expected empty map, got %v", prefs)
	}
}

func TestLoad_WithData(t *testing.T) {
	path := filepath.Join(t.TempDir(), preferencesFile)
	data := map[string]string{"key1": "val1", "key2": "val2"}
	raw, _ := json.Marshal(data)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}

	prefs, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if prefs["key1"] != "val1" || prefs["key2"] != "val2" {
		t.Errorf("unexpected prefs: %v", prefs)
	}
}

// --- Save tests (using saveTo for isolation) ---

func TestSave_CreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), preferencesFile)
	prefs := map[string]string{"new_key": "new_val"}

	if err := saveTo(path, prefs); err != nil {
		t.Fatalf("saveTo error: %v", err)
	}

	// File should exist
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// Read back and verify
	got, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["new_key"] != "new_val" {
		t.Errorf("read back %q, want 'new_val'", got["new_key"])
	}
}

func TestSave_OverwritesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), preferencesFile)

	// First write
	if err := saveTo(path, map[string]string{"a": "1"}); err != nil {
		t.Fatal(err)
	}

	// Overwrite
	if err := saveTo(path, map[string]string{"b": "2"}); err != nil {
		t.Fatal(err)
	}

	got, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["a"]; ok {
		t.Error("key 'a' should be gone after overwrite")
	}
	if got["b"] != "2" {
		t.Errorf("key 'b' = %q, want '2'", got["b"])
	}
}

func TestSave_WritesValidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), preferencesFile)
	prefs := map[string]string{
		"alpha": "1",
		"beta":  "2",
		"gamma": "3",
	}

	if err := saveTo(path, prefs); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]string
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("file is not valid JSON: %v", err)
	}
	for k, v := range prefs {
		if parsed[k] != v {
			t.Errorf("key %q = %q, want %q", k, parsed[k], v)
		}
	}
}

// --- Get tests ---

func TestGet_MissingKey(t *testing.T) {
	setTestConfigHome(t)

	val, err := Get("nonexistent_key")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if val != "" {
		t.Errorf("Get(nonexistent) = %q, want empty", val)
	}
}

func TestGet_ExistingKey(t *testing.T) {
	configDir := setTestConfigHome(t)
	os.MkdirAll(configDir, 0700)

	// Pre-write a preferences file
	prefs := map[string]string{"existing_key": "existing_val"}
	raw, _ := json.Marshal(prefs)
	os.WriteFile(filepath.Join(configDir, preferencesFile), raw, 0600)

	val, err := Get("existing_key")
	if err != nil {
		t.Fatal(err)
	}
	if val != "existing_val" {
		t.Errorf("Get(existing_key) = %q, want 'existing_val'", val)
	}
}

// --- Set tests ---

func TestSet_PersistsValue(t *testing.T) {
	setTestConfigHome(t)

	if err := Set("test_key", "test_value"); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	val, err := Get("test_key")
	if err != nil {
		t.Fatal(err)
	}
	if val != "test_value" {
		t.Errorf("Get after Set = %q, want 'test_value'", val)
	}
}

func TestSet_UpdatesExistingKey(t *testing.T) {
	setTestConfigHome(t)

	if err := Set("key", "old_value"); err != nil {
		t.Fatal(err)
	}
	if err := Set("key", "new_value"); err != nil {
		t.Fatal(err)
	}

	val, err := Get("key")
	if err != nil {
		t.Fatal(err)
	}
	if val != "new_value" {
		t.Errorf("updated key = %q, want 'new_value'", val)
	}
}

func TestSet_PreservesOtherKeys(t *testing.T) {
	setTestConfigHome(t)

	if err := Set("key1", "val1"); err != nil {
		t.Fatal(err)
	}
	if err := Set("key2", "val2"); err != nil {
		t.Fatal(err)
	}

	val1, _ := Get("key1")
	val2, _ := Get("key2")
	if val1 != "val1" || val2 != "val2" {
		t.Errorf("keys not preserved: key1=%q, key2=%q", val1, val2)
	}
}

// --- Delete tests ---

func TestDelete_RemovesKey(t *testing.T) {
	setTestConfigHome(t)

	if err := Set("to_delete", "value"); err != nil {
		t.Fatal(err)
	}

	// Verify it exists
	val, _ := Get("to_delete")
	if val != "value" {
		t.Fatalf("key should exist before delete, got %q", val)
	}

	if err := Delete("to_delete"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	val, _ = Get("to_delete")
	if val != "" {
		t.Errorf("key should be gone after delete, got %q", val)
	}
}

func TestDelete_PreservesOtherKeys(t *testing.T) {
	setTestConfigHome(t)

	Set("keep", "value")
	Set("remove", "value")

	Delete("remove")

	val, _ := Get("keep")
	if val != "value" {
		t.Errorf("'keep' should be preserved, got %q", val)
	}
	val, _ = Get("remove")
	if val != "" {
		t.Errorf("'remove' should be gone, got %q", val)
	}
}

func TestDelete_NonexistentKey(t *testing.T) {
	setTestConfigHome(t)

	// Deleting a nonexistent key should not error
	if err := Delete("never_existed"); err != nil {
		t.Fatalf("Delete nonexistent should not error: %v", err)
	}
}

// --- Integration: Set-Get-Delete cycle ---

func TestSetGetDeleteCycle(t *testing.T) {
	setTestConfigHome(t)

	// Set
	if err := Set("cycle_key", "cycle_val"); err != nil {
		t.Fatal(err)
	}

	// Get
	val, err := Get("cycle_key")
	if err != nil || val != "cycle_val" {
		t.Fatalf("Get after Set: val=%q, err=%v", val, err)
	}

	// Delete
	if err := Delete("cycle_key"); err != nil {
		t.Fatal(err)
	}

	// Get after Delete
	val, err = Get("cycle_key")
	if err != nil || val != "" {
		t.Fatalf("Get after Delete: val=%q, err=%v", val, err)
	}
}

// --- Load/Save integration via the public API ---

func TestLoad_Integration_NoFileReturnsEmpty(t *testing.T) {
	setTestConfigHome(t)

	prefs, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if len(prefs) != 0 {
		t.Errorf("expected empty prefs, got %v", prefs)
	}
}

func TestSave_Integration_CreatesConfigDir(t *testing.T) {
	configDir := setTestConfigHome(t)

	prefs := map[string]string{"key": "val"}
	if err := Save(prefs); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Config directory should exist
	if _, err := os.Stat(configDir); err != nil {
		t.Fatalf("config dir not created: %v", err)
	}

	// Preferences file should exist
	prefFile := filepath.Join(configDir, preferencesFile)
	if _, err := os.Stat(prefFile); err != nil {
		t.Fatalf("preferences file not created: %v", err)
	}
}

// --- Multiple rapid Set calls ---

func TestSet_MultipleRapidCalls(t *testing.T) {
	setTestConfigHome(t)

	for i := 0; i < 10; i++ {
		key := "key_" + strings.Repeat("x", i)
		val := "val_" + strings.Repeat("y", i)
		if err := Set(key, val); err != nil {
			t.Fatalf("Set(%q) error: %v", key, err)
		}
	}

	// Verify all keys persist
	for i := 0; i < 10; i++ {
		key := "key_" + strings.Repeat("x", i)
		want := "val_" + strings.Repeat("y", i)
		got, err := Get(key)
		if err != nil {
			t.Fatalf("Get(%q) error: %v", key, err)
		}
		if got != want {
			t.Errorf("Get(%q) = %q, want %q", key, got, want)
		}
	}
}
