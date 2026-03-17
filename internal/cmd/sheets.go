package cmd

import (
	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// SheetsCmd groups Sheets operations.
type SheetsCmd struct {
	Read   SheetsReadCmd   `cmd:"" help:"Read a range from a spreadsheet"`
	Append SheetsAppendCmd `cmd:"" help:"Append rows to a spreadsheet"`
	Update SheetsUpdateCmd `cmd:"" help:"Update cells in a spreadsheet"`
	Create SheetsCreateCmd `cmd:"" help:"Create a new spreadsheet"`
}

// SheetsReadCmd reads a range.
type SheetsReadCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	Range         string `arg:"" help:"Range (e.g. Sheet1!A1:C10)"`
}

func (c *SheetsReadCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.read"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "sheets.read", "id": c.SpreadsheetID, "range": c.Range})
		return nil
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	data, err := sheetsSvc.ReadRange(rctx.Context, c.SpreadsheetID, c.Range)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(data)
	return nil
}

// SheetsAppendCmd appends rows.
type SheetsAppendCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	Range         string `arg:"" help:"Range (e.g. Sheet1!A:C)"`
	Values        string `help:"JSON array of rows: [[\"a\",1],[\"b\",2]]" required:"" short:"v"`
}

func (c *SheetsAppendCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.append"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	values, err := api.ParseValuesJSON(c.Values)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "sheets.append", "rows": len(values)})
		return nil
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	result, err := sheetsSvc.AppendValues(rctx.Context, c.SpreadsheetID, c.Range, values)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"appended": true,
		"result":   result,
	})
	return nil
}

// SheetsUpdateCmd updates cells.
type SheetsUpdateCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	Range         string `arg:"" help:"Range (e.g. Sheet1!A1:C3)"`
	Values        string `help:"JSON array of rows" required:"" short:"v"`
}

func (c *SheetsUpdateCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.update"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	values, err := api.ParseValuesJSON(c.Values)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "sheets.update", "rows": len(values)})
		return nil
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	result, err := sheetsSvc.UpdateValues(rctx.Context, c.SpreadsheetID, c.Range, values)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"updated": true,
		"result":  result,
	})
	return nil
}

// SheetsCreateCmd creates a spreadsheet.
type SheetsCreateCmd struct {
	Title string `help:"Spreadsheet title" required:""`
}

func (c *SheetsCreateCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.create"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "sheets.create", "title": c.Title})
		return nil
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	result, err := sheetsSvc.CreateSpreadsheet(rctx.Context, c.Title)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"created":     true,
		"spreadsheet": result,
	})
	return nil
}
