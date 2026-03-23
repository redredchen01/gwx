package testutil

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/mcp"
)

// --- Mock API unit tests ---

func TestMockAPI_GmailListMessages(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	svc := api.NewGmailService(client)

	messages, total, err := svc.ListMessages(t.Context(), "", nil, 10, false)
	if err != nil {
		t.Fatalf("ListMessages error: %v", err)
	}

	if total != 1 {
		t.Errorf("expected total=1, got %d", total)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].ID != "msg_001" {
		t.Errorf("expected message ID msg_001, got %s", messages[0].ID)
	}
	if messages[0].Subject != "Q2 Roadmap Sync" {
		t.Errorf("expected subject 'Q2 Roadmap Sync', got %q", messages[0].Subject)
	}
	if messages[0].From != "alice@example.com" {
		t.Errorf("expected from alice@example.com, got %q", messages[0].From)
	}
}

func TestMockAPI_GmailGetMessage(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	svc := api.NewGmailService(client)

	msg, err := svc.GetMessage(t.Context(), "msg_001")
	if err != nil {
		t.Fatalf("GetMessage error: %v", err)
	}

	if msg.ID != "msg_001" {
		t.Errorf("expected ID msg_001, got %s", msg.ID)
	}
	if msg.Subject != "Q2 Roadmap Sync" {
		t.Errorf("expected subject 'Q2 Roadmap Sync', got %q", msg.Subject)
	}
}

