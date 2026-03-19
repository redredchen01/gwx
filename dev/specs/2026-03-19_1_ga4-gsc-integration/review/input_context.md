# Code Review Input Context

review_type: code_review
scope: code
session: s5_ga4_gsc

---

## 1. Review Standards

### Code Review 審查項目

1. **架構合規**：各層職責清晰，業務邏輯在正確的層。遵循 API Service → CMD → MCP Tool 三層架構。
2. **命名慣例**：遵循 Go 命名規範，與 codebase 既有風格一致。
3. **安全性**：敏感資料不硬編碼，輸入驗證完整。
4. **測試品質**：覆蓋 happy + error path。
5. **效能**：無 N+1，大量資料有分頁。
6. **Spec 一致性**：遵循 s1_dev_spec.md 的 API 設計、任務 DoD。

### 嚴重度
- P0：安全漏洞、資料遺失、架構根本錯誤
- P1：邏輯錯誤、缺驗證、效能瓶頸、不符規範
- P2：命名風格、註解品質、可讀性

### Output Format
Finding ID: CR-{P0|P1|P2}-{NNN}
Each finding: Severity, Category, File, Line, Description, Evidence, Recommendation, Status

Decision: pass | conditional_pass | fix_required | redesign_required

使用繁體中文。

---

## 2. S1 Dev Spec（參考）

        SVC-->>CLI: circuit breaker error
        CLI-->>U: "service temporarily unavailable"
    end
```

### 5.1 API 設計

#### AnalyticsService（`internal/api/analytics.go`）

```go
package api

import (
    "context"
    analyticsdata "google.golang.org/api/analyticsdata/v1beta"
    analyticsadmin "google.golang.org/api/analyticsadmin/v1alpha"
)

type AnalyticsService struct {
    client *Client
}

func NewAnalyticsService(client *Client) *AnalyticsService {
    return &AnalyticsService{client: client}
}

// --- Data Types ---

type ReportRequest struct {
    Property   string   // e.g. "properties/123456"
    StartDate  string   // YYYY-MM-DD
    EndDate    string   // YYYY-MM-DD
    Metrics    []string // e.g. ["sessions", "activeUsers"]
    Dimensions []string // e.g. ["date", "country"]
    Limit      int64    // max rows, default 100
}

type ReportRow struct {
    Dimensions map[string]string `json:"dimensions"`
    Metrics    map[string]string `json:"metrics"`
}

type ReportResult struct {
    Rows      []ReportRow `json:"rows"`
    RowCount  int         `json:"row_count"`
    Property  string      `json:"property"`
    DateRange string      `json:"date_range"`
}

type RealtimeResult struct {
    Rows      []ReportRow `json:"rows"`
    RowCount  int         `json:"row_count"`
    Property  string      `json:"property"`
}

type PropertySummary struct {
    Name        string `json:"name"`        // "properties/123456"
    DisplayName string `json:"display_name"`
    Industry    string `json:"industry,omitempty"`
    TimeZone    string `json:"time_zone,omitempty"`
}

type AudienceSummary struct {
    Name        string `json:"name"`
    DisplayName string `json:"display_name"`
    Description string `json:"description,omitempty"`
    MemberCount int64  `json:"member_count,omitempty"`
}

// --- Methods ---

// RunReport executes a GA4 report query.
func (s *AnalyticsService) RunReport(ctx context.Context, req *ReportRequest) (*ReportResult, error)

// RunRealtimeReport gets real-time data for a property.
func (s *AnalyticsService) RunRealtimeReport(ctx context.Context, property string, metrics []string) (*RealtimeResult, error)

// ListProperties lists GA4 properties accessible to the authenticated user.
func (s *AnalyticsService) ListProperties(ctx context.Context) ([]PropertySummary, error)

// ListAudiences lists audiences for a property.
// NOTE: Uses v1alpha Admin API — may break on API update.
func (s *AnalyticsService) ListAudiences(ctx context.Context, property string) ([]AudienceSummary, error)
```

#### SearchConsoleService（`internal/api/searchconsole.go`）

```go
package api

import (
    "context"
    searchconsole "google.golang.org/api/searchconsole/v1"
)

