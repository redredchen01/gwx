package mcp

import (
	"context"
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
)

type analyticsProvider struct{}

func (analyticsProvider) Tools() []Tool {
	return []Tool{
		{
			Name:        "analytics_report",
			Description: "Run a Google Analytics 4 report query. Returns metrics (e.g. sessions, activeUsers) grouped by dimensions (e.g. date, country) for a date range.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"property":   {Type: "string", Description: "GA4 property ID (e.g. properties/123456). If omitted, uses default from config."},
					"metrics":    {Type: "string", Description: "Comma-separated metrics to query (e.g. sessions,activeUsers,screenPageViews)"},
					"dimensions": {Type: "string", Description: "Comma-separated dimensions to group by (e.g. date,country,deviceCategory)"},
					"start_date": {Type: "string", Description: "Start date (YYYY-MM-DD or relative: today, yesterday, 7daysAgo, 30daysAgo). Default: 7daysAgo"},
					"end_date":   {Type: "string", Description: "End date (YYYY-MM-DD or relative). Default: today"},
					"limit":      {Type: "integer", Description: "Max rows to return. Default: 100"},
				},
				Required: []string{"metrics"},
			},
		},
		{
			Name:        "analytics_realtime",
			Description: "Run a Google Analytics 4 realtime report. Returns active-user counts and other real-time metrics for the current moment.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"property":   {Type: "string", Description: "GA4 property ID (e.g. properties/123456). If omitted, uses default from config."},
					"metrics":    {Type: "string", Description: "Comma-separated real-time metrics (e.g. activeUsers,screenPageViews)"},
					"dimensions": {Type: "string", Description: "Comma-separated dimensions (e.g. country,deviceCategory,unifiedScreenName)"},
					"limit":      {Type: "integer", Description: "Max rows to return. Default: 100"},
				},
				Required: []string{"metrics"},
			},
		},
		{
			Name:        "analytics_properties",
			Description: "List all Google Analytics 4 properties accessible to the authenticated account.",
			InputSchema: InputSchema{
				Type: "object",
			},
		},
		{
			Name:        "analytics_audiences",
			Description: "List all audiences defined for a GA4 property.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"property": {Type: "string", Description: "GA4 property ID (e.g. properties/123456). If omitted, uses default from config."},
				},
			},
		},
	}
}

func (analyticsProvider) Handlers(h *GWXHandler) map[string]ToolHandler {
	return map[string]ToolHandler{
		"analytics_report":     h.analyticsReport,
		"analytics_realtime":   h.analyticsRealtime,
		"analytics_properties": h.analyticsProperties,
		"analytics_audiences":  h.analyticsAudiences,
	}
}

func init() { RegisterProvider(analyticsProvider{}) }

// resolveProperty returns the property arg if provided, otherwise falls back to
// the "analytics.default-property" config key. Returns an error if neither is set.
func resolveProperty(args map[string]interface{}) (string, error) {
	if p := strArg(args, "property"); p != "" {
		return p, nil
	}
	p, err := config.Get("analytics.default-property")
	if err != nil {
		return "", fmt.Errorf("analytics: could not read default property from config: %w", err)
	}
	if p == "" {
		return "", fmt.Errorf("analytics: property not provided and analytics.default-property is not configured")
	}
	return p, nil
}

// --- Analytics handlers ---

func (h *GWXHandler) analyticsReport(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	property, err := resolveProperty(args)
	if err != nil {
		return nil, err
	}

	startDate := strArg(args, "start_date")
	if startDate == "" {
		startDate = "7daysAgo"
	}
	endDate := strArg(args, "end_date")
	if endDate == "" {
		endDate = "today"
	}

	req := api.ReportRequest{
		Property:   property,
		StartDate:  startDate,
		EndDate:    endDate,
		Metrics:    splitArg(args, "metrics"),
		Dimensions: splitArg(args, "dimensions"),
		Limit:      int64(intArg(args, "limit", 100)),
	}

	svc := api.NewAnalyticsService(h.client)
	result, err := svc.RunReport(ctx, req)
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (h *GWXHandler) analyticsRealtime(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	property, err := resolveProperty(args)
	if err != nil {
		return nil, err
	}

	metrics := splitArg(args, "metrics")
	dimensions := splitArg(args, "dimensions")
	limit := int64(intArg(args, "limit", 100))

	svc := api.NewAnalyticsService(h.client)
	result, err := svc.RunRealtimeReport(ctx, property, metrics, dimensions, limit)
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (h *GWXHandler) analyticsProperties(ctx context.Context, _ map[string]interface{}) (*ToolResult, error) {
	svc := api.NewAnalyticsService(h.client)
	properties, err := svc.ListProperties(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"properties": properties, "count": len(properties)})
}

func (h *GWXHandler) analyticsAudiences(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	property, err := resolveProperty(args)
	if err != nil {
		return nil, err
	}

	svc := api.NewAnalyticsService(h.client)
	audiences, err := svc.ListAudiences(ctx, property)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"property": property, "audiences": audiences, "count": len(audiences)})
}

