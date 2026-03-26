package cmd

import (
	"github.com/redredchen01/gwx/internal/exitcode"
	"github.com/redredchen01/gwx/internal/workflow"
)

// WeeklyDigestCmd implements gwx workflow weekly-digest.
type WeeklyDigestCmd struct {
	Weeks   int  `help:"Number of weeks to cover" default:"1" short:"w"`
	Execute bool `help:"Execute actions" name:"execute"`
}

func (c *WeeklyDigestCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "workflow.weekly-digest", []string{"gmail", "calendar", "tasks"}); done {
		return err
	}

	result, err := workflow.RunWeeklyDigest(rctx.Context, rctx.APIClient, workflow.WeeklyDigestOpts{
		Weeks: c.Weeks,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}
	rctx.Printer.Success(result)
	return nil
}