type SearchConsoleService struct {
    client *Client
}

func NewSearchConsoleService(client *Client) *SearchConsoleService {
    return &SearchConsoleService{client: client}
}

// --- Data Types ---

type SearchQueryRequest struct {
    SiteURL    string   // e.g. "https://example.com"
    StartDate  string   // YYYY-MM-DD
    EndDate    string   // YYYY-MM-DD
    Dimensions []string // e.g. ["query", "page", "country"]
    Query      string   // filter by query text (optional)
    Limit      int      // default 100, max 25000
}

type SearchQueryRow struct {
    Keys        []string `json:"keys"`
    Clicks      float64  `json:"clicks"`
    Impressions float64  `json:"impressions"`
    CTR         float64  `json:"ctr"`
    Position    float64  `json:"position"`
}

type SearchQueryResult struct {
    Rows      []SearchQueryRow `json:"rows"`
    RowCount  int              `json:"row_count"`
    SiteURL   string           `json:"site_url"`
    DateRange string           `json:"date_range"`
}

type SiteSummary struct {
    SiteURL         string `json:"site_url"`
    PermissionLevel string `json:"permission_level"`
}

type URLInspectionResult struct {
    URL             string `json:"url"`
    Verdict         string `json:"verdict"`          // PASS, PARTIAL, FAIL, NEUTRAL
    CoverageState   string `json:"coverage_state"`
    IndexingState   string `json:"indexing_state"`
    LastCrawlTime   string `json:"last_crawl_time,omitempty"`
    CrawledAs       string `json:"crawled_as,omitempty"`
    RobotsTxtState  string `json:"robots_txt_state,omitempty"`
}

type SitemapInfo struct {
    Path          string `json:"path"`
    Type          string `json:"type"`
    LastSubmitted string `json:"last_submitted,omitempty"`
    LastDownloaded string `json:"last_downloaded,omitempty"`
    IsPending     bool   `json:"is_pending"`
    Warnings      int64  `json:"warnings"`
    Errors        int64  `json:"errors"`
}

type IndexStatusSummary struct {
    SiteURL          string `json:"site_url"`
    TotalIndexed     int64  `json:"total_indexed,omitempty"`
    TotalSubmitted   int64  `json:"total_submitted,omitempty"`
    CoverageState    string `json:"coverage_state,omitempty"`
}

// --- Methods ---

// Query runs a Search Analytics query.
func (s *SearchConsoleService) Query(ctx context.Context, req *SearchQueryRequest) (*SearchQueryResult, error)

// ListSites lists all sites the user has access to.
func (s *SearchConsoleService) ListSites(ctx context.Context) ([]SiteSummary, error)

// InspectURL inspects a URL's index status.
// NOTE: GSC URL Inspection API has a 2000 requests/day quota.
func (s *SearchConsoleService) InspectURL(ctx context.Context, siteURL, inspectURL string) (*URLInspectionResult, error)

// ListSitemaps lists sitemaps for a site.
func (s *SearchConsoleService) ListSitemaps(ctx context.Context, siteURL string) ([]SitemapInfo, error)

// GetIndexStatus gets index coverage status for a site.
// NOTE: This uses Search Analytics data to approximate index status.
func (s *SearchConsoleService) GetIndexStatus(ctx context.Context, siteURL string) (*IndexStatusSummary, error)
```

#### Preferences（`internal/config/preferences.go`）

```go
package config

// Load reads preferences from `config.Dir()`/preferences.json.
// Returns empty map if file doesn't exist.
func Load() (map[string]string, error)

// Save writes preferences to `config.Dir()`/preferences.json.
func Save(prefs map[string]string) error

// Get reads a single preference key.
func Get(key string) (string, error)

// Set writes a single preference key (read-modify-write).
func Set(key, value string) error

