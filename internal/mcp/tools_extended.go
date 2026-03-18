package mcp

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/redredchen01/gwx/internal/api"
)

// ExtendedTools returns the additional tools not in the base set.
func ExtendedTools() []Tool {
	return []Tool{
		// Sheets extended
		{
			Name:        "sheets_stats",
			Description: "Column statistics: fill rates, value counts for enum columns. Returns summary like '已完成: 9, 持续中: 5'.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"range":          {Type: "string", Description: "Range (auto-detects first sheet if empty)"},
				},
				Required: []string{"spreadsheet_id"},
			},
		},
		{
			Name:        "sheets_diff",
			Description: "Compare two sheet tabs. Shows added/removed/modified rows with cell-level changes. Uses first column as key.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"range_a":        {Type: "string", Description: "First range (e.g. '第1周')"},
					"range_b":        {Type: "string", Description: "Second range (e.g. '第2周')"},
				},
				Required: []string{"spreadsheet_id", "range_a", "range_b"},
			},
		},
		{
			Name:        "sheets_copy_tab",
			Description: "Copy a tab's structure (headers + first column) to a new tab. For creating weekly planning sheets.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"source":         {Type: "string", Description: "Source tab name"},
					"name":           {Type: "string", Description: "New tab name"},
				},
				Required: []string{"spreadsheet_id", "source", "name"},
			},
		},
		{
			Name:        "sheets_export",
			Description: "Export a range to CSV or JSON format. Returns the formatted data as text.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"range":          {Type: "string", Description: "Range to export"},
					"format":         {Type: "string", Description: "csv or json", Default: "csv"},
				},
				Required: []string{"spreadsheet_id", "range"},
			},
		},
		{
			Name:        "sheets_info",
			Description: "Get spreadsheet metadata: title, all tab names with row/column counts.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
				},
				Required: []string{"spreadsheet_id"},
			},
		},
		{
			Name:        "sheets_clear",
			Description: "Clear all values in a range without deleting cells. CAUTION: Destructive.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"range":          {Type: "string", Description: "Range to clear"},
				},
				Required: []string{"spreadsheet_id", "range"},
			},
		},
		// Docs extended
		{
			Name:        "docs_search",
			Description: "Search for text within a Google Doc. Returns matching paragraphs with position.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"doc_id": {Type: "string", Description: "Document ID"},
					"query":  {Type: "string", Description: "Text to search for"},
				},
				Required: []string{"doc_id", "query"},
			},
		},
		{
			Name:        "docs_replace",
			Description: "Find and replace text in a Google Doc. CAUTION: Modifies document.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"doc_id":  {Type: "string", Description: "Document ID"},
					"find":    {Type: "string", Description: "Text to find"},
					"replace": {Type: "string", Description: "Replacement text"},
				},
				Required: []string{"doc_id", "find", "replace"},
			},
		},
		{
			Name:        "docs_create",
			Description: "Create a new Google Doc with optional body text.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"title": {Type: "string", Description: "Document title"},
					"body":  {Type: "string", Description: "Initial body text"},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "docs_append",
			Description: "Append text to the end of a Google Doc.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"doc_id": {Type: "string", Description: "Document ID"},
					"text":   {Type: "string", Description: "Text to append"},
				},
				Required: []string{"doc_id", "text"},
			},
		},
		// Calendar extended
		{
			Name:        "calendar_find_slot",
			Description: "Find free time slots for attendees. Checks FreeBusy and returns available slots during business hours.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"attendees": {Type: "string", Description: "Attendee emails, comma-separated"},
					"duration":  {Type: "string", Description: "Meeting duration (e.g. 30m, 1h)", Default: "30m"},
					"days":      {Type: "integer", Description: "Days ahead to search (default 3)"},
				},
				Required: []string{"attendees"},
			},
		},
		{
			Name:        "calendar_delete",
			Description: "Delete a calendar event. CAUTION: Destructive.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"event_id": {Type: "string", Description: "Event ID to delete"},
				},
				Required: []string{"event_id"},
			},
		},
		// Drive extended
		{
			Name:        "drive_upload",
			Description: "Upload a local file to Google Drive. CAUTION: Creates a file.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path":   {Type: "string", Description: "Local file path"},
					"folder": {Type: "string", Description: "Destination folder ID"},
					"name":   {Type: "string", Description: "Override file name"},
				},
				Required: []string{"path"},
			},
		},
		{
			Name:        "drive_share",
			Description: "Share a file with a user. CAUTION: Changes permissions.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_id": {Type: "string", Description: "File ID to share"},
					"email":   {Type: "string", Description: "Email to share with"},
					"role":    {Type: "string", Description: "Permission: reader, writer, commenter", Default: "reader"},
				},
				Required: []string{"file_id", "email"},
			},
		},
		{
			Name:        "drive_mkdir",
			Description: "Create a folder in Google Drive.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"name":   {Type: "string", Description: "Folder name"},
					"parent": {Type: "string", Description: "Parent folder ID"},
				},
				Required: []string{"name"},
			},
		},
		// Gmail extended
		{
			Name:        "gmail_labels",
			Description: "List all Gmail labels (system + user labels).",
			InputSchema: InputSchema{
				Type: "object",
			},
		},
		{
			Name:        "gmail_draft",
			Description: "Create an email draft (not sent). Safer than gmail_send.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"to":      {Type: "string", Description: "Recipient email(s)"},
					"subject": {Type: "string", Description: "Subject"},
					"body":    {Type: "string", Description: "Body text"},
				},
				Required: []string{"to", "subject", "body"},
			},
		},
	}
}

