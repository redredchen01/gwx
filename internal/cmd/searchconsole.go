package cmd

import (
	"fmt"
	"time"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// SearchConsoleCmd groups Search Console operations.
type SearchConsoleCmd struct {
	Query       SearchConsoleQueryCmd       `cmd:"" help:"Query search analytics data"`
	Sites       SearchConsoleSitesCmd       `cmd:"" help:"List verified sites"`
	Inspect     SearchConsoleInspectCmd     `cmd:"" help:"Inspect a URL's index status"`
	Sitemaps    SearchConsoleSitemapsCmd    `cmd:"" help:"List sitemaps for a site"`
	IndexStatus SearchConsoleIndexStatusCmd `cmd:"index-status" help:"Get index coverage status"`
}

// SearchConsoleQueryCmd queries search analytics data.
type SearchConsoleQueryCmd struct {
	Site        string   `help:"Site URL (e.g. https://example.com)" short:"s"`
	StartDate   string   `help:"Start date (YYYY-MM-DD)" name:"start-date" required:""`
	EndDate     string   `help:"End date (YYYY-MM-DD)" name:"end-date" default:""`
	Dimensions  []string `help:"Dimensions (query, page, country, device, date)"`
	QueryFilter string   `help:"Filter by query text" name:"query-filter"`
	Limit       int      `help:"Max rows to return" default:"100" short:"n"`
}

func (c *SearchConsoleQueryCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "searchconsole.query", []string{"searchconsole"}); done {
		return err
	}

	site := c.Site
	if site == "" {
		val, err := config.Get("searchconsole.default-site")
		if err != nil {
			return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("load config: %s", err))
		}
		site = val
	}
	if site == "" {
		return rctx.Printer.ErrExit(exitcode.UsageError, "site URL is required (use --site or set searchconsole.default-site in config)")
	}

	scSvc := api.NewSearchConsoleService(rctx.APIClient)

	result, err := scSvc.Query(rctx.Context, api.SearchQueryRequest{
		SiteURL:    site,
		StartDate:  c.StartDate,
		EndDate:    c.EndDate,
		Dimensions: c.Dimensions,
		Query:      c.QueryFilter,
		Limit:      c.Limit,
	})
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(result)
	return nil
}

// SearchConsoleSitesCmd lists all verified Search Console sites.
type SearchConsoleSitesCmd struct{}

func (c *SearchConsoleSitesCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "searchconsole.sites", []string{"searchconsole"}); done {
		return err
	}

	scSvc := api.NewSearchConsoleService(rctx.APIClient)

	sites, err := scSvc.ListSites(rctx.Context)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"sites": sites,
		"count": len(sites),
	})
	return nil
}

// SearchConsoleInspectCmd inspects a URL's index status.
type SearchConsoleInspectCmd struct {
	Site string `help:"Site URL" short:"s" required:""`
	URL  string `arg:"" help:"URL to inspect"`
}

func (c *SearchConsoleInspectCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "searchconsole.inspect", []string{"searchconsole"}); done {
		return err
	}

	scSvc := api.NewSearchConsoleService(rctx.APIClient)

	result, err := scSvc.InspectURL(rctx.Context, c.Site, c.URL)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(result)
	return nil
}

// SearchConsoleSitemapsCmd lists sitemaps for a site.
type SearchConsoleSitemapsCmd struct {
	Site string `help:"Site URL" short:"s"`
}

func (c *SearchConsoleSitemapsCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "searchconsole.sitemaps", []string{"searchconsole"}); done {
		return err
	}

	site := c.Site
	if site == "" {
		val, err := config.Get("searchconsole.default-site")
		if err != nil {
			return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("load config: %s", err))
		}
		site = val
	}
	if site == "" {
		return rctx.Printer.ErrExit(exitcode.UsageError, "site URL is required (use --site or set searchconsole.default-site in config)")
	}

	scSvc := api.NewSearchConsoleService(rctx.APIClient)

	sitemaps, err := scSvc.ListSitemaps(rctx.Context, site)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"site":     site,
		"sitemaps": sitemaps,
		"count":    len(sitemaps),
	})
	return nil
}

// SearchConsoleIndexStatusCmd gets index coverage status for a site.
type SearchConsoleIndexStatusCmd struct {
	Site      string `help:"Site URL" short:"s"`
	StartDate string `help:"Start date (YYYY-MM-DD); defaults to 28 days ago" name:"start-date"`
	EndDate   string `help:"End date (YYYY-MM-DD); defaults to today" name:"end-date"`
}

func (c *SearchConsoleIndexStatusCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "searchconsole.index-status", []string{"searchconsole"}); done {
		return err
	}

	site := c.Site
	if site == "" {
		val, err := config.Get("searchconsole.default-site")
		if err != nil {
			return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("load config: %s", err))
		}
		site = val
	}
	if site == "" {
		return rctx.Printer.ErrExit(exitcode.UsageError, "site URL is required (use --site or set searchconsole.default-site in config)")
	}

	// Resolve date range defaults.
	endDate := c.EndDate
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}
	startDate := c.StartDate
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -28).Format("2006-01-02")
	}

	scSvc := api.NewSearchConsoleService(rctx.APIClient)

	summary, err := scSvc.GetIndexStatus(rctx.Context, site, startDate, endDate)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(summary)
	return nil
}
