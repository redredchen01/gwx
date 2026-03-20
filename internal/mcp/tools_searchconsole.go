package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
)

type searchconsoleProvider struct{}

func (searchconsoleProvider) Tools() []Tool {
	return []Tool{
		{
			Name:        "searchconsole_query",
			Description: "Query Search Console search analytics data (clicks, impressions, CTR, position).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"site":         {Type: "string", Description: "Site URL (e.g. https://example.com/); uses config default if omitted"},
					"start_date":   {Type: "string", Description: "Start date (YYYY-MM-DD)"},
					"end_date":     {Type: "string", Description: "End date (YYYY-MM-DD)"},
					"dimensions":   {Type: "string", Description: "Comma-separated dimensions to group by: query, page, country, device, searchAppearance"},
					"query_filter": {Type: "string", Description: "Filter rows to queries containing this string"},
					"limit":        {Type: "integer", Description: "Max rows to return (default 100)"},
				},
				Required: []string{"start_date"},
			},
		},
		{
			Name:        "searchconsole_sites",
			Description: "List all Search Console properties the authenticated user can access.",
			InputSchema: InputSchema{
				Type: "object",
			},
		},
		{
			Name:        "searchconsole_inspect",
			Description: "Inspect a URL's index status in Search Console. NOTE: 2000 requests/day quota limit applies.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"site": {Type: "string", Description: "Site URL (e.g. https://example.com/)"},
					"url":  {Type: "string", Description: "URL to inspect"},
				},
				Required: []string{"site", "url"},
			},
		},
		{
			Name:        "searchconsole_sitemaps",
			Description: "List sitemaps submitted to Search Console for a site.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"site": {Type: "string", Description: "Site URL; uses config default if omitted"},
				},
			},
		},
		{
			Name:        "searchconsole_index_status",
			Description: "Get approximate index status (pages with impressions) for a site over a date range.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"site":       {Type: "string", Description: "Site URL; uses config default if omitted"},
					"start_date": {Type: "string", Description: "Start date (YYYY-MM-DD); defaults to 28 days ago"},
					"end_date":   {Type: "string", Description: "End date (YYYY-MM-DD); defaults to today"},
				},
			},
		},
	}
}

func (searchconsoleProvider) Handlers(h *GWXHandler) map[string]ToolHandler {
	return map[string]ToolHandler{
		"searchconsole_query":        h.searchconsoleQuery,
		"searchconsole_sites":        h.searchconsoleSites,
		"searchconsole_inspect":      h.searchconsoleInspect,
		"searchconsole_sitemaps":     h.searchconsoleSitemaps,
		"searchconsole_index_status": h.searchconsoleIndexStatus,
	}
}

func init() { RegisterProvider(searchconsoleProvider{}) }

// --- helpers ---

// resolveSearchConsoleSite returns the site URL from args or config default.
func resolveSearchConsoleSite(args map[string]interface{}) (string, error) {
	site := strArg(args, "site")
	if site != "" {
		return site, nil
	}
	val, err := config.Get("searchconsole.default-site")
	if err != nil {
		return "", err
	}
	if val == "" {
		return "", fmt.Errorf("site is required. Provide 'site' parameter or run: gwx config set searchconsole.default-site <url>")
	}
	return val, nil
}

// --- handlers ---

func (h *GWXHandler) searchconsoleQuery(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	site, err := resolveSearchConsoleSite(args)
	if err != nil {
		return nil, err
	}

	// Parse comma-separated dimensions into a slice.
	var dimensions []string
	if raw := strArg(args, "dimensions"); raw != "" {
		for _, d := range strings.Split(raw, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				dimensions = append(dimensions, d)
			}
		}
	}

	req := api.SearchQueryRequest{
		SiteURL:    site,
		StartDate:  strArg(args, "start_date"),
		EndDate:    strArg(args, "end_date"),
		Dimensions: dimensions,
		Query:      strArg(args, "query_filter"),
		Limit:      intArg(args, "limit", 0),
	}

	svc := api.NewSearchConsoleService(h.client)
	result, err := svc.Query(ctx, req)
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (h *GWXHandler) searchconsoleSites(ctx context.Context, _ map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSearchConsoleService(h.client)
	sites, err := svc.ListSites(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"sites": sites, "count": len(sites)})
}

func (h *GWXHandler) searchconsoleInspect(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSearchConsoleService(h.client)
	result, err := svc.InspectURL(ctx, strArg(args, "site"), strArg(args, "url"))
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (h *GWXHandler) searchconsoleSitemaps(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	site, err := resolveSearchConsoleSite(args)
	if err != nil {
		return nil, err
	}

	svc := api.NewSearchConsoleService(h.client)
	sitemaps, err := svc.ListSitemaps(ctx, site)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"site": site, "sitemaps": sitemaps, "count": len(sitemaps)})
}

func (h *GWXHandler) searchconsoleIndexStatus(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	site, err := resolveSearchConsoleSite(args)
	if err != nil {
		return nil, err
	}

	endDate := strArg(args, "end_date")
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}
	startDate := strArg(args, "start_date")
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -28).Format("2006-01-02")
	}

	svc := api.NewSearchConsoleService(h.client)
	summary, err := svc.GetIndexStatus(ctx, site, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return jsonResult(summary)
}

