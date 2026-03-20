package cmd

import (
	"encoding/json"

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
	Template  DocsTemplateCmd  `cmd:"" help:"Create doc from template with {{var}} replacement"`
	FromSheet DocsFromSheetCmd `cmd:"from-sheet" help:"Create doc from spreadsheet data"`
	Export    DocsExportCmd    `cmd:"" help:"Export document to file"`
}

// DocsGetCmd retrieves a document.
type DocsGetCmd struct {
	DocID string `arg:"" help:"Document ID"`
}

func (c *DocsGetCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "docs.get", []string{"docs"}); done {
		return err
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
	if done, err := Preflight(rctx, "docs.create", []string{"docs"}); done {
		return err
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
	if done, err := Preflight(rctx, "docs.append", []string{"docs"}); done {
		return err
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
// NOTE: services = ["docs", "drive"] — export writes to Drive, requires both scopes.
type DocsExportCmd struct {
	DocID     string `arg:"" help:"Document ID"`
	ExportFmt string `help:"Export format: pdf, docx, txt, html" default:"pdf" enum:"pdf,docx,txt,html" name:"export-format"`
	Output    string `help:"Output file path" short:"o"`
}

func (c *DocsExportCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "docs.export", []string{"docs", "drive"}); done {
		return err
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
	if done, err := Preflight(rctx, "docs.search", []string{"docs"}); done {
		return err
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
	if done, err := Preflight(rctx, "docs.replace", []string{"docs"}); done {
		return err
	}

	docsSvc := api.NewDocsService(rctx.APIClient)
	count, err := docsSvc.ReplaceText(rctx.Context, c.DocID, c.Find, c.Replace)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"replaced":            true,
		"occurrences_changed": count,
		"find":                c.Find,
		"replace":             c.Replace,
	})
	return nil
}

// DocsFromSheetCmd creates a doc from spreadsheet data.
// NOTE: services = ["docs", "sheets"] — reads Sheets then writes Docs, requires both scopes.
type DocsFromSheetCmd struct {
	SpreadsheetID string `arg:"" help:"Source spreadsheet ID"`
	Range         string `arg:"" help:"Range to read (e.g. Sheet1!A1:D10)"`
	Title         string `help:"Document title" required:""`
}

func (c *DocsFromSheetCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "docs.from-sheet", []string{"docs", "sheets"}); done {
		return err
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

// DocsTemplateCmd creates a doc from a template with {{var}} replacement.
// NOTE: manual Preflight — json.Unmarshal(vars) must happen before DryRun so that
// invalid --vars JSON is caught even in dry-run mode. Preflight would short-circuit
// at DryRun before the validation runs, causing silent bad input to pass.
type DocsTemplateCmd struct {
	TemplateID string `arg:"" help:"Template document ID"`
	Vars       string `help:"JSON object of variables: {\"name\":\"Alice\",\"date\":\"2026-03-17\"}" required:"" short:"v"`
	Title      string `help:"New document title (default: template title + ' (from template)')"`
}

func (c *DocsTemplateCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "docs.template"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"docs"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	// Validate --vars JSON before DryRun so bad input is caught in all modes.
	var vars map[string]string
	if err := json.Unmarshal([]byte(c.Vars), &vars); err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, "invalid --vars JSON: "+err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":     "docs.template",
			"template_id": c.TemplateID,
			"vars":        vars,
			"title":       c.Title,
		})
		return nil
	}

	docsSvc := api.NewDocsService(rctx.APIClient)
	doc, err := docsSvc.CreateFromTemplate(rctx.Context, c.TemplateID, c.Title, vars)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"created":    true,
		"document":   doc,
		"template":   c.TemplateID,
		"vars_count": len(vars),
	})
	return nil
}
