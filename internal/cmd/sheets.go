package cmd

import (
	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// SheetsCmd groups Sheets operations.
type SheetsCmd struct {
	Read   SheetsReadCmd   `cmd:"" help:"Read a range from a spreadsheet"`
	Info   SheetsInfoCmd   `cmd:"" help:"Get spreadsheet metadata and sheet tabs"`
	Search SheetsSearchCmd `cmd:"" help:"Search for text in a spreadsheet"`
	Filter SheetsFilterCmd `cmd:"" help:"Filter rows by column value"`
	Append SheetsAppendCmd `cmd:"" help:"Append rows to a spreadsheet"`
	Update SheetsUpdateCmd `cmd:"" help:"Update cells in a spreadsheet"`
	Clear  SheetsClearCmd  `cmd:"" help:"Clear a range"`
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

// SheetsInfoCmd gets spreadsheet metadata.
type SheetsInfoCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
}

func (c *SheetsInfoCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.info"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	info, err := sheetsSvc.GetInfo(rctx.Context, c.SpreadsheetID)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(info)
	return nil
}

// SheetsSearchCmd searches for text in a spreadsheet.
type SheetsSearchCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	Query         string `help:"Text to search for" required:"" short:"q"`
	Range         string `help:"Range to search (default: first sheet)" short:"r"`
}

func (c *SheetsSearchCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.search"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	searchRange := c.Range
	if searchRange == "" {
		// Auto-detect: get info first to find first sheet name
		sheetsSvc := api.NewSheetsService(rctx.APIClient)
		info, err := sheetsSvc.GetInfo(rctx.Context, c.SpreadsheetID)
		if err != nil {
			return handleAPIError(rctx, err)
		}
		if len(info.Sheets) > 0 {
			searchRange = info.Sheets[0].Title
		} else {
			searchRange = "Sheet1"
		}
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	result, err := sheetsSvc.SearchValues(rctx.Context, c.SpreadsheetID, searchRange, c.Query)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(result)
	return nil
}

// SheetsFilterCmd filters rows by column value.
type SheetsFilterCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	Range         string `arg:"" help:"Range (e.g. Sheet1!A:D)"`
	Column        int    `help:"Column index to filter (0-based)" required:"" short:"c"`
	Value         string `help:"Value to match" required:"" short:"v"`
}

func (c *SheetsFilterCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.filter"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	result, err := sheetsSvc.FilterRows(rctx.Context, c.SpreadsheetID, c.Range, c.Column, c.Value)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(result)
	return nil
}

// SheetsClearCmd clears a range.
type SheetsClearCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	Range         string `arg:"" help:"Range to clear (e.g. Sheet1!A2:D)"`
}

func (c *SheetsClearCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.clear"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "sheets.clear", "range": c.Range})
		return nil
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	if err := sheetsSvc.ClearRange(rctx.Context, c.SpreadsheetID, c.Range); err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"cleared": true,
		"range":   c.Range,
	})
	return nil
}
