package cmd

import (
	"fmt"

	"github.com/user/gwx/internal/api"
	"github.com/user/gwx/internal/exitcode"
)

// GmailCmd groups Gmail operations.
type GmailCmd struct {
	List   GmailListCmd   `cmd:"" help:"List messages"`
	Get    GmailGetCmd    `cmd:"" help:"Get a single message"`
	Search GmailSearchCmd `cmd:"" help:"Search messages"`
	Labels GmailLabelsCmd `cmd:"" help:"List labels"`
	Send   GmailSendCmd   `cmd:"" help:"Send an email"`
	Draft  GmailDraftCmd  `cmd:"" help:"Create a draft"`
	Reply  GmailReplyCmd  `cmd:"" help:"Reply to a message"`
}

// GmailListCmd lists Gmail messages.
type GmailListCmd struct {
	Limit  int64  `help:"Max messages to return" default:"10" short:"n"`
	Label  string `help:"Filter by label" short:"l"`
	Unread bool   `help:"Only show unread messages" short:"u"`
}

func (c *GmailListCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "gmail.list"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := EnsureAuth(rctx, []string{"gmail"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "gmail.list would execute"})
		return nil
	}

	gmailSvc := api.NewGmailService(rctx.APIClient)

	var labels []string
	if c.Label != "" {
		labels = []string{c.Label}
	}

	messages, total, err := gmailSvc.ListMessages(rctx.Context, "", labels, c.Limit, c.Unread)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"messages":    messages,
		"count":       len(messages),
		"total_estimate": total,
	})
	return nil
}

// GmailGetCmd gets a single message.
type GmailGetCmd struct {
	MessageID string `arg:"" help:"Message ID to retrieve"`
}

func (c *GmailGetCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "gmail.get"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := EnsureAuth(rctx, []string{"gmail"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": fmt.Sprintf("gmail.get %s would execute", c.MessageID)})
		return nil
	}

	gmailSvc := api.NewGmailService(rctx.APIClient)

	msg, err := gmailSvc.GetMessage(rctx.Context, c.MessageID)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(msg)
	return nil
}

// GmailSearchCmd searches messages.
type GmailSearchCmd struct {
	Query string `arg:"" help:"Gmail search query"`
	Limit int64  `help:"Max results" default:"10" short:"n"`
}

func (c *GmailSearchCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "gmail.search"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := EnsureAuth(rctx, []string{"gmail"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": fmt.Sprintf("gmail.search %q would execute", c.Query)})
		return nil
	}

	gmailSvc := api.NewGmailService(rctx.APIClient)

	messages, total, err := gmailSvc.SearchMessages(rctx.Context, c.Query, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"query":          c.Query,
		"messages":       messages,
		"count":          len(messages),
		"total_estimate": total,
	})
	return nil
}

// GmailLabelsCmd lists labels.
type GmailLabelsCmd struct{}

func (c *GmailLabelsCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "gmail.labels"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := EnsureAuth(rctx, []string{"gmail"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	gmailSvc := api.NewGmailService(rctx.APIClient)

	labels, err := gmailSvc.ListLabels(rctx.Context)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"labels": labels,
		"count":  len(labels),
	})
	return nil
}

// GmailSendCmd sends an email.
type GmailSendCmd struct {
	To      []string `help:"Recipients (comma-separated)" required:"" short:"t"`
	CC      []string `help:"CC recipients" short:"c"`
	BCC     []string `help:"BCC recipients"`
	Subject string   `help:"Email subject" required:"" short:"s"`
	Body    string   `help:"Email body text" required:"" short:"b"`
	Attach  []string `help:"File paths to attach" short:"A"`
}

func (c *GmailSendCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "gmail.send"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := EnsureAuth(rctx, []string{"gmail"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	input := &api.SendInput{
		To:          c.To,
		CC:          c.CC,
		BCC:         c.BCC,
		Subject:     c.Subject,
		Body:        c.Body,
		Attachments: c.Attach,
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":     "gmail.send would execute",
			"to":          input.To,
			"cc":          input.CC,
			"subject":     input.Subject,
			"body_length": len(input.Body),
			"attachments": len(input.Attachments),
		})
		return nil
	}

	gmailSvc := api.NewGmailService(rctx.APIClient)
	result, err := gmailSvc.SendMessage(rctx.Context, input)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"sent":       true,
		"message_id": result.MessageID,
		"thread_id":  result.ThreadID,
	})
	return nil
}

// GmailDraftCmd creates a draft.
type GmailDraftCmd struct {
	To      []string `help:"Recipients" required:"" short:"t"`
	CC      []string `help:"CC recipients" short:"c"`
	Subject string   `help:"Email subject" required:"" short:"s"`
	Body    string   `help:"Email body text" required:"" short:"b"`
	Attach  []string `help:"File paths to attach" short:"A"`
}

func (c *GmailDraftCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "gmail.draft"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := EnsureAuth(rctx, []string{"gmail"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	input := &api.SendInput{
		To:          c.To,
		CC:          c.CC,
		Subject:     c.Subject,
		Body:        c.Body,
		Attachments: c.Attach,
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": "gmail.draft would execute",
			"to":      input.To,
			"subject": input.Subject,
		})
		return nil
	}

	gmailSvc := api.NewGmailService(rctx.APIClient)
	result, err := gmailSvc.CreateDraft(rctx.Context, input)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"drafted":    true,
		"message_id": result.MessageID,
		"thread_id":  result.ThreadID,
	})
	return nil
}

// GmailReplyCmd replies to a message.
type GmailReplyCmd struct {
	MessageID string `arg:"" help:"Message ID to reply to"`
	Body      string `help:"Reply body text" required:"" short:"b"`
	ReplyAll  bool   `help:"Reply to all recipients" name:"reply-all"`
}

func (c *GmailReplyCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "gmail.reply"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := EnsureAuth(rctx, []string{"gmail"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	input := &api.SendInput{
		Body:     c.Body,
		ReplyAll: c.ReplyAll,
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":    "gmail.reply would execute",
			"message_id": c.MessageID,
			"reply_all":  c.ReplyAll,
		})
		return nil
	}

	gmailSvc := api.NewGmailService(rctx.APIClient)
	result, err := gmailSvc.ReplyMessage(rctx.Context, c.MessageID, input)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"replied":    true,
		"message_id": result.MessageID,
		"thread_id":  result.ThreadID,
	})
	return nil
}

// handleAPIError maps Google API errors to exit codes.
func handleAPIError(rctx *RunContext, err error) error {
	msg := err.Error()

	if _, ok := err.(*api.CircuitOpenError); ok {
		return rctx.Printer.ErrExit(exitcode.CircuitOpen, msg)
	}

	// Check for common Google API error patterns
	switch {
	case contains(msg, "401") || contains(msg, "Unauthorized"):
		return rctx.Printer.ErrExit(exitcode.AuthExpired, msg)
	case contains(msg, "403") || contains(msg, "Forbidden"):
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, msg)
	case contains(msg, "404") || contains(msg, "Not Found"):
		return rctx.Printer.ErrExit(exitcode.NotFound, msg)
	case contains(msg, "429") || contains(msg, "Rate Limit"):
		return rctx.Printer.ErrExit(exitcode.RateLimited, msg)
	default:
		return rctx.Printer.ErrExit(exitcode.GeneralError, msg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
