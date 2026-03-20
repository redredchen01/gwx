package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redredchen01/gwx/internal/api"
)

// NewTools returns the 18 additional tool definitions for gwx v0.8.0.
// Kept for backward-compat with tests.
func NewTools() []Tool { return newProvider{}.Tools() }

type newProvider struct{}

func (newProvider) Tools() []Tool {
	return []Tool{
		// Gmail
		{
			Name:        "gmail_reply",
			Description: "Reply to a Gmail message. CAUTION: Sends a real email.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"message_id": {Type: "string", Description: "Gmail message ID to reply to"},
					"body":       {Type: "string", Description: "Reply body text"},
					"reply_all":  {Type: "boolean", Description: "Reply to all recipients"},
					"cc":         {Type: "string", Description: "CC recipients, comma-separated"},
				},
				Required: []string{"message_id", "body"},
			},
		},
		// Calendar
		{
			Name:        "calendar_list",
			Description: "List calendar events in a time range across all or a specific calendar.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"time_min":    {Type: "string", Description: "Start time (RFC3339, e.g. 2026-01-01T00:00:00Z)"},
					"time_max":    {Type: "string", Description: "End time (RFC3339)"},
					"max_results": {Type: "integer", Description: "Max events to return"},
					"calendar_id": {Type: "string", Description: "Calendar ID (default: primary)"},
				},
			},
		},
		{
			Name:        "calendar_update",
			Description: "Update an existing calendar event. CAUTION: Modifies a real event.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"event_id":    {Type: "string", Description: "Event ID to update"},
					"title":       {Type: "string", Description: "New event title"},
					"start":       {Type: "string", Description: "New start time (RFC3339 or YYYY-MM-DD)"},
					"end":         {Type: "string", Description: "New end time (RFC3339 or YYYY-MM-DD)"},
					"location":    {Type: "string", Description: "New location"},
					"attendees":   {Type: "string", Description: "New attendees, comma-separated"},
					"calendar_id": {Type: "string", Description: "Calendar ID (default: primary)"},
				},
				Required: []string{"event_id"},
			},
		},
		// Contacts
		{
			Name:        "contacts_list",
			Description: "List contacts sorted by last modified.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit": {Type: "integer", Description: "Max contacts to return (default 100)"},
				},
			},
		},
		{
			Name:        "contacts_get",
			Description: "Get a specific contact by resource name.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"resource_name": {Type: "string", Description: "Contact resource name (e.g. people/c12345)"},
				},
				Required: []string{"resource_name"},
			},
		},
		// Drive
		{
			Name:        "drive_download",
			Description: "Download a file from Google Drive to local disk. Files over 100MB are rejected.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"file_id":     {Type: "string", Description: "Drive file ID"},
					"output_path": {Type: "string", Description: "Local output path (defaults to original file name)"},
				},
				Required: []string{"file_id"},
			},
		},
		// Sheets
		{
			Name:        "sheets_update",
			Description: "Update (overwrite) a range in a Google Spreadsheet. CAUTION: Overwrites existing data.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"range":          {Type: "string", Description: "Range to update (e.g. Sheet1!A1:C3)"},
					"values":         {Type: "string", Description: "JSON 2D array of values: [[\"a\",1],[\"b\",2]]"},
				},
				Required: []string{"spreadsheet_id", "range", "values"},
			},
		},
		{
			Name:        "sheets_import",
			Description: "Import data from a local CSV or JSON file into a spreadsheet range. CAUTION: Appends data.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"range":          {Type: "string", Description: "Target range (e.g. Sheet1!A1)"},
					"path":           {Type: "string", Description: "Local file path"},
					"format":         {Type: "string", Description: "File format: csv or json", Default: "csv"},
				},
				Required: []string{"spreadsheet_id", "range", "path"},
			},
		},
		{
			Name:        "sheets_create",
			Description: "Create a new Google Spreadsheet.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"title": {Type: "string", Description: "Spreadsheet title"},
				},
				Required: []string{"title"},
			},
		},
		// Tasks
		{
			Name:        "tasks_lists",
			Description: "List all Google Tasks task lists.",
			InputSchema: InputSchema{
				Type: "object",
			},
		},
		{
			Name:        "tasks_complete",
			Description: "Mark a task as completed. CAUTION: Modifies task status.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"task_id": {Type: "string", Description: "Task ID"},
					"list_id": {Type: "string", Description: "Task list ID (default: @default)"},
				},
				Required: []string{"task_id"},
			},
		},
		{
			Name:        "tasks_delete",
			Description: "Delete a task permanently. CAUTION: Destructive.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"task_id": {Type: "string", Description: "Task ID"},
					"list_id": {Type: "string", Description: "Task list ID (default: @default)"},
				},
				Required: []string{"task_id"},
			},
		},
		// Docs
		{
			Name:        "docs_template",
			Description: "Create a new Google Doc from a template by replacing {{var}} placeholders.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"template_id": {Type: "string", Description: "Template document ID"},
					"title":       {Type: "string", Description: "Title for the new document"},
					"vars":        {Type: "string", Description: "JSON object of template variables: {\"name\":\"Alice\"}"},
				},
				Required: []string{"template_id", "title"},
			},
		},
		{
			Name:        "docs_from_sheet",
			Description: "Create a Google Doc from tabular data (headers + rows).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"title":   {Type: "string", Description: "Document title"},
					"headers": {Type: "string", Description: "JSON array of column headers: [\"Name\",\"Score\"]"},
					"rows":    {Type: "string", Description: "JSON 2D array of rows: [[\"Alice\",95],[\"Bob\",87]]"},
				},
				Required: []string{"title", "headers", "rows"},
			},
		},
		{
			Name:        "docs_export",
			Description: "Export a Google Doc to a local file (pdf, docx, txt, html).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"doc_id":      {Type: "string", Description: "Document ID"},
					"format":      {Type: "string", Description: "Export format: pdf, docx, txt, html", Default: "pdf"},
					"output_path": {Type: "string", Description: "Local output path"},
				},
				Required: []string{"doc_id"},
			},
		},
		// Chat
		{
			Name:        "chat_spaces",
			Description: "List Google Chat spaces the user is a member of.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit": {Type: "integer", Description: "Max spaces to return"},
				},
			},
		},
		{
			Name:        "chat_send",
			Description: "Send a text message to a Google Chat space. CAUTION: Sends a real message.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"space": {Type: "string", Description: "Space name (e.g. spaces/XXXXXXXXX)"},
					"text":  {Type: "string", Description: "Message text"},
				},
				Required: []string{"space", "text"},
			},
		},
		{
			Name:        "chat_messages",
			Description: "List recent messages in a Google Chat space.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"space": {Type: "string", Description: "Space name (e.g. spaces/XXXXXXXXX)"},
					"limit": {Type: "integer", Description: "Max messages to return"},
				},
				Required: []string{"space"},
			},
		},
	}
}

