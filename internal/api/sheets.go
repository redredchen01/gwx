package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"
)

func (ss *SheetsService) service(ctx context.Context) (*sheets.Service, error) {
	svc, err := ss.client.GetOrCreateService("sheets:v4", func() (any, error) {
		opts, err := ss.client.ClientOptions(ctx, "sheets")
		if err != nil {
			return nil, err
		}
		return sheets.NewService(ctx, opts...)
	})
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}
	return svc.(*sheets.Service), nil
}

// SheetsService wraps Sheets API operations.
type SheetsService struct {
	client *Client
}

// NewSheetsService creates a Sheets service wrapper.
func NewSheetsService(client *Client) *SheetsService {
	return &SheetsService{client: client}
}

// SheetData holds read results.
type SheetData struct {
	Range    string          `json:"range"`
	Values   [][]interface{} `json:"values"`
	RowCount int             `json:"row_count"`
}

// ReadRange reads a range from a spreadsheet.
func (ss *SheetsService) ReadRange(ctx context.Context, spreadsheetID, readRange string) (*SheetData, error) {
	if !ss.client.NoCache {
		key := cacheKey("sheets", "ReadRange", spreadsheetID, readRange)
		if cached, ok := ss.client.cache.Get(key); ok {
			return cached.(*SheetData), nil
		}
	}

	if err := ss.client.WaitRate(ctx, "sheets"); err != nil {
		return nil, err
	}

	svc, err := ss.service(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := svc.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("read range: %w", err)
	}

	result := &SheetData{
		Range:    resp.Range,
		Values:   resp.Values,
		RowCount: len(resp.Values),
	}
	if !ss.client.NoCache {
		key := cacheKey("sheets", "ReadRange", spreadsheetID, readRange)
		ss.client.cache.Set(key, result, 10*time.Minute)
	}
	return result, nil
}

// AppendValues appends rows to a spreadsheet range.
func (ss *SheetsService) AppendValues(ctx context.Context, spreadsheetID, appendRange string, values [][]interface{}) (*SheetAppendResult, error) {
	if err := ss.client.WaitRate(ctx, "sheets"); err != nil {
		return nil, err
	}

	svc, err := ss.service(ctx)
	if err != nil {
		return nil, err
	}

	vr := &sheets.ValueRange{Values: values}

	resp, err := svc.Spreadsheets.Values.Append(spreadsheetID, appendRange, vr).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Do()
	if err != nil {
		return nil, fmt.Errorf("append values: %w", err)
	}

	result := &SheetAppendResult{
		UpdatedRange: resp.Updates.UpdatedRange,
		UpdatedRows:  resp.Updates.UpdatedRows,
		UpdatedCells: resp.Updates.UpdatedCells,
	}
	if !ss.client.NoCache {
		ss.client.cache.InvalidatePrefix("sheets:")
	}
	return result, nil
}

// UpdateValues updates cells in a spreadsheet range.
func (ss *SheetsService) UpdateValues(ctx context.Context, spreadsheetID, updateRange string, values [][]interface{}) (*SheetUpdateResult, error) {
	if err := ss.client.WaitRate(ctx, "sheets"); err != nil {
		return nil, err
	}

	svc, err := ss.service(ctx)
	if err != nil {
		return nil, err
	}

	vr := &sheets.ValueRange{Values: values}

	resp, err := svc.Spreadsheets.Values.Update(spreadsheetID, updateRange, vr).
		ValueInputOption("USER_ENTERED").
		Do()
	if err != nil {
		return nil, fmt.Errorf("update values: %w", err)
	}

	result := &SheetUpdateResult{
		UpdatedRange: resp.UpdatedRange,
		UpdatedRows:  resp.UpdatedRows,
		UpdatedCells: resp.UpdatedCells,
	}
	if !ss.client.NoCache {
		ss.client.cache.InvalidatePrefix("sheets:")
	}
	return result, nil
}

