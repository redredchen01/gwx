package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

	// 4. Config directory (with disk usage)
	checks = append(checks, checkConfigDir())

	// 5. Google Auth (actual API connectivity + token expiry)
	checks = append(checks, checkGoogleAuth(rctx))

	// 6. GitHub Auth (actual API connectivity)
	checks = append(checks, checkProviderAPI("github", "default"))

	// 7. Slack Auth (actual API connectivity)
	checks = append(checks, checkProviderAPI("slack", "default"))

	// 8. Notion Auth (actual API connectivity)
	checks = append(checks, checkProviderAPI("notion", "default"))

	// 9. Skills (with disk usage)
	checks = append(checks, checkSkills()...)

	// Summary counts.
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

	// Check writable by attempting to create and remove a temp file.
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

	// Calculate disk usage.
	size := dirSize(dir)
	sizeStr := formatBytes(size)

	// Shorten dir for display: replace $HOME with ~
	displayDir := dir
	if home, err := os.UserHomeDir(); err == nil {
		displayDir = strings.Replace(dir, home, "~", 1)
	}

	return checkResult{
		Name:    "config",
		Status:  "ok",
		Message: fmt.Sprintf("%s (%s)", displayDir, sizeStr),
		Detail:  fmt.Sprintf("path=%s size=%s", dir, sizeStr),
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
			Message: fmt.Sprintf("not authenticated (run 'gwx auth login')"),
			Detail:  fmt.Sprintf("account=%s token=missing", account),
		}
	}

	// Check token expiry.
	expiryInfo := ""
	token, err := rctx.Auth.LoadToken(account)
	if err == nil && !token.Expiry.IsZero() {
		remaining := time.Until(token.Expiry)
		if remaining < 0 {
			expiryInfo = "token expired, will auto-refresh"
		} else if remaining < 1*time.Hour {
			expiryInfo = fmt.Sprintf("expires in %s", formatDuration(remaining))
		} else {
			expiryInfo = fmt.Sprintf("expires %s", token.Expiry.Format("2006-01-02 15:04"))
		}
	}

	// Try actual API connectivity: call Gmail labels (lightweight read-only).
	apiOK := false
	apiMsg := ""
	if err := EnsureAuth(rctx, []string{"gmail"}); err == nil && rctx.APIClient != nil {
		ctx, cancel := context.WithTimeout(rctx.Context, 5*time.Second)
		defer cancel()

		opts, err := rctx.APIClient.ClientOptions(ctx, "gmail")
		if err == nil {
			_, _ = opts, ctx // We have the client options; try a lightweight call.
			// Instead of importing gmail/v1 here, we use the raw HTTP approach.
			apiOK, apiMsg = probeGoogleAPI(ctx, rctx)
		}
	}

	if apiOK {
		msg := "authenticated, API responding"
		if expiryInfo != "" {
			msg += " (" + expiryInfo + ")"
		}
		return checkResult{
			Name:    "google",
			Status:  "ok",
			Message: msg,
			Detail:  fmt.Sprintf("account=%s api=ok %s", account, expiryInfo),
		}
	}

	// Token exists but API unreachable or failing.
	if apiMsg != "" {
		msg := fmt.Sprintf("token found but API error: %s", apiMsg)
		if expiryInfo != "" {
			msg += " (" + expiryInfo + ")"
		}
		return checkResult{
			Name:    "google",
			Status:  "warning",
			Message: msg,
			Detail:  fmt.Sprintf("account=%s api=error %s", account, expiryInfo),
		}
	}

	// Fallback: token exists, no API test performed.
	msg := "authenticated"
	if expiryInfo != "" {
		msg += " (" + expiryInfo + ")"
	}
	return checkResult{
		Name:    "google",
		Status:  "ok",
		Message: msg,
		Detail:  fmt.Sprintf("account=%s %s", account, expiryInfo),
	}
}

