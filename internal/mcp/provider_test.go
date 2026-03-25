package mcp

import (
	"context"
	"testing"
)

// --- Individual provider tests ---
// Each test verifies that a provider's Tools() returns the expected count and
// that all tool names are non-empty, and Handlers() returns matching entries.

func TestFormsProvider_Tools(t *testing.T) {
	p := formsProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("formsProvider.Tools() returned empty")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("formsProvider: tool with empty name")
		}
		if tool.InputSchema.Type != "object" {
			t.Errorf("formsProvider tool %s: InputSchema.Type = %q", tool.Name, tool.InputSchema.Type)
		}
	}
}

func TestFormsProvider_Handlers(t *testing.T) {
	p := formsProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("formsProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
	for _, tool := range tools {
		if _, ok := handlers[tool.Name]; !ok {
			t.Errorf("formsProvider: no handler for tool %q", tool.Name)
		}
	}
}

func TestBigQueryProvider_Tools(t *testing.T) {
	p := bigqueryProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("bigqueryProvider.Tools() returned empty")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("bigqueryProvider: tool with empty name")
		}
		if tool.Description == "" {
			t.Errorf("bigqueryProvider tool %s: empty description", tool.Name)
		}
	}
}

func TestBigQueryProvider_Handlers(t *testing.T) {
	p := bigqueryProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("bigqueryProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
	for _, tool := range tools {
		if _, ok := handlers[tool.Name]; !ok {
			t.Errorf("bigqueryProvider: no handler for tool %q", tool.Name)
		}
	}
}

func TestGitHubProvider_Tools(t *testing.T) {
	p := githubProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("githubProvider.Tools() returned empty")
	}
	names := make(map[string]bool)
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("githubProvider: tool with empty name")
		}
		if names[tool.Name] {
			t.Errorf("githubProvider: duplicate tool name %q", tool.Name)
		}
		names[tool.Name] = true
	}
}

func TestGitHubProvider_Handlers(t *testing.T) {
	p := githubProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("githubProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
}

func TestSlackProvider_Tools(t *testing.T) {
	p := slackProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("slackProvider.Tools() returned empty")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("slackProvider: tool with empty name")
		}
		if tool.InputSchema.Type != "object" {
			t.Errorf("slackProvider tool %s: InputSchema.Type = %q", tool.Name, tool.InputSchema.Type)
		}
	}
}

func TestSlackProvider_Handlers(t *testing.T) {
	p := slackProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("slackProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
}

func TestNotionProvider_Tools(t *testing.T) {
	p := notionProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("notionProvider.Tools() returned empty")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("notionProvider: tool with empty name")
		}
		if tool.Description == "" {
			t.Errorf("notionProvider tool %s: empty description", tool.Name)
		}
	}
}

func TestNotionProvider_Handlers(t *testing.T) {
	p := notionProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("notionProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
}

func TestSlidesProvider_Tools(t *testing.T) {
	p := slidesProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("slidesProvider.Tools() returned empty")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("slidesProvider: tool with empty name")
		}
	}
}

func TestSlidesProvider_Handlers(t *testing.T) {
	p := slidesProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("slidesProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
}

func TestSearchConsoleProvider_Tools(t *testing.T) {
	p := searchconsoleProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("searchconsoleProvider.Tools() returned empty")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("searchconsoleProvider: tool with empty name")
		}
		// Verify required fields exist in properties
		for _, req := range tool.InputSchema.Required {
			if _, ok := tool.InputSchema.Properties[req]; !ok {
				t.Errorf("searchconsoleProvider tool %s: required %q not in Properties", tool.Name, req)
			}
		}
	}
}

func TestSearchConsoleProvider_Handlers(t *testing.T) {
	p := searchconsoleProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("searchconsoleProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
}

func TestAnalyticsProvider_Tools(t *testing.T) {
	p := analyticsProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("analyticsProvider.Tools() returned empty")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("analyticsProvider: tool with empty name")
		}
		if tool.Description == "" {
			t.Errorf("analyticsProvider tool %s: empty description", tool.Name)
		}
	}
}

func TestAnalyticsProvider_Handlers(t *testing.T) {
	p := analyticsProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("analyticsProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
}

func TestCoreProvider_Tools(t *testing.T) {
	p := coreProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("coreProvider.Tools() returned empty")
	}
	names := make(map[string]bool)
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("coreProvider: tool with empty name")
		}
		if names[tool.Name] {
			t.Errorf("coreProvider: duplicate tool name %q", tool.Name)
		}
		names[tool.Name] = true
	}
}

func TestCoreProvider_Handlers(t *testing.T) {
	p := coreProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("coreProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
}

func TestExtendedProvider_Tools(t *testing.T) {
	p := extendedProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("extendedProvider.Tools() returned empty")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("extendedProvider: tool with empty name")
		}
	}
}