func (newProvider) Handlers(h *GWXHandler) map[string]ToolHandler {
	return map[string]ToolHandler{
		"gmail_reply":     h.gmailReply,
		"calendar_list":   h.calendarList,
		"calendar_update": h.calendarUpdate,
		"contacts_list":   h.contactsList,
		"contacts_get":    h.contactsGet,
		"drive_download":  h.driveDownload,
		"sheets_update":   h.sheetsUpdate,
		"sheets_import":   h.sheetsImport,
		"sheets_create":   h.sheetsCreate,
		"tasks_lists":     h.tasksLists,
		"tasks_complete":  h.tasksComplete,
		"tasks_delete":    h.tasksDelete,
		"docs_template":   h.docsTemplate,
		"docs_from_sheet": h.docsFromSheet,
		"docs_export":     h.docsExport,
		"chat_spaces":     h.chatSpaces,
		"chat_send":       h.chatSend,
		"chat_messages":   h.chatMessages,
	}
}

func init() { RegisterProvider(newProvider{}) }

// --- Gmail ---

func (h *GWXHandler) gmailReply(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewGmailService(h.client)
	messageID := strArg(args, "message_id")
	input := &api.SendInput{
		Body:     strArg(args, "body"),
		ReplyAll: boolArg(args, "reply_all"),
		CC:       splitArg(args, "cc"),
	}
	result, err := svc.ReplyMessage(ctx, messageID, input)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"replied": true, "message_id": result.MessageID, "thread_id": result.ThreadID})
}

// --- Calendar ---

func (h *GWXHandler) calendarList(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewCalendarService(h.client)
	calendarID := strArg(args, "calendar_id")
	if calendarID == "" {
		calendarID = "primary"
	}
	maxResults := int64(intArg(args, "max_results", 50))

	now := time.Now()
	timeMin := now
	timeMax := now.AddDate(0, 0, 30)

	if s := strArg(args, "time_min"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("invalid time_min %q: must be RFC3339", s)
		}
		timeMin = t
	}
	if s := strArg(args, "time_max"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("invalid time_max %q: must be RFC3339", s)
		}
		timeMax = t
	}

	events, err := svc.ListEvents(ctx, calendarID, timeMin, timeMax, maxResults)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"events": events, "count": len(events)})
}

func (h *GWXHandler) calendarUpdate(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewCalendarService(h.client)
	calendarID := strArg(args, "calendar_id")
	if calendarID == "" {
		calendarID = "primary"
	}
	input := &api.EventInput{
		Title:     strArg(args, "title"),
		Start:     strArg(args, "start"),
		End:       strArg(args, "end"),
		Location:  strArg(args, "location"),
		Attendees: splitArg(args, "attendees"),
	}
	event, err := svc.UpdateEvent(ctx, calendarID, strArg(args, "event_id"), input)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"updated": true, "event": event})
}

