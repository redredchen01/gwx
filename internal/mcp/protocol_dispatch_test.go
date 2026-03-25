package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// --- Server.dispatch tests ---
// These target the Server.Run path (full stdin→parse→dispatch→stdout)
// to cover Run, handleRequest, sendResult, sendError as called from the
// stdio loop rather than from direct handleRequest calls.

func TestServer_Dispatch_Initialize(t *testing.T) {
	req := Request{JSONRPC: "2.0", ID: float64(1), Method: "initialize"}
	resp := dispatchSingle(t, &mockHandler{}, req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	raw, _ := json.Marshal(resp.Result)
	var result InitializeResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal InitializeResult: %v", err)
	}
	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("ProtocolVersion = %q, want 2024-11-05", result.ProtocolVersion)
	}
	if result.ServerInfo.Name != "gwx" {
		t.Errorf("ServerInfo.Name = %q, want gwx", result.ServerInfo.Name)
	}
	if result.Capabilities.Tools == nil {
		t.Error("Capabilities.Tools should be non-nil")
	}
}

func TestServer_Dispatch_ToolsList(t *testing.T) {
	h := &mockHandler{
		tools: []Tool{
			{Name: "alpha", Description: "A", InputSchema: InputSchema{Type: "object"}},
			{Name: "beta", Description: "B", InputSchema: InputSchema{Type: "object"}},
		},
	}
	resp := dispatchSingle(t, h, Request{JSONRPC: "2.0", ID: float64(2), Method: "tools/list"})

	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	raw, _ := json.Marshal(resp.Result)
	var result ListToolsResult
	json.Unmarshal(raw, &result)

	if len(result.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(result.Tools))
	}
	if result.Tools[0].Name != "alpha" {
		t.Errorf("tools[0].Name = %q, want alpha", result.Tools[0].Name)
	}
}

func TestServer_Dispatch_ToolsCall(t *testing.T) {
	h := &mockHandler{
		callFn: func(name string, args map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{
				Content: []ContentBlock{{Type: "text", Text: "called:" + name}},
			}, nil
		},
	}
	params, _ := json.Marshal(ToolCallParams{Name: "my_tool", Arguments: map[string]interface{}{"x": "1"}})
	resp := dispatchSingle(t, h, Request{JSONRPC: "2.0", ID: float64(3), Method: "tools/call", Params: params})

	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	raw, _ := json.Marshal(resp.Result)
	var result ToolResult
	json.Unmarshal(raw, &result)
	if len(result.Content) != 1 || result.Content[0].Text != "called:my_tool" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestServer_Dispatch_ToolsCall_HandlerError(t *testing.T) {
	h := &mockHandler{
		callFn: func(name string, args map[string]interface{}) (*ToolResult, error) {
			return nil, fmt.Errorf("handler failed: %s", name)
		},
	}
	params, _ := json.Marshal(ToolCallParams{Name: "fail_tool"})
	resp := dispatchSingle(t, h, Request{JSONRPC: "2.0", ID: float64(4), Method: "tools/call", Params: params})

	// Tool errors should be in result, not in RPC error
	if resp.Error != nil {
		t.Fatalf("handler errors should be in result, not RPC error")
	}
	raw, _ := json.Marshal(resp.Result)
	var result ToolResult
	json.Unmarshal(raw, &result)
	if !result.IsError {
		t.Error("expected IsError=true for handler error")
	}
	if len(result.Content) == 0 || !strings.Contains(result.Content[0].Text, "handler failed") {
		t.Errorf("expected error message, got: %+v", result)
	}
}

func TestServer_Dispatch_Unknown(t *testing.T) {
	resp := dispatchSingle(t, &mockHandler{}, Request{JSONRPC: "2.0", ID: float64(5), Method: "bogus/method"})

	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", resp.Error.Code)
	}
	if !strings.Contains(resp.Error.Message, "Method not found") {
		t.Errorf("error message = %q, want 'Method not found'", resp.Error.Message)
	}
}

