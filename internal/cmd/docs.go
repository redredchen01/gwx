package cmd

import (
	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// DocsCmd groups Docs operations.
type DocsCmd struct {
	Get       DocsGetCmd       `cmd:"" help:"Get document content"`
	Create    DocsCreateCmd    `cmd:"" help:"Create a document"`
	Append    DocsAppendCmd    `cmd:"" help:"Append text to a document"`
	Search    DocsSearchCmd    `cmd:"" help:"Search text in a document"`
	Replace   DocsReplaceCmd   `cmd:"" help:"Find and replace text"`
	FromSheet DocsFromSheetCmd `cmd:"from-sheet" help:"Create doc from spreadsheet data"`
	Export    DocsExportCmd    `cmd:"" help:"Export document to file"`
}

// DocsGetCmd retrieves a document.
type DocsGetCmd struct {
	DocID string `arg:"" help:"Document ID"`
}

func (c *DocsGetCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "docs.get"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"docs"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "docs.get", "doc_id": c.DocID})
		return nil
	}

	docsSvc := api.NewDocsService(rctx.APIClient)
	doc, err := docsSvc.GetDocument(rctx.Context, c.DocID)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(doc)
	return nil
}

// DocsCreateCmd creates a document.
type DocsCreateCmd struct {
	Title string `help:"Document title" required:""`
	Body  string `help:"Initial body text" short:"b"`
}

func (c *DocsCreateCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "docs.create"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"docs"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "docs.create", "title": c.Title})
		return nil
	}

	docsSvc := api.NewDocsService(rctx.APIClient)
	doc, err := docsSvc.CreateDocument(rctx.Context, c.Title, c.Body)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"created":  true,
		"document": doc,
	})
	return nil
}

// DocsAppendCmd appends text to a document.
type DocsAppendCmd struct {
	DocID string `arg:"" help:"Document ID"`
	Text  string `help:"Text to append" required:"" short:"t"`
}

func (c *DocsAppendCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "docs.append"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"docs"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "docs.append", "doc_id": c.DocID})
		return nil
	}

	docsSvc := api.NewDocsService(rctx.APIClient)
	if err := docsSvc.AppendText(rctx.Context, c.DocID, c.Text); err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"appended": true,
		"doc_id":   c.DocID,
	})
	return nil
}

// DocsExportCmd exports a document.
type DocsExportCmd struct {
	DocID  string `arg:"" help:"Document ID"`
	ExportFmt string `help:"Export format: pdf, docx, txt, html" default:"pdf" enum:"pdf,docx,txt,html" name:"export-format"`
	Output string `help:"Output file path" short:"o"`
}

func (c *DocsExportCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "docs.export"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"docs", "drive"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "docs.export", "doc_id": c.DocID, "format": c.ExportFmt})
		return nil
	}

	docsSvc := api.NewDocsService(rctx.APIClient)
	path, err := docsSvc.ExportDocument(rctx.Context, c.DocID, c.ExportFmt, c.Output)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"exported": true,
		"path":     path,
		"format":   c.ExportFmt,
	})
	return nil
}

// DocsSearchCmd searches text within a document.
type DocsSearchCmd struct {
	DocID string `arg:"" help:"Document ID"`
	Query string `help:"Text to search for" required:"" short:"q"`
}

func (c *DocsSearchCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "docs.search"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"docs"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	docsSvc := api.NewDocsService(rctx.APIClient)
	result, err := docsSvc.SearchDocument(rctx.Context, c.DocID, c.Query)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(result)
	return nil
}

// DocsReplaceCmd finds and replaces text.
type DocsReplaceCmd struct {
	DocID   string `arg:"" help:"Document ID"`
	Find    string `help:"Text to find" required:""`
	Replace string `help:"Replacement text" required:"" name:"with"`
}

func (c *DocsReplaceCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "docs.replace"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"docs"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": "docs.replace",
			"doc_id":  c.DocID,
			"find":    c.Find,
			"replace": c.Replace,
		})
		return nil
	}

	docsSvc := api.NewDocsService(rctx.APIClient)
	count, err := docsSvc.ReplaceText(rctx.Context, c.DocID, c.Find, c.Replace)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"replaced":           true,
		"occurrences_changed": count,
		"find":               c.Find,
		"replace":            c.Replace,
	})
	return nil
}

// DocsFromSheetCmd creates a doc from spreadsheet data.
type DocsFromSheetCmd struct {
	SpreadsheetID string `arg:"" help:"Source spreadsheet ID"`
	Range         string `arg:"" help:"Range to read (e.g. Sheet1!A1:D10)"`
	Title         string `help:"Document title" required:""`
}

func (c *DocsFromSheetCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "docs.from-sheet"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"docs", "sheets"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":        "docs.from-sheet",
			"spreadsheet_id": c.SpreadsheetID,
			"range":          c.Range,
			"title":          c.Title,
		})
		return nil
	}

	// Read sheet data
	sheetsSvc := api.NewSheetsService(rctx.APIClient)
	data, err := sheetsSvc.ReadRange(rctx.Context, c.SpreadsheetID, c.Range)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	if len(data.Values) == 0 {
		return rctx.Printer.ErrExit(exitcode.NotFound, "no data in the specified range")
	}

	// First row as headers, rest as data
	headers := data.Values[0]
	rows := data.Values[1:]

	// Create doc
	docsSvc := api.NewDocsService(rctx.APIClient)
	doc, err := docsSvc.CreateDocFromTable(rctx.Context, c.Title, headers, rows)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"created":    true,
		"document":   doc,
		"source":     c.SpreadsheetID,
		"rows_count": len(rows),
	})
	return nil
}
