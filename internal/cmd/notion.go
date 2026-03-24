package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/auth"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// NotionCmd groups Notion operations.
type NotionCmd struct {
	Login     NotionLoginCmd     `cmd:"" help:"Save Notion integration token"`
	Status    NotionStatusCmd    `cmd:"" help:"Check Notion auth status"`
	Search    NotionSearchCmd    `cmd:"" help:"Search pages"`
	Page      NotionPageCmd      `cmd:"" help:"Get a page"`
	Create    NotionCreateCmd    `cmd:"" help:"Create a page in a database"`
	Databases NotionDatabasesCmd `cmd:"" help:"List databases"`
	Query     NotionQueryCmd     `cmd:"" help:"Query a database"`
}

// notionClient loads the Notion token and returns an authenticated client.
func notionClient(rctx *RunContext) (*api.NotionClient, error) {
	token, err := auth.LoadProviderToken("notion", rctx.Account)
	if err != nil {
		return nil, fmt.Errorf("not authenticated to Notion. Run 'gwx notion login' first")
	}
	return api.NewNotionClient(token), nil
}

// NotionLoginCmd saves a Notion integration token.
type NotionLoginCmd struct {
	Token string `arg:"" help:"Notion integration token (ntn_...)"`
}

func (c *NotionLoginCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "notion.login"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": true, "command": "notion.login"})
		return nil
	}

	if err := auth.SaveProviderToken("notion", rctx.Account, c.Token); err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("save token: %s", err))
	}

	rctx.Printer.Success(map[string]interface{}{
		"provider": "notion",
		"account":  rctx.Account,
		"status":   "authenticated",
	})
	return nil
}

// NotionStatusCmd checks Notion auth status.
type NotionStatusCmd struct{}

func (c *NotionStatusCmd) Run(rctx *RunContext) error {
	if auth.HasProviderToken("notion", rctx.Account) {
		rctx.Printer.Success(map[string]interface{}{
			"provider": "notion",
			"account":  rctx.Account,
			"status":   "authenticated",
		})
		return nil
	}
	return rctx.Printer.ErrExit(exitcode.AuthRequired, "not authenticated to Notion. Run 'gwx notion login <token>'")
}

// NotionSearchCmd searches Notion pages.
type NotionSearchCmd struct {
	Query string `arg:"" help:"Search query" default:""`
	Limit int    `help:"Max results" default:"20" short:"n"`
}

func (c *NotionSearchCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "notion.search"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": true, "command": "notion.search"})
		return nil
	}

	client, err := notionClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	results, err := client.SearchPages(rctx.Context, c.Query, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"query":   c.Query,
		"results": results,
		"count":   len(results),
	})
	return nil
}

// NotionPageCmd retrieves a Notion page.
type NotionPageCmd struct {
	PageID string `arg:"" help:"Page ID"`
}

func (c *NotionPageCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "notion.page"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": true, "command": "notion.page"})
		return nil
	}

	client, err := notionClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	page, err := client.GetPage(rctx.Context, c.PageID)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(page)
	return nil
}

// NotionCreateCmd creates a page in a Notion database.
type NotionCreateCmd struct {
	ParentID   string `help:"Parent database ID" required:"" name:"parent"`
	Title      string `help:"Page title" required:"" short:"t"`
	Properties string `help:"Extra properties as JSON object" name:"props" default:""`
}

func (c *NotionCreateCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "notion.create"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":   true,
			"command":   "notion.create",
			"parent_id": c.ParentID,
			"title":     c.Title,
		})
		return nil
	}

	client, err := notionClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	var props map[string]interface{}
	if c.Properties != "" {
		if err := json.Unmarshal([]byte(c.Properties), &props); err != nil {
			return rctx.Printer.ErrExit(exitcode.InvalidInput, fmt.Sprintf("invalid properties JSON: %s", err))
		}
	}

	page, err := client.CreatePage(rctx.Context, c.ParentID, c.Title, props)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"created": true,
		"page":    page,
	})
	return nil
}

// NotionDatabasesCmd lists databases visible to the integration.
type NotionDatabasesCmd struct {
	Limit int `help:"Max databases to return" default:"20" short:"n"`
}

func (c *NotionDatabasesCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "notion.databases"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": true, "command": "notion.databases"})
		return nil
	}

	client, err := notionClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	databases, err := client.ListDatabases(rctx.Context, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"databases": databases,
		"count":     len(databases),
	})
	return nil
}

// NotionQueryCmd queries a Notion database.
type NotionQueryCmd struct {
	DatabaseID string `arg:"" help:"Database ID to query"`
	Filter     string `help:"Filter as JSON object" default:""`
	Limit      int    `help:"Max results" default:"20" short:"n"`
}

func (c *NotionQueryCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "notion.query"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": true, "command": "notion.query"})
		return nil
	}

	client, err := notionClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	var filter map[string]interface{}
	if c.Filter != "" {
		if err := json.Unmarshal([]byte(c.Filter), &filter); err != nil {
			return rctx.Printer.ErrExit(exitcode.InvalidInput, fmt.Sprintf("invalid filter JSON: %s", err))
		}
	}

	results, err := client.QueryDatabase(rctx.Context, c.DatabaseID, filter, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"database_id": c.DatabaseID,
		"results":     results,
		"count":       len(results),
	})
	return nil
}
