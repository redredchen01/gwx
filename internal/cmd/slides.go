package cmd

import (
	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// SlidesCmd groups Google Slides operations.
type SlidesCmd struct {
	Get       SlidesGetCmd       `cmd:"" help:"Get presentation structure"`
	List      SlidesListCmd      `cmd:"" help:"List presentations in Drive"`
	Create    SlidesCreateCmd    `cmd:"" help:"Create a new presentation"`
	Duplicate SlidesDuplicateCmd `cmd:"" help:"Duplicate a presentation"`
	Export    SlidesExportCmd    `cmd:"" help:"Export presentation to PDF or PPTX"`
	FromSheet SlidesFromSheetCmd `cmd:"from-sheet" help:"Generate presentation from Sheet data + template"`
}

// SlidesGetCmd gets a presentation's structure.
type SlidesGetCmd struct {
	PresentationID string `arg:"" help:"Presentation ID"`
}

func (c *SlidesGetCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "slides.get", []string{"slides"}); done {
		return err
	}
	svc := api.NewSlidesService(rctx.APIClient)
	result, err := svc.GetPresentation(rctx.Context, c.PresentationID)
	if err != nil {
		return handleAPIError(rctx, err)
	}
	rctx.Printer.Success(result)
	return nil
}

// SlidesListCmd lists presentations.
type SlidesListCmd struct {
	Limit int64 `help:"Max results (e.g. --limit 10)" default:"20" short:"n"`
}

func (c *SlidesListCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "slides.list", []string{"drive"}); done {
		return err
	}
	svc := api.NewSlidesService(rctx.APIClient)
	files, err := svc.ListPresentations(rctx.Context, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}
	rctx.Printer.Success(map[string]interface{}{
		"presentations": files,
		"count":         len(files),
	})
	return nil
}

// SlidesCreateCmd creates a new presentation.
type SlidesCreateCmd struct {
	Title string `help:"Presentation title" required:""`
}

func (c *SlidesCreateCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "slides.create", []string{"slides"}); done {
		return err
	}
	svc := api.NewSlidesService(rctx.APIClient)
	result, err := svc.CreatePresentation(rctx.Context, c.Title)
	if err != nil {
		return handleAPIError(rctx, err)
	}
	rctx.Printer.Success(result)
	return nil
}

// SlidesDuplicateCmd duplicates a presentation.
type SlidesDuplicateCmd struct {
	PresentationID string `arg:"" help:"Presentation ID to duplicate"`
	Title          string `help:"Title for the copy" required:""`
}

func (c *SlidesDuplicateCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "slides.duplicate", []string{"drive"}); done {
		return err
	}
	svc := api.NewSlidesService(rctx.APIClient)
	result, err := svc.DuplicatePresentation(rctx.Context, c.PresentationID, c.Title)
	if err != nil {
		return handleAPIError(rctx, err)
	}
	rctx.Printer.Success(result)
	return nil
}

// SlidesExportCmd exports a presentation.
type SlidesExportCmd struct {
	PresentationID string `arg:"" help:"Presentation ID to export"`
	Format         string `help:"Export format: pdf or pptx" default:"pdf" enum:"pdf,pptx" name:"export-format"`
	Output         string `help:"Output file path" short:"o"`
}

func (c *SlidesExportCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "slides.export", []string{"drive"}); done {
		return err
	}
	svc := api.NewSlidesService(rctx.APIClient)
	outputPath, err := svc.ExportPresentation(rctx.Context, c.PresentationID, c.Format, c.Output)
	if err != nil {
		return handleAPIError(rctx, err)
	}
	rctx.Printer.Success(map[string]interface{}{
		"exported": true,
		"format":   c.Format,
		"path":     outputPath,
	})
	return nil
}

// SlidesFromSheetCmd generates a presentation from Sheet data.
type SlidesFromSheetCmd struct {
	TemplateID string `help:"Template presentation ID (with {{placeholders}})" required:"" name:"template"`
	SheetID    string `help:"Source Sheet ID" required:"" name:"sheet-id"`
	Range      string `help:"Sheet range (e.g. Sheet1!A:D)" default:"Sheet1" name:"range"`
}

func (c *SlidesFromSheetCmd) Run(rctx *RunContext) error {
	if done, err := Preflight(rctx, "slides.from-sheet", []string{"slides", "sheets", "drive"}); done {
		return err
	}
	svc := api.NewSlidesService(rctx.APIClient)
	result, err := svc.FromSheet(rctx.Context, c.TemplateID, c.SheetID, c.Range)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}
	rctx.Printer.Success(result)
	return nil
}