// CreateSpreadsheet creates a new spreadsheet.
func (ss *SheetsService) CreateSpreadsheet(ctx context.Context, title string) (*SheetCreateResult, error) {
	if err := ss.client.WaitRate(ctx, "sheets"); err != nil {
		return nil, err
	}

	svc, err := ss.service(ctx)
	if err != nil {
		return nil, err
	}

	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{Title: title},
	}

	created, err := svc.Spreadsheets.Create(spreadsheet).Do()
	if err != nil {
		return nil, fmt.Errorf("create spreadsheet: %w", err)
	}

	return &SheetCreateResult{
		SpreadsheetID:  created.SpreadsheetId,
		Title:          created.Properties.Title,
		SpreadsheetURL: created.SpreadsheetUrl,
	}, nil
}

// SheetInfo holds spreadsheet metadata.
type SheetInfo struct {
	SpreadsheetID  string      `json:"spreadsheet_id"`
	Title          string      `json:"title"`
	SpreadsheetURL string      `json:"spreadsheet_url"`
	Sheets         []SheetTab  `json:"sheets"`
	SheetCount     int         `json:"sheet_count"`
}

// SheetTab holds individual sheet/tab metadata.
type SheetTab struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Index    int64  `json:"index"`
	RowCount int64  `json:"row_count"`
	ColCount int64  `json:"col_count"`
}

// GetInfo returns metadata about a spreadsheet including all sheet tabs.
func (ss *SheetsService) GetInfo(ctx context.Context, spreadsheetID string) (*SheetInfo, error) {
	if !ss.client.NoCache {
		key := cacheKey("sheets", "GetInfo", spreadsheetID)
		if cached, ok := ss.client.cache.Get(key); ok {
			return cached.(*SheetInfo), nil
		}
	}

	if err := ss.client.WaitRate(ctx, "sheets"); err != nil {
		return nil, err
	}

	svc, err := ss.service(ctx)
	if err != nil {
		return nil, err
	}

	spreadsheet, err := svc.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return nil, fmt.Errorf("get spreadsheet info: %w", err)
	}

	var tabs []SheetTab
	for _, s := range spreadsheet.Sheets {
		tab := SheetTab{
			ID:    s.Properties.SheetId,
			Title: s.Properties.Title,
			Index: s.Properties.Index,
		}
		if s.Properties.GridProperties != nil {
			tab.RowCount = s.Properties.GridProperties.RowCount
			tab.ColCount = s.Properties.GridProperties.ColumnCount
		}
		tabs = append(tabs, tab)
	}

	info := &SheetInfo{
		SpreadsheetID:  spreadsheet.SpreadsheetId,
		Title:          spreadsheet.Properties.Title,
		SpreadsheetURL: spreadsheet.SpreadsheetUrl,
		Sheets:         tabs,
		SheetCount:     len(tabs),
	}
	if !ss.client.NoCache {
		key := cacheKey("sheets", "GetInfo", spreadsheetID)
		ss.client.cache.Set(key, info, 10*time.Minute)
	}
	return info, nil
}

// SearchValues searches for a keyword across all cells in a range and returns matching rows.
func (ss *SheetsService) SearchValues(ctx context.Context, spreadsheetID, searchRange, query string) (*SheetSearchResult, error) {
	data, err := ss.ReadRange(ctx, spreadsheetID, searchRange)
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var matchedRows []SheetMatchedRow

	for rowIdx, row := range data.Values {
		for colIdx, cell := range row {
			cellStr := fmt.Sprintf("%v", cell)
			if strings.Contains(strings.ToLower(cellStr), queryLower) {
				matchedRows = append(matchedRows, SheetMatchedRow{
					RowIndex:   rowIdx,
					ColIndex:   colIdx,
					MatchedCell: cellStr,
					FullRow:    row,
				})
				break // one match per row is enough
			}
		}
	}

	return &SheetSearchResult{
		Query:       query,
		Range:       data.Range,
		TotalRows:   data.RowCount,
		MatchedRows: matchedRows,
		MatchCount:  len(matchedRows),
	}, nil
}

