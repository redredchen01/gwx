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
	services := []string{"gmail", "calendar", "tasks"}
	if c.Push != "" {
		services = append(services, "chat")
	}
	if done, err := Preflight(rctx, "workflow.standup", services); done {
		return err
	}

	result, err := workflow.RunStandup(rctx.Context, rctx.APIClient, workflow.StandupOpts{
		Days:    c.Days,
		Execute: c.Execute,
		NoInput: rctx.NoInput,
		Push:    c.Push,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(result)
	return nil
}
