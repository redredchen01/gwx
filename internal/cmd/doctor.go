package cmd

import (
	"fmt"
	"os"
	"runtime"
	"sort"

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
}

func (c *DoctorCmd) Run(rctx *RunContext) error {
	var checks []checkResult

	// 1. Version
	checks = append(checks, checkResult{
		Name:    "version",
		Status:  "ok",
		Message: version,
	})

	// 2. Go version
	checks = append(checks, checkResult{
		Name:    "go_version",
		Status:  "ok",
		Message: runtime.Version(),
	})

	// 3. OS / Arch
	checks = append(checks, checkResult{
		Name:    "os",
		Status:  "ok",
		Message: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	})

	// 4. Config directory
	checks = append(checks, checkConfigDir())

	// 5. Google Auth
	checks = append(checks, checkGoogleAuth(rctx))

	// 6. GitHub Auth
	checks = append(checks, checkProviderAuth("github", "default"))

	// 7. Slack Auth
	checks = append(checks, checkProviderAuth("slack", "default"))

	// 8. Notion Auth
	checks = append(checks, checkProviderAuth("notion", "default"))

	// 9. Skills
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
		"status":   overallStatus,
		"checks":   checks,
		"summary": map[string]int{
			"ok":      okCount,
			"warning": warnCount,
			"error":   errCount,
			"total":   len(checks),
		},
	})
	return nil
}

func checkConfigDir() checkResult {
	dir, err := config.Dir()
	if err != nil {
		return checkResult{
			Name:    "config_dir",
			Status:  "error",
			Message: fmt.Sprintf("cannot determine config dir: %s", err),
		}
	}

	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return checkResult{
			Name:    "config_dir",
			Status:  "warning",
			Message: fmt.Sprintf("%s does not exist (run 'gwx onboard' to create)", dir),
		}
	}
	if err != nil {
		return checkResult{
			Name:    "config_dir",
			Status:  "error",
			Message: fmt.Sprintf("cannot stat %s: %s", dir, err),
		}
	}
	if !info.IsDir() {
		return checkResult{
			Name:    "config_dir",
			Status:  "error",
			Message: fmt.Sprintf("%s exists but is not a directory", dir),
		}
	}

	// Check writable by attempting to create and remove a temp file.
	testFile := dir + "/.doctor-write-test"
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		return checkResult{
			Name:    "config_dir",
			Status:  "error",
			Message: fmt.Sprintf("%s is not writable: %s", dir, err),
		}
	}
	os.Remove(testFile)

	return checkResult{
		Name:    "config_dir",
		Status:  "ok",
		Message: dir,
	}
}

func checkGoogleAuth(rctx *RunContext) checkResult {
	account := rctx.Account
	if account == "" {
		account = "default"
	}

	if rctx.Auth.HasToken(account) {
		return checkResult{
			Name:    "google_auth",
			Status:  "ok",
			Message: fmt.Sprintf("token found for account %q", account),
		}
	}
	return checkResult{
		Name:    "google_auth",
		Status:  "warning",
		Message: fmt.Sprintf("no token for account %q (run 'gwx auth login')", account),
	}
}

func checkProviderAuth(provider, account string) checkResult {
	name := provider + "_auth"
	if auth.HasProviderToken(provider, account) {
		return checkResult{
			Name:    name,
			Status:  "ok",
			Message: fmt.Sprintf("token found for %s account %q", provider, account),
		}
	}
	return checkResult{
		Name:    name,
		Status:  "warning",
		Message: fmt.Sprintf("no %s token (run 'gwx %s login')", provider, provider),
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

	// Sort for deterministic output.
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })

	names := make([]string, 0, len(skills))
	for _, s := range skills {
		names = append(names, s.Name)
	}

	return []checkResult{{
		Name:    "skills",
		Status:  "ok",
		Message: fmt.Sprintf("%d skill(s) loaded: %s", len(skills), joinNames(names)),
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
