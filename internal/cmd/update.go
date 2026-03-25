package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// UpdateCmd checks for and installs updates from GitHub Releases.
type UpdateCmd struct {
	Check bool `help:"Only check for updates, don't install" name:"check"`
}

// githubRelease represents the relevant fields from GitHub's release API.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

const (
	releaseAPIURL   = "https://api.github.com/repos/redredchen01/gwx/releases/latest"
	releaseDownload = "https://github.com/redredchen01/gwx/releases/download/%s/%s"
)

// Run executes the update command.
func (c *UpdateCmd) Run(rctx *RunContext) error {
	currentVersion := version

	// Fetch latest release info from GitHub
	latest, err := fetchLatestRelease()
	if err != nil {
		return rctx.Printer.ErrExit(1, fmt.Sprintf("failed to check for updates: %v", err))
	}

	latestVersion := strings.TrimPrefix(latest.TagName, "v")
	currentClean := strings.TrimPrefix(currentVersion, "v")

	if latestVersion == currentClean {
		rctx.Printer.Success(map[string]interface{}{
			"current": currentClean,
			"latest":  latestVersion,
			"status":  "up_to_date",
			"message": "You are running the latest version.",
		})
		return nil
	}

	// Update available
	if c.Check {
		rctx.Printer.Success(map[string]interface{}{
			"current":          currentClean,
			"latest":           latestVersion,
			"status":           "update_available",
			"message":          fmt.Sprintf("Update available: %s → %s", currentClean, latestVersion),
			"update_command":   "gwx update",
		})
		return nil
	}

	// Perform the update
	tag := latest.TagName
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}

	binaryName := buildBinaryName(tag)
	downloadURL := fmt.Sprintf(releaseDownload, tag, binaryName)

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return rctx.Printer.ErrExit(1, fmt.Sprintf("failed to determine executable path: %v", err))
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return rctx.Printer.ErrExit(1, fmt.Sprintf("failed to resolve executable path: %v", err))
	}

	// Download to temp file in the same directory (ensures same filesystem for rename)
	dir := filepath.Dir(execPath)
	tmpFile, err := os.CreateTemp(dir, "gwx-update-*")
	if err != nil {
		return rctx.Printer.ErrExit(1, fmt.Sprintf("failed to create temp file: %v", err))
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // clean up on failure

	if err := downloadBinary(downloadURL, tmpFile); err != nil {
		tmpFile.Close()
		return rctx.Printer.ErrExit(1, fmt.Sprintf("failed to download update: %v", err))
	}
	tmpFile.Close()

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return rctx.Printer.ErrExit(1, fmt.Sprintf("failed to set permissions: %v", err))
	}

	// Verify the downloaded binary works
	if err := verifyBinary(tmpPath); err != nil {
		return rctx.Printer.ErrExit(1, fmt.Sprintf("downloaded binary verification failed: %v", err))
	}

	// Replace current binary
	if err := os.Rename(tmpPath, execPath); err != nil {
		return rctx.Printer.ErrExit(1, fmt.Sprintf("failed to replace binary: %v", err))
	}

	rctx.Printer.Success(map[string]interface{}{
		"current":  currentClean,
		"latest":   latestVersion,
		"status":   "updated",
		"message":  fmt.Sprintf("Successfully updated gwx: %s → %s", currentClean, latestVersion),
		"path":     execPath,
	})
	return nil
}

// fetchLatestRelease queries the GitHub API for the latest release.
func fetchLatestRelease() (*githubRelease, error) {
	req, err := http.NewRequest("GET", releaseAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gwx-updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if release.TagName == "" {
		return nil, fmt.Errorf("no tag_name in release response")
	}

	return &release, nil
}

// buildBinaryName constructs the expected binary filename for the current platform.
func buildBinaryName(tag string) string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	name := fmt.Sprintf("gwx_%s_%s_%s", tag, goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// downloadBinary downloads from url into the given file, following redirects.
func downloadBinary(url string, dst *os.File) error {
	// http.DefaultClient follows redirects by default (up to 10)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d for %s", resp.StatusCode, url)
	}

	if _, err := io.Copy(dst, resp.Body); err != nil {
		return fmt.Errorf("writing binary: %w", err)
	}

	return nil
}

// verifyBinary runs "gwx version" on the downloaded binary to confirm it's valid.
func verifyBinary(path string) error {
	out, err := exec.Command(path, "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("binary failed verification (output: %s): %w", string(out), err)
	}
	return nil
}
