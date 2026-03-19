package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redredchen01/gwx/internal/api"
)

// GWXHandler implements the MCP Handler interface for Google Workspace tools.
type GWXHandler struct {
	client   *api.Client
	registry map[string]ToolHandler // lazy-init tool dispatch map
}

// NewGWXHandler creates a handler with an authenticated API client.
func NewGWXHandler(client *api.Client) *GWXHandler {
	return &GWXHandler{client: client}
}

func (h *GWXHandler) ListTools() []Tool {
	tools := []Tool{
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
		{
			Name:        "sheets_describe",
			Description: "Analyze a spreadsheet's column structure: header names, data types (enum/text/url/number), required vs optional, sample values. Use this BEFORE writing data to understand what each column expects.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"range":          {Type: "string", Description: "Sheet range (auto-detects first sheet if empty)"},
				},
				Required: []string{"spreadsheet_id"},
			},
		},
		{
			Name:        "sheets_smart_append",
			Description: "Validate data against sheet structure then append. First analyzes columns (types, required fields, enum values), validates proposed rows, then writes. Returns validation errors if data doesn't match structure.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"range":          {Type: "string", Description: "Range to append to (e.g. Sheet1!A:F)"},
					"values":         {Type: "string", Description: "JSON array of rows"},
				},
				Required: []string{"spreadsheet_id", "range", "values"},
			},
		},
		{
			Name:        "sheets_search",
			Description: "Search for text across all cells in a spreadsheet range. Returns matching rows with row/column indices.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"query":          {Type: "string", Description: "Text to search for"},
					"range":          {Type: "string", Description: "Range to search (auto-detects if empty)"},
				},
				Required: []string{"spreadsheet_id", "query"},
			},
		},
		{
			Name:        "sheets_filter",
			Description: "Filter rows where a specific column matches a value (like SQL WHERE). Returns header + matching rows.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"spreadsheet_id": {Type: "string", Description: "Spreadsheet ID"},
					"range":          {Type: "string", Description: "Range (e.g. Sheet1!A:F)"},
					"column":         {Type: "integer", Description: "Column index (0-based)"},
					"value":          {Type: "string", Description: "Value to match"},
				},
				Required: []string{"spreadsheet_id", "range", "column", "value"},
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
		// Digest & Context
		{
			Name:        "gmail_digest",
			Description: "Smart digest of recent emails — groups by sender, categorizes CI/transactional/personal, generates summary.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit":  {Type: "integer", Description: "Max messages to analyze (default 30)"},
					"unread": {Type: "boolean", Description: "Only unread messages"},
				},
			},
		},
		{
			Name:        "gmail_archive",
			Description: "Batch archive messages matching a query. CAUTION: Modifies mailbox.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query":     {Type: "string", Description: "Gmail search query for messages to archive"},
					"limit":     {Type: "integer", Description: "Max messages to archive (default 50)"},
					"read_only": {Type: "boolean", Description: "Only mark as read without archiving"},
				},
				Required: []string{"query"},
			},
		},
		{
			Name:        "context_gather",
			Description: "Gather all context for a topic — searches Gmail, Drive, and Calendar in parallel and returns unified results.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"topic": {Type: "string", Description: "Topic or keyword to gather context for"},
					"days":  {Type: "integer", Description: "Days of calendar events to include (default 7)"},
					"limit": {Type: "integer", Description: "Max results per service (default 5)"},
				},
				Required: []string{"topic"},
			},
		},
		{
			Name:        "unified_search",
			Description: "Search across Gmail + Drive simultaneously. Returns combined results.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {Type: "string", Description: "Search query"},
					"limit": {Type: "integer", Description: "Max results per service (default 5)"},
				},
				Required: []string{"query"},
			},
		},
	}
	// Append extended tools
	tools = append(tools, ExtendedTools()...)
	// Append new tools (v0.8.0)
	tools = append(tools, NewTools()...)
	// Append batch tools (v0.8.0)
	tools = append(tools, BatchTools()...)
	// Append analytics tools (v0.8.0)
	tools = append(tools, AnalyticsTools()...)
	// Append Search Console tools (v0.8.0)
	tools = append(tools, SearchConsoleTools()...)
	// Append config tools (v0.8.0)
	tools = append(tools, ConfigTools()...)
	// Append workflow tools
	tools = append(tools, WorkflowTools()...)
	return tools
}