// Delete removes a single preference key.
func Delete(key string) error
```

---

## 3. Code Changes (git diff + new files)

### Modified Files (git diff)
```diff
diff --git a/internal/api/ratelimiter.go b/internal/api/ratelimiter.go
index 0382f0a..b91e90e 100644
--- a/internal/api/ratelimiter.go
+++ b/internal/api/ratelimiter.go
@@ -18,7 +18,9 @@ var defaultRates = map[string]rate.Limit{
 	"docs":     rate.Every(500 * time.Millisecond),  // 2 QPS
 	"tasks":    rate.Every(250 * time.Millisecond),  // 4 QPS
 	"people":   rate.Every(250 * time.Millisecond),  // 4 QPS
-	"chat":     rate.Every(250 * time.Millisecond),  // 4 QPS
+	"chat":          rate.Every(250 * time.Millisecond),  // 4 QPS
+	"analytics":     rate.Every(500 * time.Millisecond),  // 2 QPS (GA4 quota: 10 concurrent)
+	"searchconsole": rate.Every(500 * time.Millisecond),  // 2 QPS (GSC quota: ~5 QPS)
 }
 
 // ServiceRateLimiter manages per-service token bucket rate limiters.
diff --git a/internal/auth/scopes.go b/internal/auth/scopes.go
index 01c0f5e..bea3d37 100644
--- a/internal/auth/scopes.go
+++ b/internal/auth/scopes.go
@@ -32,6 +32,12 @@ var ServiceScopes = map[string][]string{
 		"https://www.googleapis.com/auth/chat.messages",
 		"https://www.googleapis.com/auth/chat.spaces.readonly",
 	},
+	"analytics": {
+		"https://www.googleapis.com/auth/analytics.readonly",
+	},
+	"searchconsole": {
+		"https://www.googleapis.com/auth/webmasters.readonly",
+	},
 }
 
 // ReadOnlyScopes returns read-only scopes for services that support it.
@@ -43,7 +49,9 @@ var ReadOnlyScopes = map[string][]string{
 	"sheets":   {"https://www.googleapis.com/auth/spreadsheets.readonly"},
 	"tasks":    {"https://www.googleapis.com/auth/tasks.readonly"},
 	"people":   {"https://www.googleapis.com/auth/contacts.readonly"},
-	"chat":     {"https://www.googleapis.com/auth/chat.spaces.readonly"},
+	"chat":          {"https://www.googleapis.com/auth/chat.spaces.readonly"},
+	"analytics":     {"https://www.googleapis.com/auth/analytics.readonly"},
+	"searchconsole": {"https://www.googleapis.com/auth/webmasters.readonly"},
 }
 
 // AllScopes returns the union of all scopes for the given services.
