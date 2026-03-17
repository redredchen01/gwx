package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/user/gwx/internal/api"
	"github.com/user/gwx/internal/auth"
	"github.com/user/gwx/internal/config"
	"github.com/user/gwx/internal/exitcode"
	"github.com/user/gwx/internal/output"
)

// CLI is the root command struct for gwx.
type CLI struct {
	// Global flags
	Format  string `help:"Output format: json, plain, table" short:"f" default:"json" enum:"json,plain,table"`
	Account string `help:"Account email to use" short:"a" default:"default"`
	DryRun  bool   `help:"Validate without executing" name:"dry-run"`
	NoInput bool   `help:"Disable interactive prompts" name:"no-input"`

	// Service commands
	Auth     AuthCmd     `cmd:"" help:"Authentication management"`
	Onboard  OnboardCmd  `cmd:"" help:"Interactive setup wizard"`
	Gmail    GmailCmd    `cmd:"" help:"Gmail operations"`
	Calendar CalendarCmd `cmd:"" help:"Calendar operations"`
	Drive    DriveCmd    `cmd:"" help:"Google Drive operations"`
	Docs     DocsCmd     `cmd:"" help:"Google Docs operations"`
	Sheets   SheetsCmd   `cmd:"" help:"Google Sheets operations"`
	Tasks    TasksCmd    `cmd:"" help:"Google Tasks operations"`
	Contacts ContactsCmd `cmd:"" help:"Contacts operations"`
	Chat     ChatCmd     `cmd:"" help:"Google Chat operations"`
	Agent    AgentCmd    `cmd:"" help:"Agent automation helpers"`
	Schema   SchemaCmd   `cmd:"" help:"Print full command schema (for agent introspection)"`
	Version  VersionCmd  `cmd:"" help:"Print version"`
}

// RunContext holds shared state for command execution.
type RunContext struct {
	Context   context.Context
	Printer   *output.Printer
	Auth      *auth.Manager
	APIClient *api.Client
	Account   string
	DryRun    bool
	Allowlist *config.Allowlist
}

// Execute is the main entry point.
func Execute() int {
	var cli CLI

	parser, err := kong.New(&cli,
		kong.Name("gwx"),
		kong.Description("Google Workspace CLI for humans and agents"),
		kong.UsageOnError(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return exitcode.UsageError
	}

	kctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		parser.FatalIfErrorf(err)
		return exitcode.UsageError
	}

	// Auto-JSON for non-TTY (agent-friendly)
	if !output.IsTTY() && os.Getenv("GWX_AUTO_JSON") != "" {
		cli.Format = "json"
	}

	printer := output.NewPrinter(output.ParseFormat(cli.Format))
	authMgr := auth.NewManager()
	allowlist := config.LoadAllowlist()

	rctx := &RunContext{
		Context:   context.Background(),
		Printer:   printer,
		Auth:      authMgr,
		Account:   cli.Account,
		DryRun:    cli.DryRun,
		Allowlist: allowlist,
	}

	if err := kctx.Run(rctx); err != nil {
		// If it's already an ExitError (from Printer.ErrExit), extract the code
		var ee *output.ExitError
		if errors.As(err, &ee) {
			return ee.Code
		}
		return printer.Err(exitcode.GeneralError, err.Error())
	}
	return exitcode.OK
}

// CheckAllowlist verifies a command is permitted by the allowlist.
func CheckAllowlist(rctx *RunContext, command string) error {
	if rctx.Allowlist != nil && !rctx.Allowlist.IsAllowed(command) {
		return fmt.Errorf("command %q is not in the allowlist (GWX_ENABLE_COMMANDS)", command)
	}
	return nil
}

// EnsureAuth loads credentials and creates an API client.
func EnsureAuth(rctx *RunContext, services []string) error {
	scopes := auth.AllScopes(services, false)

	// Check for direct access token
	if token := os.Getenv("GWX_ACCESS_TOKEN"); token != "" {
		ts := auth.TokenFromDirect(token)
		rctx.APIClient = api.NewClient(ts)
		return nil
	}

	// Try loading from keyring
	ts, err := rctx.Auth.TokenSource(rctx.Context, rctx.Account)
	if err != nil {
		// Try loading config and report auth needed
		if loadErr := rctx.Auth.LoadConfigFromKeyring(scopes); loadErr != nil {
			return fmt.Errorf("not authenticated. Run 'gwx onboard' to set up, or 'gwx auth login' to sign in")
		}
		ts, err = rctx.Auth.TokenSource(rctx.Context, rctx.Account)
		if err != nil {
			return fmt.Errorf("not authenticated. Run 'gwx auth login' to sign in")
		}
	}

	rctx.APIClient = api.NewClient(ts)
	return nil
}
