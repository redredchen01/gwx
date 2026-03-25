package skill

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setMarketplaceConfigHome sets up an isolated config home for marketplace tests.
func setMarketplaceConfigHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", tmp)
		return filepath.Join(tmp, "Library", "Application Support", "gwx", "skills")
	}
	t.Setenv("XDG_CONFIG_HOME", tmp)
	return filepath.Join(tmp, "gwx", "skills")
}

const installableSkillYAML = `name: installable
description: An installable skill
version: "1.0"
steps:
  - id: s1
    tool: gmail_list
    args:
      limit: "10"
`

func TestInstallFromFile(t *testing.T) {
	skillsDir := setMarketplaceConfigHome(t)

	// Create a temporary skill file
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "my-skill.yaml")
	if err := os.WriteFile(srcPath, []byte(installableSkillYAML), 0644); err != nil {
		t.Fatal(err)
	}

	dest, err := InstallFromFile(srcPath)
	if err != nil {
		t.Fatalf("InstallFromFile failed: %v", err)
	}

	// Verify destination path is in skills directory
	if !strings.HasPrefix(dest, skillsDir) {
		t.Errorf("dest = %q, want prefix %q", dest, skillsDir)
	}

	// Verify file was copied
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read installed file: %v", err)
	}
	if !strings.Contains(string(data), "installable") {
		t.Error("installed file should contain skill content")
	}

	// Verify the installed skill can be parsed
	s, err := ParseFile(dest)
	if err != nil {
		t.Fatalf("installed skill not parseable: %v", err)
	}
	if s.Name != "installable" {
		t.Errorf("name = %q, want installable", s.Name)
	}
}

func TestInstallFromFile_InvalidSkill(t *testing.T) {
	setMarketplaceConfigHome(t)

	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "bad.yaml")
	// Invalid skill: missing steps
	if err := os.WriteFile(srcPath, []byte("name: bad\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := InstallFromFile(srcPath)
	if err == nil {
		t.Fatal("expected error for invalid skill")
	}
	if !strings.Contains(err.Error(), "validate skill file") {
		t.Errorf("error = %q, want containing 'validate skill file'", err)
	}
}

func TestInstallFromFile_FileNotFound(t *testing.T) {
	setMarketplaceConfigHome(t)

	_, err := InstallFromFile("/nonexistent/skill.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "read skill file") {
		t.Errorf("error = %q, want containing 'read skill file'", err)
	}
}

func TestUninstallSkill(t *testing.T) {
	skillsDir := setMarketplaceConfigHome(t)

	// Install a skill first
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "removable.yaml")
	if err := os.WriteFile(srcPath, []byte(installableSkillYAML), 0644); err != nil {
		t.Fatal(err)
	}

	dest, err := InstallFromFile(srcPath)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("installed file should exist: %v", err)
	}

	// Uninstall
	if err := UninstallSkill("installable"); err != nil {
		t.Fatalf("UninstallSkill failed: %v", err)
	}

	// Verify file is removed
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("skill file should be removed after uninstall")
	}

	// Verify the skills directory still exists
	if _, err := os.Stat(skillsDir); err != nil {
		t.Error("skills directory should still exist")
	}
}

func TestUninstallSkill_NotFound(t *testing.T) {
	setMarketplaceConfigHome(t)

	// Create the skills directory but don't install anything
	dir, err := SkillsDir()
	if err != nil {
		t.Fatal(err)
	}
	_ = dir

	err = UninstallSkill("nonexistent-skill")
	if err == nil {
		t.Fatal("expected error for uninstalling nonexistent skill")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want containing 'not found'", err)
	}
}

