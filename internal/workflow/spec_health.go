package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
)

// SpecHealthResult is the output of RunSpecHealth.
type SpecHealthResult struct {
	Action       string      `json:"action"`
	SheetID      string      `json:"sheet_id"`
	SheetURL     string      `json:"sheet_url,omitempty"`
	Stats        interface{} `json:"stats,omitempty"`
	RecordResult interface{} `json:"record_result,omitempty"`
}

// SpecHealthOpts configures the spec-health workflow.
type SpecHealthOpts struct {
	Action     string // "init", "record", "stats"
	Feature    string
	SheetID    string
	SpecFolder string
	IsMCP      bool
}

var specHealthHeaders = []interface{}{"Spec Folder", "Feature", "Stage", "Status", "Last Updated", "Author", "Notes"}

// RunSpecHealth tracks spec status in a Google Sheet.
func RunSpecHealth(ctx context.Context, client *api.Client, opts SpecHealthOpts) (*SpecHealthResult, error) {
	svc := api.NewSheetsService(client)

	switch opts.Action {
	case "init":
		if opts.IsMCP {
			return &SpecHealthResult{Action: "init", SheetID: "(preview — not created in MCP mode)"}, nil
		}
		title := fmt.Sprintf("%s — Spec Health", opts.Feature)
		created, err := svc.CreateSpreadsheet(ctx, title)
		if err != nil {
			return nil, fmt.Errorf("create spreadsheet: %w", err)
		}
		headerRow := [][]interface{}{specHealthHeaders}
		if _, err := svc.AppendValues(ctx, created.SpreadsheetID, "Sheet1!A1", headerRow); err != nil {
			return nil, fmt.Errorf("write headers: %w", err)
		}
		if err := config.SetWorkflowConfig("spec-health.sheet-id", created.SpreadsheetID); err != nil {
			return nil, fmt.Errorf("save config: %w", err)
		}
		return &SpecHealthResult{
			Action:   "init",
			SheetID:  created.SpreadsheetID,
			SheetURL: created.SpreadsheetURL,
		}, nil

	case "record":
		sheetID, err := resolveSheetID(opts.SheetID, "spec-health")
		if err != nil {
			return nil, err
		}
		if opts.IsMCP {
			return &SpecHealthResult{Action: "record", SheetID: sheetID}, nil
		}
		// Read sdd_context.json from spec folder
		ctxPath := filepath.Join(opts.SpecFolder, "sdd_context.json")
		data, err := os.ReadFile(ctxPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", ctxPath, err)
		}
		var sddCtx struct {
			SDDContext struct {
				Feature      string `json:"feature"`
				CurrentStage string `json:"current_stage"`
				Status       string `json:"status"`
			} `json:"sdd_context"`
		}
		if err := json.Unmarshal(data, &sddCtx); err != nil {
			return nil, fmt.Errorf("parse sdd_context.json: %w", err)
		}

		row := [][]interface{}{{
			opts.SpecFolder,
			sddCtx.SDDContext.Feature,
			sddCtx.SDDContext.CurrentStage,
			sddCtx.SDDContext.Status,
			time.Now().Format("2006-01-02 15:04"),
			"", // Author — not available from sdd_context
			"",
		}}
		result, err := svc.AppendValues(ctx, sheetID, "Sheet1!A:G", row)
		if err != nil {
			return nil, fmt.Errorf("append values: %w", err)
		}
		return &SpecHealthResult{Action: "record", SheetID: sheetID, RecordResult: result}, nil

	case "stats":
		sheetID, err := resolveSheetID(opts.SheetID, "spec-health")
		if err != nil {
			return nil, err
		}
		data, err := svc.ReadRange(ctx, sheetID, "Sheet1!A:G")
		if err != nil {
			return nil, fmt.Errorf("read sheet: %w", err)
		}
		stats := computeSpecStats(data)
		return &SpecHealthResult{Action: "stats", SheetID: sheetID, Stats: stats}, nil

	default:
		return nil, fmt.Errorf("unknown action: %s (use init, record, or stats)", opts.Action)
	}
}

func computeSpecStats(data *api.SheetData) map[string]int {
	stats := map[string]int{"in_progress": 0, "completed": 0, "cancelled": 0, "total": 0}
	if data == nil {
		return stats
	}
	for i, row := range data.Values {
		if i == 0 {
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
