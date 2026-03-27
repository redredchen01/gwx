package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/redredchen01/gwx/internal/auth"
)

var defaultServices = []string{
	"gmail", "calendar", "drive", "docs", "sheets",
	"tasks", "people", "chat", "analytics", "searchconsole", "slides",
}

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

// OnboardCmd runs the setup wizard with support for --json flag and stdin pipe.
//
// Credential sources (highest to lowest priority):
//  1. --json flag
//  2. stdin pipe (non-TTY)
//  3. GWX_OAUTH_JSON environment variable
//  4. GWX_OAUTH_FILE environment variable
//  5. Interactive prompt (TTY only)
type OnboardCmd struct {
	JSON       string `help:"OAuth credentials JSON string (highest priority)" name:"json"`
	Services   string `help:"Comma-separated services to authorize (default: all)" name:"services"`
	AuthMethod string `help:"Auth method: browser, manual, or remote (default: remote for non-interactive)" name:"auth-method"`
}

// isStdinPipe checks if stdin is a pipe (non-TTY).
func isStdinPipe() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// resolveCredentials resolves OAuth credentials JSON from multiple sources.
// Returns (credJSON, source, error). Nil credJSON means no non-interactive source found.
func (c *OnboardCmd) resolveCredentials(isPipe bool) ([]byte, string, error) {
	// 1. --json flag (highest priority)
	if c.JSON != "" {
		data := []byte(c.JSON)
		if !json.Valid(data) {
			return nil, "", fmt.Errorf("--json: invalid JSON")
		}
		return data, "flag", nil
	}

	// 2. stdin pipe (non-TTY)
	if isPipe {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, "", fmt.Errorf("read stdin: %w", err)
		}
		trimmed := strings.TrimSpace(string(data))
		if trimmed == "" {
			return nil, "", fmt.Errorf("stdin pipe is empty")
		}
		b := []byte(trimmed)
		if !json.Valid(b) {
			return nil, "", fmt.Errorf("stdin: invalid JSON")
		}
		return b, "pipe", nil
	}

	// 3. GWX_OAUTH_JSON environment variable
	if jsonStr := os.Getenv("GWX_OAUTH_JSON"); jsonStr != "" {
		return []byte(jsonStr), "env", nil
	}

	// 4. GWX_OAUTH_FILE environment variable
	if filePath := os.Getenv("GWX_OAUTH_FILE"); filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, "", fmt.Errorf("read GWX_OAUTH_FILE %q: %w", filePath, err)
		}
		return data, "env-file", nil
	}

	// 5. No non-interactive source — caller decides (interactive or error)
	return nil, "", nil
}

// resolveServices resolves the service list from --services flag, GWX_SERVICES env, or default.
func (c *OnboardCmd) resolveServices() []string {
	input := c.Services
	if input == "" {
		input = os.Getenv("GWX_SERVICES")
	}
	if input == "" {
		return defaultServices
	}
	var services []string
	for _, s := range strings.Split(input, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			services = append(services, s)
		}
	}
	return services
}

// resolveAuthMethod resolves the auth method from --auth-method flag, GWX_AUTH_METHOD env, or default.
// Returns empty string when no explicit method is set (interactive mode will ask user).
func (c *OnboardCmd) resolveAuthMethod() string {
	if c.AuthMethod != "" {
		return strings.ToLower(c.AuthMethod)
	}
	if env := os.Getenv("GWX_AUTH_METHOD"); env != "" {
		return strings.ToLower(env)
	}
	return ""
}

func (c *OnboardCmd) Run(rctx *RunContext) error {
	isPipe := isStdinPipe()

	// Resolve credentials from all sources (5-level priority)
	credJSON, source, err := c.resolveCredentials(isPipe)
	if err != nil {
		return err
	}

	// No non-interactive credentials found
	if credJSON == nil {
		if rctx.DryRun {
			return fmt.Errorf("dry-run requires credentials: use --json, stdin pipe, or GWX_OAUTH_JSON/GWX_OAUTH_FILE env")
		}
		return c.runInteractive(rctx)
	}

	// --- Non-interactive path ---

	services := c.resolveServices()
	scopes := auth.AllScopes(services, false)

	if err := rctx.Auth.LoadConfigFromJSON(credJSON, scopes); err != nil {
		return fmt.Errorf("load credentials: %w", err)
	}
	fmt.Fprintln(os.Stderr, "  ✓ Credentials saved to OS Keyring")

	// DryRun: output result and stop before login
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":  true,
			"command":  "onboard",
			"services": services,
			"source":   source,
		})
		return nil
	}

	// Resolve auth method
	method := c.resolveAuthMethod()
	if method == "" {
		method = "remote" // default for non-interactive (VPS-friendly)
	}

	// Validate auth method
	switch method {
	case "browser", "b", "manual", "m", "remote", "r":
		// valid
	default:
		return fmt.Errorf("invalid --auth-method %q: must be browser, manual, or remote", method)
	}

	// Conflict: pipe consumed stdin, but remote auth also needs stdin
	if isPipe && (method == "remote" || method == "r") {
		return fmt.Errorf("pipe mode conflicts with remote auth (stdin already consumed). Use --auth-method browser or --auth-method manual")
	}

	// Login
	return c.doLogin(rctx, method, services)
}

func (c *OnboardCmd) doLogin(rctx *RunContext, method string, services []string) error {
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
	default: // browser
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

// runInteractive is the original interactive setup wizard (TTY mode).
func (c *OnboardCmd) runInteractive(rctx *RunContext) error {
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
	allServices := "gmail, calendar, drive, docs, sheets, tasks, people, chat, analytics, searchconsole, slides"
	fmt.Fprintln(os.Stderr, "  Available: "+allServices)
	fmt.Fprintln(os.Stderr, "  Default:   ALL (recommended)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprint(os.Stderr, "  Services (comma-separated, or press Enter for default): ")

	svcInput, _ := reader.ReadString('\n')
	svcInput = strings.TrimSpace(svcInput)

	var services []string
	if svcInput == "" {
		services = defaultServices
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
	if method == "" {
		method = "browser"
	}

	return c.doLogin(rctx, method, services)
}
