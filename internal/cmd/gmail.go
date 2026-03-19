package cmd

import (
	"errors"
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
	"google.golang.org/api/googleapi"
)

// GmailCmd groups Gmail operations.
type GmailCmd struct {
	List    GmailListCmd    `cmd:"" help:"List messages"`
	Get     GmailGetCmd     `cmd:"" help:"Get a single message"`
	Search  GmailSearchCmd  `cmd:"" help:"Search messages"`
	Labels  GmailLabelsCmd  `cmd:"" help:"List labels"`
	Send    GmailSendCmd    `cmd:"" help:"Send an email"`
	Draft   GmailDraftCmd   `cmd:"" help:"Create a draft"`
	Reply   GmailReplyCmd   `cmd:"" help:"Reply to a message"`
	Digest  GmailDigestCmd  `cmd:"" help:"Smart digest of recent messages"`
	Archive GmailArchiveCmd `cmd:"" help:"Batch archive messages"`
	Label   GmailLabelCmd   `cmd:"" help:"Batch add/remove labels on messages"`
	Forward GmailForwardCmd `cmd:"" help:"Forward a message to new recipients"`
}

// GmailListCmd lists Gmail messages.
type GmailListCmd struct {
	Limit  int64  `help:"Max messages to return (e.g. --limit 5)" default:"10" short:"n"`
	Label  string `help:"Filter by label (e.g. --label INBOX)" short:"l"`
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
	Query string `arg:"" help:"Gmail search query (e.g. 'from:boss subject:urgent')"`
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
	To      []string `help:"Recipients, comma-separated (e.g. --to a@b.com,c@d.com)" required:"" short:"t"`
	CC      []string `help:"CC recipients" short:"c"`
	BCC     []string `help:"BCC recipients"`
	Subject string   `help:"Email subject (e.g. --subject 'Hello')" required:"" short:"s"`
	Body    string   `help:"Email body text (e.g. --body 'Hi there')" required:"" short:"b"`
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

// GmailDigestCmd produces a smart digest.
type GmailDigestCmd struct {
	Limit  int64 `help:"Max messages to analyze" default:"30" short:"n"`
	Unread bool  `help:"Only unread messages" short:"u"`
}

func (c *GmailDigestCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "gmail.digest"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"gmail"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "gmail.digest"})
		return nil
	}

	gmailSvc := api.NewGmailService(rctx.APIClient)
	digest, err := gmailSvc.DigestMessages(rctx.Context, c.Limit, c.Unread)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(digest)
	return nil
}

// GmailArchiveCmd batch archives messages.
type GmailArchiveCmd struct {
	Query    string `arg:"" help:"Gmail search query for messages to archive"`
	Limit    int64  `help:"Max messages to archive" default:"50" short:"n"`
	ReadOnly bool   `help:"Only mark as read, don't remove from inbox" name:"read-only"`
}

func (c *GmailArchiveCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "gmail.archive"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"gmail"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": "gmail.archive",
			"query":   c.Query,
			"limit":   c.Limit,
		})
		return nil
	}

	gmailSvc := api.NewGmailService(rctx.APIClient)

	var count int
	var err error
	if c.ReadOnly {
		count, err = gmailSvc.MarkRead(rctx.Context, c.Query, c.Limit)
	} else {
		count, err = gmailSvc.ArchiveMessages(rctx.Context, c.Query, c.Limit)
	}
	if err != nil {
		return handleAPIError(rctx, err)
	}

	action := "archived"
	if c.ReadOnly {
		action = "marked_read"
	}
	rctx.Printer.Success(map[string]interface{}{
		"action": action,
		"count":  count,
		"query":  c.Query,
	})
	return nil
}

// GmailLabelCmd batch modifies labels on messages.
type GmailLabelCmd struct {
	Query  string   `arg:"" help:"Gmail search query (e.g. 'subject:CI from:github')"`
	Add    []string `help:"Labels to add (e.g. --add CI,Important)" name:"add"`
	Remove []string `help:"Labels to remove (e.g. --remove INBOX)" name:"remove"`
	Limit  int64    `help:"Max messages to modify" default:"50" short:"n"`
}

func (c *GmailLabelCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "gmail.label"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if len(c.Add) == 0 && len(c.Remove) == 0 {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, "at least one of --add or --remove is required")
	}
	if err := EnsureAuth(rctx, []string{"gmail"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": true,
			"command": "gmail.label",
			"query":   c.Query,
			"add":     c.Add,
			"remove":  c.Remove,
			"limit":   c.Limit,
		})
		return nil
	}

	svc := api.NewGmailService(rctx.APIClient)
	count, err := svc.BatchModifyLabels(rctx.Context, c.Query, c.Add, c.Remove, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"action":   "labels_modified",
		"count":    count,
		"query":    c.Query,
		"added":    c.Add,
		"removed":  c.Remove,
	})
	return nil
}

// GmailForwardCmd forwards a message.
type GmailForwardCmd struct {
	MessageID string   `arg:"" help:"Message ID to forward"`
	To        []string `help:"Forward recipients (e.g. --to a@b.com,c@d.com)" required:"" short:"t"`
}

func (c *GmailForwardCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "gmail.forward"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"gmail"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":    true,
			"command":    "gmail.forward",
			"message_id": c.MessageID,
			"to":         c.To,
		})
		return nil
	}

	svc := api.NewGmailService(rctx.APIClient)
	result, err := svc.ForwardMessage(rctx.Context, c.MessageID, c.To)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(result)
	return nil
}

// handleAPIError maps Google API errors to exit codes.
func handleAPIError(rctx *RunContext, err error) error {
	msg := err.Error()

	var circuitErr *api.CircuitOpenError
	if errors.As(err, &circuitErr) {
		return rctx.Printer.ErrExit(exitcode.CircuitOpen, msg)
	}

	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		switch gErr.Code {
		case 401:
			return rctx.Printer.ErrExit(exitcode.AuthExpired, msg)
		case 403:
			return rctx.Printer.ErrExit(exitcode.PermissionDenied, msg)
		case 404:
			return rctx.Printer.ErrExit(exitcode.NotFound, msg)
		case 429:
			return rctx.Printer.ErrExit(exitcode.RateLimited, msg)
		case 409:
			return rctx.Printer.ErrExit(exitcode.Conflict, msg)
		}
	}

	return rctx.Printer.ErrExit(exitcode.GeneralError, msg)
}

// ensure fmt is used (for Sprintf in dry-run messages)
var _ = fmt.Sprintf