// ToolHandler is a function that handles a tool call.
type ToolHandler func(ctx context.Context, args map[string]interface{}) (*ToolResult, error)

// buildRegistry builds a map of tool name → handler for O(1) dispatch.
// Called once lazily on first CallTool.
func (h *GWXHandler) buildRegistry() map[string]ToolHandler {
	r := map[string]ToolHandler{
		// Core tools (tools.go)
		"gmail_list":      h.gmailList,
		"gmail_get":       h.gmailGet,
		"gmail_search":    h.gmailSearch,
		"gmail_send":      h.gmailSend,
		"calendar_agenda": h.calendarAgenda,
		"calendar_create": h.calendarCreate,
		"drive_list":      h.driveList,
		"drive_search":    h.driveSearch,
		"docs_get":        h.docsGet,
		"sheets_read":     h.sheetsRead,
		"sheets_append":   h.sheetsAppend,
		"sheets_describe": h.sheetsDescribe,
		"sheets_smart_append": h.sheetsSmartAppend,
		"sheets_search":   h.sheetsSearch,
		"sheets_filter":   h.sheetsFilter,
		"tasks_list":      h.tasksList,
		"tasks_create":    h.tasksCreate,
		"contacts_search": h.contactsSearch,
		"gmail_digest":    h.gmailDigest,
		"gmail_archive":   h.gmailArchive,
		"context_gather":  h.contextGather,
		"unified_search":  h.unifiedSearch,
	}

	// Register extended tools
	h.registerExtendedTools(r)
	// Register new tools
	h.registerNewTools(r)
	// Register batch tools
	h.registerBatchTools(r)
	// Register analytics tools
	h.registerAnalyticsTools(r)
	// Register search console tools
	h.registerSearchConsoleTools(r)
	// Register config tools
	h.registerConfigTools(r)
	// Register workflow tools
	h.registerWorkflowTools(r)

	return r
}

func (h *GWXHandler) CallTool(name string, args map[string]interface{}) (*ToolResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Lazy-init registry
	if h.registry == nil {
		h.registry = h.buildRegistry()
	}

	handler, ok := h.registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, args)
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

