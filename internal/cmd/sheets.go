package cmd

import (
	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// SheetsCmd groups Sheets operations.
type SheetsCmd struct {
	Read        SheetsReadCmd        `cmd:"" help:"Read a range from a spreadsheet"`
	Info        SheetsInfoCmd        `cmd:"" help:"Get spreadsheet metadata and sheet tabs"`
	Describe    SheetsDescribeCmd    `cmd:"" help:"Analyze column structure and fill rules"`
	Stats       SheetsStatsCmd       `cmd:"" help:"Column statistics and value counts"`
	Search      SheetsSearchCmd      `cmd:"" help:"Search for text in a spreadsheet"`
	Filter      SheetsFilterCmd      `cmd:"" help:"Filter rows by column value"`
	Diff        SheetsDiffCmd        `cmd:"" help:"Compare two tabs/ranges"`
	Append      SheetsAppendCmd      `cmd:"" help:"Append rows to a spreadsheet"`
	SmartAppend SheetsSmartAppendCmd `cmd:"smart-append" help:"Validate + append with structure awareness"`
	Update      SheetsUpdateCmd      `cmd:"" help:"Update cells in a spreadsheet"`
	Clear       SheetsClearCmd       `cmd:"" help:"Clear a range"`
	CopyTab     SheetsCopyTabCmd     `cmd:"copy-tab" help:"Copy tab structure to a new tab"`
	Export      SheetsExportCmd      `cmd:"" help:"Export range to CSV or JSON"`
	Import      SheetsImportCmd      `cmd:"" help:"Import CSV or JSON file into a sheet"`
	Create      SheetsCreateCmd      `cmd:"" help:"Create a new spreadsheet"`
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

// SheetsDescribeCmd analyzes column structure and fill rules.
type SheetsDescribeCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	Range         string `help:"Sheet range (auto-detects first sheet if empty)" short:"r"`
	Samples       int    `help:"Number of data rows to analyze" default:"20" short:"n"`
}

func (c *SheetsDescribeCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.describe"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	schema, err := sheetsSvc.DescribeSheet(rctx.Context, c.SpreadsheetID, c.Range, c.Samples)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(schema)
	return nil
}

// SheetsSmartAppendCmd validates data against schema then appends.
type SheetsSmartAppendCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	Range         string `arg:"" help:"Range to append to (e.g. Sheet1!A:F)"`
	Values        string `help:"JSON array of rows" required:"" short:"v"`
}

func (c *SheetsSmartAppendCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.smart-append"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	values, err := api.ParseValuesJSON(c.Values)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, err.Error())
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)

	// Step 1: Describe the sheet to get schema
	schema, err := sheetsSvc.DescribeSheet(rctx.Context, c.SpreadsheetID, c.Range, 20)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	// Step 2: Validate each row
	var allIssues []map[string]interface{}
	allValid := true
	for i, row := range values {
		vr := api.ValidateRow(schema, row)
		if !vr.Valid {
			allValid = false
			for _, issue := range vr.Issues {
				allIssues = append(allIssues, map[string]interface{}{
					"row":     i,
					"column":  issue.Column,
					"value":   issue.Value,
					"message": issue.Message,
				})
			}
		}
	}

	if !allValid {
		rctx.Printer.Success(map[string]interface{}{
			"valid":      false,
			"issues":     allIssues,
			"schema":     schema,
			"action":     "none",
			"suggestion": "Fix the issues above and retry, or use 'sheets append' to skip validation.",
		})
		return nil
	}

	// Step 3: Dry-run shows what would be written
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"valid":      true,
			"dry_run":    "sheets.smart-append",
			"rows":       len(values),
			"values":     values,
			"schema":     schema,
		})
		return nil
	}

	// Step 4: Append
	result, err := sheetsSvc.AppendValues(rctx.Context, c.SpreadsheetID, c.Range, values)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"valid":    true,
		"appended": true,
		"result":   result,
		"rows":     len(values),
	})
	return nil
}

// SheetsStatsCmd shows column statistics.
type SheetsStatsCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	Range         string `help:"Range (auto-detects first sheet if empty)" short:"r"`
}

func (c *SheetsStatsCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.stats"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	stats, err := sheetsSvc.StatsRange(rctx.Context, c.SpreadsheetID, c.Range)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(stats)
	return nil
}

// SheetsDiffCmd compares two ranges.
type SheetsDiffCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	RangeA        string `help:"First range (e.g. '第1周规划及完成情况')" required:"" name:"from"`
	RangeB        string `help:"Second range (e.g. '第2周规划及完成情况')" required:"" name:"to"`
}

func (c *SheetsDiffCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.diff"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	diff, err := sheetsSvc.DiffRanges(rctx.Context, c.SpreadsheetID, c.RangeA, c.RangeB)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(diff)
	return nil
}

// SheetsCopyTabCmd copies a tab structure.
type SheetsCopyTabCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	Source        string `help:"Source tab name" required:""`
	Name          string `help:"New tab name" required:""`
}

func (c *SheetsCopyTabCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.copy-tab"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": "sheets.copy-tab",
			"source":  c.Source,
			"name":    c.Name,
		})
		return nil
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	if err := sheetsSvc.CopyTab(rctx.Context, c.SpreadsheetID, c.Source, c.Name); err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"copied": true,
		"source": c.Source,
		"name":   c.Name,
	})
	return nil
}

// SheetsExportCmd exports a range to CSV or JSON.
type SheetsExportCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	Range         string `arg:"" help:"Range to export"`
	ExportFmt     string `help:"Format: csv or json" default:"csv" name:"export-format"`
	Output        string `help:"Output file (default: stdout)" short:"o"`
}

func (c *SheetsExportCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.export"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	count, err := sheetsSvc.ExportToFile(rctx.Context, c.SpreadsheetID, c.Range, c.ExportFmt, c.Output)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	if c.Output != "" && c.Output != "-" {
		rctx.Printer.Success(map[string]interface{}{
			"exported": true,
			"format":   c.ExportFmt,
			"path":     c.Output,
			"rows":     count,
		})
	}
	return nil
}

// SheetsImportCmd imports a CSV or JSON file into a sheet.
type SheetsImportCmd struct {
	SpreadsheetID string `arg:"" help:"Spreadsheet ID"`
	Range         string `arg:"" help:"Range to import into (e.g. Sheet1!A1)"`
	File          string `help:"File path to import" required:"" short:"i"`
	ImportFmt     string `help:"Format: csv or json" default:"csv" name:"import-format"`
}

func (c *SheetsImportCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "sheets.import"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": "sheets.import",
			"file":    c.File,
			"format":  c.ImportFmt,
		})
		return nil
	}

	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	result, err := sheetsSvc.ImportFromFile(rctx.Context, c.SpreadsheetID, c.Range, c.ImportFmt, c.File)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"imported": true,
		"result":   result,
	})
	return nil
}
