package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const appName = "gwx"

// Dir returns the gwx config directory (~/.config/gwx on macOS/Linux).
func Dir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config dir: %w", err)
	}
	return filepath.Join(configDir, appName), nil
}

// EnsureDir creates the config directory if it doesn't exist.
func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return dir, nil
}
