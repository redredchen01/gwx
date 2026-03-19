package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
)

// BatchTools returns the batch tool definitions for drive_batch_upload and sheets_batch_append.
func BatchTools() []Tool {
	return []Tool{
		{
			Name:        "drive_batch_upload",
			Description: "Upload multiple local files to Google Drive in parallel. CAUTION: Uploads real files.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"paths":       {Type: "string", Description: "Local file paths, comma-separated"},
					"folder":      {Type: "string", Description: "Target folder ID (optional)"},
					"concurrency": {Type: "integer", Description: "Parallel upload count (default 3, max 5)"},
				},
				Required: []string{"paths"},
			},
		},
		{
			Name:        "sheets_batch_append",
			Description: "Append values to multiple ranges of the same spreadsheet in parallel. CAUTION: Modifies data.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"entries":        {Type: "string", Description: `JSON array of range+values pairs: [{"range":"Sheet1!A:C","values":[["a",1]]}]`},
					"concurrency":    {Type: "integer", Description: "Parallel append count (default 3, max 5)"},
				},
				Required: []string{"spreadsheet_id", "entries"},
			},
		},
	}
}

// CallBatchTool routes batch tool calls.
// Returns (result, error, handled). handled=true means the tool name was recognised.
func (h *GWXHandler) CallBatchTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error, bool) {
	switch name {
	case "drive_batch_upload":
		result, err := h.driveBatchUpload(ctx, args)
		return result, err, true
	case "sheets_batch_append":
		result, err := h.sheetsBatchAppend(ctx, args)
		return result, err, true
	default:
		return nil, nil, false
	}
}

// --- Batch handlers ---

func (h *GWXHandler) driveBatchUpload(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	paths := splitArg(args, "paths")
	if len(paths) == 0 {
		return nil, fmt.Errorf("paths is required and must not be empty")
	}
	folder := strArg(args, "folder")
	concurrency := intArg(args, "concurrency", 0)
	if concurrency == 0 {
		concurrency = 3
	}

	svc := api.NewDriveService(h.client)
	result, err := svc.BatchUploadFiles(ctx, paths, folder, concurrency)
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (h *GWXHandler) sheetsBatchAppend(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	spreadsheetID := strArg(args, "spreadsheet_id")
	if spreadsheetID == "" {
		return nil, fmt.Errorf("spreadsheet_id is required")
	}
	entriesJSON := strArg(args, "entries")
	if entriesJSON == "" {
		return nil, fmt.Errorf("entries is required")
	}

	var entries []api.BatchAppendEntry
	if err := json.Unmarshal([]byte(entriesJSON), &entries); err != nil {
		return nil, fmt.Errorf("invalid entries JSON: %w", err)
	}

	concurrency := intArg(args, "concurrency", 0)
	if concurrency == 0 {
		concurrency = 3
	}

	svc := api.NewSheetsService(h.client)
	result, err := svc.BatchAppendValues(ctx, spreadsheetID, entries, concurrency)
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (h *GWXHandler) registerBatchTools(r map[string]ToolHandler) {
	r["drive_batch_upload"] = h.driveBatchUpload
	r["sheets_batch_append"] = h.sheetsBatchAppend
}
