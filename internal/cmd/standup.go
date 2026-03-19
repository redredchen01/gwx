package cmd

import (
	"github.com/redredchen01/gwx/internal/exitcode"
	"github.com/redredchen01/gwx/internal/workflow"
)

// StandupCmd implements gwx standup.
type StandupCmd struct {
	Days    int    `help:"Days of git history" default:"1" short:"d"`
	Execute bool   `help:"Execute actions (e.g. push to chat/email)" name:"execute"`
	Push    string `help:"Push target (chat:spaces/XXX or email:addr@example.com)" name:"push"`
}

func (c *StandupCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "workflow.standup"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	services := []string{"gmail", "calendar", "tasks"}
	if c.Push != "" {
		services = append(services, "chat")
	}
	if err := EnsureAuth(rctx, services); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "standup"})
		return nil
	}

	result, err := workflow.RunStandup(rctx.Context, rctx.APIClient, workflow.StandupOpts{
		Days:    c.Days,
		Execute: c.Execute,
		NoInput: rctx.DryRun,
		Push:    c.Push,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(result)
	return nil
}