diff --git a/internal/cmd/auth.go b/internal/cmd/auth.go
index 6bc4d03..1e3482e 100644
--- a/internal/cmd/auth.go
+++ b/internal/cmd/auth.go
@@ -19,7 +19,7 @@ type AuthCmd struct {
 type AuthLoginCmd struct {
 	CredentialsFile string   `help:"Path to OAuth credentials JSON" name:"credentials" short:"c"`
 	Manual          bool     `help:"Use manual (headless) auth flow" name:"manual"`
-	Services        []string `help:"Services to authorize" default:"gmail,calendar,drive,docs,sheets,tasks,people,chat"`
+	Services        []string `help:"Services to authorize" default:"gmail,calendar,drive,docs,sheets,tasks,people,chat,analytics,searchconsole"`
 }
 
 func (c *AuthLoginCmd) Run(rctx *RunContext) error {
diff --git a/internal/cmd/mcpserver.go b/internal/cmd/mcpserver.go
index 427085e..08d9a31 100644
--- a/internal/cmd/mcpserver.go
+++ b/internal/cmd/mcpserver.go
@@ -18,7 +18,7 @@ func (c *MCPServerCmd) Run(rctx *RunContext) error {
 	slog.SetDefault(logger)
 
 	// MCP server needs auth — load token silently
-	if err := EnsureAuth(rctx, []string{"gmail", "calendar", "drive", "docs", "sheets", "tasks", "people", "chat"}); err != nil {
+	if err := EnsureAuth(rctx, []string{"gmail", "calendar", "drive", "docs", "sheets", "tasks", "people", "chat", "analytics", "searchconsole"}); err != nil {
 		// Try loading with whatever scopes are available
 		if token := os.Getenv("GWX_ACCESS_TOKEN"); token != "" {
 			ts := auth.TokenFromDirect(token)
diff --git a/internal/cmd/onboard.go b/internal/cmd/onboard.go
index 3bb0248..e222600 100644
--- a/internal/cmd/onboard.go
+++ b/internal/cmd/onboard.go
@@ -48,7 +48,7 @@ func (c *OnboardCmd) Run(rctx *RunContext) error {
 	fmt.Fprintln(os.Stderr, "")
 	fmt.Fprintln(os.Stderr, "Step 2/3: Select Services")
 	fmt.Fprintln(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━")
-	allServices := "gmail, calendar, drive, docs, sheets, tasks, people, chat"
+	allServices := "gmail, calendar, drive, docs, sheets, tasks, people, chat, analytics, searchconsole"
 	fmt.Fprintln(os.Stderr, "  Available: "+allServices)
 	fmt.Fprintln(os.Stderr, "  Default:   ALL (recommended)")
 	fmt.Fprintln(os.Stderr, "")
@@ -59,7 +59,7 @@ func (c *OnboardCmd) Run(rctx *RunContext) error {
 
 	var services []string
 	if svcInput == "" {
-		services = []string{"gmail", "calendar", "drive", "docs", "sheets", "tasks", "people", "chat"}
+		services = []string{"gmail", "calendar", "drive", "docs", "sheets", "tasks", "people", "chat", "analytics", "searchconsole"}
 	} else {
 		for _, s := range strings.Split(svcInput, ",") {
 			s = strings.TrimSpace(s)
diff --git a/internal/cmd/root.go b/internal/cmd/root.go
index 18b924d..4710a14 100644
--- a/internal/cmd/root.go
+++ b/internal/cmd/root.go
@@ -44,7 +44,10 @@ type CLI struct {
 	Sheets   SheetsCmd   `cmd:"" help:"Google Sheets operations"`
 	Tasks    TasksCmd    `cmd:"" help:"Google Tasks operations"`
 	Contacts ContactsCmd `cmd:"" help:"Contacts operations"`
-	Chat     ChatCmd     `cmd:"" help:"Google Chat operations"`
+	Chat          ChatCmd          `cmd:"" help:"Google Chat operations"`
+	Analytics     AnalyticsCmd     `cmd:"" help:"Google Analytics 4 operations"`
+	SearchConsole SearchConsoleCmd `cmd:"searchconsole" help:"Google Search Console operations"`
+	Config        ConfigCmd        `cmd:"" help:"Configuration management"`
 	Agent     AgentCmd     `cmd:"" help:"Agent automation helpers"`
 	Schema    SchemaCmd    `cmd:"" help:"Print full command schema (for agent introspection)"`
 	MCPServer MCPServerCmd `cmd:"mcp-server" help:"Start MCP server (stdio) for Claude integration"`
diff --git a/internal/mcp/tools.go b/internal/mcp/tools.go
index 27efe1b..fa17725 100644
--- a/internal/mcp/tools.go
+++ b/internal/mcp/tools.go
@@ -307,6 +307,12 @@ func (h *GWXHandler) ListTools() []Tool {
 	tools = append(tools, NewTools()...)
 	// Append batch tools (v0.8.0)
 	tools = append(tools, BatchTools()...)
+	// Append analytics tools (v0.8.0)
+	tools = append(tools, AnalyticsTools()...)
+	// Append Search Console tools (v0.8.0)
+	tools = append(tools, SearchConsoleTools()...)
+	// Append config tools (v0.8.0)
+	tools = append(tools, ConfigTools()...)
 	return tools
 }
 
@@ -372,6 +378,18 @@ func (h *GWXHandler) CallTool(name string, args map[string]interface{}) (*ToolRe
 		if result, err, handled := h.CallBatchTool(ctx, name, args); handled {
 			return result, err
 		}
+		// Try analytics tools (v0.8.0)
+		if result, err, handled := h.CallAnalyticsTool(ctx, name, args); handled {
+			return result, err
+		}
+		// Try Search Console tools (v0.8.0)
+		if result, err, handled := h.CallSearchConsoleTool(ctx, name, args); handled {
+			return result, err
+		}
+		// Try config tools (v0.8.0)
+		if result, err, handled := h.CallConfigTool(ctx, name, args); handled {
+			return result, err
+		}
 		return nil, fmt.Errorf("unknown tool: %s", name)
 	}
 }
diff --git a/internal/mcp/tools_test.go b/internal/mcp/tools_test.go
index b2d4e51..8ac77ad 100644
--- a/internal/mcp/tools_test.go
+++ b/internal/mcp/tools_test.go
@@ -9,8 +9,8 @@ func TestListTools_Count(t *testing.T) {
 	tools := h.ListTools()
 
 	// Verify total tool count matches actual registration
-	if len(tools) != 59 {
-		t.Errorf("expected 59 tools, got %d", len(tools))
+	if len(tools) != 71 {
+		t.Errorf("expected 71 tools, got %d", len(tools))
 	}
 }
 
```

### New File: internal/config/preferences.go
```go
package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

const preferencesFile = "preferences.json"

// Load reads preferences from config.Dir()/preferences.json.
// Returns empty map if file doesn't exist or is malformed.
func Load() (map[string]string, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	return loadFrom(filepath.Join(dir, preferencesFile))
}

// loadFrom reads preferences from an explicit path (used internally for testing).
func loadFrom(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	var prefs map[string]string
	if err := json.Unmarshal(data, &prefs); err != nil {
		slog.Warn("preferences.json is malformed, returning empty map", "error", err)
		return make(map[string]string), nil
	}
	return prefs, nil
}

// Save writes preferences to config.Dir()/preferences.json.
func Save(prefs map[string]string) error {
	dir, err := EnsureDir()
	if err != nil {
		return err
	}
	return saveTo(filepath.Join(dir, preferencesFile), prefs)
}

// saveTo writes preferences to an explicit path (used internally for testing).
func saveTo(path string, prefs map[string]string) error {
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Get reads a single preference key.
func Get(key string) (string, error) {
	prefs, err := Load()
	if err != nil {
		return "", err
	}
	return prefs[key], nil
}

// Set writes a single preference key (read-modify-write).
func Set(key, value string) error {
	prefs, err := Load()
	if err != nil {
		return err
	}
	prefs[key] = value
	return Save(prefs)
}

// Delete removes a single preference key.
func Delete(key string) error {
	prefs, err := Load()
	if err != nil {
		return err
	}
	delete(prefs, key)
	return Save(prefs)
}
```

### New File: internal/api/analytics.go
```go
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
	if err := as.client.WaitRate(ctx, "analyticsdata"); err != nil {
		return nil, err
	}

	opts, err := as.client.ClientOptions(ctx, "analyticsdata")
	if err != nil {
		return nil, err
	}

	svc, err := analyticsdata.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("RunReport: create analyticsdata service: %w", err)
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
	if err := as.client.WaitRate(ctx, "analyticsdata"); err != nil {
		return nil, err
	}

	opts, err := as.client.ClientOptions(ctx, "analyticsdata")
	if err != nil {
		return nil, err
	}

	svc, err := analyticsdata.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("RunRealtimeReport: create analyticsdata service: %w", err)
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
	if err := as.client.WaitRate(ctx, "analyticsadmin"); err != nil {
		return nil, err
	}

	opts, err := as.client.ClientOptions(ctx, "analyticsadmin")
	if err != nil {
		return nil, err
	}

	svc, err := analyticsadmin.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("ListProperties: create analyticsadmin service: %w", err)
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
	if err := as.client.WaitRate(ctx, "analyticsadmin"); err != nil {
		return nil, err
	}

	opts, err := as.client.ClientOptions(ctx, "analyticsadmin")
	if err != nil {
		return nil, err
	}

	svc, err := analyticsadmin.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("ListAudiences: property %q: create analyticsadmin service: %w", property, err)
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
```

### New File: internal/api/searchconsole.go
```go
package api

import (
	"context"
	"fmt"

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

	apiReq := &searchconsole.SearchAnalyticsQueryRequest{
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
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
```

### New File: internal/cmd/analytics.go
```go
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
	if err := CheckAllowlist(rctx, "analytics.report"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"analytics"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
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

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":    "analytics.report",
			"property":   property,
			"metrics":    c.Metrics,
			"dimensions": c.Dimensions,
			"start_date": c.StartDate,
			"end_date":   c.EndDate,
			"limit":      c.Limit,
		})
		return nil
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
	if err := CheckAllowlist(rctx, "analytics.realtime"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"analytics"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
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

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":    "analytics.realtime",
			"property":   property,
			"metrics":    c.Metrics,
			"dimensions": c.Dimensions,
			"limit":      c.Limit,
		})
		return nil
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
	if err := CheckAllowlist(rctx, "analytics.properties"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"analytics"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "analytics.properties"})
		return nil
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
	if err := CheckAllowlist(rctx, "analytics.audiences"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"analytics"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
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

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":  "analytics.audiences",
			"property": property,
		})
		return nil
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
```

### New File: internal/cmd/config.go
```go
package cmd

import (
	"fmt"
	"sort"

	"github.com/redredchen01/gwx/internal/config"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// ConfigCmd groups configuration management operations.
type ConfigCmd struct {
	Set  ConfigSetCmd  `cmd:"" help:"Set a configuration value"`
	Get  ConfigGetCmd  `cmd:"" help:"Get a configuration value"`
	List ConfigListCmd `cmd:"" help:"List all configuration values"`
}

// ConfigSetCmd sets a single configuration key.
type ConfigSetCmd struct {
	Key   string `arg:"" help:"Configuration key (e.g. analytics.default-property)"`
	Value string `arg:"" help:"Configuration value"`
}

func (c *ConfigSetCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "config.set"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{
			"dry_run": "config.set",
			"key":     c.Key,
			"value":   c.Value,
		})
		return nil
	}
	if err := config.Set(c.Key, c.Value); err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("set config: %s", err))
	}
	rctx.Printer.Success(map[string]string{"set": c.Key, "value": c.Value})
	return nil
}

// ConfigGetCmd retrieves a single configuration key.
type ConfigGetCmd struct {
	Key string `arg:"" help:"Configuration key to retrieve"`
}

func (c *ConfigGetCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "config.get"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "config.get", "key": c.Key})
		return nil
	}
	val, err := config.Get(c.Key)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("get config: %s", err))
	}
	rctx.Printer.Success(map[string]string{"key": c.Key, "value": val})
	return nil
}

// ConfigListCmd lists all configuration key-value pairs.
type ConfigListCmd struct{}

func (c *ConfigListCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "config.list"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "config.list"})
		return nil
	}
	prefs, err := config.Load()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("load config: %s", err))
	}

	// Stable output: sort keys.
	keys := make([]string, 0, len(prefs))
	for k := range prefs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	items := make([]map[string]string, 0, len(prefs))
	for _, k := range keys {
		items = append(items, map[string]string{"key": k, "value": prefs[k]})
	}
	rctx.Printer.Success(map[string]interface{}{"preferences": items, "count": len(items)})
	return nil
}
```

### New File: internal/cmd/searchconsole.go
```go
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
	if err := CheckAllowlist(rctx, "searchconsole.query"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := EnsureAuth(rctx, []string{"searchconsole"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
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

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":    "searchconsole.query would execute",
			"site":       site,
			"start_date": c.StartDate,
			"end_date":   c.EndDate,
			"dimensions": c.Dimensions,
			"limit":      c.Limit,
		})
		return nil
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
	if err := CheckAllowlist(rctx, "searchconsole.sites"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := EnsureAuth(rctx, []string{"searchconsole"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "searchconsole.sites would execute"})
		return nil
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
	if err := CheckAllowlist(rctx, "searchconsole.inspect"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := EnsureAuth(rctx, []string{"searchconsole"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": "searchconsole.inspect would execute",
			"site":    c.Site,
			"url":     c.URL,
		})
		return nil
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
	if err := CheckAllowlist(rctx, "searchconsole.sitemaps"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := EnsureAuth(rctx, []string{"searchconsole"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
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

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": "searchconsole.sitemaps would execute",
			"site":    site,
		})
		return nil
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
	if err := CheckAllowlist(rctx, "searchconsole.index-status"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := EnsureAuth(rctx, []string{"searchconsole"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
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

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":    "searchconsole.index-status would execute",
			"site":       site,
			"start_date": startDate,
			"end_date":   endDate,
		})
		return nil
	}

	scSvc := api.NewSearchConsoleService(rctx.APIClient)

	summary, err := scSvc.GetIndexStatus(rctx.Context, site, startDate, endDate)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(summary)
	return nil
}
```

### New File: internal/mcp/tools_analytics.go
```go
package mcp

import (
	"context"
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
)

// AnalyticsTools returns the 4 Google Analytics 4 tool definitions.
func AnalyticsTools() []Tool {
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

// CallAnalyticsTool routes a tool call to the appropriate analytics handler.
// Returns (result, error, handled). handled=true means the tool name was recognized.
func (h *GWXHandler) CallAnalyticsTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error, bool) {
	switch name {
	case "analytics_report":
		r, err := h.analyticsReport(ctx, args)
		return r, err, true
	case "analytics_realtime":
		r, err := h.analyticsRealtime(ctx, args)
		return r, err, true
	case "analytics_properties":
		r, err := h.analyticsProperties(ctx, args)
		return r, err, true
	case "analytics_audiences":
		r, err := h.analyticsAudiences(ctx, args)
		return r, err, true
	default:
		return nil, nil, false
	}
}

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
```

### New File: internal/mcp/tools_searchconsole.go
```go
package mcp

import (
	"context"
	"strings"
	"time"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
)

// SearchConsoleTools returns MCP tool definitions for Google Search Console.
func SearchConsoleTools() []Tool {
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

// CallSearchConsoleTool routes a tool call to the appropriate Search Console handler.
// Returns (result, error, handled). handled=false means the tool name was not recognised.
func (h *GWXHandler) CallSearchConsoleTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error, bool) {
	switch name {
	case "searchconsole_query":
		r, err := h.searchconsoleQuery(ctx, args)
		return r, err, true
	case "searchconsole_sites":
		r, err := h.searchconsoleSites(ctx, args)
		return r, err, true
	case "searchconsole_inspect":
		r, err := h.searchconsoleInspect(ctx, args)
		return r, err, true
	case "searchconsole_sitemaps":
		r, err := h.searchconsoleSitemaps(ctx, args)
		return r, err, true
	case "searchconsole_index_status":
		r, err := h.searchconsoleIndexStatus(ctx, args)
		return r, err, true
	default:
		return nil, nil, false
	}
}

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
```

### New File: internal/mcp/tools_config.go
```go
package mcp

import (
	"context"
	"fmt"

	"github.com/redredchen01/gwx/internal/config"
)

// ConfigTools returns the 3 config management tool definitions.
func ConfigTools() []Tool {
	return []Tool{
		{
			Name:        "config_set",
			Description: "Set a configuration preference key-value pair. Persists to local preferences file.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"key":   {Type: "string", Description: "Preference key (e.g. analytics.default-property)"},
					"value": {Type: "string", Description: "Value to store"},
				},
				Required: []string{"key", "value"},
			},
		},
		{
			Name:        "config_get",
			Description: "Get a single configuration preference value by key.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"key": {Type: "string", Description: "Preference key to retrieve"},
				},
				Required: []string{"key"},
			},
		},
		{
			Name:        "config_list",
			Description: "List all configuration preferences as a key-value map.",
			InputSchema: InputSchema{
				Type: "object",
			},
		},
	}
}

