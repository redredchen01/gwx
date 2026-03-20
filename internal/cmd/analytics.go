package cmd

import (
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// AnalyticsCmd groups Google Analytics 4 operations.
type AnalyticsCmd struct {
	Report     AnalyticsReportCmd     `cmd:"" help:"Run a GA4 report query"`
	Realtime   AnalyticsRealtimeCmd   `cmd:"" help:"Get real-time analytics data"`
	Properties AnalyticsPropertiesCmd `cmd:"" help:"List GA4 properties"`
	Audiences  AnalyticsAudiencesCmd  `cmd:"" help:"List audiences for a property"`
}

// AnalyticsReportCmd runs a standard GA4 report.
type AnalyticsReportCmd struct {
	Property   string   `help:"GA4 property ID (e.g. properties/123456)" short:"p"`
	Metrics    []string `help:"Metrics to query (e.g. sessions,activeUsers)" required:""`
	Dimensions []string `help:"Dimensions to group by (e.g. date,country)"`
	StartDate  string   `help:"Start date (YYYY-MM-DD or relative: today, yesterday, 7daysAgo)" name:"start-date" default:"7daysAgo"`
	EndDate    string   `help:"End date (YYYY-MM-DD or relative)" name:"end-date" default:"today"`
	Limit      int64    `help:"Max rows to return" default:"100" short:"n"`
}

func (c *AnalyticsReportCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "analytics.report", []string{"analytics"}); done {
		return err
	}

	// Resolve property: flag > config default.
	property := c.Property
	if property == "" {
		val, err := config.Get("analytics.default-property")
		if err != nil || val == "" {
			return rctx.Printer.ErrExit(exitcode.InvalidInput,
				"property is required. Use --property or 'gwx config set analytics.default-property <id>'")
		}
		property = val
	}

	svc := api.NewAnalyticsService(rctx.APIClient)
	result, err := svc.RunReport(rctx.Context, api.ReportRequest{
		Property:   property,
		StartDate:  c.StartDate,
		EndDate:    c.EndDate,
		Metrics:    c.Metrics,
		Dimensions: c.Dimensions,
		Limit:      c.Limit,
	})
	if err != nil {
		return handleAPIError(rctx, err)
	}
	rctx.Printer.Success(result)
	return nil
}

// AnalyticsRealtimeCmd retrieves real-time GA4 data.
type AnalyticsRealtimeCmd struct {
	Property   string   `help:"GA4 property ID (e.g. properties/123456)" short:"p"`
	Metrics    []string `help:"Metrics to query (e.g. activeUsers)" required:""`
	Dimensions []string `help:"Dimensions to group by (e.g. country)"`
	Limit      int64    `help:"Max rows to return" default:"100" short:"n"`
}

func (c *AnalyticsRealtimeCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "analytics.realtime", []string{"analytics"}); done {
		return err
	}

	// Resolve property: flag > config default.
	property := c.Property
	if property == "" {
		val, err := config.Get("analytics.default-property")
		if err != nil || val == "" {
			return rctx.Printer.ErrExit(exitcode.InvalidInput,
				"property is required. Use --property or 'gwx config set analytics.default-property <id>'")
		}
		property = val
	}

	svc := api.NewAnalyticsService(rctx.APIClient)
	result, err := svc.RunRealtimeReport(rctx.Context, property, c.Metrics, c.Dimensions, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}
	rctx.Printer.Success(result)
	return nil
}

// AnalyticsPropertiesCmd lists all accessible GA4 properties.
type AnalyticsPropertiesCmd struct{}

func (c *AnalyticsPropertiesCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "analytics.properties", []string{"analytics"}); done {
		return err
	}

	svc := api.NewAnalyticsService(rctx.APIClient)
	props, err := svc.ListProperties(rctx.Context)
	if err != nil {
		return handleAPIError(rctx, err)
	}
	rctx.Printer.Success(map[string]interface{}{
		"properties": props,
		"count":      len(props),
	})
	return nil
}

// AnalyticsAudiencesCmd lists audiences for a specified GA4 property.
type AnalyticsAudiencesCmd struct {
	Property string `help:"GA4 property ID (e.g. properties/123456)" short:"p"`
}

func (c *AnalyticsAudiencesCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "analytics.audiences", []string{"analytics"}); done {
		return err
	}

	// Resolve property: flag > config default.
	property := c.Property
	if property == "" {
		val, err := config.Get("analytics.default-property")
		if err != nil || val == "" {
			return rctx.Printer.ErrExit(exitcode.InvalidInput,
				"property is required. Use --property or 'gwx config set analytics.default-property <id>'")
		}
		property = val
	}

	svc := api.NewAnalyticsService(rctx.APIClient)
	audiences, err := svc.ListAudiences(rctx.Context, property)
	if err != nil {
		return handleAPIError(rctx, err)
	}
	rctx.Printer.Success(map[string]interface{}{
		"property":  property,
		"audiences": audiences,
		"count":     len(audiences),
	})
	return nil
}

// ensure fmt is used.
var _ = fmt.Sprintf
