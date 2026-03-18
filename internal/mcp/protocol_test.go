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
	if result.ServerInfo.Version != "0.8.0" {
		t.Errorf("expected version 0.8.0, got %s", result.ServerInfo.Version)
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

// java_style_error is a simple error for testing.
type java_style_error string

func (e java_style_error) Error() string { return string(e) }
