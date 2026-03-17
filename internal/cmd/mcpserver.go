package cmd

import (
	"fmt"
	"os"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/auth"
	"github.com/redredchen01/gwx/internal/mcp"
)

// MCPServerCmd starts the MCP server over stdio.
type MCPServerCmd struct{}

func (c *MCPServerCmd) Run(rctx *RunContext) error {
	// MCP server needs auth — load token silently
	if err := EnsureAuth(rctx, []string{"gmail", "calendar", "drive", "docs", "sheets", "tasks", "people", "chat"}); err != nil {
		// Try loading with whatever scopes are available
		if token := os.Getenv("GWX_ACCESS_TOKEN"); token != "" {
			ts := auth.TokenFromDirect(token)
			rctx.APIClient = api.NewClient(ts)
		} else {
			fmt.Fprintf(os.Stderr, "gwx mcp-server: not authenticated. Run 'gwx onboard' first.\n")
			return err
		}
	}

	handler := mcp.NewGWXHandler(rctx.APIClient)
	server := mcp.NewServer(handler)

	fmt.Fprintf(os.Stderr, "gwx MCP server started (stdio)\n")
	return server.Run()
}