// --- Contacts ---

func (h *GWXHandler) contactsList(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewContactsService(h.client)
	contacts, err := svc.ListContacts(ctx, intArg(args, "limit", 100))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"contacts": contacts, "count": len(contacts)})
}

func (h *GWXHandler) contactsGet(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewContactsService(h.client)
	contact, err := svc.GetContact(ctx, strArg(args, "resource_name"))
	if err != nil {
		return nil, err
	}
	return jsonResult(contact)
}

// --- Drive ---

func (h *GWXHandler) driveDownload(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDriveService(h.client)
	fileID := strArg(args, "file_id")
	outputPath := strArg(args, "output_path")

	// Pre-check: reject files over 100MB
	if err := svc.CheckDownloadSize(ctx, fileID, 100*1024*1024); err != nil {
		return nil, err
	}

	savedPath, err := svc.DownloadFile(ctx, fileID, outputPath)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"downloaded": true, "path": savedPath})
}

// --- Sheets ---

func (h *GWXHandler) sheetsUpdate(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	values, err := api.ParseValuesJSON(strArg(args, "values"))
	if err != nil {
		return nil, err
	}
	result, err := svc.UpdateValues(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"), values)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"updated": true, "result": result})
}

func (h *GWXHandler) sheetsImport(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	format := strArg(args, "format")
	if format == "" {
		format = "csv"
	}
	result, err := svc.ImportFromFile(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"), format, strArg(args, "path"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"imported": true, "result": result})
}

func (h *GWXHandler) sheetsCreate(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	result, err := svc.CreateSpreadsheet(ctx, strArg(args, "title"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"created": true, "spreadsheet": result})
}

// --- Tasks ---

func (h *GWXHandler) tasksLists(ctx context.Context, _ map[string]interface{}) (*ToolResult, error) {
	svc := api.NewTasksService(h.client)
	lists, err := svc.ListTaskLists(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"lists": lists, "count": len(lists)})
}

func (h *GWXHandler) tasksComplete(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewTasksService(h.client)
	listID := strArg(args, "list_id")
	if listID == "" {
		listID = "@default"
	}
	task, err := svc.CompleteTask(ctx, listID, strArg(args, "task_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"completed": true, "task": task})
}

func (h *GWXHandler) tasksDelete(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewTasksService(h.client)
	listID := strArg(args, "list_id")
	if listID == "" {
		listID = "@default"
	}
	if err := svc.DeleteTask(ctx, listID, strArg(args, "task_id")); err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"deleted": true})
}

// --- Docs ---

func (h *GWXHandler) docsTemplate(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDocsService(h.client)
	vars := make(map[string]string)
	if varsStr := strArg(args, "vars"); varsStr != "" {
		if err := json.Unmarshal([]byte(varsStr), &vars); err != nil {
			return nil, fmt.Errorf("invalid vars JSON: %w", err)
		}
	}
	doc, err := svc.CreateFromTemplate(ctx, strArg(args, "template_id"), strArg(args, "title"), vars)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"created": true, "document": doc})
}

func (h *GWXHandler) docsFromSheet(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDocsService(h.client)

	var headers []interface{}
	if err := json.Unmarshal([]byte(strArg(args, "headers")), &headers); err != nil {
		return nil, fmt.Errorf("invalid headers JSON: %w", err)
	}

	var rows [][]interface{}
	if err := json.Unmarshal([]byte(strArg(args, "rows")), &rows); err != nil {
		return nil, fmt.Errorf("invalid rows JSON: %w", err)
	}

	doc, err := svc.CreateDocFromTable(ctx, strArg(args, "title"), headers, rows)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"created": true, "document": doc})
}

func (h *GWXHandler) docsExport(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDocsService(h.client)
	format := strArg(args, "format")
	if format == "" {
		format = "pdf"
	}
	savedPath, err := svc.ExportDocument(ctx, strArg(args, "doc_id"), format, strArg(args, "output_path"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"exported": true, "path": savedPath, "format": format})
}

// --- Chat ---

func (h *GWXHandler) chatSpaces(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewChatService(h.client)
	spaces, err := svc.ListSpaces(ctx, intArg(args, "limit", 0))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"spaces": spaces, "count": len(spaces)})
}

func (h *GWXHandler) chatSend(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewChatService(h.client)
	result, err := svc.SendMessage(ctx, strArg(args, "space"), strArg(args, "text"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"sent": true, "message": result})
}

func (h *GWXHandler) chatMessages(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewChatService(h.client)
	messages, err := svc.ListMessages(ctx, strArg(args, "space"), intArg(args, "limit", 0))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"messages": messages, "count": len(messages)})
}

