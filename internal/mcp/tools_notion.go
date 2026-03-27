package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/auth"
)

type notionProvider struct{}

func (notionProvider) Tools() []Tool {
	return []Tool{
		{
			Name:        "notion_search",
			Description: "Search Notion pages and databases by title.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {Type: "string", Description: "Search query (empty string returns all accessible pages)"},
					"limit": {Type: "integer", Description: "Max results (default 20)"},
				},
			},
		},
		{
			Name:        "notion_page",
			Description: "Get a Notion page by ID.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"page_id": {Type: "string", Description: "Notion page ID"},
				},
				Required: []string{"page_id"},
			},
		},
		{
			Name:        "notion_create_page",
			Description: "Create a new page in a Notion database.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"parent_id":  {Type: "string", Description: "Parent database ID"},
					"title":      {Type: "string", Description: "Page title"},
					"properties": {Type: "string", Description: "Extra properties as JSON object"},
				},
				Required: []string{"parent_id", "title"},
			},
		},
		{
			Name:        "notion_databases",
			Description: "List Notion databases visible to the integration.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit": {Type: "integer", Description: "Max databases to return (default 20)"},
				},
			},
		},
		{
			Name:        "notion_query",
			Description: "Query a Notion database with optional filter.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"database_id": {Type: "string", Description: "Database ID to query"},
					"filter":      {Type: "string", Description: "Filter as JSON object (Notion filter format)"},
					"limit":       {Type: "integer", Description: "Max results (default 20)"},
				},
				Required: []string{"database_id"},
			},
		},
	}
}

func (notionProvider) Handlers(h *GWXHandler) map[string]ToolHandler {
	return map[string]ToolHandler{
		"notion_search":      h.notionSearch,
		"notion_page":        h.notionPage,
		"notion_create_page": h.notionCreatePage,
		"notion_databases":   h.notionDatabases,
		"notion_query":       h.notionQuery,
	}
}

func init() { RegisterProvider(notionProvider{}) }

// resolveNotionClient loads the Notion token from keyring or environment
// and returns an authenticated NotionClient. Caches on the handler.
func (h *GWXHandler) resolveNotionClient() (*api.NotionClient, error) {
	h.notionMu.Lock()
	defer h.notionMu.Unlock()

	if h.notionClient != nil && time.Now().Before(h.notionExpiry) {
		return h.notionClient, nil
	}

	// Check environment variable first (agent-friendly).
	if token := os.Getenv("GWX_NOTION_TOKEN"); token != "" {
		h.notionClient = api.NewNotionClient(token)
		h.notionExpiry = time.Now().Add(5 * time.Minute)
		return h.notionClient, nil
	}
	// Fall back to keyring.
	token, err := auth.LoadProviderToken("notion", "default")
	if err != nil {
		return nil, fmt.Errorf("notion not authenticated — run 'gwx notion login <token>' or set GWX_NOTION_TOKEN")
	}
	h.notionClient = api.NewNotionClient(token)
	h.notionExpiry = time.Now().Add(5 * time.Minute)
	return h.notionClient, nil
}

func (h *GWXHandler) notionSearch(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	client, err := h.resolveNotionClient()
	if err != nil {
		return nil, err
	}
	results, err := client.SearchPages(ctx, strArg(args, "query"), intArg(args, "limit", 20))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{
		"query":   strArg(args, "query"),
		"results": results,
		"count":   len(results),
	})
}

func (h *GWXHandler) notionPage(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	client, err := h.resolveNotionClient()
	if err != nil {
		return nil, err
	}
	page, err := client.GetPage(ctx, strArg(args, "page_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(page)
}

func (h *GWXHandler) notionCreatePage(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	client, err := h.resolveNotionClient()
	if err != nil {
		return nil, err
	}

	var props map[string]interface{}
	if raw := strArg(args, "properties"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &props); err != nil {
			return nil, fmt.Errorf("invalid properties JSON: %w", err)
		}
	}

	page, err := client.CreatePage(ctx, strArg(args, "parent_id"), strArg(args, "title"), props)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"created": true, "page": page})
}

func (h *GWXHandler) notionDatabases(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	client, err := h.resolveNotionClient()
	if err != nil {
		return nil, err
	}
	databases, err := client.ListDatabases(ctx, intArg(args, "limit", 20))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"databases": databases, "count": len(databases)})
}

func (h *GWXHandler) notionQuery(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	client, err := h.resolveNotionClient()
	if err != nil {
		return nil, err
	}

	var filter map[string]interface{}
	if raw := strArg(args, "filter"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &filter); err != nil {
			return nil, fmt.Errorf("invalid filter JSON: %w", err)
		}
	}

	results, err := client.QueryDatabase(ctx, strArg(args, "database_id"), filter, intArg(args, "limit", 20))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{
		"database_id": strArg(args, "database_id"),
		"results":     results,
		"count":       len(results),
	})
}