// CallExtendedTool handles calls to extended tools.
// Returns (result, error, handled). Handled=true means the tool name was recognized.
func (h *GWXHandler) CallExtendedTool(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error, bool) {
	switch name {
	case "sheets_stats":
		r, err := h.sheetsStats(ctx, args)
		return r, err, true
	case "sheets_diff":
		r, err := h.sheetsDiff(ctx, args)
		return r, err, true
	case "sheets_copy_tab":
		r, err := h.sheetsCopyTab(ctx, args)
		return r, err, true
	case "sheets_export":
		r, err := h.sheetsExport(ctx, args)
		return r, err, true
	case "sheets_info":
		r, err := h.sheetsInfo(ctx, args)
		return r, err, true
	case "sheets_clear":
		r, err := h.sheetsClear(ctx, args)
		return r, err, true
	case "docs_search":
		r, err := h.docsSearch(ctx, args)
		return r, err, true
	case "docs_replace":
		r, err := h.docsReplace(ctx, args)
		return r, err, true
	case "docs_create":
		r, err := h.docsCreate(ctx, args)
		return r, err, true
	case "docs_append":
		r, err := h.docsAppend(ctx, args)
		return r, err, true
	case "calendar_find_slot":
		r, err := h.calendarFindSlot(ctx, args)
		return r, err, true
	case "calendar_delete":
		r, err := h.calendarDelete(ctx, args)
		return r, err, true
	case "drive_upload":
		r, err := h.driveUpload(ctx, args)
		return r, err, true
	case "drive_share":
		r, err := h.driveShare(ctx, args)
		return r, err, true
	case "drive_mkdir":
		r, err := h.driveMkdir(ctx, args)
		return r, err, true
	case "gmail_labels":
		r, err := h.gmailLabels(ctx, args)
		return r, err, true
	case "gmail_draft":
		r, err := h.gmailDraft(ctx, args)
		return r, err, true
	default:
		return nil, nil, false
	}
}

// --- Implementations ---

