package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

const preferencesFile = "preferences.json"

// Load reads preferences from config.Dir()/preferences.json.
// Returns empty map if file doesn't exist or is malformed.
func Load() (map[string]string, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	return loadFrom(filepath.Join(dir, preferencesFile))
}

// loadFrom reads preferences from an explicit path (used internally for testing).
func loadFrom(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	var prefs map[string]string
	if err := json.Unmarshal(data, &prefs); err != nil {
		slog.Warn("preferences.json is malformed, returning empty map", "error", err)
		return make(map[string]string), nil
	}
	return prefs, nil
}

// Save writes preferences to config.Dir()/preferences.json.
func Save(prefs map[string]string) error {
	dir, err := EnsureDir()
	if err != nil {
		return err
	}
	return saveTo(filepath.Join(dir, preferencesFile), prefs)
}

// saveTo writes preferences to an explicit path (used internally for testing).
func saveTo(path string, prefs map[string]string) error {
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Get reads a single preference key.
func Get(key string) (string, error) {
	prefs, err := Load()
	if err != nil {
		return "", err
	}
	return prefs[key], nil
}

// Set writes a single preference key (read-modify-write).
func Set(key, value string) error {
	prefs, err := Load()
	if err != nil {
		return err
	}
	prefs[key] = value
	return Save(prefs)
}

// Delete removes a single preference key.
func Delete(key string) error {
	prefs, err := Load()
	if err != nil {
		return err
	}
	delete(prefs, key)
	return Save(prefs)
}
