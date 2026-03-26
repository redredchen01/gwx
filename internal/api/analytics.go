package api

import (
	"context"
	"fmt"

	analyticsadmin "google.golang.org/api/analyticsadmin/v1alpha"
	analyticsdata "google.golang.org/api/analyticsdata/v1beta"
)

// AnalyticsService wraps Google Analytics Data and Admin API operations.
type AnalyticsService struct {
	client *Client
}

// NewAnalyticsService creates an Analytics service wrapper.
func NewAnalyticsService(client *Client) *AnalyticsService {
	return &AnalyticsService{client: client}
}

func (as *AnalyticsService) dataService(ctx context.Context) (*analyticsdata.Service, error) {
	svc, err := as.client.GetOrCreateService("analyticsdata:v1beta", func() (any, error) {
		opts, err := as.client.ClientOptions(ctx, "analytics")
		if err != nil {
			return nil, err
		}
		return analyticsdata.NewService(ctx, opts...)
	})
	if err != nil {
		return nil, fmt.Errorf("create analyticsdata service: %w", err)
	}
	return svc.(*analyticsdata.Service), nil
}

func (as *AnalyticsService) adminService(ctx context.Context) (*analyticsadmin.Service, error) {
	svc, err := as.client.GetOrCreateService("analyticsadmin:v1alpha", func() (any, error) {
		opts, err := as.client.ClientOptions(ctx, "analytics")
		if err != nil {
			return nil, err
		}
		return analyticsadmin.NewService(ctx, opts...)
	})
	if err != nil {
		return nil, fmt.Errorf("create analyticsadmin service: %w", err)
	}
	return svc.(*analyticsadmin.Service), nil
}

// ReportRequest holds parameters for a GA4 report request.
type ReportRequest struct {
	Property   string
	StartDate  string
	EndDate    string
	Metrics    []string
	Dimensions []string
	Limit      int64
}

// ReportRow holds a single row of dimension and metric values.
type ReportRow struct {
	Dimensions map[string]string `json:"dimensions"`
	Metrics    map[string]string `json:"metrics"`
}

// ReportResult holds the result of a RunReport call.
type ReportResult struct {
	Rows      []ReportRow `json:"rows"`
	RowCount  int         `json:"row_count"`
	Property  string      `json:"property"`
	DateRange string      `json:"date_range"`
}

// RealtimeResult holds the result of a RunRealtimeReport call.
type RealtimeResult struct {
	Rows     []ReportRow `json:"rows"`
	RowCount int         `json:"row_count"`
	Property string      `json:"property"`
}

// PropertySummary is a simplified GA4 property representation.
type PropertySummary struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Industry    string `json:"industry,omitempty"`
	TimeZone    string `json:"time_zone,omitempty"`
}

// AudienceSummary is a simplified GA4 audience representation.
type AudienceSummary struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
	MemberCount int64  `json:"member_count,omitempty"`
}

// RunReport runs a standard GA4 report for the given property.
func (as *AnalyticsService) RunReport(ctx context.Context, req ReportRequest) (*ReportResult, error) {
	if err := as.client.WaitRate(ctx, "analytics"); err != nil {
		return nil, err
	}

	svc, err := as.dataService(ctx)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}

	apiReq := &analyticsdata.RunReportRequest{
		DateRanges: []*analyticsdata.DateRange{
			{StartDate: req.StartDate, EndDate: req.EndDate},
		},
		Limit: limit,
	}
	for _, m := range req.Metrics {
		apiReq.Metrics = append(apiReq.Metrics, &analyticsdata.Metric{Name: m})
	}
	for _, d := range req.Dimensions {
		apiReq.Dimensions = append(apiReq.Dimensions, &analyticsdata.Dimension{Name: d})
	}

	resp, err := svc.Properties.RunReport(req.Property, apiReq).Do()
	if err != nil {
		return nil, fmt.Errorf("RunReport: property %q: %w", req.Property, err)
	}

	// Build dimension and metric name index from response headers.
	dimNames := make([]string, len(resp.DimensionHeaders))
	for i, h := range resp.DimensionHeaders {
		dimNames[i] = h.Name
	}
	metNames := make([]string, len(resp.MetricHeaders))
	for i, h := range resp.MetricHeaders {
		metNames[i] = h.Name
	}

	rows := make([]ReportRow, 0, len(resp.Rows))
	for _, row := range resp.Rows {
		r := ReportRow{
			Dimensions: make(map[string]string, len(row.DimensionValues)),
			Metrics:    make(map[string]string, len(row.MetricValues)),
		}
		for i, dv := range row.DimensionValues {
			if i < len(dimNames) {
				r.Dimensions[dimNames[i]] = dv.Value
			}
		}
		for i, mv := range row.MetricValues {
			if i < len(metNames) {
				r.Metrics[metNames[i]] = mv.Value
			}
		}
		rows = append(rows, r)
	}

	dateRange := req.StartDate + "/" + req.EndDate
	return &ReportResult{
		Rows:      rows,
		RowCount:  int(resp.RowCount),
		Property:  req.Property,
		DateRange: dateRange,
	}, nil
}

