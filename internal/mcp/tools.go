package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redredchen01/gwx/internal/api"
)

// GWXHandler implements the MCP Handler interface for Google Workspace tools.
type GWXHandler struct {
	client *api.Client
}

// NewGWXHandler creates a handler with an authenticated API client.
func NewGWXHandler(client *api.Client) *GWXHandler {
	return &GWXHandler{client: client}
}

func (h *GWXHandler) ListTools() []Tool {
	return []Tool{
		// Gmail
		{
			Name:        "gmail_list",
			Description: "List Gmail messages. Returns id, from, subject, date, snippet, unread status.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit":  {Type: "integer", Description: "Max messages (default 10)"},
					"unread": {Type: "boolean", Description: "Only unread messages"},
					"label":  {Type: "string", Description: "Filter by label (e.g. INBOX, STARRED)"},
				},
			},
		},
		{
			Name:        "gmail_get",
			Description: "Get a single Gmail message by ID. Returns full body, headers, and labels.",
			InputSchema: InputSchema{
				Type:     "object",
				Properties: map[string]Property{
					"message_id": {Type: "string", Description: "Gmail message ID"},
				},
				Required: []string{"message_id"},
			},
		},
		{
			Name:        "gmail_search",
			Description: "Search Gmail using query syntax (e.g. from:user@example.com, subject:invoice, has:attachment).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {Type: "string", Description: "Gmail search query"},
					"limit": {Type: "integer", Description: "Max results (default 10)"},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "gmail_send",
			Description: "Send an email. CAUTION: This sends a real email.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"to":      {Type: "string", Description: "Recipient email(s), comma-separated"},
					"subject": {Type: "string", Description: "Email subject"},
					"body":    {Type: "string", Description: "Email body text"},
					"cc":      {Type: "string", Description: "CC recipients, comma-separated"},
				},
				Required: []string{"to", "subject", "body"},
			},
		},
		// Calendar
		{
			Name:        "calendar_agenda",
			Description: "Show upcoming calendar events for today or N days ahead.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"days": {Type: "integer", Description: "Number of days (default 1)"},
				},
			},
		},
		{
			Name:        "calendar_create",
			Description: "Create a calendar event. CAUTION: This creates a real event.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"title":     {Type: "string", Description: "Event title"},
					"start":     {Type: "string", Description: "Start time (RFC3339 or YYYY-MM-DD)"},
					"end":       {Type: "string", Description: "End time (RFC3339 or YYYY-MM-DD)"},
					"attendees": {Type: "string", Description: "Attendee emails, comma-separated"},
					"location":  {Type: "string", Description: "Event location"},
				},
				Required: []string{"title", "start", "end"},
			},
		},
		// Drive
		{
			Name:        "drive_list",
			Description: "List files in Google Drive.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"folder": {Type: "string", Description: "Folder ID to list"},
					"limit":  {Type: "integer", Description: "Max files (default 20)"},
				},
			},
		},
		{
			Name:        "drive_search",
			Description: "Search files in Google Drive using Drive query syntax.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {Type: "string", Description: "Drive search query (e.g. name contains 'report')"},
					"limit": {Type: "integer", Description: "Max results (default 20)"},
				},
				Required: []string{"query"},
			},
		},
		// Docs
		{
			Name:        "docs_get",
			Description: "Get the content of a Google Doc as plain text.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"doc_id": {Type: "string", Description: "Google Doc ID"},
				},
				Required: []string{"doc_id"},
			},
		},
		// Sheets
		{
			Name:        "sheets_read",
			Description: "Read a range from a Google Spreadsheet.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"range":          {Type: "string", Description: "Range (e.g. Sheet1!A1:C10)"},
				},
				Required: []string{"spreadsheet_id", "range"},
			},
		},
		{
			Name:        "sheets_append",
			Description: "Append rows to a Google Spreadsheet. CAUTION: Modifies data.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"range":          {Type: "string", Description: "Range (e.g. Sheet1!A:C)"},
					"values":         {Type: "string", Description: "JSON array of rows: [[\"a\",1],[\"b\",2]]"},
				},
				Required: []string{"spreadsheet_id", "range", "values"},
			},
		},
		// Tasks
		{
			Name:        "tasks_list",
			Description: "List tasks from Google Tasks.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"list_id":        {Type: "string", Description: "Task list ID (default: primary)"},
					"show_completed": {Type: "boolean", Description: "Include completed tasks"},
				},
			},
		},
		{
			Name:        "tasks_create",
			Description: "Create a new task. CAUTION: Creates a real task.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"title": {Type: "string", Description: "Task title"},
					"notes": {Type: "string", Description: "Task notes"},
					"due":   {Type: "string", Description: "Due date (YYYY-MM-DD)"},
				},
				Required: []string{"title"},
			},
		},
		// Contacts
		{
			Name:        "contacts_search",
			Description: "Search contacts by name or email.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {Type: "string", Description: "Search query"},
					"limit": {Type: "integer", Description: "Max results (default 20)"},
				},
				Required: []string{"query"},
			},
		},
	}
}