// FilterRows filters rows where a specific column matches a value.
func (ss *SheetsService) FilterRows(ctx context.Context, spreadsheetID, filterRange string, colIndex int, value string) (*SheetFilterResult, error) {
	data, err := ss.ReadRange(ctx, spreadsheetID, filterRange)
	if err != nil {
		return nil, err
	}

	valueLower := strings.ToLower(value)
	var matched [][]interface{}
	var header []interface{}

	for i, row := range data.Values {
		if i == 0 {
			header = row
			continue
		}
		if colIndex < len(row) {
			cellStr := strings.ToLower(fmt.Sprintf("%v", row[colIndex]))
			if strings.Contains(cellStr, valueLower) {
				matched = append(matched, row)
			}
		}
	}

	return &SheetFilterResult{
		Range:       data.Range,
		Header:      header,
		MatchedRows: matched,
		MatchCount:  len(matched),
		TotalRows:   data.RowCount - 1, // exclude header
	}, nil
}

// ClearRange clears all values in a range without deleting cells.
func (ss *SheetsService) ClearRange(ctx context.Context, spreadsheetID, clearRange string) error {
	if err := ss.client.WaitRate(ctx, "sheets"); err != nil {
		return err
	}

	opts, err := ss.client.ClientOptions(ctx, "sheets")
	if err != nil {
		return err
	}

	svc, err := sheets.NewService(ctx, opts...)
	if err != nil {
		return fmt.Errorf("create sheets service: %w", err)
	}

	req := &sheets.ClearValuesRequest{}
	if _, err := svc.Spreadsheets.Values.Clear(spreadsheetID, clearRange, req).Do(); err != nil {
		return fmt.Errorf("clear range: %w", err)
	}
	if !ss.client.NoCache {
		ss.client.cache.InvalidatePrefix("sheets:")
	}
	return nil
}

// SheetSearchResult holds search results.
type SheetSearchResult struct {
	Query       string            `json:"query"`
	Range       string            `json:"range"`
	TotalRows   int               `json:"total_rows"`
	MatchedRows []SheetMatchedRow `json:"matched_rows"`
	MatchCount  int               `json:"match_count"`
}

// SheetMatchedRow is a row that matched a search.
type SheetMatchedRow struct {
	RowIndex    int             `json:"row_index"`
	ColIndex    int             `json:"col_index"`
	MatchedCell string          `json:"matched_cell"`
	FullRow     []interface{}   `json:"full_row"`
}

// SheetFilterResult holds filter results.
type SheetFilterResult struct {
	Range       string          `json:"range"`
	Header      []interface{}   `json:"header"`
	MatchedRows [][]interface{} `json:"matched_rows"`
	MatchCount  int             `json:"match_count"`
	TotalRows   int             `json:"total_rows"`
}

// ParseValuesJSON parses a JSON string into [][]interface{} for Sheets API.
// Automatically sanitizes formula injection (values starting with =, +, -, @).
func ParseValuesJSON(raw string) ([][]interface{}, error) {
	var values [][]interface{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("parse values JSON: %w (expected [[\"a\",1],[\"b\",2]])", err)
	}
	return SanitizeValues(values), nil
}

// SanitizeValues escapes potential formula injection in cell values.
// Prefixes strings starting with =, +, -, @ with a single quote to prevent
// them from being interpreted as formulas by Google Sheets.
func SanitizeValues(values [][]interface{}) [][]interface{} {
	for i, row := range values {
		for j, cell := range row {
			if s, ok := cell.(string); ok && len(s) > 0 {
				switch s[0] {
				case '=', '+', '-', '@':
					values[i][j] = "'" + s
				}
			}
		}
	}
	return values
}

// SheetAppendResult holds append results.
type SheetAppendResult struct {
	UpdatedRange string `json:"updated_range"`
	UpdatedRows  int64  `json:"updated_rows"`
	UpdatedCells int64  `json:"updated_cells"`
}

// SheetUpdateResult holds update results.
type SheetUpdateResult struct {
	UpdatedRange string `json:"updated_range"`
	UpdatedRows  int64  `json:"updated_rows"`
	UpdatedCells int64  `json:"updated_cells"`
}

// SheetCreateResult holds creation results.
type SheetCreateResult struct {
	SpreadsheetID  string `json:"spreadsheet_id"`
	Title          string `json:"title"`
	SpreadsheetURL string `json:"spreadsheet_url"`
}
