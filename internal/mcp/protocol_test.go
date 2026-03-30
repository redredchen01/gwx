package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// mockHandler implements Handler for testing.
type mockHandler struct {
	tools     []Tool
	callFn    func(name string, args map[string]interface{}) (*ToolResult, error)
}

func (m *mockHandler) ListTools() []Tool { return m.tools }

func (m *mockHandler) CallTool(name string, args map[string]interface{}) (*ToolResult, error) {
	if m.callFn != nil {
		return m.callFn(name, args)
	}
	return &ToolResult{Content: []ContentBlock{{Type: "text", Text: "ok"}}}, nil
}

func newTestServer(h Handler) (*Server, *bytes.Buffer) {
	var out bytes.Buffer
	s := &Server{
		handler: h,
		reader:  bufio.NewReader(strings.NewReader("")),
		writer:  &out,
	}
	return s, &out
}

func parseResponse(t *testing.T, data []byte) Response {
	t.Helper()
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to parse response: %v\nraw: %s", err, data)
	}
	return resp
}

func TestHandleRequest_Initialize(t *testing.T) {
	s, out := newTestServer(&mockHandler{})
	req := &Request{JSONRPC: "2.0", ID: float64(1), Method: "initialize"}
	s.handleRequest(req)

	resp := parseResponse(t, out.Bytes())
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	raw, _ := json.Marshal(resp.Result)
	var result InitializeResult
	json.Unmarshal(raw, &result)

	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("expected protocol version 2024-11-05, got %s", result.ProtocolVersion)
	}
	if result.ServerInfo.Name != "gwx" {
		t.Errorf("expected server name gwx, got %s", result.ServerInfo.Name)
	}
	if result.ServerInfo.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", result.ServerInfo.Version)
	}
}

func TestHandleRequest_ToolsList(t *testing.T) {
	tools := []Tool{
		{Name: "test_tool", Description: "A test tool", InputSchema: InputSchema{Type: "object"}},
	}
	s, out := newTestServer(&mockHandler{tools: tools})
	req := &Request{JSONRPC: "2.0", ID: float64(2), Method: "tools/list"}
	s.handleRequest(req)

	resp := parseResponse(t, out.Bytes())
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	raw, _ := json.Marshal(resp.Result)
	var result ListToolsResult
	json.Unmarshal(raw, &result)

	if len(result.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result.Tools))
	}
	if result.Tools[0].Name != "test_tool" {
		t.Errorf("expected tool name test_tool, got %s", result.Tools[0].Name)
	}
}

