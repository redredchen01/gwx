package mcp

import (
	"testing"
)

// --- GWXHandler tests ---

func TestGWXHandler_ListTools_NonEmpty(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()
	if len(tools) == 0 {
		t.Fatal("ListTools() returned empty; expected registered tools")
	}
}

func TestGWXHandler_CallTool_UnknownReturnsError(t *testing.T) {
	h := &GWXHandler{}
	_, err := h.CallTool("totally_nonexistent_tool", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if err.Error() != "unknown tool: totally_nonexistent_tool" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGWXHandler_CallTool_UnknownWithArgs(t *testing.T) {
	h := &GWXHandler{}
	_, err := h.CallTool("fake_tool_xyz", map[string]interface{}{"key": "value"})
	if err == nil {
		t.Fatal("expected error for unknown tool with args")
	}
}

func TestGWXHandler_BuildRegistry_HasEntries(t *testing.T) {
	h := &GWXHandler{}
	registry := h.buildRegistry()
	if len(registry) == 0 {
		t.Fatal("buildRegistry() returned empty map")
	}
}

func TestGWXHandler_BuildRegistry_AllToolsHaveHandlers(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()
	registry := h.buildRegistry()

	for _, tool := range tools {
		if _, ok := registry[tool.Name]; !ok {
			t.Errorf("tool %q has no handler in registry", tool.Name)
		}
	}
}

func TestGWXHandler_BuildRegistry_AllHandlersHaveTools(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()
	registry := h.buildRegistry()

	toolSet := make(map[string]bool)
	for _, tool := range tools {
		toolSet[tool.Name] = true
	}

	for name := range registry {
		if !toolSet[name] {
			t.Errorf("handler %q has no corresponding tool definition", name)
		}
	}
}

func TestGWXHandler_RegistryLazyInit(t *testing.T) {
	h := &GWXHandler{}
	// Before any CallTool, registry should be nil
	if h.registry != nil {
		t.Fatal("registry should be nil before first CallTool")
	}
	// After CallTool (even failing), registry should be initialized via sync.Once
	h.CallTool("nonexistent", nil)
	if h.registry == nil {
		t.Fatal("registry should be initialized after first CallTool")
	}
	// Calling again should reuse the same registry (sync.Once guarantees this)
	sizeBefore := len(h.registry)
	h.CallTool("nonexistent_again", nil)
	if len(h.registry) != sizeBefore {
		t.Fatal("registry size should remain stable across calls")
	}
}

// --- ToolProvider tests ---

func TestToolProviders_AllRegistered(t *testing.T) {
	globalRegistry.mu.RLock()
	count := len(globalRegistry.providers)
	globalRegistry.mu.RUnlock()

	if count == 0 {
		t.Fatal("no providers registered in globalRegistry")
	}
}

func TestToolProviders_EachReturnsNonEmptyTools(t *testing.T) {
	globalRegistry.mu.RLock()
	providers := make([]ToolProvider, len(globalRegistry.providers))
	copy(providers, globalRegistry.providers)
	globalRegistry.mu.RUnlock()

	for i, p := range providers {
		tools := p.Tools()
		if len(tools) == 0 {
			t.Errorf("provider %d returned empty Tools()", i)
		}
	}
}

func TestToolProviders_EachReturnsNonEmptyHandlers(t *testing.T) {
	h := &GWXHandler{}
	globalRegistry.mu.RLock()
	providers := make([]ToolProvider, len(globalRegistry.providers))
	copy(providers, globalRegistry.providers)
	globalRegistry.mu.RUnlock()

	for i, p := range providers {
		handlers := p.Handlers(h)
		if len(handlers) == 0 {
			t.Errorf("provider %d returned empty Handlers()", i)
		}
	}
}

func TestToolProviders_ToolCountMatchesHandlerCount(t *testing.T) {
	h := &GWXHandler{}
	globalRegistry.mu.RLock()
	providers := make([]ToolProvider, len(globalRegistry.providers))
	copy(providers, globalRegistry.providers)
	globalRegistry.mu.RUnlock()

	for i, p := range providers {
		tools := p.Tools()
		handlers := p.Handlers(h)
		if len(tools) != len(handlers) {
			t.Errorf("provider %d: %d tools vs %d handlers", i, len(tools), len(handlers))
		}
	}
}

// --- Server protocol tests ---

func TestServer_HandleRequest_InitializeContent(t *testing.T) {
	s, out := newTestServer(&mockHandler{})
	req := &Request{JSONRPC: "2.0", ID: float64(10), Method: "initialize"}
	s.handleRequest(req)

	resp := parseResponse(t, out.Bytes())
	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want 2.0", resp.JSONRPC)
	}
	if resp.ID != float64(10) {
		t.Errorf("ID = %v, want 10", resp.ID)
	}
}

// --- Edge case: calling same unknown tool twice ---

func TestGWXHandler_CallTool_UnknownTwice(t *testing.T) {
	h := &GWXHandler{}
	_, err1 := h.CallTool("no_such_tool", nil)
	_, err2 := h.CallTool("no_such_tool", nil)
	if err1 == nil || err2 == nil {
		t.Fatal("both calls should return error")
	}
	if err1.Error() != err2.Error() {
		t.Errorf("error messages differ: %q vs %q", err1.Error(), err2.Error())
	}
}
