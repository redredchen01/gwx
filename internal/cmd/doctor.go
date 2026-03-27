package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/redredchen01/gwx/internal/auth"
	"github.com/redredchen01/gwx/internal/config"
	"github.com/redredchen01/gwx/internal/skill"
)

// DoctorCmd runs a health check on the gwx installation.
type DoctorCmd struct{}

// checkResult holds the outcome of a single diagnostic check.
type checkResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "warning", "error"
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (c *DoctorCmd) Run(rctx *RunContext) error {
	var checks []checkResult

	// 1. Version (with latest release check)
	checks = append(checks, checkVersion())

	// 2. Go version
	checks = append(checks, checkResult{
		Name:    "go",
		Status:  "ok",
		Message: runtime.Version(),
		Detail:  runtime.Version(),
	})

	// 3. OS / Arch
	osInfo := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	checks = append(checks, checkResult{
		Name:    "os",
		Status:  "ok",
		Message: osInfo,
		Detail:  osInfo,
	})

	// 4. Config directory
	checks = append(checks, checkConfigDir())

	// 5. Google Auth
	checks = append(checks, checkGoogleAuth(rctx))

	// 6. GitHub Auth
	checks = append(checks, checkProviderAPI("github", "default"))

	// 7. Slack Auth
	checks = append(checks, checkProviderAPI("slack", "default"))

	// 8. Notion Auth
	checks = append(checks, checkProviderAPI("notion", "default"))

	// 9. Skills
	checks = append(checks, checkSkills()...)

	var okCount, warnCount, errCount int
	for _, ch := range checks {
		switch ch.Status {
		case "ok":
			okCount++
		case "warning":
			warnCount++
		case "error":
			errCount++
		}
	}

	overallStatus := "healthy"
	if errCount > 0 {
		overallStatus = "unhealthy"
	} else if warnCount > 0 {
		overallStatus = "degraded"
	}

	rctx.Printer.Success(map[string]interface{}{
		"status": overallStatus,
		"checks": checks,
		"summary": map[string]int{
			"ok":      okCount,
			"warning": warnCount,
			"error":   errCount,
			"total":   len(checks),
		},
	})
	return nil
}

// checkVersion checks the current version and compares against the latest GitHub release.
func checkVersion() checkResult {
	latest := fetchLatestVersion()
	if latest == "" {
		return checkResult{
			Name:    "version",
			Status:  "ok",
			Message: fmt.Sprintf("v%s (latest: unknown)", version),
			Detail:  fmt.Sprintf("v%s", version),
		}
	}

	latestClean := strings.TrimPrefix(latest, "v")
	currentClean := strings.TrimPrefix(version, "v")

	if latestClean == currentClean {
		return checkResult{
			Name:    "version",
			Status:  "ok",
			Message: fmt.Sprintf("v%s (latest: v%s)", currentClean, latestClean),
			Detail:  fmt.Sprintf("v%s", currentClean),
		}
	}

	return checkResult{
		Name:    "version",
		Status:  "warning",
		Message: fmt.Sprintf("v%s (latest: v%s, run 'brew upgrade gwx' or reinstall)", currentClean, latestClean),
		Detail:  fmt.Sprintf("current=v%s latest=v%s", currentClean, latestClean),
	}
}

// fetchLatestVersion queries GitHub API for the latest release tag.
// Returns empty string on any failure (2s timeout to avoid hanging).
func fetchLatestVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/redredchen01/gwx/releases/latest", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return ""
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return ""
	}
	return release.TagName
}

func checkConfigDir() checkResult {
	dir, err := config.Dir()
	if err != nil {
		return checkResult{
			Name:    "config",
			Status:  "error",
			Message: fmt.Sprintf("cannot determine config dir: %s", err),
		}
	}

	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return checkResult{
			Name:    "config",
			Status:  "warning",
			Message: fmt.Sprintf("%s does not exist (run 'gwx onboard' to create)", dir),
			Detail:  dir,
		}
	}
	if err != nil {
		return checkResult{
			Name:    "config",
			Status:  "error",
			Message: fmt.Sprintf("cannot stat %s: %s", dir, err),
			Detail:  dir,
		}
	}
	if !info.IsDir() {
		return checkResult{
			Name:    "config",
			Status:  "error",
			Message: fmt.Sprintf("%s exists but is not a directory", dir),
			Detail:  dir,
		}
	}

	testFile := dir + "/.doctor-write-test"
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		return checkResult{
			Name:    "config",
			Status:  "error",
			Message: fmt.Sprintf("%s is not writable: %s", dir, err),
			Detail:  dir,
		}
	}
	os.Remove(testFile)

	displayDir := dir
	if home, err := os.UserHomeDir(); err == nil {
		displayDir = strings.Replace(dir, home, "~", 1)
	}

	return checkResult{
		Name:    "config",
		Status:  "ok",
		Message: displayDir,
		Detail:  fmt.Sprintf("path=%s", dir),
	}
}

func checkGoogleAuth(rctx *RunContext) checkResult {
	account := rctx.Account
	if account == "" {
		account = "default"
	}

	if !rctx.Auth.HasToken(account) {
		return checkResult{
			Name:    "google",
			Status:  "warning",
			Message: "not authenticated (run 'gwx auth login')",
			Detail:  fmt.Sprintf("account=%s token=missing", account),
		}
	}

	expiryInfo := ""
	token, err := rctx.Auth.LoadToken(account)
	if err == nil && !token.Expiry.IsZero() {
		remaining := time.Until(token.Expiry)
		if remaining < 0 {
			expiryInfo = "token expired, will auto-refresh"
		} else {
			expiryInfo = fmt.Sprintf("expires in %s", remaining.Round(time.Minute))
		}
	}

	msg := fmt.Sprintf("authenticated as %q", account)
	if expiryInfo != "" {
		msg += " — " + expiryInfo
	}
	return checkResult{Name: "google", Status: "ok", Message: msg, Detail: fmt.Sprintf("account=%s", account)}
}

func checkProviderAPI(provider, account string) checkResult {
	name := provider
	if auth.HasProviderToken(provider, account) {
		return checkResult{
			Name:    name,
			Status:  "ok",
			Message: fmt.Sprintf("%s token present for %q", provider, account),
			Detail:  fmt.Sprintf("provider=%s account=%s", provider, account),
		}
	}
	return checkResult{
		Name:    name,
		Status:  "warning",
		Message: fmt.Sprintf("no %s token (run 'gwx %s login')", provider, provider),
		Detail:  fmt.Sprintf("provider=%s account=%s", provider, account),
	}
}

func checkSkills() []checkResult {
	skills, err := skill.LoadAll()
	if err != nil {
		return []checkResult{{
			Name:    "skills",
			Status:  "error",
			Message: fmt.Sprintf("failed to load skills: %s", err),
		}}
	}

	if len(skills) == 0 {
		return []checkResult{{
			Name:    "skills",
			Status:  "ok",
			Message: "no skills loaded (this is fine — create with 'gwx skill create <name>')",
		}}
	}

	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })

	names := make([]string, 0, len(skills))
	for _, s := range skills {
		names = append(names, s.Name)
	}

	return []checkResult{{
		Name:    "skills",
		Status:  "ok",
		Message: fmt.Sprintf("%d skill(s) loaded: %s", len(skills), joinNames(names)),
		Detail:  fmt.Sprintf("count=%d", len(skills)),
	}}
}

func joinNames(names []string) string {
	var sb strings.Builder
	limit := len(names)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(names[i])
	}
	if len(names) > 5 {
		fmt.Fprintf(&sb, " (and %d more)", len(names)-5)
	}
	return sb.String()
}
