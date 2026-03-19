package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/auth"
	"github.com/redredchen01/gwx/internal/config"
	"github.com/redredchen01/gwx/internal/exitcode"
	gwxlog "github.com/redredchen01/gwx/internal/log"
	"github.com/redredchen01/gwx/internal/output"
)

// CLI is the root command struct for gwx.
type CLI struct {
	// Global flags
	Format  string `help:"Output format: json, plain, table" short:"f" default:"json" enum:"json,plain,table"`
	Account string `help:"Account email to use" short:"a" default:"default"`
	Fields  string `help:"Comma-separated fields to include in output (e.g. id,name,subject)" name:"fields"`
	DryRun  bool   `help:"Validate without executing" name:"dry-run"`
	NoInput bool   `help:"Disable interactive prompts" name:"no-input"`
	NoCache bool   `help:"Disable caching" name:"no-cache" default:"false"`

	// Shortcuts (desire paths)
	Send   GmailSendCmd    `cmd:"" help:"Send an email (shortcut for gmail send)" hidden:""`
	Ls     DriveListCmd    `cmd:"" help:"List Drive files (shortcut for drive list)" hidden:""`
	Search GmailSearchCmd  `cmd:"" help:"Search Gmail (shortcut for gmail search)" hidden:""`
	Find    UnifiedSearchCmd `cmd:"" help:"Search across Gmail + Drive + Contacts"`
	Context ContextCmd       `cmd:"" help:"Gather all context for a topic across services"`

	// Service commands
	Auth     AuthCmd     `cmd:"" help:"Authentication management"`
	Onboard  OnboardCmd  `cmd:"" help:"Interactive setup wizard"`
	Gmail    GmailCmd    `cmd:"" help:"Gmail operations"`
	Calendar CalendarCmd `cmd:"" help:"Calendar operations"`
	Drive    DriveCmd    `cmd:"" help:"Google Drive operations"`
	Docs     DocsCmd     `cmd:"" help:"Google Docs operations"`
	Sheets   SheetsCmd   `cmd:"" help:"Google Sheets operations"`
	Tasks    TasksCmd    `cmd:"" help:"Google Tasks operations"`
	Contacts      ContactsCmd      `cmd:"" help:"Contacts operations"`
	Chat          ChatCmd          `cmd:"" help:"Google Chat operations"`
	Analytics     AnalyticsCmd     `cmd:"" help:"Google Analytics 4 operations"`
	SearchConsole SearchConsoleCmd `cmd:"searchconsole" help:"Google Search Console operations"`
	Config        ConfigCmd        `cmd:"" help:"Configuration management"`
	// Workflow commands
	Standup     StandupCmd     `cmd:"" help:"Daily standup report (aggregate Git + Gmail + Calendar + Tasks)"`
	MeetingPrep MeetingPrepCmd `cmd:"meeting-prep" help:"Prepare context for an upcoming meeting"`
	Workflow    WorkflowCmd    `cmd:"" help:"Workflow commands (test-matrix, sprint-board, etc.)"`

	Pipe      PipeCmd      `cmd:"" help:"Chain gwx commands via JSON pipeline (e.g. 'gmail search X | sheets append ID A:C')"`
	Agent     AgentCmd     `cmd:"" help:"Agent automation helpers"`
	Schema    SchemaCmd    `cmd:"" help:"Print full command schema (for agent introspection)"`
	MCPServer MCPServerCmd `cmd:"mcp-server" help:"Start MCP server (stdio) for Claude integration"`
	Version   VersionCmd   `cmd:"" help:"Print version"`
}

// RunContext holds shared state for command execution.
type RunContext struct {
	Context   context.Context
	Printer   *output.Printer
	Auth      *auth.Manager
	APIClient *api.Client
	Account   string
	DryRun    bool
	NoCache   bool
	Allowlist *config.Allowlist
}

// Execute is the main entry point.
func Execute() int {
	var cli CLI

	slog.SetDefault(gwxlog.SetupCLILogger())

	parser, err := kong.New(&cli,
		kong.Name("gwx"),
		kong.Description("Google Workspace CLI for humans and agents"),
		kong.UsageOnError(),
	)
	if err != nil {
		slog.Error("command failed", "error", err)
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
	if cli.Fields != "" {
		for _, f := range strings.Split(cli.Fields, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				printer.Fields = append(printer.Fields, f)
			}
		}
	}
	authMgr := auth.NewManager()
	allowlist := config.LoadAllowlist()

	rctx := &RunContext{
		Context:   context.Background(),
		Printer:   printer,
		Auth:      authMgr,
		Account:   cli.Account,
		DryRun:    cli.DryRun,
		NoCache:   cli.NoCache,
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

// Preflight checks allowlist + auth + dry-run in one call.
// If dry-run is active, it prints a standard response and returns a sentinel error
// that the caller should return directly (not an actual error — the response is already printed).
//
// Usage in Run():
//
//	if done, err := Preflight(rctx, "gmail.list", []string{"gmail"}); done {
//	    return err
//	}
func Preflight(rctx *RunContext, command string, services []string) (done bool, err error) {
	if err := CheckAllowlist(rctx, command); err != nil {
		return true, rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, services); err != nil {
		return true, rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": true,
			"command": command,
		})
		return true, nil
	}
	return false, nil
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
		apiClient := api.NewClient(ts)
		apiClient.NoCache = rctx.NoCache
		rctx.APIClient = apiClient
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

	apiClient := api.NewClient(ts)
	apiClient.NoCache = rctx.NoCache
	rctx.APIClient = apiClient
	return nil
}
