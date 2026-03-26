package cmd

import (
	"github.com/redredchen01/gwx/internal/exitcode"
	"github.com/redredchen01/gwx/internal/workflow"
)

// ContextBoostCmd implements gwx workflow context-boost.
type ContextBoostCmd struct {
	Topic string `arg:"" help:"Topic to gather context for"`
	Days  int    `help:"Days of history" default:"14" short:"d"`
	Limit int    `help:"Max results per service" default:"10" short:"n"`
}

func (c *ContextBoostCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "workflow.context-boost", []string{"gmail", "drive", "calendar", "contacts"}); done {
		return err
	}

	result, err := workflow.RunContextBoost(rctx.Context, rctx.APIClient, workflow.ContextBoostOpts{
		Topic: c.Topic,
		Days:  c.Days,
		Limit: c.Limit,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}
	rctx.Printer.Success(result)
	return nil
}

// BugIntakeCmd implements gwx workflow bug-intake.
type BugIntakeCmd struct {
	BugID   string `help:"Bug ID or keyword to search" name:"bug-id"`
	After   string `help:"Date filter (e.g. 2026/03/15)" name:"after"`
	Execute bool   `help:"Execute actions" name:"execute"`
}

func (c *BugIntakeCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "workflow.bug-intake", []string{"gmail", "drive"}); done {
		return err
	}

	result, err := workflow.RunBugIntake(rctx.Context, rctx.APIClient, workflow.BugIntakeOpts{
		BugID: c.BugID,
		After: c.After,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}
	rctx.Printer.Success(result)
	return nil
}