func (h *GWXHandler) sheetsStats(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	stats, err := svc.StatsRange(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"))
	if err != nil {
		return nil, err
	}
	return jsonResult(stats)
}

func (h *GWXHandler) sheetsDiff(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	diff, err := svc.DiffRanges(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range_a"), strArg(args, "range_b"))
	if err != nil {
		return nil, err
	}
	return jsonResult(diff)
}

func (h *GWXHandler) sheetsCopyTab(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	err := svc.CopyTab(ctx, strArg(args, "spreadsheet_id"), strArg(args, "source"), strArg(args, "name"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"copied": true, "source": strArg(args, "source"), "name": strArg(args, "name")})
}

func (h *GWXHandler) sheetsExport(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	var buf bytes.Buffer
	format := strArg(args, "format")
	if format == "" {
		format = "csv"
	}
	switch format {
	case "csv":
		_, err := svc.ExportCSV(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"), &buf)
		if err != nil {
			return nil, err
		}
	case "json":
		_, err := svc.ExportJSON(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"), &buf)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format %q: use \"csv\" or \"json\"", format)
	}
	return &ToolResult{Content: []ContentBlock{{Type: "text", Text: buf.String()}}}, nil
}

func (h *GWXHandler) sheetsInfo(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	info, err := svc.GetInfo(ctx, strArg(args, "spreadsheet_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(info)
}

func (h *GWXHandler) sheetsClear(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	err := svc.ClearRange(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"cleared": true})
}

func (h *GWXHandler) docsSearch(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDocsService(h.client)
	result, err := svc.SearchDocument(ctx, strArg(args, "doc_id"), strArg(args, "query"))
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (h *GWXHandler) docsReplace(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDocsService(h.client)
	count, err := svc.ReplaceText(ctx, strArg(args, "doc_id"), strArg(args, "find"), strArg(args, "replace"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"replaced": true, "occurrences_changed": count})
}

func (h *GWXHandler) docsCreate(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDocsService(h.client)
	doc, err := svc.CreateDocument(ctx, strArg(args, "title"), strArg(args, "body"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"created": true, "document": doc})
}

func (h *GWXHandler) docsAppend(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDocsService(h.client)
	err := svc.AppendText(ctx, strArg(args, "doc_id"), strArg(args, "text"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"appended": true})
}

func (h *GWXHandler) calendarFindSlot(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewCalendarService(h.client)
	dur := 30 * time.Minute
	if ds := strArg(args, "duration"); ds != "" {
		parsed, err := time.ParseDuration(ds)
		if err != nil {
			return nil, fmt.Errorf("invalid duration %q: use Go duration format like 30m, 1h, 1h30m", ds)
		}
		dur = parsed
	}
	slots, err := svc.FindSlot(ctx, splitArg(args, "attendees"), dur, intArg(args, "days", 3))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"slots": slots, "count": len(slots)})
}

func (h *GWXHandler) calendarDelete(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewCalendarService(h.client)
	err := svc.DeleteEvent(ctx, "primary", strArg(args, "event_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"deleted": true})
}

func (h *GWXHandler) driveUpload(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDriveService(h.client)
	file, err := svc.UploadFile(ctx, strArg(args, "path"), strArg(args, "folder"), strArg(args, "name"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"uploaded": true, "file": file})
}

func (h *GWXHandler) driveShare(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDriveService(h.client)
	role := strArg(args, "role")
	if role == "" {
		role = "reader"
	}
	err := svc.ShareFile(ctx, strArg(args, "file_id"), strArg(args, "email"), role)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"shared": true})
}

func (h *GWXHandler) driveMkdir(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDriveService(h.client)
	folder, err := svc.CreateFolder(ctx, strArg(args, "name"), strArg(args, "parent"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"created": true, "folder": folder})
}

func (h *GWXHandler) gmailLabels(ctx context.Context, _ map[string]interface{}) (*ToolResult, error) {
	svc := api.NewGmailService(h.client)
	labels, err := svc.ListLabels(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"labels": labels, "count": len(labels)})
}

func (h *GWXHandler) gmailDraft(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewGmailService(h.client)
	input := &api.SendInput{
		To:      splitArg(args, "to"),
		Subject: strArg(args, "subject"),
		Body:    strArg(args, "body"),
	}
	result, err := svc.CreateDraft(ctx, input)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"drafted": true, "message_id": result.MessageID})
}