// probeGoogleAPI makes a lightweight Gmail labels call to verify API connectivity.
func probeGoogleAPI(ctx context.Context, rctx *RunContext) (ok bool, errMsg string) {
	httpClient := rctx.APIClient.HTTPClient("gmail")

	req, err := http.NewRequestWithContext(ctx, "GET", "https://gmail.googleapis.com/gmail/v1/users/me/labels?fields=labels(id)", nil)
	if err != nil {
		return false, err.Error()
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()
	io.ReadAll(io.LimitReader(resp.Body, 1024)) //nolint:errcheck

	if resp.StatusCode == 200 {
		return true, ""
	}
	return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

// checkProviderAPI checks a provider token and tests actual API connectivity.
func checkProviderAPI(provider, account string) checkResult {
	name := provider

	if !auth.HasProviderToken(provider, account) {
		loginCmd := fmt.Sprintf("gwx %s login --token <TOKEN>", provider)
		return checkResult{
			Name:    name,
			Status:  "warning",
			Message: fmt.Sprintf("not configured (%s)", loginCmd),
			Detail:  "token=missing",
		}
	}

	token, err := auth.LoadProviderToken(provider, account)
	if err != nil {
		return checkResult{
			Name:    name,
			Status:  "error",
			Message: fmt.Sprintf("token load error: %s", err),
			Detail:  "token=error",
		}
	}

	// Test actual API connectivity with 3s timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	ok, apiMsg := probeProviderAPI(ctx, provider, token)
	if ok {
		return checkResult{
			Name:    name,
			Status:  "ok",
			Message: "authenticated, API responding",
			Detail:  fmt.Sprintf("account=%s api=ok", account),
		}
	}

	if apiMsg != "" {
		return checkResult{
			Name:    name,
			Status:  "warning",
			Message: fmt.Sprintf("token found but API error: %s", apiMsg),
			Detail:  fmt.Sprintf("account=%s api=error error=%s", account, apiMsg),
		}
	}

	return checkResult{
		Name:    name,
		Status:  "ok",
		Message: fmt.Sprintf("token found for %s account %q", provider, account),
		Detail:  fmt.Sprintf("account=%s api=untested", account),
	}
}

// probeProviderAPI tests actual API connectivity for a given provider.
func probeProviderAPI(ctx context.Context, provider, token string) (ok bool, errMsg string) {
	switch provider {
	case "github":
		return probeGitHub(ctx, token)
	case "slack":
		return probeSlack(ctx, token)
	case "notion":
		return probeNotion(ctx, token)
	default:
		return false, ""
	}
}

func probeGitHub(ctx context.Context, token string) (bool, string) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()
	io.ReadAll(io.LimitReader(resp.Body, 1024)) //nolint:errcheck

	if resp.StatusCode == 200 {
		return true, ""
	}
	if resp.StatusCode == 401 {
		return false, "invalid or expired token"
	}
	return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

func probeSlack(ctx context.Context, token string) (bool, string) {
	req, err := http.NewRequestWithContext(ctx, "POST", "https://slack.com/api/auth.test", nil)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return false, err.Error()
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, "invalid response"
	}
	if result.OK {
		return true, ""
	}
	if result.Error != "" {
		return false, result.Error
	}
	return false, "auth.test returned ok=false"
}

func probeNotion(ctx context.Context, token string) (bool, string) {
	body := strings.NewReader(`{"query":"","page_size":1}`)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.notion.com/v1/search", body)
	if err != nil {
		return false, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()
	io.ReadAll(io.LimitReader(resp.Body, 1024)) //nolint:errcheck

	if resp.StatusCode == 200 {
		return true, ""
	}
	if resp.StatusCode == 401 {
		return false, "invalid or expired token"
	}
	return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
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
		// Calculate skills dir size if it exists.
		sizeStr := ""
		configDir, err := config.Dir()
		if err == nil {
			skillsDir := filepath.Join(configDir, "skills")
			if s := dirSize(skillsDir); s > 0 {
				sizeStr = fmt.Sprintf(" (%s on disk)", formatBytes(s))
			}
		}
		return []checkResult{{
			Name:    "skills",
			Status:  "ok",
			Message: fmt.Sprintf("no skills loaded%s (create with 'gwx skill create <name>')", sizeStr),
			Detail:  "count=0",
		}}
	}

	// Sort for deterministic output.
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })

	names := make([]string, 0, len(skills))
	for _, s := range skills {
		names = append(names, s.Name)
	}

	// Calculate skills directories size.
	var totalSize int64
	configDir, err := config.Dir()
	if err == nil {
		totalSize += dirSize(filepath.Join(configDir, "skills"))
	}
	totalSize += dirSize("skills")
	sizeStr := ""
	if totalSize > 0 {
		sizeStr = fmt.Sprintf(" (%s on disk)", formatBytes(totalSize))
	}

	return []checkResult{{
		Name:    "skills",
		Status:  "ok",
		Message: fmt.Sprintf("%d skill(s) loaded%s", len(skills), sizeStr),
		Detail:  fmt.Sprintf("count=%d names=%s", len(skills), joinNames(names)),
	}}
}

func joinNames(names []string) string {
	if len(names) <= 5 {
		result := ""
		for i, n := range names {
			if i > 0 {
				result += ", "
			}
			result += n
		}
		return result
	}
	result := ""
	for i := 0; i < 5; i++ {
		if i > 0 {
			result += ", "
		}
		result += names[i]
	}
	return fmt.Sprintf("%s (and %d more)", result, len(names)-5)
}

// dirSize recursively calculates the total size of a directory in bytes.
func dirSize(path string) int64 {
	var total int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error { //nolint:errcheck
		if err != nil {
			return nil // skip errors
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// formatBytes formats a byte count into a human-readable string.
func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// formatDuration formats a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}