// RunRealtimeReport runs a realtime GA4 report for the given property.
func (as *AnalyticsService) RunRealtimeReport(ctx context.Context, property string, metrics []string, dimensions []string, limit int64) (*RealtimeResult, error) {
	if err := as.client.WaitRate(ctx, "analytics"); err != nil {
		return nil, err
	}

	svc, err := as.dataService(ctx)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 100
	}

	apiReq := &analyticsdata.RunRealtimeReportRequest{
		Limit: limit,
	}
	for _, m := range metrics {
		apiReq.Metrics = append(apiReq.Metrics, &analyticsdata.Metric{Name: m})
	}
	for _, d := range dimensions {
		apiReq.Dimensions = append(apiReq.Dimensions, &analyticsdata.Dimension{Name: d})
	}

	resp, err := svc.Properties.RunRealtimeReport(property, apiReq).Do()
	if err != nil {
		return nil, fmt.Errorf("RunRealtimeReport: property %q: %w", property, err)
	}

	dimNames := make([]string, len(resp.DimensionHeaders))
	for i, h := range resp.DimensionHeaders {
		dimNames[i] = h.Name
	}
	metNames := make([]string, len(resp.MetricHeaders))
	for i, h := range resp.MetricHeaders {
		metNames[i] = h.Name
	}

	rows := make([]ReportRow, 0, len(resp.Rows))
	for _, row := range resp.Rows {
		r := ReportRow{
			Dimensions: make(map[string]string, len(row.DimensionValues)),
			Metrics:    make(map[string]string, len(row.MetricValues)),
		}
		for i, dv := range row.DimensionValues {
			if i < len(dimNames) {
				r.Dimensions[dimNames[i]] = dv.Value
			}
		}
		for i, mv := range row.MetricValues {
			if i < len(metNames) {
				r.Metrics[metNames[i]] = mv.Value
			}
		}
		rows = append(rows, r)
	}

	return &RealtimeResult{
		Rows:     rows,
		RowCount: int(resp.RowCount),
		Property: property,
	}, nil
}

// ListProperties lists all GA4 properties accessible to the caller by iterating
// over account summaries and flattening the nested account → property structure.
func (as *AnalyticsService) ListProperties(ctx context.Context) ([]PropertySummary, error) {
	if err := as.client.WaitRate(ctx, "analytics"); err != nil {
		return nil, err
	}

	svc, err := as.adminService(ctx)
	if err != nil {
		return nil, err
	}

	var properties []PropertySummary
	pageToken := ""

	for {
		call := svc.AccountSummaries.List()
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("ListProperties: list account summaries: %w", err)
		}

		// Flatten: account → property summaries.
		for _, acct := range resp.AccountSummaries {
			for _, ps := range acct.PropertySummaries {
				properties = append(properties, PropertySummary{
					Name:        ps.Property,
					DisplayName: ps.DisplayName,
				})
			}
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return properties, nil
}

// ListAudiences lists all audiences for the given GA4 property.
//
// NOTE: v1alpha — may break on API update.
func (as *AnalyticsService) ListAudiences(ctx context.Context, property string) ([]AudienceSummary, error) {
	if err := as.client.WaitRate(ctx, "analytics"); err != nil {
		return nil, err
	}

	svc, err := as.adminService(ctx)
	if err != nil {
		return nil, err
	}

	var audiences []AudienceSummary
	pageToken := ""

	for {
		call := svc.Properties.Audiences.List(property)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("ListAudiences: property %q: %w", property, err)
		}

		for _, a := range resp.Audiences {
			audiences = append(audiences, AudienceSummary{
				Name:        a.Name,
				DisplayName: a.DisplayName,
				Description: a.Description,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	return audiences, nil
}