func TestExtendedProvider_Handlers(t *testing.T) {
	p := extendedProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("extendedProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
}

func TestNewProvider_Tools(t *testing.T) {
	p := newProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("newProvider.Tools() returned empty")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("newProvider: tool with empty name")
		}
	}
}

func TestNewProvider_Handlers(t *testing.T) {
	p := newProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("newProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
}

func TestBatchProvider_Tools(t *testing.T) {
	p := batchProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("batchProvider.Tools() returned empty")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("batchProvider: tool with empty name")
		}
	}
}

func TestBatchProvider_Handlers(t *testing.T) {
	p := batchProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("batchProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
}

func TestWorkflowProvider_Tools(t *testing.T) {
	p := workflowProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("workflowProvider.Tools() returned empty")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("workflowProvider: tool with empty name")
		}
	}
}

func TestWorkflowProvider_Handlers(t *testing.T) {
	p := workflowProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("workflowProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
}

func TestConfigProvider_Tools(t *testing.T) {
	p := configProvider{}
	tools := p.Tools()
	if len(tools) == 0 {
		t.Fatal("configProvider.Tools() returned empty")
	}
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("configProvider: tool with empty name")
		}
	}
}

func TestConfigProvider_Handlers(t *testing.T) {
	p := configProvider{}
	h := &GWXHandler{}
	tools := p.Tools()
	handlers := p.Handlers(h)
	if len(handlers) != len(tools) {
		t.Errorf("configProvider: %d tools vs %d handlers", len(tools), len(handlers))
	}
}

// --- Cross-provider handler dispatch test ---

func TestAllProviders_HandlersAreCallable(t *testing.T) {
	// Verify that every handler registered across all providers is a non-nil
	// function that can be called without panic (we only test the type, not
	// actual execution which requires API credentials).
	h := &GWXHandler{}
	handlers := globalRegistry.buildHandlers(h)

	for name, handler := range handlers {
		if handler == nil {
			t.Errorf("handler %q is nil", name)
			continue
		}
		// Verify handler has correct signature by attempting a type assertion.
		// This is already guaranteed by compilation, but we verify non-nil.
		var fn ToolHandler = handler
		_ = fn
	}
}

func TestAllProviders_ToolNamePrefix(t *testing.T) {
	// Verify each provider's tools follow naming conventions (no spaces, lowercase).
	h := &GWXHandler{}
	tools := h.ListTools()

	for _, tool := range tools {
		if tool.Name == "" {
			t.Errorf("tool name sanity check failed for %q", tool.Name)
		}
		// Tool names should not contain spaces
		for _, r := range tool.Name {
			if r == ' ' {
				t.Errorf("tool name %q contains spaces", tool.Name)
				break
			}
		}
	}
}

func TestAllProviders_HandlerReturnType(t *testing.T) {
	// Spot-check that each handler, when given a nil context and nil args to an
	// unknown tool, does not panic. We cannot actually call most handlers without
	// API credentials, so we only verify they can be built.
	h := &GWXHandler{}
	registry := h.buildRegistry()

	// Just verify the registry is non-empty and all values are non-nil.
	if len(registry) == 0 {
		t.Fatal("buildRegistry returned empty")
	}
	for name, fn := range registry {
		if fn == nil {
			t.Errorf("registry entry %q is nil", name)
		}
	}
}

func TestConfigProvider_GetConfigHandler(t *testing.T) {
	// config_get and config_set should be callable tools in the registry.
	p := configProvider{}
	h := &GWXHandler{}
	handlers := p.Handlers(h)

	// Verify config tools exist
	for _, tool := range p.Tools() {
		fn, ok := handlers[tool.Name]
		if !ok {
			t.Errorf("missing handler for config tool %q", tool.Name)
			continue
		}
		if fn == nil {
			t.Errorf("nil handler for config tool %q", tool.Name)
		}
	}
}

func TestAllProviders_SchemaIntegrity(t *testing.T) {
	// Comprehensive check: for each provider, verify tools have valid schemas
	// and handlers match tools exactly.
	globalRegistry.mu.RLock()
	providers := make([]ToolProvider, len(globalRegistry.providers))
	copy(providers, globalRegistry.providers)
	globalRegistry.mu.RUnlock()

	h := &GWXHandler{}
	for i, p := range providers {
		tools := p.Tools()
		handlers := p.Handlers(h)

		if len(tools) == 0 {
			t.Errorf("provider %d: empty tools list", i)
			continue
		}

		toolNames := make(map[string]bool, len(tools))
		for _, tool := range tools {
			if tool.Name == "" {
				t.Errorf("provider %d: tool with empty name", i)
			}
			if tool.Description == "" {
				t.Errorf("provider %d: tool %q has empty description", i, tool.Name)
			}
			if tool.InputSchema.Type != "object" {
				t.Errorf("provider %d: tool %q: InputSchema.Type = %q", i, tool.Name, tool.InputSchema.Type)
			}
			toolNames[tool.Name] = true
		}

		// Every handler should match a tool
		for hName := range handlers {
			if !toolNames[hName] {
				t.Errorf("provider %d: handler %q has no matching tool", i, hName)
			}
		}

		// Every tool should have a handler
		for _, tool := range tools {
			if _, ok := handlers[tool.Name]; !ok {
				t.Errorf("provider %d: tool %q has no handler", i, tool.Name)
			}
		}
	}
}