func TestHandleRequest_ToolsCall(t *testing.T) {
	h := &mockHandler{
		callFn: func(name string, args map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{
				Content: []ContentBlock{{Type: "text", Text: "hello " + name}},
			}, nil
		},
	}
	s, out := newTestServer(h)

	params, _ := json.Marshal(ToolCallParams{Name: "greet", Arguments: map[string]interface{}{"x": "y"}})
	req := &Request{JSONRPC: "2.0", ID: float64(3), Method: "tools/call", Params: params}
	s.handleRequest(req)

	resp := parseResponse(t, out.Bytes())
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	raw, _ := json.Marshal(resp.Result)
	var result ToolResult
	json.Unmarshal(raw, &result)

	if len(result.Content) != 1 || result.Content[0].Text != "hello greet" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestHandleRequest_ToolsCallError(t *testing.T) {
	h := &mockHandler{
		callFn: func(name string, args map[string]interface{}) (*ToolResult, error) {
			return nil, java_style_error("something broke")
		},
	}
	s, out := newTestServer(h)

	params, _ := json.Marshal(ToolCallParams{Name: "fail"})
	req := &Request{JSONRPC: "2.0", ID: float64(4), Method: "tools/call", Params: params}
	s.handleRequest(req)

	resp := parseResponse(t, out.Bytes())
	if resp.Error != nil {
		t.Fatalf("tool errors should be in result, not RPC error")
	}

	raw, _ := json.Marshal(resp.Result)
	var result ToolResult
	json.Unmarshal(raw, &result)

	if !result.IsError {
		t.Error("expected IsError=true")
	}
	if len(result.Content) == 0 || result.Content[0].Text != "something broke" {
		t.Errorf("expected error text 'something broke', got %+v", result)
	}
}

func TestHandleRequest_Ping(t *testing.T) {
	s, out := newTestServer(&mockHandler{})
	req := &Request{JSONRPC: "2.0", ID: float64(5), Method: "ping"}
	s.handleRequest(req)

	resp := parseResponse(t, out.Bytes())
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestHandleRequest_UnknownMethod(t *testing.T) {
	s, out := newTestServer(&mockHandler{})
	req := &Request{JSONRPC: "2.0", ID: float64(6), Method: "unknown/method"}
	s.handleRequest(req)

	resp := parseResponse(t, out.Bytes())
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestHandleRequest_InvalidToolsCallParams(t *testing.T) {
	s, out := newTestServer(&mockHandler{})
	req := &Request{JSONRPC: "2.0", ID: float64(7), Method: "tools/call", Params: json.RawMessage(`{invalid`)}
	s.handleRequest(req)

	resp := parseResponse(t, out.Bytes())
	if resp.Error == nil {
		t.Fatal("expected error for invalid params")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("expected error code -32602, got %d", resp.Error.Code)
	}
}

func TestHandleRequest_NotificationNoResponse(t *testing.T) {
	s, out := newTestServer(&mockHandler{})
	req := &Request{JSONRPC: "2.0", Method: "notifications/initialized"}
	s.handleRequest(req)

	if out.Len() != 0 {
		t.Errorf("notifications should not produce output, got: %s", out.String())
	}
}

func TestRun_ParsesMultipleMessages(t *testing.T) {
	ping1, _ := json.Marshal(Request{JSONRPC: "2.0", ID: float64(1), Method: "ping"})
	ping2, _ := json.Marshal(Request{JSONRPC: "2.0", ID: float64(2), Method: "ping"})
	input := string(ping1) + "\n" + string(ping2) + "\n"

	var out bytes.Buffer
	s := &Server{
		handler: &mockHandler{},
		reader:  bufio.NewReader(strings.NewReader(input)),
		writer:  &out,
	}

	if err := s.Run(); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 response lines, got %d: %v", len(lines), lines)
	}
}

func TestRun_SkipsEmptyLines(t *testing.T) {
	ping, _ := json.Marshal(Request{JSONRPC: "2.0", ID: float64(1), Method: "ping"})
	input := "\n\n" + string(ping) + "\n\n"

	var out bytes.Buffer
	s := &Server{
		handler: &mockHandler{},
		reader:  bufio.NewReader(strings.NewReader(input)),
		writer:  &out,
	}

	if err := s.Run(); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 response, got %d", len(lines))
	}
}

func TestRun_InvalidJSON(t *testing.T) {
	input := "not json\n"

	var out bytes.Buffer
	s := &Server{
		handler: &mockHandler{},
		reader:  bufio.NewReader(strings.NewReader(input)),
		writer:  &out,
	}

	s.Run()

	resp := parseResponse(t, out.Bytes())
	if resp.Error == nil || resp.Error.Code != -32700 {
		t.Errorf("expected parse error (-32700), got: %+v", resp.Error)
	}
}

// --- JSON roundtrip tests for protocol types ---

func TestRequest_JSONRoundtrip(t *testing.T) {
	original := Request{
		JSONRPC: "2.0",
		ID:      float64(42),
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"gmail_list"}`),
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", decoded.JSONRPC)
	}
	if decoded.Method != "tools/call" {
		t.Errorf("Method = %q, want tools/call", decoded.Method)
	}
	if string(decoded.Params) != `{"name":"gmail_list"}` {
		t.Errorf("Params = %s, want {\"name\":\"gmail_list\"}", decoded.Params)
	}
}

func TestResponse_JSONRoundtrip_Success(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      float64(1),
		Result:  map[string]string{"status": "ok"},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Error != nil {
		t.Error("expected nil Error for success response")
	}
	if decoded.Result == nil {
		t.Error("expected non-nil Result")
	}
}

func TestResponse_JSONRoundtrip_Error(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      float64(1),
		Error:   &RPCError{Code: -32601, Message: "Method not found", Data: "bad_method"},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Response
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Error == nil {
		t.Fatal("expected non-nil Error")
	}
	if decoded.Error.Code != -32601 {
		t.Errorf("Error.Code = %d, want -32601", decoded.Error.Code)
	}
	if decoded.Error.Message != "Method not found" {
		t.Errorf("Error.Message = %q, want 'Method not found'", decoded.Error.Message)
	}
}

func TestTool_JSONSerialization(t *testing.T) {
	tool := Tool{
		Name:        "gmail_send",
		Description: "Send an email",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"to":      {Type: "string", Description: "Recipient"},
				"subject": {Type: "string", Description: "Subject line"},
			},
			Required: []string{"to", "subject"},
		},
	}

	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Tool
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Name != "gmail_send" {
		t.Errorf("Name = %q, want gmail_send", decoded.Name)
	}
	if decoded.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type = %q, want object", decoded.InputSchema.Type)
	}
	if len(decoded.InputSchema.Properties) != 2 {
		t.Errorf("Properties count = %d, want 2", len(decoded.InputSchema.Properties))
	}
	if len(decoded.InputSchema.Required) != 2 {
		t.Errorf("Required count = %d, want 2", len(decoded.InputSchema.Required))
	}
	toProp, ok := decoded.InputSchema.Properties["to"]
	if !ok {
		t.Fatal("missing 'to' property")
	}
	if toProp.Type != "string" {
		t.Errorf("to.Type = %q, want string", toProp.Type)
	}
}

func TestToolResult_JSONSerialization(t *testing.T) {
	tr := ToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: `{"count": 5}`},
		},
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ToolResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.IsError {
		t.Error("IsError should be false")
	}
	if len(decoded.Content) != 1 {
		t.Fatalf("Content count = %d, want 1", len(decoded.Content))
	}
	if decoded.Content[0].Type != "text" {
		t.Errorf("Content[0].Type = %q, want text", decoded.Content[0].Type)
	}
	if decoded.Content[0].Text != `{"count": 5}` {
		t.Errorf("Content[0].Text = %q, unexpected", decoded.Content[0].Text)
	}
}

func TestToolResult_IsError_JSONSerialization(t *testing.T) {
	tr := ToolResult{
		Content: []ContentBlock{{Type: "text", Text: "something failed"}},
		IsError: true,
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ToolResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !decoded.IsError {
		t.Error("IsError should be true")
	}
}

func TestToolCallParams_JSONRoundtrip(t *testing.T) {
	p := ToolCallParams{
		Name: "gmail_search",
		Arguments: map[string]interface{}{
			"query": "from:boss",
			"limit": float64(5),
		},
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ToolCallParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Name != "gmail_search" {
		t.Errorf("Name = %q, want gmail_search", decoded.Name)
	}
	if decoded.Arguments["query"] != "from:boss" {
		t.Errorf("Arguments[query] = %v, want from:boss", decoded.Arguments["query"])
	}
}

func TestNewServer_NotNil(t *testing.T) {
	// NewServer uses os.Stdin/os.Stdout, so we just verify it doesn't panic
	// and returns a non-nil server.
	s := NewServer(&mockHandler{})
	if s == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestProperty_EnumSerialization(t *testing.T) {
	p := Property{
		Type:        "string",
		Description: "Output format",
		Enum:        []string{"json", "text", "table"},
		Default:     "json",
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded Property
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded.Enum) != 3 {
		t.Errorf("Enum count = %d, want 3", len(decoded.Enum))
	}
	if decoded.Default != "json" {
		t.Errorf("Default = %q, want json", decoded.Default)
	}
}

func TestInitializeResult_JSONRoundtrip(t *testing.T) {
	ir := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{ListChanged: true},
		},
		ServerInfo: ServerInfo{Name: "gwx", Version: "0.8.0"},
	}
	data, err := json.Marshal(ir)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded InitializeResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ProtocolVersion != "2024-11-05" {
		t.Errorf("ProtocolVersion = %q", decoded.ProtocolVersion)
	}
	if decoded.Capabilities.Tools == nil {
		t.Fatal("Tools capability nil")
	}
	if decoded.ServerInfo.Name != "gwx" {
		t.Errorf("ServerInfo.Name = %q", decoded.ServerInfo.Name)
	}
}

// java_style_error is a simple error for testing.
type java_style_error string

func (e java_style_error) Error() string { return string(e) }
