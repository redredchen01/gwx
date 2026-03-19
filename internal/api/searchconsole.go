package api

import (
	"context"
	"fmt"
	"time"

	searchconsole "google.golang.org/api/searchconsole/v1"
)

// SearchConsoleService wraps Google Search Console API operations.
type SearchConsoleService struct {
	client *Client
}

// NewSearchConsoleService creates a Search Console service wrapper.
func NewSearchConsoleService(client *Client) *SearchConsoleService {
	return &SearchConsoleService{client: client}
}

// SearchQueryRequest specifies parameters for a Search Analytics query.
type SearchQueryRequest struct {
	SiteURL    string
	StartDate  string
	EndDate    string
	Dimensions []string
	Query      string
	Limit      int
}

// SearchQueryRow represents a single row of Search Analytics data.
type SearchQueryRow struct {
	Keys        []string `json:"keys"`
	Clicks      float64  `json:"clicks"`
	Impressions float64  `json:"impressions"`
	CTR         float64  `json:"ctr"`
	Position    float64  `json:"position"`
}

// SearchQueryResult is the full result of a Search Analytics query.
type SearchQueryResult struct {
	Rows      []SearchQueryRow `json:"rows"`
	RowCount  int              `json:"row_count"`
	SiteURL   string           `json:"site_url"`
	DateRange string           `json:"date_range"`
}

// SiteSummary is a simplified representation of a Search Console site entry.
type SiteSummary struct {
	SiteURL         string `json:"site_url"`
	PermissionLevel string `json:"permission_level"`
}

// URLInspectionResult contains the index inspection outcome for a URL.
type URLInspectionResult struct {
	URL            string `json:"url"`
	Verdict        string `json:"verdict"`
	CoverageState  string `json:"coverage_state"`
	IndexingState  string `json:"indexing_state"`
	LastCrawlTime  string `json:"last_crawl_time,omitempty"`
	CrawledAs      string `json:"crawled_as,omitempty"`
	RobotsTxtState string `json:"robots_txt_state,omitempty"`
}

// SitemapInfo contains summarised information about a submitted sitemap.
type SitemapInfo struct {
	Path           string `json:"path"`
	Type           string `json:"type"`
	LastSubmitted  string `json:"last_submitted,omitempty"`
	LastDownloaded string `json:"last_downloaded,omitempty"`
	IsPending      bool   `json:"is_pending"`
	Warnings       int64  `json:"warnings"`
	Errors         int64  `json:"errors"`
}

// IndexStatusSummary provides a high-level view of indexing status for a site.
type IndexStatusSummary struct {
	SiteURL        string `json:"site_url"`
	TotalIndexed   int64  `json:"total_indexed,omitempty"`
	TotalSubmitted int64  `json:"total_submitted,omitempty"`
	CoverageState  string `json:"coverage_state,omitempty"`
}

// Query executes a Search Analytics query and returns the matching rows.
// If req.Limit is zero, the default of 100 is used.
func (sc *SearchConsoleService) Query(ctx context.Context, req SearchQueryRequest) (*SearchQueryResult, error) {
	if err := sc.client.WaitRate(ctx, "searchconsole"); err != nil {
		return nil, err
	}

	opts, err := sc.client.ClientOptions(ctx, "searchconsole")
	if err != nil {
		return nil, err
	}

	svc, err := searchconsole.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create searchconsole service: %w", err)
	}

	limit := int64(req.Limit)
	if limit <= 0 {
		limit = 100
	}
	if limit > 25000 {
		limit = 25000
	}

	endDate := req.EndDate
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	apiReq := &searchconsole.SearchAnalyticsQueryRequest{
		StartDate:  req.StartDate,
		EndDate:    endDate,
		Dimensions: req.Dimensions,
		RowLimit:   limit,
	}

	if req.Query != "" {
		apiReq.DimensionFilterGroups = []*searchconsole.ApiDimensionFilterGroup{
			{
				GroupType: "AND",
				Filters: []*searchconsole.ApiDimensionFilter{
					{
						Dimension:  "QUERY",
						Operator:   "CONTAINS",
						Expression: req.Query,
					},
				},
			},
		}
	}

	resp, err := svc.Searchanalytics.Query(req.SiteURL, apiReq).Do()
	if err != nil {
		return nil, fmt.Errorf("searchconsole query %s: %w", req.SiteURL, err)
	}

	rows := make([]SearchQueryRow, 0, len(resp.Rows))
	for _, r := range resp.Rows {
		rows = append(rows, SearchQueryRow{
			Keys:        r.Keys,
			Clicks:      r.Clicks,
			Impressions: r.Impressions,
			CTR:         r.Ctr,
			Position:    r.Position,
		})
	}

	return &SearchQueryResult{
		Rows:      rows,
		RowCount:  len(rows),
		SiteURL:   req.SiteURL,
		DateRange: req.StartDate + "/" + req.EndDate,
	}, nil
}