// CallConfigTool routes a tool call to the appropriate config handler.
// Returns (result, error, handled). handled=true means the tool name was recognized.
func (h *GWXHandler) CallConfigTool(_ context.Context, name string, args map[string]interface{}) (*ToolResult, error, bool) {
	switch name {
	case "config_set":
		r, err := h.configSet(args)
		return r, err, true
	case "config_get":
		r, err := h.configGet(args)
		return r, err, true
	case "config_list":
		r, err := h.configList()
		return r, err, true
	default:
		return nil, nil, false
	}
}

// --- Config handlers ---

func (h *GWXHandler) configSet(args map[string]interface{}) (*ToolResult, error) {
	key := strArg(args, "key")
	if key == "" {
		return nil, fmt.Errorf("config_set: key is required")
	}
	value := strArg(args, "value")
	if err := config.Set(key, value); err != nil {
		return nil, fmt.Errorf("config_set: %w", err)
	}
	return jsonResult(map[string]interface{}{"set": true, "key": key, "value": value})
}

func (h *GWXHandler) configGet(args map[string]interface{}) (*ToolResult, error) {
	key := strArg(args, "key")
	if key == "" {
		return nil, fmt.Errorf("config_get: key is required")
	}
	value, err := config.Get(key)
	if err != nil {
		return nil, fmt.Errorf("config_get: %w", err)
	}
	return jsonResult(map[string]interface{}{"key": key, "value": value})
}