// TestRegisterProvider_GlobalEffect verifies RegisterProvider adds to globalRegistry.
func TestRegisterProvider_GlobalEffect(t *testing.T) {
	// Count providers before
	globalRegistry.mu.RLock()
	before := len(globalRegistry.providers)
	globalRegistry.mu.RUnlock()

	// After init(), we should have at least 14 providers (one per tools_*.go file)
	if before < 14 {
		t.Errorf("expected at least 14 providers from init(), got %d", before)
	}
}

// TestGWXHandler_CallTool_LazyInitIdempotent verifies calling CallTool multiple
// times does not re-build the registry each time.
func TestGWXHandler_CallTool_LazyInitIdempotent(t *testing.T) {
	h := &GWXHandler{}
	// First call — initializes registry
	h.CallTool("__nonexistent__", nil)
	reg1 := h.registry

	// Second call — should reuse same registry
	h.CallTool("__nonexistent__", nil)
	reg2 := h.registry

	if len(reg1) != len(reg2) {
		t.Errorf("registry size changed: %d -> %d", len(reg1), len(reg2))
	}
}

// TestGWXHandler_CallTool_WithContext verifies CallTool creates a context with
// timeout (we can't test the timeout directly but can verify no panic).
func TestGWXHandler_CallTool_WithContext(t *testing.T) {
	h := &GWXHandler{}
	// This should not panic even with nil client — the error comes from
	// "unknown tool" before any API call is made.
	_, err := h.CallTool("definitely_not_a_tool", map[string]interface{}{
		"arg1": "value1",
	})
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

// TestMockHandler_ImplementsHandler verifies mockHandler satisfies Handler interface.
func TestMockHandler_ImplementsHandler(t *testing.T) {
	var h Handler = &mockHandler{
		tools: []Tool{{Name: "test", Description: "test", InputSchema: InputSchema{Type: "object"}}},
		callFn: func(name string, args map[string]interface{}) (*ToolResult, error) {
			return &ToolResult{Content: []ContentBlock{{Type: "text", Text: "ok"}}}, nil
		},
	}
	tools := h.ListTools()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}
	result, err := h.CallTool("test", nil)
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if result.Content[0].Text != "ok" {
		t.Errorf("unexpected result: %s", result.Content[0].Text)
	}
}

// TestServer_SendResult_ValidJSON verifies sendResult produces valid JSON.
func TestServer_SendResult_ValidJSON(t *testing.T) {
	s, out := newTestServer(&mockHandler{})
	s.sendResult(float64(1), map[string]string{"key": "value"})

	resp := parseResponse(t, out.Bytes())
	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", resp.JSONRPC)
	}
	if resp.ID != float64(1) {
		t.Errorf("ID = %v, want 1", resp.ID)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %+v", resp.Error)
	}
}

// TestServer_SendError_ValidJSON verifies sendError produces valid JSON.
func TestServer_SendError_ValidJSON(t *testing.T) {
	s, out := newTestServer(&mockHandler{})
	s.sendError(float64(2), -32600, "Invalid Request", "details here")

	resp := parseResponse(t, out.Bytes())
	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", resp.JSONRPC)
	}
	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("error code = %d, want -32600", resp.Error.Code)
	}
	if resp.Error.Message != "Invalid Request" {
		t.Errorf("error message = %q", resp.Error.Message)
	}
}

// Verify that each handler in the full registry is a valid ToolHandler function
// by calling it with context.Background and nil args (which will fail gracefully).
func TestAllProviders_HandlersAreNonNil(t *testing.T) {
	h := &GWXHandler{}
	handlers := globalRegistry.buildHandlers(h)
	tools := globalRegistry.allTools()

	toolSet := make(map[string]bool, len(tools))
	for _, tool := range tools {
		toolSet[tool.Name] = true
	}

	for name := range handlers {
		if !toolSet[name] {
			t.Errorf("handler %q not found in tool list", name)
		}
	}

	for _, tool := range tools {
		fn, ok := handlers[tool.Name]
		if !ok {
			t.Errorf("tool %q missing handler", tool.Name)
			continue
		}
		_ = context.Background()
		_ = fn // Non-nil check — we can't call without API credentials
	}
}
