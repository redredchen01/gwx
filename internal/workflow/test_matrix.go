package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
)

// TestMatrixResult is the output of RunTestMatrix.
type TestMatrixResult struct {
	Action     string      `json:"action"`
	SheetID    string      `json:"sheet_id"`
	SheetURL   string      `json:"sheet_url,omitempty"`
	Stats      interface{} `json:"stats,omitempty"`
	SyncResult interface{} `json:"sync_result,omitempty"`
}

// TestMatrixOpts configures the test-matrix workflow.
type TestMatrixOpts struct {
	Action  string // "init", "sync", "stats"
	Feature string
	SheetID string
	File    string
	IsMCP   bool
}

var testMatrixHeaders = []interface{}{"Test ID", "Test Name", "Module", "Status", "Last Run", "Duration (ms)", "Notes"}

// RunTestMatrix manages test results in a Google Sheet.
func RunTestMatrix(ctx context.Context, client *api.Client, opts TestMatrixOpts) (*TestMatrixResult, error) {
	svc := api.NewSheetsService(client)

	switch opts.Action {
	case "init":
		if opts.IsMCP {
			return &TestMatrixResult{Action: "init", SheetID: "(preview — not created in MCP mode)"}, nil
		}
		title := fmt.Sprintf("%s — Test Matrix", opts.Feature)
		created, err := svc.CreateSpreadsheet(ctx, title)
		if err != nil {
			return nil, fmt.Errorf("create spreadsheet: %w", err)
		}
		// Write header row
		headerRow := [][]interface{}{testMatrixHeaders}
		if _, err := svc.AppendValues(ctx, created.SpreadsheetID, "Sheet1!A1", headerRow); err != nil {
			return nil, fmt.Errorf("write headers: %w", err)
		}
		// Save Sheet ID to config
		if err := config.SetWorkflowConfig("test-matrix.sheet-id", created.SpreadsheetID); err != nil {
			return nil, fmt.Errorf("save config: %w", err)
		}
		return &TestMatrixResult{
			Action:   "init",
			SheetID:  created.SpreadsheetID,
			SheetURL: created.SpreadsheetURL,
		}, nil

	case "sync":
		sheetID, err := resolveSheetID(opts.SheetID, "test-matrix")
		if err != nil {
			return nil, err
		}
		if opts.IsMCP {
			return &TestMatrixResult{Action: "sync", SheetID: sheetID}, nil
		}
		// Read test results from file
		data, err := os.ReadFile(opts.File)
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", opts.File, err)
		}
		var rows [][]interface{}
		if err := json.Unmarshal(data, &rows); err != nil {
			return nil, fmt.Errorf("parse test results: %w", err)
		}
		result, err := svc.AppendValues(ctx, sheetID, "Sheet1!A:G", rows)
		if err != nil {
			return nil, fmt.Errorf("append values: %w", err)
		}
		return &TestMatrixResult{Action: "sync", SheetID: sheetID, SyncResult: result}, nil

	case "stats":
		sheetID, err := resolveSheetID(opts.SheetID, "test-matrix")
		if err != nil {
			return nil, err
		}
		data, err := svc.ReadRange(ctx, sheetID, "Sheet1!A:G")
		if err != nil {
			return nil, fmt.Errorf("read sheet: %w", err)
		}
		stats := computeTestStats(data)
		return &TestMatrixResult{Action: "stats", SheetID: sheetID, Stats: stats}, nil

	default:
		return nil, fmt.Errorf("unknown action: %s (use init, sync, or stats)", opts.Action)
	}
}

func resolveSheetID(explicit, workflowName string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	id, err := config.GetWorkflowConfig(workflowName + ".sheet-id")
	if err != nil {
		return "", err
	}
	if id == "" {
		return "", fmt.Errorf("no sheet-id configured. Run 'gwx workflow %s init' first", workflowName)
	}
	return id, nil
}

func computeTestStats(data *api.SheetData) map[string]int {
	stats := map[string]int{"pass": 0, "fail": 0, "skip": 0, "pending": 0, "total": 0}
	if data == nil {
		return stats
	}
	for i, row := range data.Values {
		if i == 0 { // skip header
			continue
		}
		if len(row) > 3 {
			status := fmt.Sprintf("%v", row[3])
			stats[status]++
			stats["total"]++
		}
	}
	return stats
}
