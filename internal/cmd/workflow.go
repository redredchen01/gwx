package cmd

import (
	"strings"

	"github.com/redredchen01/gwx/internal/exitcode"
	"github.com/redredchen01/gwx/internal/workflow"
)

// WorkflowCmd is the command group for gwx workflow subcommands.
type WorkflowCmd struct {
	WeeklyDigest     WeeklyDigestCmd     `cmd:"weekly-digest" help:"Weekly activity digest"`
	ContextBoost     ContextBoostCmd     `cmd:"context-boost" help:"Deep context gathering for a topic"`
	BugIntake        BugIntakeCmd        `cmd:"bug-intake" help:"Gather context for a bug report"`
	TestMatrix       TestMatrixCmd       `cmd:"test-matrix" help:"Manage test results in Sheets"`
	SpecHealth       SpecHealthCmd       `cmd:"spec-health" help:"Track spec status in Sheets"`
	SprintBoard      SprintBoardCmd      `cmd:"sprint-board" help:"Sprint board in Sheets"`
	ReviewNotify     ReviewNotifyCmd     `cmd:"review-notify" help:"Notify reviewers about a spec"`
	EmailFromDoc     EmailFromDocCmd     `cmd:"email-from-doc" help:"Send email from a Google Doc"`
	SheetToEmail     SheetToEmailCmd     `cmd:"sheet-to-email" help:"Send personalized emails from Sheet data"`
	ParallelSchedule ParallelScheduleCmd `cmd:"parallel-schedule" help:"Schedule parallel 1-on-1 reviews"`
	Digest           WeeklyDigestCmd     `cmd:"digest" help:"Alias for weekly-digest" hidden:""`
}

// --- FA-B: Data Aggregation Workflows ---

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

// --- FA-C: Sheets State Workflows ---

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

// --- FA-D: Action Workflows ---

// ReviewNotifyCmd implements gwx workflow review-notify.
type ReviewNotifyCmd struct {
	SpecFolder string `help:"Spec folder path" name:"spec-folder" required:""`
	Reviewers  string `help:"Comma-separated reviewer emails" name:"reviewers" required:""`
	Channel    string `help:"Notification channel (email or chat:spaces/XXX)" name:"channel"`
	Execute    bool   `help:"Execute actions" name:"execute"`
}

func (c *ReviewNotifyCmd) Run(rctx *RunContext) error {
	if c.Execute && c.Channel == "" {
		return rctx.Printer.ErrExit(exitcode.UsageError, "--channel required when --execute is set")
	}
	services := []string{"gmail"}
	if strings.HasPrefix(c.Channel, "chat:") {
		services = append(services, "chat")
	}
	if done, err := Preflight(rctx, "workflow.review-notify", services); done {
		return err
	}

	result, err := workflow.RunReviewNotify(rctx.Context, rctx.APIClient, workflow.ReviewNotifyOpts{
		SpecFolder: c.SpecFolder,
		Reviewers:  strings.Split(c.Reviewers, ","),
		Channel:    c.Channel,
		Execute:    c.Execute,
		NoInput:    rctx.NoInput,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}
	rctx.Printer.Success(result)
	return nil
}

// EmailFromDocCmd implements gwx workflow email-from-doc.
type EmailFromDocCmd struct {
	DocID      string `help:"Google Doc ID" name:"doc-id" required:""`
	Recipients string `help:"Comma-separated recipient emails" name:"recipients"`
	Subject    string `help:"Email subject override" name:"subject"`
	Execute    bool   `help:"Execute actions" name:"execute"`
}

func (c *EmailFromDocCmd) Run(rctx *RunContext) error {
	if c.Execute && c.Recipients == "" {
		return rctx.Printer.ErrExit(exitcode.UsageError, "--recipients required when --execute is set")
	}
	if done, err := Preflight(rctx, "workflow.email-from-doc", []string{"docs", "gmail"}); done {
		return err
	}

	var recipientList []string
	if c.Recipients != "" {
		recipientList = strings.Split(c.Recipients, ",")
	}

	result, err := workflow.RunEmailFromDoc(rctx.Context, rctx.APIClient, workflow.EmailFromDocOpts{
		DocID:      c.DocID,
		Recipients: recipientList,
		Subject:    c.Subject,
		Execute:    c.Execute,
		NoInput:    rctx.NoInput,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}
	rctx.Printer.Success(result)
	return nil
}

// SheetToEmailCmd implements gwx workflow sheet-to-email.
type SheetToEmailCmd struct {
	SheetID    string `help:"Sheet ID" name:"sheet-id" required:""`
	Range      string `help:"Sheet range (e.g. Sheet1!A:F)" name:"range" required:""`
	EmailCol   int    `help:"Column index for email address (0-based)" name:"email-col" required:""`
	SubjectCol int    `help:"Column index for subject" name:"subject-col" required:""`
	BodyCol    int    `help:"Column index for body" name:"body-col" required:""`
	Execute    bool   `help:"Execute actions" name:"execute"`
}

func (c *SheetToEmailCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "workflow.sheet-to-email", []string{"sheets", "gmail"}); done {
		return err
	}

	result, err := workflow.RunSheetToEmail(rctx.Context, rctx.APIClient, workflow.SheetToEmailOpts{
		SheetID:    c.SheetID,
		Range:      c.Range,
		EmailCol:   c.EmailCol,
		SubjectCol: c.SubjectCol,
		BodyCol:    c.BodyCol,
		Execute:    c.Execute,
		NoInput:    rctx.NoInput,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}
	rctx.Printer.Success(result)
	return nil
}

// ParallelScheduleCmd implements gwx workflow parallel-schedule.
type ParallelScheduleCmd struct {
	Title     string `help:"Meeting title" name:"title" required:""`
	Attendees string `help:"Comma-separated attendee emails" name:"attendees" required:""`
	Duration  string `help:"Meeting duration (e.g. 30m, 1h)" name:"duration" required:""`
	DaysAhead int    `help:"Days ahead to search for slots" name:"days-ahead" default:"7"`
	Execute   bool   `help:"Execute actions" name:"execute"`
}

func (c *ParallelScheduleCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "workflow.parallel-schedule", []string{"calendar"}); done {
		return err
	}

	result, err := workflow.RunParallelSchedule(rctx.Context, rctx.APIClient, workflow.ParallelScheduleOpts{
		Title:     c.Title,
		Attendees: strings.Split(c.Attendees, ","),
		Duration:  c.Duration,
		DaysAhead: c.DaysAhead,
		Execute:   c.Execute,
		NoInput:   rctx.NoInput,
	})
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}
	rctx.Printer.Success(result)
	return nil
}