func (h *GWXHandler) configList() (*ToolResult, error) {
	prefs, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config_list: %w", err)
	}
	return jsonResult(map[string]interface{}{"preferences": prefs, "count": len(prefs)})
}
```

### New File: internal/config/preferences_test.go
```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// prefPath returns path to preferences.json inside a temp dir.
func prefPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), preferencesFile)
}

func TestPreferencesLoad_FileNotExist(t *testing.T) {
	path := prefPath(t)
	prefs, err := loadFrom(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(prefs) != 0 {
		t.Fatalf("expected empty map, got: %v", prefs)
	}
}

func TestPreferencesLoad_MalformedJSON(t *testing.T) {
	path := prefPath(t)
	if err := os.WriteFile(path, []byte("not json {{{{"), 0600); err != nil {
		t.Fatal(err)
	}
	prefs, err := loadFrom(path)
	if err != nil {
		t.Fatalf("expected no error for malformed JSON, got: %v", err)
	}
	if len(prefs) != 0 {
		t.Fatalf("expected empty map for malformed JSON, got: %v", prefs)
	}
}

func TestPreferencesSetGet(t *testing.T) {
	path := prefPath(t)

	// Set a value via saveTo/loadFrom directly.
	prefs := map[string]string{}
	prefs["analytics.default-property"] = "UA-12345"
	if err := saveTo(path, prefs); err != nil {
		t.Fatalf("saveTo: %v", err)
	}

	got, err := loadFrom(path)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}
	if got["analytics.default-property"] != "UA-12345" {
		t.Fatalf("expected UA-12345, got %q", got["analytics.default-property"])
	}
}

