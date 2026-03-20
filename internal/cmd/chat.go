package cmd

import (
	"github.com/redredchen01/gwx/internal/api"
)

// ChatCmd groups Chat operations.
type ChatCmd struct {
	Spaces   ChatSpacesCmd   `cmd:"" help:"List Chat spaces"`
	Send     ChatSendCmd     `cmd:"" help:"Send a message to a space"`
	Messages ChatMessagesCmd `cmd:"" help:"List messages in a space"`
}

// ChatSpacesCmd lists spaces.
type ChatSpacesCmd struct {
	Limit int `help:"Max spaces to return" default:"50" short:"n"`
}

func (c *ChatSpacesCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "chat.spaces", []string{"chat"}); done {
		return err
	}

	chatSvc := api.NewChatService(rctx.APIClient)
	spaces, err := chatSvc.ListSpaces(rctx.Context, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"spaces": spaces,
		"count":  len(spaces),
	})
	return nil
}

// ChatSendCmd sends a message.
type ChatSendCmd struct {
	Space string `arg:"" help:"Space name (e.g. spaces/AAAA)"`
	Text  string `help:"Message text" required:"" short:"t"`
}

func (c *ChatSendCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "chat.send", []string{"chat"}); done {
		return err
	}

	chatSvc := api.NewChatService(rctx.APIClient)
	result, err := chatSvc.SendMessage(rctx.Context, c.Space, c.Text)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"sent":    true,
		"message": result,
	})
	return nil
}

// ChatMessagesCmd lists messages.
type ChatMessagesCmd struct {
	Space string `arg:"" help:"Space name (e.g. spaces/AAAA)"`
	Limit int    `help:"Max messages" default:"20" short:"n"`
}

func (c *ChatMessagesCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "chat.messages", []string{"chat"}); done {
		return err
	}

	chatSvc := api.NewChatService(rctx.APIClient)
	messages, err := chatSvc.ListMessages(rctx.Context, c.Space, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"messages": messages,
		"count":    len(messages),
	})
	return nil
}
