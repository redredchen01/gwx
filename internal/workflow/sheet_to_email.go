package workflow

import (
	"context"
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
)

const sheetToEmailLimit = 50

// SheetToEmailResult is the output of RunSheetToEmail.
type SheetToEmailResult struct {
	SheetID  string         `json:"sheet_id"`
	RowCount int            `json:"row_count"`
	Limit    int            `json:"limit"`
	Preview  []string       `json:"preview"`
	Execute  *ExecuteResult `json:"execute,omitempty"`
}

// SheetToEmailOpts configures the sheet-to-email workflow.
type SheetToEmailOpts struct {
	SheetID    string
	Range      string
	EmailCol   int
	SubjectCol int
	BodyCol    int
	Execute    bool
	NoInput    bool
	IsMCP      bool
}

// RunSheetToEmail reads a Sheet and sends personalized emails.
func RunSheetToEmail(ctx context.Context, client *api.Client, opts SheetToEmailOpts) (*SheetToEmailResult, error) {
	sheetsSvc := api.NewSheetsService(client)
	data, err := sheetsSvc.ReadRange(ctx, opts.SheetID, opts.Range)
	if err != nil {
		return nil, fmt.Errorf("read sheet: %w", err)
	}

	if data == nil || len(data.Values) <= 1 {
		return &SheetToEmailResult{SheetID: opts.SheetID, RowCount: 0, Limit: sheetToEmailLimit}, nil
	}

	// Skip header row
	rows := data.Values[1:]
	rowCount := len(rows)

	// Hard limit check — reject even preview if over limit
	if rowCount > sheetToEmailLimit {
		return nil, fmt.Errorf("row count %d exceeds limit %d", rowCount, sheetToEmailLimit)
	}

	// Build preview (first 3)
	var preview []string
	for i, row := range rows {
		if i >= 3 {
			break
		}
		email := safeCol(row, opts.EmailCol)
		subject := safeCol(row, opts.SubjectCol)
		preview = append(preview, fmt.Sprintf("To: %s | Subject: %s", email, subject))
	}

	result := &SheetToEmailResult{
		SheetID:  opts.SheetID,
		RowCount: rowCount,
		Limit:    sheetToEmailLimit,
		Preview:  preview,
	}

	if opts.Execute {
		var actions []Action
		for _, row := range rows {
			email := safeCol(row, opts.EmailCol)
			subject := safeCol(row, opts.SubjectCol)
			body := safeCol(row, opts.BodyCol)
			if email == "" {
				continue
			}
			// Capture for closure
			e, s, b := email, subject, body
			actions = append(actions, Action{
				Name:        "send_email",
				Description: fmt.Sprintf("Send to %s: %s", e, s),
				Fn: func(ctx context.Context) (interface{}, error) {
					svc := api.NewGmailService(client)
					return svc.SendMessage(ctx, &api.SendInput{
						To:      []string{e},
						Subject: s,
						Body:    b,
					})
				},
			})
		}

		execResult, err := Dispatch(ctx, actions, ExecuteOpts{
			Execute: opts.Execute,
			NoInput: opts.NoInput,
			IsMCP:   opts.IsMCP,
		})
		if err != nil {
			return nil, err
		}
		result.Execute = execResult
	}

	return result, nil
}

func safeCol(row []interface{}, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return fmt.Sprintf("%v", row[idx])
}