func (h *GWXHandler) CallTool(name string, args map[string]interface{}) (*ToolResult, error) {
	ctx := context.Background()

	switch name {
	case "gmail_list":
		return h.gmailList(ctx, args)
	case "gmail_get":
		return h.gmailGet(ctx, args)
	case "gmail_search":
		return h.gmailSearch(ctx, args)
	case "gmail_send":
		return h.gmailSend(ctx, args)
	case "calendar_agenda":
		return h.calendarAgenda(ctx, args)
	case "calendar_create":
		return h.calendarCreate(ctx, args)
	case "drive_list":
		return h.driveList(ctx, args)
	case "drive_search":
		return h.driveSearch(ctx, args)
	case "docs_get":
		return h.docsGet(ctx, args)
	case "sheets_read":
		return h.sheetsRead(ctx, args)
	case "sheets_append":
		return h.sheetsAppend(ctx, args)
	case "tasks_list":
		return h.tasksList(ctx, args)
	case "tasks_create":
		return h.tasksCreate(ctx, args)
	case "contacts_search":
		return h.contactsSearch(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// --- Tool implementations ---

func (h *GWXHandler) gmailList(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewGmailService(h.client)
	limit := intArg(args, "limit", 10)
	unread := boolArg(args, "unread")
	var labels []string
	if l := strArg(args, "label"); l != "" {
		labels = []string{l}
	}
	messages, total, err := svc.ListMessages(ctx, "", labels, int64(limit), unread)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"messages": messages, "count": len(messages), "total_estimate": total})
}

func (h *GWXHandler) gmailGet(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewGmailService(h.client)
	msg, err := svc.GetMessage(ctx, strArg(args, "message_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(msg)
}

func (h *GWXHandler) gmailSearch(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewGmailService(h.client)
	messages, total, err := svc.SearchMessages(ctx, strArg(args, "query"), int64(intArg(args, "limit", 10)))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"messages": messages, "count": len(messages), "total_estimate": total})
}

func (h *GWXHandler) gmailSend(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewGmailService(h.client)
	input := &api.SendInput{
		To:      splitArg(args, "to"),
		Subject: strArg(args, "subject"),
		Body:    strArg(args, "body"),
		CC:      splitArg(args, "cc"),
	}
	result, err := svc.SendMessage(ctx, input)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"sent": true, "message_id": result.MessageID, "thread_id": result.ThreadID})
}

func (h *GWXHandler) calendarAgenda(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewCalendarService(h.client)
	days := intArg(args, "days", 1)
	events, err := svc.Agenda(ctx, days)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"events": events, "count": len(events), "days": days})
}

func (h *GWXHandler) calendarCreate(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewCalendarService(h.client)
	input := &api.EventInput{
		Title:     strArg(args, "title"),
		Start:     strArg(args, "start"),
		End:       strArg(args, "end"),
		Location:  strArg(args, "location"),
		Attendees: splitArg(args, "attendees"),
	}
	event, err := svc.CreateEvent(ctx, "primary", input)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"created": true, "event": event})
}

func (h *GWXHandler) driveList(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDriveService(h.client)
	files, err := svc.ListFiles(ctx, strArg(args, "folder"), int64(intArg(args, "limit", 20)))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"files": files, "count": len(files)})
}

func (h *GWXHandler) driveSearch(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDriveService(h.client)
	files, err := svc.SearchFiles(ctx, strArg(args, "query"), int64(intArg(args, "limit", 20)))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"files": files, "count": len(files)})
}

func (h *GWXHandler) docsGet(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewDocsService(h.client)
	doc, err := svc.GetDocument(ctx, strArg(args, "doc_id"))
	if err != nil {
		return nil, err
	}
	return jsonResult(doc)
}

func (h *GWXHandler) sheetsRead(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	data, err := svc.ReadRange(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"))
	if err != nil {
		return nil, err
	}
	return jsonResult(data)
}

func (h *GWXHandler) sheetsAppend(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	values, err := api.ParseValuesJSON(strArg(args, "values"))
	if err != nil {
		return nil, err
	}
	result, err := svc.AppendValues(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"), values)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"appended": true, "result": result})
}

func (h *GWXHandler) tasksList(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewTasksService(h.client)
	items, err := svc.ListTasks(ctx, strArg(args, "list_id"), boolArg(args, "show_completed"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"tasks": items, "count": len(items)})
}

func (h *GWXHandler) tasksCreate(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewTasksService(h.client)
	item, err := svc.CreateTask(ctx, strArg(args, "list_id"), strArg(args, "title"), strArg(args, "notes"), strArg(args, "due"))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"created": true, "task": item})
}

func (h *GWXHandler) contactsSearch(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewContactsService(h.client)
	contacts, err := svc.SearchContacts(ctx, strArg(args, "query"), intArg(args, "limit", 20))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"contacts": contacts, "count": len(contacts)})
}

// --- Helpers ---

func jsonResult(data interface{}) (*ToolResult, error) {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}
	return &ToolResult{
		Content: []ContentBlock{{Type: "text", Text: string(raw)}},
	}, nil
}

func strArg(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intArg(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
}

func boolArg(args map[string]interface{}, key string) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func splitArg(args map[string]interface{}, key string) []string {
	s := strArg(args, key)
	if s == "" {
		return nil
	}
	var parts []string
	for _, p := range splitComma(s) {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, trimSpace(s[start:i]))
			start = i + 1
		}
	}
	result = append(result, trimSpace(s[start:]))
	return result
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && s[i] == ' ' {
		i++
	}
	for j > i && s[j-1] == ' ' {
		j--
	}
	return s[i:j]
}

