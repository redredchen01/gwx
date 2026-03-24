package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/auth"
)

type slackProvider struct{}

func (slackProvider) Tools() []Tool {
	return []Tool{
		{
			Name:        "slack_channels",
			Description: "List Slack channels visible to the bot.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit": {Type: "integer", Description: "Max channels to return (default 100)"},
				},
			},
		},
		{
			Name:        "slack_send",
			Description: "Send a message to a Slack channel.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"channel": {Type: "string", Description: "Channel ID or name (e.g. C01234567 or #general)"},
					"text":    {Type: "string", Description: "Message text to send"},
				},
				Required: []string{"channel", "text"},
			},
		},
		{
			Name:        "slack_messages",
			Description: "List recent messages from a Slack channel.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"channel": {Type: "string", Description: "Channel ID"},
					"limit":   {Type: "integer", Description: "Max messages to return (default 20)"},
				},
				Required: []string{"channel"},
			},
		},
		{
			Name:        "slack_search",
			Description: "Search messages across the Slack workspace.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {Type: "string", Description: "Search query"},
					"limit": {Type: "integer", Description: "Max results (default 20)"},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "slack_users",
			Description: "List Slack workspace members.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit": {Type: "integer", Description: "Max users to return (default 100)"},
				},
			},
		},
		{
			Name:        "slack_user",
			Description: "Get profile information for a single Slack user.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"user_id": {Type: "string", Description: "Slack user ID (e.g. U01234567)"},
				},
				Required: []string{"user_id"},
			},
		},
	}
}

func (slackProvider) Handlers(h *GWXHandler) map[string]ToolHandler {
	return map[string]ToolHandler{
		"slack_channels": h.slackChannels,
		"slack_send":     h.slackSend,
		"slack_messages": h.slackMessages,
		"slack_search":   h.slackSearch,
		"slack_users":    h.slackUsers,
		"slack_user":     h.slackUser,
	}
}

func init() { RegisterProvider(slackProvider{}) }

// resolveSlackClient loads the Slack token from keyring or environment
// and returns an authenticated SlackClient.
func resolveSlackClient() (*api.SlackClient, error) {
	// Check environment variable first (agent-friendly).
	if token := os.Getenv("GWX_SLACK_TOKEN"); token != "" {
		return api.NewSlackClient(token), nil
	}
	// Fall back to keyring.
	token, err := auth.LoadProviderToken("slack", "default")
	if err != nil {
		return nil, fmt.Errorf("Slack not authenticated. Run 'gwx slack login <token>' or set GWX_SLACK_TOKEN")
	}
	return api.NewSlackClient(token), nil
}

func (h *GWXHandler) slackChannels(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	client, err := resolveSlackClient()
	if err != nil {
		return nil, err
	}
	channels, err := client.ListChannels(ctx, intArg(args, "limit", 100))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"channels": channels, "count": len(channels)})
}

func (h *GWXHandler) slackSend(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	client, err := resolveSlackClient()
	if err != nil {
		return nil, err
	}
	result, err := client.SendMessage(ctx, strArg(args, "channel"), strArg(args, "text"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"sent": true, "result": result})
}

func (h *GWXHandler) slackMessages(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	client, err := resolveSlackClient()
	if err != nil {
		return nil, err
	}
	messages, err := client.ListMessages(ctx, strArg(args, "channel"), intArg(args, "limit", 20))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{
		"channel":  strArg(args, "channel"),
		"messages": messages,
		"count":    len(messages),
	})
}

func (h *GWXHandler) slackSearch(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	client, err := resolveSlackClient()
	if err != nil {
		return nil, err
	}
	matches, err := client.SearchMessages(ctx, strArg(args, "query"), intArg(args, "limit", 20))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{
		"query":   strArg(args, "query"),
		"matches": matches,
		"count":   len(matches),
	})
}

func (h *GWXHandler) slackUsers(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	client, err := resolveSlackClient()
	if err != nil {
		return nil, err
	}
	users, err := client.ListUsers(ctx, intArg(args, "limit", 100))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"users": users, "count": len(users)})
}

func (h *GWXHandler) slackUser(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	client, err := resolveSlackClient()
	if err != nil {
		return nil, err
	}
	user, err := client.GetUser(ctx, strArg(args, "user_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(user)
}