// ListSites returns all Search Console properties the authenticated user can access.
func (sc *SearchConsoleService) ListSites(ctx context.Context) ([]SiteSummary, error) {
	if err := sc.client.WaitRate(ctx, "searchconsole"); err != nil {
		return nil, err
	}

	opts, err := sc.client.ClientOptions(ctx, "searchconsole")
	if err != nil {
		return nil, err
	}

	svc, err := searchconsole.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create searchconsole service: %w", err)
	}

	resp, err := svc.Sites.List().Do()
	if err != nil {
		return nil, fmt.Errorf("searchconsole list sites: %w", err)
	}

	sites := make([]SiteSummary, 0, len(resp.SiteEntry))
	for _, s := range resp.SiteEntry {
		sites = append(sites, SiteSummary{
			SiteURL:         s.SiteUrl,
			PermissionLevel: s.PermissionLevel,
		})
	}

	return sites, nil
}

// InspectURL inspects the given URL's index status in Search Console.
// NOTE: 2000 requests/day quota applies to the URL Inspection API.
func (sc *SearchConsoleService) InspectURL(ctx context.Context, siteURL, inspectionURL string) (*URLInspectionResult, error) {
	if err := sc.client.WaitRate(ctx, "searchconsole"); err != nil {
		return nil, err
	}

	opts, err := sc.client.ClientOptions(ctx, "searchconsole")
	if err != nil {
		return nil, err
	}

	svc, err := searchconsole.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create searchconsole service: %w", err)
	}

	apiReq := &searchconsole.InspectUrlIndexRequest{
		SiteUrl:       siteURL,
		InspectionUrl: inspectionURL,
	}

	resp, err := svc.UrlInspection.Index.Inspect(apiReq).Do()
	if err != nil {
		return nil, fmt.Errorf("searchconsole inspect url %s: %w", inspectionURL, err)
	}

	result := &URLInspectionResult{
		URL: inspectionURL,
	}

	if resp.InspectionResult != nil && resp.InspectionResult.IndexStatusResult != nil {
		isr := resp.InspectionResult.IndexStatusResult
		result.Verdict = isr.Verdict
		result.CoverageState = isr.CoverageState
		result.IndexingState = isr.IndexingState
		result.LastCrawlTime = isr.LastCrawlTime
		result.CrawledAs = isr.CrawledAs
		result.RobotsTxtState = isr.RobotsTxtState
	}

	return result, nil
}

// ListSitemaps returns the sitemaps submitted for the given site.
func (sc *SearchConsoleService) ListSitemaps(ctx context.Context, siteURL string) ([]SitemapInfo, error) {
	if err := sc.client.WaitRate(ctx, "searchconsole"); err != nil {
		return nil, err
	}

	opts, err := sc.client.ClientOptions(ctx, "searchconsole")
	if err != nil {
		return nil, err
	}

	svc, err := searchconsole.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create searchconsole service: %w", err)
	}

	resp, err := svc.Sitemaps.List(siteURL).Do()
	if err != nil {
		return nil, fmt.Errorf("searchconsole list sitemaps %s: %w", siteURL, err)
	}

	sitemaps := make([]SitemapInfo, 0, len(resp.Sitemap))
	for _, s := range resp.Sitemap {
		sitemaps = append(sitemaps, SitemapInfo{
			Path:           s.Path,
			Type:           s.Type,
			LastSubmitted:  s.LastSubmitted,
			LastDownloaded: s.LastDownloaded,
			IsPending:      s.IsPending,
			Warnings:       s.Warnings,
			Errors:         s.Errors,
		})
	}

	return sitemaps, nil
}

// GetIndexStatus returns an approximate index status summary for the given site.
// It uses Search Analytics data (dimension=page) and counts pages with
// impressions > 0 as an approximation for TotalIndexed.
func (sc *SearchConsoleService) GetIndexStatus(ctx context.Context, siteURL, startDate, endDate string) (*IndexStatusSummary, error) {
	if err := sc.client.WaitRate(ctx, "searchconsole"); err != nil {
		return nil, err
	}

	opts, err := sc.client.ClientOptions(ctx, "searchconsole")
	if err != nil {
		return nil, err
	}

	svc, err := searchconsole.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create searchconsole service: %w", err)
	}

	apiReq := &searchconsole.SearchAnalyticsQueryRequest{
		StartDate:  startDate,
		EndDate:    endDate,
		Dimensions: []string{"PAGE"},
		RowLimit:   25000,
	}

	resp, err := svc.Searchanalytics.Query(siteURL, apiReq).Do()
	if err != nil {
		return nil, fmt.Errorf("searchconsole get index status %s: %w", siteURL, err)
	}

	var totalIndexed int64
	for _, row := range resp.Rows {
		if row.Impressions > 0 {
			totalIndexed++
		}
	}

	return &IndexStatusSummary{
		SiteURL:       siteURL,
		TotalIndexed:  totalIndexed,
		CoverageState: fmt.Sprintf("approx %d pages with impressions (%s to %s)", totalIndexed, startDate, endDate),
	}, nil
}
