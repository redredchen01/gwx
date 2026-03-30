package skill

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/redredchen01/gwx/internal/config"
)

// gistPattern matches GitHub Gist URLs: https://gist.github.com/USER/ID
var gistPattern = regexp.MustCompile(`^https://gist\.github\.com/([^/]+)/([a-f0-9]+)`)

// FetchSkill downloads a skill YAML from a URL and validates it.
func FetchSkill(url string) (*Skill, []byte, error) {
	raw := resolveRawURL(url)

	// Create HTTP client with 10-second timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(raw) //nolint:gosec
	if err != nil {
		return nil, nil, fmt.Errorf("fetch skill from %s: %w", raw, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("fetch skill from %s: HTTP %d", raw, resp.StatusCode)
	}

	// Limit response to 1MB
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, nil, fmt.Errorf("read skill response from %s: %w", raw, err)
	}

	s, err := Parse(data)
	if err != nil {
		return nil, nil, fmt.Errorf("validate fetched skill: %w", err)
	}

	return s, data, nil
}

// InstallFromURL downloads and installs a skill to the user's skills directory.
// Returns the destination file path.
func InstallFromURL(url string) (string, error) {
	s, data, err := FetchSkill(url)
	if err != nil {
		return "", err
	}

	dir, err := SkillsDir()
	if err != nil {
		return "", err
	}

	dest := filepath.Join(dir, sanitizeFilename(s.Name)+".yaml")
	if err := os.WriteFile(dest, data, 0644); err != nil {
		return "", fmt.Errorf("write skill file %s: %w", dest, err)
	}

	return dest, nil
}

// InstallFromFile copies a local skill file to the user's skills directory.
// Returns the destination file path.
func InstallFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read skill file %s: %w", path, err)
	}

	s, err := Parse(data)
	if err != nil {
		return "", fmt.Errorf("validate skill file: %w", err)
	}

	dir, err := SkillsDir()
	if err != nil {
		return "", err
	}

	dest := filepath.Join(dir, sanitizeFilename(s.Name)+".yaml")
	if err := os.WriteFile(dest, data, 0644); err != nil {
		return "", fmt.Errorf("write skill file %s: %w", dest, err)
	}

	return dest, nil
}

// UninstallSkill removes a skill from the user's skills directory.
func UninstallSkill(name string) error {
	dir, err := SkillsDir()
	if err != nil {
		return err
	}

	// Try both .yaml and .yml extensions.
	candidates := []string{
		filepath.Join(dir, sanitizeFilename(name)+".yaml"),
		filepath.Join(dir, sanitizeFilename(name)+".yml"),
	}

	var removed bool
	for _, path := range candidates {
		if err := os.Remove(path); err == nil {
			removed = true
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("remove skill file %s: %w", path, err)
		}
	}

	if !removed {
		return fmt.Errorf("skill %q not found in %s", name, dir)
	}

	return nil
}

// SkillsDir returns the user's skills directory path, creating it if needed.
func SkillsDir() (string, error) {
	configDir, err := config.Dir()
	if err != nil {
		return "", fmt.Errorf("get config dir: %w", err)
	}

	dir := filepath.Join(configDir, skillsDirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create skills dir %s: %w", dir, err)
	}

	return dir, nil
}

// resolveRawURL converts known URL patterns to their raw content URLs.
func resolveRawURL(url string) string {
	// GitHub Gist → raw
	if m := gistPattern.FindStringSubmatch(url); m != nil {
		return fmt.Sprintf("https://gist.githubusercontent.com/%s/%s/raw", m[1], m[2])
	}
	return url
}

// sanitizeFilename converts a skill name into a safe filename.
// Replaces non-alphanumeric characters (except hyphens and underscores) with hyphens,
// lowercases the result, and trims edge hyphens.
func sanitizeFilename(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
