package mcp

type coreProvider struct{}

func (coreProvider) Tools() []Tool {
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
				Type: "object",
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
}

func (coreProvider) Handlers(h *GWXHandler) map[string]ToolHandler {
	return map[string]ToolHandler{
		"gmail_list":          h.gmailList,
		"gmail_get":           h.gmailGet,
		"gmail_search":        h.gmailSearch,
		"gmail_send":          h.gmailSend,
		"calendar_agenda":     h.calendarAgenda,
		"calendar_create":     h.calendarCreate,
		"drive_list":          h.driveList,
		"drive_search":        h.driveSearch,
		"docs_get":            h.docsGet,
		"sheets_read":         h.sheetsRead,
		"sheets_append":       h.sheetsAppend,
		"sheets_describe":     h.sheetsDescribe,
		"sheets_smart_append": h.sheetsSmartAppend,
		"sheets_search":       h.sheetsSearch,
		"sheets_filter":       h.sheetsFilter,
		"tasks_list":          h.tasksList,
		"tasks_create":        h.tasksCreate,
		"contacts_search":     h.contactsSearch,
		"gmail_digest":        h.gmailDigest,
		"gmail_archive":       h.gmailArchive,
		"context_gather":      h.contextGather,
		"unified_search":      h.unifiedSearch,
	}
}

func init() { RegisterProvider(coreProvider{}) }
