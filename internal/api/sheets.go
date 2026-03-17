package api

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/api/sheets/v4"
)

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
	if err := ss.client.WaitRate(ctx, "sheets"); err != nil {
		return nil, err
	}

	opts, err := ss.client.ClientOptions(ctx, "sheets")
	if err != nil {
		return nil, err
	}

	svc, err := sheets.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}

	resp, err := svc.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return nil, fmt.Errorf("read range: %w", err)
	}

	return &SheetData{
		Range:    resp.Range,
		Values:   resp.Values,
		RowCount: len(resp.Values),
	}, nil
}

// AppendValues appends rows to a spreadsheet range.
func (ss *SheetsService) AppendValues(ctx context.Context, spreadsheetID, appendRange string, values [][]interface{}) (*SheetAppendResult, error) {
	if err := ss.client.WaitRate(ctx, "sheets"); err != nil {
		return nil, err
	}

	opts, err := ss.client.ClientOptions(ctx, "sheets")
	if err != nil {
		return nil, err
	}

	svc, err := sheets.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}

	vr := &sheets.ValueRange{Values: values}

	resp, err := svc.Spreadsheets.Values.Append(spreadsheetID, appendRange, vr).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Do()
	if err != nil {
		return nil, fmt.Errorf("append values: %w", err)
	}

	return &SheetAppendResult{
		UpdatedRange: resp.Updates.UpdatedRange,
		UpdatedRows:  resp.Updates.UpdatedRows,
		UpdatedCells: resp.Updates.UpdatedCells,
	}, nil
}

// UpdateValues updates cells in a spreadsheet range.
func (ss *SheetsService) UpdateValues(ctx context.Context, spreadsheetID, updateRange string, values [][]interface{}) (*SheetUpdateResult, error) {
	if err := ss.client.WaitRate(ctx, "sheets"); err != nil {
		return nil, err
	}

	opts, err := ss.client.ClientOptions(ctx, "sheets")
	if err != nil {
		return nil, err
	}

	svc, err := sheets.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}

	vr := &sheets.ValueRange{Values: values}

	resp, err := svc.Spreadsheets.Values.Update(spreadsheetID, updateRange, vr).
		ValueInputOption("USER_ENTERED").
		Do()
	if err != nil {
		return nil, fmt.Errorf("update values: %w", err)
	}

	return &SheetUpdateResult{
		UpdatedRange: resp.UpdatedRange,
		UpdatedRows:  resp.UpdatedRows,
		UpdatedCells: resp.UpdatedCells,
	}, nil
}

// CreateSpreadsheet creates a new spreadsheet.
func (ss *SheetsService) CreateSpreadsheet(ctx context.Context, title string) (*SheetCreateResult, error) {
	if err := ss.client.WaitRate(ctx, "sheets"); err != nil {
		return nil, err
	}

	opts, err := ss.client.ClientOptions(ctx, "sheets")
	if err != nil {
		return nil, err
	}

	svc, err := sheets.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
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
