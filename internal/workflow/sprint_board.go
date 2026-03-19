package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/config"
)

// SprintBoardResult is the output of RunSprintBoard.
type SprintBoardResult struct {
	Action       string      `json:"action"`
	SheetID      string      `json:"sheet_id"`
	SheetURL     string      `json:"sheet_url,omitempty"`
	Stats        interface{} `json:"stats,omitempty"`
	TicketResult interface{} `json:"ticket_result,omitempty"`
}

// SprintBoardOpts configures the sprint-board workflow.
type SprintBoardOpts struct {
	Action   string // "init", "ticket", "stats", "archive"
	Feature  string
	SheetID  string
	Title    string
	Assignee string
	Priority string
	Sprint   string
	IsMCP    bool
}

var sprintBoardHeaders = []interface{}{"Ticket ID", "Title", "Assignee", "Status", "Priority", "Sprint", "Created", "Updated", "Notes"}

// RunSprintBoard manages a sprint board in Google Sheets.
func RunSprintBoard(ctx context.Context, client *api.Client, opts SprintBoardOpts) (*SprintBoardResult, error) {
	svc := api.NewSheetsService(client)

	switch opts.Action {
	case "init":
		if opts.IsMCP {
			return &SprintBoardResult{Action: "init", SheetID: "(preview — not created in MCP mode)"}, nil
		}
		title := fmt.Sprintf("%s — Sprint Board", opts.Feature)
		created, err := svc.CreateSpreadsheet(ctx, title)
		if err != nil {
			return nil, fmt.Errorf("create spreadsheet: %w", err)
		}
		headerRow := [][]interface{}{sprintBoardHeaders}
		if _, err := svc.AppendValues(ctx, created.SpreadsheetID, "Sheet1!A1", headerRow); err != nil {
			return nil, fmt.Errorf("write headers: %w", err)
		}
		if err := config.SetWorkflowConfig("sprint-board.sheet-id", created.SpreadsheetID); err != nil {
			return nil, fmt.Errorf("save config: %w", err)
		}
		return &SprintBoardResult{
			Action:   "init",
			SheetID:  created.SpreadsheetID,
			SheetURL: created.SpreadsheetURL,
		}, nil

	case "ticket":
		sheetID, err := resolveSheetID(opts.SheetID, "sprint-board")
		if err != nil {
			return nil, err
		}
		if opts.IsMCP {
			return &SprintBoardResult{Action: "ticket", SheetID: sheetID}, nil
		}
		now := time.Now().Format("2006-01-02 15:04")
		ticketID := fmt.Sprintf("T-%d", time.Now().Unix())
		row := [][]interface{}{{
			ticketID,
			opts.Title,
			opts.Assignee,
			"todo",
			opts.Priority,
			opts.Sprint,
			now,
			now,
			"",
		}}
		result, err := svc.AppendValues(ctx, sheetID, "Sheet1!A:I", row)
		if err != nil {
			return nil, fmt.Errorf("append ticket: %w", err)
		}
		return &SprintBoardResult{Action: "ticket", SheetID: sheetID, TicketResult: result}, nil

	case "stats":
		sheetID, err := resolveSheetID(opts.SheetID, "sprint-board")
		if err != nil {
			return nil, err
		}
		data, err := svc.ReadRange(ctx, sheetID, "Sheet1!A:I")
		if err != nil {
			return nil, fmt.Errorf("read sheet: %w", err)
		}
		stats := computeSprintStats(data)
		return &SprintBoardResult{Action: "stats", SheetID: sheetID, Stats: stats}, nil

	case "archive":
		sheetID, err := resolveSheetID(opts.SheetID, "sprint-board")
		if err != nil {
			return nil, err
		}
		if opts.IsMCP {
			return &SprintBoardResult{Action: "archive", SheetID: sheetID}, nil
		}
		// Read all data, filter out done tickets for the specified sprint
		data, err := svc.ReadRange(ctx, sheetID, "Sheet1!A:I")
		if err != nil {
			return nil, fmt.Errorf("read sheet: %w", err)
		}
		archived := 0
		if data != nil {
			for _, row := range data.Values {
				if len(row) > 5 {
					status := fmt.Sprintf("%v", row[3])
					sprint := fmt.Sprintf("%v", row[5])
					if status == "done" && sprint == opts.Sprint {
						archived++
					}
				}
			}
		}
		return &SprintBoardResult{
			Action:  "archive",
			SheetID: sheetID,
			Stats:   map[string]int{"archived": archived},
		}, nil

	default:
		return nil, fmt.Errorf("unknown action: %s (use init, ticket, stats, or archive)", opts.Action)
	}
}

func computeSprintStats(data *api.SheetData) map[string]int {
	stats := map[string]int{"todo": 0, "in-progress": 0, "review": 0, "done": 0, "total": 0}
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