func TestMockAPI_CalendarAgenda(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	svc := api.NewCalendarService(client)

	events, err := svc.Agenda(t.Context(), 1)
	if err != nil {
		t.Fatalf("Agenda error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ID != "evt_001" {
		t.Errorf("expected event ID evt_001, got %s", events[0].ID)
	}
	if events[0].Title != "Sprint Planning" {
		t.Errorf("expected title 'Sprint Planning', got %q", events[0].Title)
	}
}

func TestMockAPI_DriveListFiles(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	svc := api.NewDriveService(client)

	files, err := svc.ListFiles(t.Context(), "", 20)
	if err != nil {
		t.Fatalf("ListFiles error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].ID != "file_001" {
		t.Errorf("expected file ID file_001, got %s", files[0].ID)
	}
	if files[0].Name != "Q2 Roadmap.docx" {
		t.Errorf("expected name 'Q2 Roadmap.docx', got %q", files[0].Name)
	}
}

func TestMockAPI_MultipleDriveFiles(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	mock.DriveFiles = []map[string]interface{}{
		SampleDriveFile(),
		SampleDriveFile2(),
	}

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	svc := api.NewDriveService(client)

	files, err := svc.ListFiles(t.Context(), "", 20)
	if err != nil {
		t.Fatalf("ListFiles error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestMockAPI_SheetsReadRange(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	svc := api.NewSheetsService(client)

	data, err := svc.ReadRange(t.Context(), "sheet_001", "Tasks!A1:C3")
	if err != nil {
		t.Fatalf("ReadRange error: %v", err)
	}

	if data.RowCount != 3 {
		t.Errorf("expected 3 rows, got %d", data.RowCount)
	}
	if data.Range != "Tasks!A1:C3" {
		t.Errorf("expected range 'Tasks!A1:C3', got %q", data.Range)
	}
}

func TestMockAPI_RequestRecording(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	svc := api.NewDriveService(client)

	mock.ResetRequests()
	_, _ = svc.ListFiles(t.Context(), "", 20)

	reqs := mock.Requests()
	if len(reqs) == 0 {
		t.Fatal("expected at least 1 recorded request")
	}

	found := false
	for _, r := range reqs {
		// Drive SDK uses relative path /files (not /drive/v3/files) with option.WithEndpoint
		if r.Path == "/files" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a request to /files, got: %+v", reqs)
	}
}

// --- MCP Tool E2E tests ---

func TestE2E_GmailList(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	handler := mcp.NewGWXHandler(client)

	result, err := handler.CallTool("gmail_list", map[string]interface{}{
		"limit": float64(10),
	})
	if err != nil {
		t.Fatalf("CallTool gmail_list error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}

	var data map[string]interface{}
	if err := ParseToolResultJSON(result, &data); err != nil {
		t.Fatalf("parse result JSON: %v", err)
	}

	count, ok := data["count"].(float64)
	if !ok || count < 1 {
		t.Errorf("expected count >= 1, got %v", data["count"])
	}

	messages, ok := data["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		t.Fatalf("expected messages array with entries, got %v", data["messages"])
	}

	firstMsg, ok := messages[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected message to be a map, got %T", messages[0])
	}
	if firstMsg["id"] != "msg_001" {
		t.Errorf("expected first message id=msg_001, got %v", firstMsg["id"])
	}
}

func TestE2E_CalendarAgenda(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	handler := mcp.NewGWXHandler(client)

	result, err := handler.CallTool("calendar_agenda", map[string]interface{}{
		"days": float64(1),
	})
	if err != nil {
		t.Fatalf("CallTool calendar_agenda error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}

	var data map[string]interface{}
	if err := ParseToolResultJSON(result, &data); err != nil {
		t.Fatalf("parse result JSON: %v", err)
	}

	events, ok := data["events"].([]interface{})
	if !ok {
		t.Fatalf("expected events array, got %T (%v)", data["events"], data["events"])
	}
	if len(events) == 0 {
		t.Fatal("expected at least 1 event")
	}

	firstEvent, ok := events[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected event to be a map, got %T", events[0])
	}
	if firstEvent["id"] != "evt_001" {
		t.Errorf("expected event id=evt_001, got %v", firstEvent["id"])
	}
	if firstEvent["title"] != "Sprint Planning" {
		t.Errorf("expected title='Sprint Planning', got %v", firstEvent["title"])
	}
}

func TestE2E_DriveList(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	handler := mcp.NewGWXHandler(client)

	result, err := handler.CallTool("drive_list", map[string]interface{}{
		"limit": float64(20),
	})
	if err != nil {
		t.Fatalf("CallTool drive_list error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}

	var data map[string]interface{}
	if err := ParseToolResultJSON(result, &data); err != nil {
		t.Fatalf("parse result JSON: %v", err)
	}

	files, ok := data["files"].([]interface{})
	if !ok {
		t.Fatalf("expected files array, got %T (%v)", data["files"], data["files"])
	}
	if len(files) == 0 {
		t.Fatal("expected at least 1 file")
	}

	firstFile, ok := files[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected file to be a map, got %T", files[0])
	}
	if firstFile["id"] != "file_001" {
		t.Errorf("expected file id=file_001, got %v", firstFile["id"])
	}
}

func TestE2E_SheetsRead(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	handler := mcp.NewGWXHandler(client)

	result, err := handler.CallTool("sheets_read", map[string]interface{}{
		"spreadsheet_id": "sheet_001",
		"range":          "Tasks!A1:C3",
	})
	if err != nil {
		t.Fatalf("CallTool sheets_read error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}

	var data map[string]interface{}
	if err := ParseToolResultJSON(result, &data); err != nil {
		t.Fatalf("parse result JSON: %v", err)
	}

	rowCount, ok := data["row_count"].(float64)
	if !ok || rowCount != 3 {
		t.Errorf("expected row_count=3, got %v", data["row_count"])
	}
}

func TestE2E_UnknownTool(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	handler := mcp.NewGWXHandler(client)

	_, err := handler.CallTool("nonexistent_tool", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("expected 'unknown tool' error, got: %v", err)
	}
}

func TestE2E_CustomMockData(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	// Override with custom data
	mock.DriveFiles = []map[string]interface{}{
		SampleDriveFile(),
		SampleDriveFile2(),
	}
	mock.CalendarEvents = []map[string]interface{}{
		SampleCalendarEvent(),
		SampleCalendarEvent2(),
	}

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	handler := mcp.NewGWXHandler(client)

	// Drive should return 2 files
	driveResult, err := handler.CallTool("drive_list", nil)
	if err != nil {
		t.Fatalf("drive_list error: %v", err)
	}
	var driveData map[string]interface{}
	ParseToolResultJSON(driveResult, &driveData)
	if driveData["count"] != float64(2) {
		t.Errorf("expected 2 drive files, got %v", driveData["count"])
	}

	// Calendar should return 2 events
	calResult, err := handler.CallTool("calendar_agenda", map[string]interface{}{"days": float64(1)})
	if err != nil {
		t.Fatalf("calendar_agenda error: %v", err)
	}
	var calData map[string]interface{}
	ParseToolResultJSON(calResult, &calData)
	count, _ := calData["count"].(float64)
	if count != 2 {
		t.Errorf("expected 2 calendar events, got %v", calData["count"])
	}
}

// --- MCP Protocol E2E tests ---

func TestMCPProtocol_Initialize(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	handler := mcp.NewGWXHandler(client)

	mcpClient := NewMCPClient(handler)
	defer mcpClient.Close()

	result, err := mcpClient.Initialize()
	if err != nil {
		t.Fatalf("Initialize error: %v", err)
	}

	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("expected protocol version 2024-11-05, got %s", result.ProtocolVersion)
	}
	if result.ServerInfo.Name != "gwx" {
		t.Errorf("expected server name gwx, got %s", result.ServerInfo.Name)
	}
	if result.ServerInfo.Version != "0.8.0" {
		t.Errorf("expected version 0.8.0, got %s", result.ServerInfo.Version)
	}
	if result.Capabilities.Tools == nil {
		t.Error("expected tools capability to be present")
	}
}

func TestMCPProtocol_ListTools(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	handler := mcp.NewGWXHandler(client)

	mcpClient := NewMCPClient(handler)
	defer mcpClient.Close()

	tools, err := mcpClient.ListTools()
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}

	if len(tools) == 0 {
		t.Fatal("expected at least 1 tool")
	}

	// Verify well-known tools exist
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	requiredTools := []string{
		"gmail_list", "gmail_get", "gmail_search",
		"calendar_agenda", "calendar_create",
		"drive_list", "drive_search",
		"sheets_read",
	}
	for _, name := range requiredTools {
		if !toolNames[name] {
			t.Errorf("missing required tool: %s", name)
		}
	}

	// All tools should have description and object input schema
	for _, tool := range tools {
		if tool.Description == "" {
			t.Errorf("tool %s has empty description", tool.Name)
		}
		if tool.InputSchema.Type != "object" {
			t.Errorf("tool %s has InputSchema.Type=%q, want 'object'", tool.Name, tool.InputSchema.Type)
		}
	}
}

func TestMCPProtocol_CallTool(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	handler := mcp.NewGWXHandler(client)

	mcpClient := NewMCPClient(handler)
	defer mcpClient.Close()

	// Initialize first
	if _, err := mcpClient.Initialize(); err != nil {
		t.Fatalf("Initialize error: %v", err)
	}

	// Call drive_list through the protocol
	result, err := mcpClient.CallTool("drive_list", map[string]interface{}{
		"limit": float64(10),
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}

	var data map[string]interface{}
	if err := ParseToolResultJSON(result, &data); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if data["files"] == nil {
		t.Error("expected files in result")
	}
}

func TestMCPProtocol_CallTool_Gmail(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	handler := mcp.NewGWXHandler(client)

	mcpClient := NewMCPClient(handler)
	defer mcpClient.Close()

	result, err := mcpClient.CallTool("gmail_list", map[string]interface{}{
		"limit": float64(5),
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %s", result.Content[0].Text)
	}

	var data map[string]interface{}
	if err := ParseToolResultJSON(result, &data); err != nil {
		t.Fatalf("parse result: %v", err)
	}

	messages, ok := data["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		t.Fatal("expected messages in result")
	}
}

func TestMCPProtocol_Ping(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	handler := mcp.NewGWXHandler(client)

	mcpClient := NewMCPClient(handler)
	defer mcpClient.Close()

	if err := mcpClient.Ping(); err != nil {
		t.Fatalf("Ping error: %v", err)
	}
}

func TestMCPProtocol_FullHandshake(t *testing.T) {
	mock := NewMockAPI()
	defer mock.Close()

	client := api.NewTestClient(mock.HTTPClient(), mock.URL())
	handler := mcp.NewGWXHandler(client)

	mcpClient := NewMCPClient(handler)
	defer mcpClient.Close()

	// Step 1: Initialize
	initResult, err := mcpClient.Initialize()
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if initResult.ServerInfo.Name != "gwx" {
		t.Fatalf("unexpected server name: %s", initResult.ServerInfo.Name)
	}

	// Step 2: List tools
	tools, err := mcpClient.ListTools()
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) < 10 {
		t.Fatalf("expected 10+ tools, got %d", len(tools))
	}

	// Step 3: Call a tool
	result, err := mcpClient.CallTool("gmail_list", map[string]interface{}{
		"limit": float64(5),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", result.Content[0].Text)
	}

	// Step 4: Parse and verify
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &data); err != nil {
		t.Fatalf("parse result JSON: %v", err)
	}
	if data["messages"] == nil {
		t.Error("expected messages in result")
	}
}

// --- Fixture tests ---

func TestFixtures_SampleGmailMessage(t *testing.T) {
	msg := SampleGmailMessage()
	if msg["id"] != "msg_001" {
		t.Errorf("expected id=msg_001, got %v", msg["id"])
	}
	payload, ok := msg["payload"].(map[string]interface{})
	if !ok {
		t.Fatal("expected payload map")
	}
	headers, ok := payload["headers"].([]interface{})
	if !ok || len(headers) < 4 {
		t.Fatal("expected at least 4 headers")
	}
}

func TestFixtures_SampleCalendarEvent(t *testing.T) {
	evt := SampleCalendarEvent()
	if evt["id"] != "evt_001" {
		t.Errorf("expected id=evt_001, got %v", evt["id"])
	}
	if evt["summary"] != "Sprint Planning" {
		t.Errorf("expected summary=Sprint Planning, got %v", evt["summary"])
	}
	start, ok := evt["start"].(map[string]interface{})
	if !ok || start["dateTime"] == "" {
		t.Error("expected start.dateTime")
	}
}

func TestFixtures_SampleDriveFile(t *testing.T) {
	f := SampleDriveFile()
	if f["id"] != "file_001" {
		t.Errorf("expected id=file_001, got %v", f["id"])
	}
	if f["name"] != "Q2 Roadmap.docx" {
		t.Errorf("expected name=Q2 Roadmap.docx, got %v", f["name"])
	}
}

func TestFixtures_SampleSpreadsheet(t *testing.T) {
	ss := SampleSpreadsheet()
	if ss["spreadsheetId"] != "sheet_001" {
		t.Errorf("expected spreadsheetId=sheet_001, got %v", ss["spreadsheetId"])
	}
	sheets, ok := ss["sheets"].([]interface{})
	if !ok || len(sheets) != 2 {
		t.Errorf("expected 2 sheets, got %v", ss["sheets"])
	}
}

func TestFixtures_SampleSpreadsheetValues(t *testing.T) {
	sv := SampleSpreadsheetValues()
	values, ok := sv["values"].([]interface{})
	if !ok || len(values) != 3 {
		t.Errorf("expected 3 rows, got %v", sv["values"])
	}
}

// --- ParseToolResultJSON test ---

func TestParseToolResultJSON_Valid(t *testing.T) {
	result := &mcp.ToolResult{
		Content: []mcp.ContentBlock{{Type: "text", Text: `{"count":42}`}},
	}
	var data map[string]interface{}
	if err := ParseToolResultJSON(result, &data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["count"] != float64(42) {
		t.Errorf("expected count=42, got %v", data["count"])
	}
}

func TestParseToolResultJSON_Empty(t *testing.T) {
	result := &mcp.ToolResult{Content: []mcp.ContentBlock{}}
	var data map[string]interface{}
	err := ParseToolResultJSON(result, &data)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}
