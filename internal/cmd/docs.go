package cmd

import (
	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// DocsCmd groups Docs operations.
type DocsCmd struct {
	Get    DocsGetCmd    `cmd:"" help:"Get document content"`
	Create DocsCreateCmd `cmd:"" help:"Create a document"`
	Append DocsAppendCmd `cmd:"" help:"Append text to a document"`
	Export DocsExportCmd `cmd:"" help:"Export document to file"`
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
