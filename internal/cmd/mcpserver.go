package cmd

import (
	"log/slog"
	"os"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/auth"
	gwxlog "github.com/redredchen01/gwx/internal/log"
	"github.com/redredchen01/gwx/internal/mcp"
	"github.com/redredchen01/gwx/internal/skill"
)

// MCPServerCmd starts the MCP server over stdio.
type MCPServerCmd struct{}

func (c *MCPServerCmd) Run(rctx *RunContext) error {
	logger := gwxlog.SetupMCPLogger()
	slog.SetDefault(logger)

	// MCP server needs auth — load token silently
	if err := EnsureAuth(rctx, []string{"gmail", "calendar", "drive", "docs", "sheets", "tasks", "people", "chat", "analytics", "searchconsole", "slides"}); err != nil {
		// Try loading with whatever scopes are available
		if token := os.Getenv("GWX_ACCESS_TOKEN"); token != "" {
			ts := auth.TokenFromDirect(token)
			rctx.APIClient = api.NewClient(ts)
		} else {
			slog.Error("not authenticated", "hint", "run gwx onboard")
			return err
		}
	}

	// Load skill DSL definitions so they appear as MCP tools.
	if err := skill.Reload(); err != nil {
		slog.Warn("failed to load skills", "error", err)
	}

	handler := mcp.NewGWXHandler(rctx.APIClient)
	server := mcp.NewServer(handler)

	slog.Info("MCP server started", "transport", "stdio")
	return server.Run()
}
