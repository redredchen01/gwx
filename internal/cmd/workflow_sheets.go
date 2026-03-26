package cmd

import (
	"github.com/redredchen01/gwx/internal/exitcode"
	"github.com/redredchen01/gwx/internal/workflow"
)

// TestMatrixCmd implements gwx workflow test-matrix.
type TestMatrixCmd struct {
	Action  string `arg:"" help:"Action: init, sync, stats" enum:"init,sync,stats"`
	Feature string `help:"Feature name (for init)" name:"feature"`
	SheetID string `help:"Sheet ID (for sync/stats)" name:"sheet-id"`
	File    string `help:"Test results file (for sync)" name:"file"`
}

func (c *TestMatrixCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "workflow.test-matrix", []string{"sheets"}); done {
		return err
	}

	result, err := workflow.RunTestMatrix(rctx.Context, rctx.APIClient, workflow.TestMatrixOpts{
		Action:  c.Action,
		Feature: c.Feature,
		SheetID: c.SheetID,
		File:    c.File,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}
	rctx.Printer.Success(result)
	return nil
}

// SpecHealthCmd implements gwx workflow spec-health.
type SpecHealthCmd struct {
	Action     string `arg:"" help:"Action: init, record, stats" enum:"init,record,stats"`
	Feature    string `help:"Feature name (for init)" name:"feature"`
	SheetID    string `help:"Sheet ID (for record/stats)" name:"sheet-id"`
	SpecFolder string `help:"Spec folder path (for record)" name:"spec-folder"`
}

func (c *SpecHealthCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "workflow.spec-health", []string{"sheets"}); done {
		return err
	}

	result, err := workflow.RunSpecHealth(rctx.Context, rctx.APIClient, workflow.SpecHealthOpts{
		Action:     c.Action,
		Feature:    c.Feature,
		SheetID:    c.SheetID,
		SpecFolder: c.SpecFolder,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}
	rctx.Printer.Success(result)
	return nil
}

// SprintBoardCmd implements gwx workflow sprint-board.
type SprintBoardCmd struct {
	Action   string `arg:"" help:"Action: init, ticket, stats, archive" enum:"init,ticket,stats,archive"`
	Feature  string `help:"Feature name (for init)" name:"feature"`
	SheetID  string `help:"Sheet ID" name:"sheet-id"`
	Title    string `help:"Ticket title (for ticket)" name:"title"`
	Assignee string `help:"Assignee (for ticket)" name:"assignee"`
	Priority string `help:"Priority P0-P3 (for ticket)" name:"priority" default:"P2"`
	Sprint   string `help:"Sprint name (for archive)" name:"sprint"`
}

func (c *SprintBoardCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "workflow.sprint-board", []string{"sheets"}); done {
		return err
	}

	result, err := workflow.RunSprintBoard(rctx.Context, rctx.APIClient, workflow.SprintBoardOpts{
		Action:   c.Action,
		Feature:  c.Feature,
		SheetID:  c.SheetID,
		Title:    c.Title,
		Assignee: c.Assignee,
		Priority: c.Priority,
		Sprint:   c.Sprint,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}
	rctx.Printer.Success(result)
	return nil
}
