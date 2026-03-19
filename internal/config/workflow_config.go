package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const workflowConfigFile = "workflow.json"

var workflowMu sync.Mutex

// GetWorkflowConfig reads a key from the workflow config file.
// Returns ("", nil) if the key does not exist.
func GetWorkflowConfig(key string) (string, error) {
	m, err := loadWorkflowConfig()
	if err != nil {
		return "", err
	}
	return m[key], nil
}

// SetWorkflowConfig writes a key-value pair to the workflow config file.
// Creates the file and directory if they don't exist.
// Uses temp file + rename for atomic writes.
func SetWorkflowConfig(key, value string) error {
	workflowMu.Lock()
	defer workflowMu.Unlock()

	m, err := loadWorkflowConfig()
	if err != nil {
		return err
	}
	m[key] = value
	return saveWorkflowConfig(m)
}

// GetAllWorkflowConfig returns all workflow config key-value pairs.
func GetAllWorkflowConfig() (map[string]string, error) {
	return loadWorkflowConfig()
}

func workflowConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, workflowConfigFile), nil
}

func loadWorkflowConfig() (map[string]string, error) {
	path, err := workflowConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	m := make(map[string]string)
	if err := json.Unmarshal(data, &m); err != nil {
		// Malformed JSON → start fresh
		return make(map[string]string), nil
	}
	return m, nil
}

func saveWorkflowConfig(m map[string]string) error {
	path, err := workflowConfigPath()
	if err != nil {
		return err
	}
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	// Atomic write: temp file + rename
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
