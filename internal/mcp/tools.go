package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redredchen01/gwx/internal/api"
)

// GWXHandler implements the MCP Handler interface for Google Workspace tools.
type GWXHandler struct {
	client   *api.Client
	registry map[string]ToolHandler // lazy-init tool dispatch map
	once     sync.Once
}

// NewGWXHandler creates a handler with an authenticated API client.
func NewGWXHandler(client *api.Client) *GWXHandler {
	return &GWXHandler{client: client}
}

func (h *GWXHandler) ListTools() []Tool {
	return globalRegistry.allTools()
}

// ToolHandler is a function that handles a tool call.
type ToolHandler func(ctx context.Context, args map[string]interface{}) (*ToolResult, error)

// buildRegistry builds a map of tool name → handler for O(1) dispatch.
// Called once lazily on first CallTool.
func (h *GWXHandler) buildRegistry() map[string]ToolHandler {
	return globalRegistry.buildHandlers(h)
}

func (h *GWXHandler) CallTool(name string, args map[string]interface{}) (*ToolResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Build registry exactly once via sync.Once (thread-safe).
	h.once.Do(func() {
		h.registry = h.buildRegistry()
	})

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

