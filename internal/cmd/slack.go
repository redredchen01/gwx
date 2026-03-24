package cmd

import (
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/auth"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// SlackCmd groups Slack operations.
type SlackCmd struct {
	Login    SlackLoginCmd    `cmd:"" help:"Save Slack bot token"`
	Status   SlackStatusCmd   `cmd:"" help:"Check Slack auth status"`
	Channels SlackChannelsCmd `cmd:"" help:"List channels"`
	Send     SlackSendCmd     `cmd:"" help:"Send a message"`
	Messages SlackMessagesCmd `cmd:"" help:"List channel messages"`
	Search   SlackSearchCmd   `cmd:"" help:"Search messages"`
	Users    SlackUsersCmd    `cmd:"" help:"List users"`
}

// slackClient loads the Slack token and returns an authenticated client.
func slackClient(rctx *RunContext) (*api.SlackClient, error) {
	token, err := auth.LoadProviderToken("slack", rctx.Account)
	if err != nil {
		return nil, fmt.Errorf("not authenticated to Slack. Run 'gwx slack login' first")
	}
	return api.NewSlackClient(token), nil
}

// SlackLoginCmd saves a Slack bot token.
type SlackLoginCmd struct {
	Token string `arg:"" help:"Slack bot token (xoxb-...)"`
}

func (c *SlackLoginCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "slack.login"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": true, "command": "slack.login"})
		return nil
	}

	if err := auth.SaveProviderToken("slack", rctx.Account, c.Token); err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("save token: %s", err))
	}

	rctx.Printer.Success(map[string]interface{}{
		"provider": "slack",
		"account":  rctx.Account,
		"status":   "authenticated",
	})
	return nil
}

// SlackStatusCmd checks Slack auth status.
type SlackStatusCmd struct{}

func (c *SlackStatusCmd) Run(rctx *RunContext) error {
	if auth.HasProviderToken("slack", rctx.Account) {
		rctx.Printer.Success(map[string]interface{}{
			"provider": "slack",
			"account":  rctx.Account,
			"status":   "authenticated",
		})
		return nil
	}
	return rctx.Printer.ErrExit(exitcode.AuthRequired, "not authenticated to Slack. Run 'gwx slack login <token>'")
}

// SlackChannelsCmd lists Slack channels.
type SlackChannelsCmd struct {
	Limit int `help:"Max channels to return" default:"100" short:"n"`
}

func (c *SlackChannelsCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "slack.channels"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": true, "command": "slack.channels"})
		return nil
	}

	client, err := slackClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	channels, err := client.ListChannels(rctx.Context, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"channels": channels,
		"count":    len(channels),
	})
	return nil
}

// SlackSendCmd sends a Slack message.
type SlackSendCmd struct {
	Channel string `help:"Channel ID or name (e.g. C01234567 or #general)" required:"" short:"c"`
	Text    string `arg:"" help:"Message text"`
}

func (c *SlackSendCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "slack.send"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": true,
			"command": "slack.send",
			"channel": c.Channel,
			"text":    c.Text,
		})
		return nil
	}

	client, err := slackClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	result, err := client.SendMessage(rctx.Context, c.Channel, c.Text)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"sent":    true,
		"channel": c.Channel,
		"result":  result,
	})
	return nil
}

// SlackMessagesCmd lists channel messages.
type SlackMessagesCmd struct {
	Channel string `arg:"" help:"Channel ID"`
	Limit   int    `help:"Max messages to return" default:"20" short:"n"`
}

func (c *SlackMessagesCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "slack.messages"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": true, "command": "slack.messages"})
		return nil
	}

	client, err := slackClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	messages, err := client.ListMessages(rctx.Context, c.Channel, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"channel":  c.Channel,
		"messages": messages,
		"count":    len(messages),
	})
	return nil
}

// SlackSearchCmd searches Slack messages.
type SlackSearchCmd struct {
	Query string `arg:"" help:"Search query"`
	Limit int    `help:"Max results" default:"20" short:"n"`
}

func (c *SlackSearchCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "slack.search"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": true, "command": "slack.search"})
		return nil
	}

	client, err := slackClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	matches, err := client.SearchMessages(rctx.Context, c.Query, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"query":   c.Query,
		"matches": matches,
		"count":   len(matches),
	})
	return nil
}

// SlackUsersCmd lists workspace users.
type SlackUsersCmd struct {
	Limit int `help:"Max users to return" default:"100" short:"n"`
}

func (c *SlackUsersCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "slack.users"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": true, "command": "slack.users"})
		return nil
	}

	client, err := slackClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	users, err := client.ListUsers(rctx.Context, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"users": users,
		"count": len(users),
	})
	return nil
}