func TestUninstallSkill_YMLExtension(t *testing.T) {
	setMarketplaceConfigHome(t)

	dir, err := SkillsDir()
	if err != nil {
		t.Fatal(err)
	}

	// Manually create a .yml file
	ymlPath := filepath.Join(dir, "yml-skill.yml")
	if err := os.WriteFile(ymlPath, []byte(installableSkillYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Uninstall should find .yml extension too
	if err := UninstallSkill("yml-skill"); err != nil {
		t.Fatalf("UninstallSkill(.yml) failed: %v", err)
	}

	if _, err := os.Stat(ymlPath); !os.IsNotExist(err) {
		t.Error(".yml file should be removed")
	}
}

func TestFetchSkill_InvalidURL(t *testing.T) {
	_, _, err := FetchSkill("http://127.0.0.1:1/nonexistent")
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
	if !strings.Contains(err.Error(), "fetch skill") {
		t.Errorf("error = %q, want containing 'fetch skill'", err)
	}
}

func TestFetchSkill_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, _, err := FetchSkill(server.URL + "/skill.yaml")
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("error = %q, want containing 'HTTP 404'", err)
	}
}

func TestFetchSkill_InvalidYAML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not valid yaml content {{{{"))
	}))
	defer server.Close()

	_, _, err := FetchSkill(server.URL + "/skill.yaml")
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "validate") {
		t.Errorf("error = %q, want containing 'validate'", err)
	}
}

func TestFetchSkill_ValidSkill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(installableSkillYAML))
	}))
	defer server.Close()

	skill, data, err := FetchSkill(server.URL + "/skill.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if skill.Name != "installable" {
		t.Errorf("name = %q, want installable", skill.Name)
	}
	if len(data) == 0 {
		t.Error("data should not be empty")
	}
}

func TestInstallFromURL(t *testing.T) {
	skillsDir := setMarketplaceConfigHome(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(installableSkillYAML))
	}))
	defer server.Close()

	dest, err := InstallFromURL(server.URL + "/skill.yaml")
	if err != nil {
		t.Fatalf("InstallFromURL failed: %v", err)
	}

	if !strings.HasPrefix(dest, skillsDir) {
		t.Errorf("dest = %q, want prefix %q", dest, skillsDir)
	}

	// Verify file exists and is valid
	s, err := ParseFile(dest)
	if err != nil {
		t.Fatalf("installed skill not parseable: %v", err)
	}
	if s.Name != "installable" {
		t.Errorf("name = %q, want installable", s.Name)
	}
}

func TestInstallFromURL_InvalidSkill(t *testing.T) {
	setMarketplaceConfigHome(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("name: no-steps\n"))
	}))
	defer server.Close()

	_, err := InstallFromURL(server.URL + "/bad.yaml")
	if err == nil {
		t.Fatal("expected error for invalid skill from URL")
	}
}

// --- resolveRawURL tests ---

func TestResolveRawURL_GistURL(t *testing.T) {
	url := "https://gist.github.com/user123/abc123def456"
	got := resolveRawURL(url)
	want := "https://gist.githubusercontent.com/user123/abc123def456/raw"
	if got != want {
		t.Errorf("resolveRawURL(%q) = %q, want %q", url, got, want)
	}
}

func TestResolveRawURL_NonGistURL(t *testing.T) {
	url := "https://example.com/skill.yaml"
	got := resolveRawURL(url)
	if got != url {
		t.Errorf("resolveRawURL(%q) = %q, want unchanged", url, got)
	}
}

func TestResolveRawURL_PlainURL(t *testing.T) {
	url := "http://localhost:8080/skills/test.yaml"
	got := resolveRawURL(url)
	if got != url {
		t.Errorf("resolveRawURL(%q) = %q, want unchanged", url, got)
	}
}

// --- sanitizeFilename tests ---

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"with-dash", "with-dash"},
		{"with_underscore", "with_underscore"},
		{"With Spaces", "with-spaces"},
		{"UPPERCASE", "uppercase"},
		{"special!@#chars", "special---chars"},
		{"---leading-trailing---", "leading-trailing"},
		{"mixed.dots.and-dashes", "mixed-dots-and-dashes"},
		{"123numbers", "123numbers"},
	}
	for _, tt := range tests {
		got := sanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