func TestPreferencesDelete(t *testing.T) {
	path := prefPath(t)

	// Write two keys.
	initial := map[string]string{"key1": "val1", "key2": "val2"}
	if err := saveTo(path, initial); err != nil {
		t.Fatal(err)
	}

	// Delete key1.
	prefs, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	delete(prefs, "key1")
	if err := saveTo(path, prefs); err != nil {
		t.Fatal(err)
	}

	// Reload and verify.
	got, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := got["key1"]; ok {
		t.Fatalf("key1 should be deleted, got %q", v)
	}
	if got["key2"] != "val2" {
		t.Fatalf("key2 should remain, got %q", got["key2"])
	}
}

func TestPreferencesOverwrite(t *testing.T) {
	path := prefPath(t)

	// First write.
	if err := saveTo(path, map[string]string{"foo": "bar"}); err != nil {
		t.Fatal(err)
	}

	// Second write with same key, different value.
	prefs, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	prefs["foo"] = "baz"
	if err := saveTo(path, prefs); err != nil {
		t.Fatal(err)
	}

	// Reload and verify second value wins.
	got, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["foo"] != "baz" {
		t.Fatalf("expected baz, got %q", got["foo"])
	}
}

// TestPreferencesFilePermissions verifies the file is written with 0600.
func TestPreferencesFilePermissions(t *testing.T) {
	path := prefPath(t)
	if err := saveTo(path, map[string]string{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600, got %o", info.Mode().Perm())
	}
}

// TestPreferencesValidJSON verifies the written file is valid JSON.
func TestPreferencesValidJSON(t *testing.T) {
	path := prefPath(t)
	data := map[string]string{"alpha": "1", "beta": "2"}
	if err := saveTo(path, data); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("file is not valid JSON: %v", err)
	}
	if out["alpha"] != "1" || out["beta"] != "2" {
		t.Fatalf("unexpected content: %v", out)
	}
}
```

