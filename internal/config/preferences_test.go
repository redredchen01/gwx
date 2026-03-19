package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// prefPath returns path to preferences.json inside a temp dir.
func prefPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), preferencesFile)
}

func TestPreferencesLoad_FileNotExist(t *testing.T) {
	path := prefPath(t)
	prefs, err := loadFrom(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(prefs) != 0 {
		t.Fatalf("expected empty map, got: %v", prefs)
	}
}

func TestPreferencesLoad_MalformedJSON(t *testing.T) {
	path := prefPath(t)
	if err := os.WriteFile(path, []byte("not json {{{{"), 0600); err != nil {
		t.Fatal(err)
	}
	prefs, err := loadFrom(path)
	if err != nil {
		t.Fatalf("expected no error for malformed JSON, got: %v", err)
	}
	if len(prefs) != 0 {
		t.Fatalf("expected empty map for malformed JSON, got: %v", prefs)
	}
}

func TestPreferencesSetGet(t *testing.T) {
	path := prefPath(t)

	// Set a value via saveTo/loadFrom directly.
	prefs := map[string]string{}
	prefs["analytics.default-property"] = "UA-12345"
	if err := saveTo(path, prefs); err != nil {
		t.Fatalf("saveTo: %v", err)
	}

	got, err := loadFrom(path)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}
	if got["analytics.default-property"] != "UA-12345" {
		t.Fatalf("expected UA-12345, got %q", got["analytics.default-property"])
	}
}

func TestPreferencesDelete(t *testing.T) {
	path := prefPath(t)

	// Write two keys.
	initial := map[string]string{"key1": "val1", "key2": "val2"}
	if err := saveTo(path, initial); err != nil {
		t.Fatal(err)
	}

	// Delete key1.
	prefs, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	delete(prefs, "key1")
	if err := saveTo(path, prefs); err != nil {
		t.Fatal(err)
	}

	// Reload and verify.
	got, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := got["key1"]; ok {
		t.Fatalf("key1 should be deleted, got %q", v)
	}
	if got["key2"] != "val2" {
		t.Fatalf("key2 should remain, got %q", got["key2"])
	}
}

func TestPreferencesOverwrite(t *testing.T) {
	path := prefPath(t)

	// First write.
	if err := saveTo(path, map[string]string{"foo": "bar"}); err != nil {
		t.Fatal(err)
	}

	// Second write with same key, different value.
	prefs, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	prefs["foo"] = "baz"
	if err := saveTo(path, prefs); err != nil {
		t.Fatal(err)
	}

	// Reload and verify second value wins.
	got, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["foo"] != "baz" {
		t.Fatalf("expected baz, got %q", got["foo"])
	}
}

// TestPreferencesFilePermissions verifies the file is written with 0600.
func TestPreferencesFilePermissions(t *testing.T) {
	path := prefPath(t)
	if err := saveTo(path, map[string]string{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600, got %o", info.Mode().Perm())
	}
}

// TestPreferencesValidJSON verifies the written file is valid JSON.
func TestPreferencesValidJSON(t *testing.T) {
	path := prefPath(t)
	data := map[string]string{"alpha": "1", "beta": "2"}
	if err := saveTo(path, data); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("file is not valid JSON: %v", err)
	}
	if out["alpha"] != "1" || out["beta"] != "2" {
		t.Fatalf("unexpected content: %v", out)
	}
}
