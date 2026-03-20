package cmd

import (
	"github.com/redredchen01/gwx/internal/exitcode"
	"github.com/redredchen01/gwx/internal/workflow"
)

// MeetingPrepCmd implements gwx meeting-prep.
type MeetingPrepCmd struct {
	Meeting string `arg:"" help:"Meeting title or keyword to match"`
	Days    int    `help:"Days ahead to search" default:"1" short:"d"`
	Execute bool   `help:"Execute actions" name:"execute"`
}

func (c *MeetingPrepCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "meeting-prep", []string{"calendar", "contacts", "gmail", "drive"}); done {
		return err
	}

	result, err := workflow.RunMeetingPrep(rctx.Context, rctx.APIClient, workflow.MeetingPrepOpts{
		Meeting: c.Meeting,
		Days:    c.Days,
		Execute: c.Execute,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(result)
	return nil
}
