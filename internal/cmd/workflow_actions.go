package cmd

import (
	"strings"

	"github.com/redredchen01/gwx/internal/exitcode"
	"github.com/redredchen01/gwx/internal/workflow"
)

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
