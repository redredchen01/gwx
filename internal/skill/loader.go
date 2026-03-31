package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/redredchen01/gwx/internal/config"
)

const skillsDirName = "skills"

// LoadAll discovers and parses all .yaml/.yml skill files from:
//  1. <config-dir>/skills/  (user skills)
//  2. ./skills/             (project-local skills)
//
// Duplicate names are resolved by last-wins (project overrides user).
func LoadAll() ([]*Skill, error) {
	seen := make(map[string]*Skill)

	// 1. User-level skills (~/.config/gwx/skills/)
	configDir, err := config.Dir()
	if err == nil {
		userDir := filepath.Join(configDir, skillsDirName)
		if err := loadDir(userDir, seen); err != nil {
			// Non-fatal: dir may not exist.
			_ = err
		}
	}

	// 2. Project-local skills (./skills/)
	if err := loadDir(skillsDirName, seen); err != nil {
		// Non-fatal: dir may not exist.
		_ = err
	}

	skills := make([]*Skill, 0, len(seen))
	for _, s := range seen {
		skills = append(skills, s)
	}
	return skills, nil
}

// LoadFile loads a single skill from a specific path.
func LoadFile(path string) (*Skill, error) {
	return ParseFile(path)
}

func loadDir(dir string, seen map[string]*Skill) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read skills dir %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		s, err := ParseFile(path)
		if err != nil {
			// Log but don't abort loading other skills.
			fmt.Fprintf(os.Stderr, "gwx: skip invalid skill %s: %s\n", path, err)
			continue
		}
		seen[s.Name] = s
	}
	return nil
}
