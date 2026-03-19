package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/redredchen01/gwx/internal/auth"
)

// OnboardCmd runs the interactive setup wizard.
type OnboardCmd struct{}

func (c *OnboardCmd) Run(rctx *RunContext) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "╔══════════════════════════════════════════╗")
	fmt.Fprintln(os.Stderr, "║       gwx - Google Workspace Setup       ║")
	fmt.Fprintln(os.Stderr, "╚══════════════════════════════════════════╝")
	fmt.Fprintln(os.Stderr, "")

	// Step 1: Check for Google Cloud Project
	fmt.Fprintln(os.Stderr, "Step 1/3: OAuth Credentials")
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(os.Stderr, "  You need an OAuth 2.0 Client ID (Desktop type).")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  1. Go to https://console.cloud.google.com/apis/credentials")
	fmt.Fprintln(os.Stderr, "  2. Create Credentials → OAuth Client ID → Desktop App")
	fmt.Fprintln(os.Stderr, "  3. Download the JSON file")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprint(os.Stderr, "  Path to credentials JSON: ")

	credPath, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	credPath = strings.TrimSpace(credPath)

	// Expand ~ if present
	if strings.HasPrefix(credPath, "~/") {
		home, _ := os.UserHomeDir()
		credPath = home + credPath[1:]
	}

	// Step 2: Select services
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Step 2/3: Select Services")
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━")
	allServices := "gmail, calendar, drive, docs, sheets, tasks, people, chat, analytics, searchconsole"
	fmt.Fprintln(os.Stderr, "  Available: "+allServices)
	fmt.Fprintln(os.Stderr, "  Default:   ALL (recommended)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprint(os.Stderr, "  Services (comma-separated, or press Enter for default): ")

	svcInput, _ := reader.ReadString('\n')
	svcInput = strings.TrimSpace(svcInput)

	var services []string
	if svcInput == "" {
		services = []string{"gmail", "calendar", "drive", "docs", "sheets", "tasks", "people", "chat", "analytics", "searchconsole"}
	} else {
		for _, s := range strings.Split(svcInput, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				services = append(services, s)
			}
		}
	}

	scopes := auth.AllScopes(services, false)

	// Load credentials
	if err := rctx.Auth.LoadConfigFromFile(credPath, scopes); err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}
	fmt.Fprintln(os.Stderr, "  ✓ Credentials saved to OS Keyring")

	// Step 3: Login
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Step 3/3: Sign In")
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━")
	fmt.Fprint(os.Stderr, "  Auth method - (b)rowser or (m)anual? [b]: ")

	method, _ := reader.ReadString('\n')
	method = strings.TrimSpace(strings.ToLower(method))

	var token interface{}
	if method == "m" || method == "manual" {
		t, err := rctx.Auth.LoginManual(rctx.Context)
		if err != nil {
			return fmt.Errorf("manual login: %w", err)
		}
		token = t
		if err := rctx.Auth.SaveToken(rctx.Account, t); err != nil {
			return fmt.Errorf("save token: %w", err)
		}
	} else {
		t, err := rctx.Auth.LoginBrowser(rctx.Context)
		if err != nil {
			return fmt.Errorf("browser login: %w", err)
		}
		token = t
		if err := rctx.Auth.SaveToken(rctx.Account, t); err != nil {
			return fmt.Errorf("save token: %w", err)
		}
	}
	_ = token

	fmt.Fprintln(os.Stderr, "  ✓ Token saved to OS Keyring (never written to disk)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(os.Stderr, "✓ Setup complete! Try:")
	fmt.Fprintln(os.Stderr, "  gwx gmail list --limit 5")
	fmt.Fprintln(os.Stderr, "  gwx auth status")
	fmt.Fprintln(os.Stderr, "")

	rctx.Printer.Success(map[string]interface{}{
		"status":   "onboarded",
		"account":  rctx.Account,
		"services": services,
	})
	return nil
}