func TestServer_Dispatch_InvalidJSON(t *testing.T) {
	input := "this is not valid JSON\n"
	var out bytes.Buffer
	s := &Server{
		handler: &mockHandler{},
		reader:  bufio.NewReader(strings.NewReader(input)),
		writer:  &out,
	}
	s.Run()

	resp := parseResponse(t, out.Bytes())
	if resp.Error == nil {
		t.Fatal("expected parse error")
	}
	if resp.Error.Code != -32700 {
		t.Errorf("error code = %d, want -32700 (parse error)", resp.Error.Code)
	}
}

func TestServer_Dispatch_MissingID(t *testing.T) {
	// A notification (no ID) should not produce output for notifications/initialized.
	// But for an unknown method without an ID, it should still produce an error response.
	req := Request{JSONRPC: "2.0", Method: "notifications/initialized"}
	reqData, _ := json.Marshal(req)
	input := string(reqData) + "\n"

	var out bytes.Buffer
	s := &Server{
		handler: &mockHandler{},
		reader:  bufio.NewReader(strings.NewReader(input)),
		writer:  &out,
	}
	s.Run()

	// notifications/initialized is a valid notification — should produce no output
	if out.Len() != 0 {
		t.Errorf("notification should not produce output, got: %s", out.String())
	}
}

func TestServer_Dispatch_InvalidToolsCallParams(t *testing.T) {
	// We must send raw bytes through the Run loop because json.Marshal
	// rejects invalid json.RawMessage. Construct the line manually.
	line := `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":"not-an-object"}` + "\n"

	var out bytes.Buffer
	s := &Server{
		handler: &mockHandler{},
		reader:  bufio.NewReader(strings.NewReader(line)),
		writer:  &out,
	}
	s.Run()

	resp := parseResponse(t, out.Bytes())
	if resp.Error == nil {
		t.Fatal("expected error for invalid params")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("error code = %d, want -32602", resp.Error.Code)
	}
}

func TestServer_Dispatch_Ping(t *testing.T) {
	resp := dispatchSingle(t, &mockHandler{}, Request{JSONRPC: "2.0", ID: float64(8), Method: "ping"})

	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	// Ping returns empty map
	if resp.Result == nil {
		t.Error("ping result should not be nil")
	}
}

func TestServer_Dispatch_MultipleMessages(t *testing.T) {
	r1, _ := json.Marshal(Request{JSONRPC: "2.0", ID: float64(1), Method: "ping"})
	r2, _ := json.Marshal(Request{JSONRPC: "2.0", ID: float64(2), Method: "initialize"})
	r3, _ := json.Marshal(Request{JSONRPC: "2.0", ID: float64(3), Method: "ping"})
	input := string(r1) + "\n" + string(r2) + "\n" + string(r3) + "\n"

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
	if len(lines) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(lines))
	}

	// Verify each response has the correct ID
	for i, line := range lines {
		var resp Response
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("response %d: parse error: %v", i, err)
		}
		expectedID := float64(i + 1)
		if resp.ID != expectedID {
			t.Errorf("response %d: ID = %v, want %v", i, resp.ID, expectedID)
		}
	}
}

func TestServer_Dispatch_ResponsePreservesID(t *testing.T) {
	// Test that the response ID matches the request ID
	for _, id := range []interface{}{float64(42), float64(0), float64(999)} {
		s, out := newTestServer(&mockHandler{})
		req := &Request{JSONRPC: "2.0", ID: id, Method: "ping"}
		s.handleRequest(req)

		resp := parseResponse(t, out.Bytes())
		if resp.ID != id {
			t.Errorf("request ID=%v, response ID=%v", id, resp.ID)
		}
		out.Reset()
	}
}

// dispatchSingle sends a single request through the full Run loop and returns the response.
func dispatchSingle(t *testing.T, h Handler, req Request) Response {
	t.Helper()
	reqData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var out bytes.Buffer
	s := &Server{
		handler: h,
		reader:  bufio.NewReader(strings.NewReader(string(reqData) + "\n")),
		writer:  &out,
	}
	if err := s.Run(); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		t.Fatal("no output from server")
	}
	return parseResponse(t, []byte(output))
}
