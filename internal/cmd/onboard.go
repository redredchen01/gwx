package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/redredchen01/gwx/internal/auth"
)

// readPastedJSON reads multi-line JSON from stdin, starting with firstLine.
// Reads until braces are balanced or valid JSON is formed.
func readPastedJSON(firstLine string, reader *bufio.Reader) ([]byte, error) {
	var sb strings.Builder
	sb.WriteString(firstLine)
	sb.WriteString("\n")

	// Check if first line is already complete JSON
	if json.Valid([]byte(firstLine)) {
		return []byte(firstLine), nil
	}

	// Read more lines until we have valid JSON
	for i := 0; i < 200; i++ { // safety cap
		line, err := reader.ReadString('\n')
		if err != nil {
			// EOF — try what we have
			sb.WriteString(line)
			break
		}
		sb.WriteString(line)

		if json.Valid([]byte(sb.String())) {
			return []byte(sb.String()), nil
		}
	}

	data := []byte(sb.String())
	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid JSON (pasted content is not valid JSON)")
	}
	return data, nil
}

// OnboardCmd runs the interactive setup wizard.
type OnboardCmd struct{}

func (c *OnboardCmd) Run(rctx *RunContext) error {
	// Non-interactive mode: read from environment variables
	if rctx.DryRun || os.Getenv("GWX_OAUTH_JSON") != "" {
		return c.runNonInteractive(rctx)
	}

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
	fmt.Fprintln(os.Stderr, "  Enter file path, or paste JSON directly (starts with '{'):")
	fmt.Fprint(os.Stderr, "  > ")

	firstLine, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	firstLine = strings.TrimSpace(firstLine)

	var credJSON []byte
	if strings.HasPrefix(firstLine, "{") {
		// Paste JSON mode — read until braces are balanced
		credJSON, err = readPastedJSON(firstLine, reader)
		if err != nil {
			return fmt.Errorf("read pasted JSON: %w", err)
		}
		fmt.Fprintln(os.Stderr, "  ✓ JSON received (paste mode)")
	} else {
		// File path mode
		credPath := firstLine
		if strings.HasPrefix(credPath, "~/") {
			home, _ := os.UserHomeDir()
			credPath = home + credPath[1:]
		}
		credJSON, err = os.ReadFile(credPath)
		if err != nil {
			return fmt.Errorf("read credentials file: %w", err)
		}
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
	if err := rctx.Auth.LoadConfigFromJSON(credJSON, scopes); err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}
	fmt.Fprintln(os.Stderr, "  ✓ Credentials saved to OS Keyring")

	// Step 3: Login
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Step 3/3: Sign In")
	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(os.Stderr, "  (b)rowser  — opens browser on this machine (default)")
	fmt.Fprintln(os.Stderr, "  (m)anual   — localhost redirect, copy URL manually")
	fmt.Fprintln(os.Stderr, "  (r)emote   — for VPS/SSH: paste auth code from your local browser")
	fmt.Fprint(os.Stderr, "  Auth method [b]: ")

	method, _ := reader.ReadString('\n')
	method = strings.TrimSpace(strings.ToLower(method))

	var loginErr error
	switch method {
	case "r", "remote":
		t, err := rctx.Auth.LoginRemote(rctx.Context)
		if err != nil {
			loginErr = fmt.Errorf("remote login: %w", err)
		} else if err := rctx.Auth.SaveToken(rctx.Account, t); err != nil {
			loginErr = fmt.Errorf("save token: %w", err)
		}
	case "m", "manual":
		t, err := rctx.Auth.LoginManual(rctx.Context)
		if err != nil {
			loginErr = fmt.Errorf("manual login: %w", err)
		} else if err := rctx.Auth.SaveToken(rctx.Account, t); err != nil {
			loginErr = fmt.Errorf("save token: %w", err)
		}
	default:
		t, err := rctx.Auth.LoginBrowser(rctx.Context)
		if err != nil {
			loginErr = fmt.Errorf("browser login: %w", err)
		} else if err := rctx.Auth.SaveToken(rctx.Account, t); err != nil {
			loginErr = fmt.Errorf("save token: %w", err)
		}
	}
	if loginErr != nil {
		return loginErr
	}

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

// runNonInteractive handles onboard via environment variables (CI/VPS/Agent).
//
// Environment variables:
//   - GWX_OAUTH_JSON: OAuth credentials JSON string (required)
//   - GWX_OAUTH_FILE: Path to OAuth credentials JSON file (alternative to GWX_OAUTH_JSON)
//   - GWX_SERVICES: Comma-separated services (default: all)
//   - GWX_AUTH_METHOD: "browser", "manual", or "remote" (default: "remote" in --no-input)
func (c *OnboardCmd) runNonInteractive(rctx *RunContext) error {
	// Step 1: Load credentials
	var credJSON []byte
	if jsonStr := os.Getenv("GWX_OAUTH_JSON"); jsonStr != "" {
		credJSON = []byte(jsonStr)
	} else if filePath := os.Getenv("GWX_OAUTH_FILE"); filePath != "" {
		var err error
		credJSON, err = os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read GWX_OAUTH_FILE %q: %w", filePath, err)
		}
	} else {
		return fmt.Errorf("non-interactive onboard requires GWX_OAUTH_JSON or GWX_OAUTH_FILE environment variable")
	}

	// Step 2: Services
	services := []string{"gmail", "calendar", "drive", "docs", "sheets", "tasks", "people", "chat", "analytics", "searchconsole"}
	if svcEnv := os.Getenv("GWX_SERVICES"); svcEnv != "" {
		services = nil
		for _, s := range strings.Split(svcEnv, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				services = append(services, s)
			}
		}
	}

	scopes := auth.AllScopes(services, false)

	if err := rctx.Auth.LoadConfigFromJSON(credJSON, scopes); err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":  true,
			"command":  "onboard",
			"services": services,
			"source":   "environment",
		})
		return nil
	}

	// Step 3: Login
	method := os.Getenv("GWX_AUTH_METHOD")
	if method == "" {
		method = "remote" // default for non-interactive (VPS-friendly)
	}

	switch method {
	case "remote":
		t, err := rctx.Auth.LoginRemote(rctx.Context)
		if err != nil {
			return fmt.Errorf("remote login: %w", err)
		}
		if err := rctx.Auth.SaveToken(rctx.Account, t); err != nil {
			return fmt.Errorf("save token: %w", err)
		}
	case "manual":
		t, err := rctx.Auth.LoginManual(rctx.Context)
		if err != nil {
			return fmt.Errorf("manual login: %w", err)
		}
		if err := rctx.Auth.SaveToken(rctx.Account, t); err != nil {
			return fmt.Errorf("save token: %w", err)
		}
	default:
		t, err := rctx.Auth.LoginBrowser(rctx.Context)
		if err != nil {
			return fmt.Errorf("browser login: %w", err)
		}
		if err := rctx.Auth.SaveToken(rctx.Account, t); err != nil {
			return fmt.Errorf("save token: %w", err)
		}
	}

	rctx.Printer.Success(map[string]interface{}{
		"status":   "onboarded",
		"account":  rctx.Account,
		"services": services,
		"mode":     "non-interactive",
	})
	return nil
}