func (h *GWXHandler) sheetsDescribe(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	schema, err := svc.DescribeSheet(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"), 20)
	if err != nil {
		return nil, err
	}
	return jsonResult(schema)
}

func (h *GWXHandler) sheetsSmartAppend(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	values, err := api.ParseValuesJSON(strArg(args, "values"))
	if err != nil {
		return nil, err
	}

	schema, err := svc.DescribeSheet(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"), 20)
	if err != nil {
		return nil, err
	}

	// Validate
	allValid := true
	var issues []map[string]interface{}
	for i, row := range values {
		vr := api.ValidateRow(schema, row)
		if !vr.Valid {
			allValid = false
			for _, issue := range vr.Issues {
				issues = append(issues, map[string]interface{}{"row": i, "column": issue.Column, "message": issue.Message})
			}
		}
	}

	if !allValid {
		return jsonResult(map[string]interface{}{"valid": false, "issues": issues, "schema": schema})
	}

	result, err := svc.AppendValues(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"), values)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"valid": true, "appended": true, "result": result})
}

func (h *GWXHandler) sheetsSearch(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	result, err := svc.SearchValues(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"), strArg(args, "query"))
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
}

func (h *GWXHandler) sheetsFilter(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewSheetsService(h.client)
	result, err := svc.FilterRows(ctx, strArg(args, "spreadsheet_id"), strArg(args, "range"), intArg(args, "column", 0), strArg(args, "value"))
	if err != nil {
		return nil, err
	}
	return jsonResult(result)
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

func (h *GWXHandler) gmailDigest(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewGmailService(h.client)
	digest, err := svc.DigestMessages(ctx, int64(intArg(args, "limit", 30)), boolArg(args, "unread"))
	if err != nil {
		return nil, err
	}
	return jsonResult(digest)
}

func (h *GWXHandler) gmailArchive(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	svc := api.NewGmailService(h.client)
	query := strArg(args, "query")
	limit := int64(intArg(args, "limit", 50))

	var count int
	var err error
	if boolArg(args, "read_only") {
		count, err = svc.MarkRead(ctx, query, limit)
	} else {
		count, err = svc.ArchiveMessages(ctx, query, limit)
	}
	if err != nil {
		return nil, err
	}
	action := "archived"
	if boolArg(args, "read_only") {
		action = "marked_read"
	}
	return jsonResult(map[string]interface{}{"action": action, "count": count, "query": query})
}

func (h *GWXHandler) contextGather(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	topic := strArg(args, "topic")
	limit := intArg(args, "limit", 5)
	days := intArg(args, "days", 7)

	type result struct {
		key string
		val interface{}
	}
	ch := make(chan result, 3)

	// Gmail
	go func() {
		svc := api.NewGmailService(h.client)
		msgs, total, err := svc.SearchMessages(ctx, topic, int64(limit))
		if err != nil {
			ch <- result{"emails", map[string]interface{}{"error": err.Error()}}
			return
		}
		ch <- result{"emails", map[string]interface{}{"count": len(msgs), "total_estimate": total, "messages": msgs}}
	}()

	// Drive
	go func() {
		svc := api.NewDriveService(h.client)
		safeTopic := strings.ReplaceAll(topic, "'", "\\'")
		files, err := svc.SearchFiles(ctx, "fullText contains '"+safeTopic+"'", int64(limit))
		if err != nil {
			files, err = svc.SearchFiles(ctx, "name contains '"+safeTopic+"'", int64(limit))
			if err != nil {
				ch <- result{"files", map[string]interface{}{"error": err.Error()}}
				return
			}
		}
		ch <- result{"files", map[string]interface{}{"count": len(files), "files": files}}
	}()

	// Calendar
	go func() {
		svc := api.NewCalendarService(h.client)
		events, err := svc.Agenda(ctx, days)
		if err != nil {
			ch <- result{"events", map[string]interface{}{"error": err.Error()}}
			return
		}
		ch <- result{"events", map[string]interface{}{"count": len(events), "events": events}}
	}()

	data := map[string]interface{}{"topic": topic}
	for i := 0; i < 3; i++ {
		r := <-ch
		data[r.key] = r.val
	}
	return jsonResult(data)
}

func (h *GWXHandler) unifiedSearch(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	query := strArg(args, "query")
	limit := intArg(args, "limit", 5)

	type result struct {
		service string
		data    interface{}
	}
	ch := make(chan result, 2)

	go func() {
		svc := api.NewGmailService(h.client)
		msgs, _, err := svc.SearchMessages(ctx, query, int64(limit))
		if err != nil {
			ch <- result{"gmail", map[string]interface{}{"error": err.Error()}}
			return
		}
		ch <- result{"gmail", map[string]interface{}{"count": len(msgs), "messages": msgs}}
	}()

	go func() {
		svc := api.NewDriveService(h.client)
		safeQuery := strings.ReplaceAll(query, "'", "\\'")
		files, err := svc.SearchFiles(ctx, "fullText contains '"+safeQuery+"'", int64(limit))
		if err != nil {
			ch <- result{"drive", map[string]interface{}{"error": err.Error()}}
			return
		}
		ch <- result{"drive", map[string]interface{}{"count": len(files), "files": files}}
	}()

	results := make(map[string]interface{})
	for i := 0; i < 2; i++ {
		r := <-ch
		results[r.service] = r.data
	}
	return jsonResult(map[string]interface{}{"query": query, "results": results})
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
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

