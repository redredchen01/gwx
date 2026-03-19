package mcp

import (
	"context"

	"github.com/redredchen01/gwx/internal/api"
)

// SlidesTools returns MCP tool definitions for Google Slides.
func SlidesTools() []Tool {
	return []Tool{
		{
			Name:        "slides_get",
			Description: "Get a Google Slides presentation structure (slides, text elements).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"presentation_id": {Type: "string", Description: "Presentation ID"},
				},
				Required: []string{"presentation_id"},
			},
		},
		{
			Name:        "slides_list",
			Description: "List Google Slides presentations in Drive.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit": {Type: "integer", Description: "Max results (default 20)"},
				},
			},
		},
		{
			Name:        "slides_create",
			Description: "Create a new Google Slides presentation. CAUTION: Creates a real file.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"title": {Type: "string", Description: "Presentation title"},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "slides_duplicate",
			Description: "Duplicate a Google Slides presentation. CAUTION: Creates a real file.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"presentation_id": {Type: "string", Description: "Presentation ID to duplicate"},
					"title":           {Type: "string", Description: "Title for the copy"},
				},
				Required: []string{"presentation_id", "title"},
			},
		},
		{
			Name:        "slides_export",
			Description: "Export a Google Slides presentation to PDF or PPTX.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"presentation_id": {Type: "string", Description: "Presentation ID"},
					"format":          {Type: "string", Description: "Export format: pdf or pptx (default pdf)"},
					"output":          {Type: "string", Description: "Output file path"},
				},
				Required: []string{"presentation_id"},
			},
		},
		{
			Name:        "slides_from_sheet",
			Description: "Generate a presentation from a template + Sheet data. Replaces {{placeholders}} with Sheet values. CAUTION: Creates a real file.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"template_id": {Type: "string", Description: "Template presentation ID (with {{placeholders}})"},
					"sheet_id":    {Type: "string", Description: "Source Sheet ID"},
					"range":       {Type: "string", Description: "Sheet range (default Sheet1)"},
				},
				Required: []string{"template_id", "sheet_id"},
			},
		},
	}
}

// registerSlidesTools registers Slides tool handlers into the registry.
func (h *GWXHandler) registerSlidesTools(r map[string]ToolHandler) {
	r["slides_get"] = h.slidesGet
	r["slides_list"] = h.slidesList
	r["slides_create"] = h.slidesCreate
	r["slides_duplicate"] = h.slidesDuplicate
	r["slides_export"] = h.slidesExport
	r["slides_from_sheet"] = h.slidesFromSheet
}

func (h *GWXHandler) slidesGet(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSlidesService(h.client)
	result, err := svc.GetPresentation(ctx, strArg(args, "presentation_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (h *GWXHandler) slidesList(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSlidesService(h.client)
	files, err := svc.ListPresentations(ctx, int64(intArg(args, "limit", 20)))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"presentations": files, "count": len(files)})
}

func (h *GWXHandler) slidesCreate(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSlidesService(h.client)
	result, err := svc.CreatePresentation(ctx, strArg(args, "title"))
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (h *GWXHandler) slidesDuplicate(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSlidesService(h.client)
	result, err := svc.DuplicatePresentation(ctx, strArg(args, "presentation_id"), strArg(args, "title"))
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (h *GWXHandler) slidesExport(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSlidesService(h.client)
	format := strArg(args, "format")
	if format == "" {
		format = "pdf"
	}
	path, err := svc.ExportPresentation(ctx, strArg(args, "presentation_id"), format, strArg(args, "output"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"exported": true, "format": format, "path": path})
}

func (h *GWXHandler) slidesFromSheet(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSlidesService(h.client)
	sheetRange := strArg(args, "range")
	if sheetRange == "" {
		sheetRange = "Sheet1"
	}
	result, err := svc.FromSheet(ctx, strArg(args, "template_id"), strArg(args, "sheet_id"), sheetRange)
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}
