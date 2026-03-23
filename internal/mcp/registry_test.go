package mcp

import (
	"context"
	"testing"
)

// stubProvider is a minimal ToolProvider for registry tests.
type stubProvider struct {
	tools    []Tool
	handlers map[string]ToolHandler
}

func (s stubProvider) Tools() []Tool { return s.tools }
func (s stubProvider) Handlers(_ *GWXHandler) map[string]ToolHandler {
	return s.handlers
}

func TestToolRegistry_RegisterAndAllTools(t *testing.T) {
	r := &ToolRegistry{}
	p1 := stubProvider{
		tools: []Tool{
			{Name: "tool_a", Description: "A", InputSchema: InputSchema{Type: "object"}},
		},
	}
	p2 := stubProvider{
		tools: []Tool{
			{Name: "tool_b", Description: "B", InputSchema: InputSchema{Type: "object"}},
			{Name: "tool_c", Description: "C", InputSchema: InputSchema{Type: "object"}},
		},
	}

	r.mu.Lock()
	r.providers = append(r.providers, p1, p2)
	r.mu.Unlock()

	tools := r.allTools()
	if len(tools) != 3 {
		t.Fatalf("allTools() returned %d, want 3", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"tool_a", "tool_b", "tool_c"} {
		if !names[want] {
			t.Errorf("missing tool %q in allTools()", want)
		}
	}
}

func TestToolRegistry_AllToolsPreservesOrder(t *testing.T) {
	r := &ToolRegistry{}
	p1 := stubProvider{
		tools: []Tool{
			{Name: "first", Description: "1st", InputSchema: InputSchema{Type: "object"}},
		},
	}
	p2 := stubProvider{
		tools: []Tool{
			{Name: "second", Description: "2nd", InputSchema: InputSchema{Type: "object"}},
		},
	}

	r.mu.Lock()
	r.providers = append(r.providers, p1, p2)
	r.mu.Unlock()

	tools := r.allTools()
	if tools[0].Name != "first" {
		t.Errorf("first tool = %q, want 'first'", tools[0].Name)
	}
	if tools[1].Name != "second" {
		t.Errorf("second tool = %q, want 'second'", tools[1].Name)
	}
}

func TestToolRegistry_EmptyRegistry(t *testing.T) {
	r := &ToolRegistry{}
	tools := r.allTools()
	if len(tools) != 0 {
		t.Errorf("allTools() on empty registry returned %d, want 0", len(tools))
	}

	handlers := r.buildHandlers(nil)
	if len(handlers) != 0 {
		t.Errorf("buildHandlers() on empty registry returned %d, want 0", len(handlers))
	}
}

func TestToolRegistry_BuildHandlers(t *testing.T) {
	r := &ToolRegistry{}
	dummyHandler := func(_ context.Context, _ map[string]interface{}) (*ToolResult, error) {
		return &ToolResult{Content: []ContentBlock{{Type: "text", Text: "ok"}}}, nil
	}

	p := stubProvider{
		tools: []Tool{
			{Name: "tool_x", Description: "X", InputSchema: InputSchema{Type: "object"}},
		},
		handlers: map[string]ToolHandler{
			"tool_x": dummyHandler,
		},
	}

	r.mu.Lock()
	r.providers = append(r.providers, p)
	r.mu.Unlock()

	handlers := r.buildHandlers(nil)
	if len(handlers) != 1 {
		t.Fatalf("buildHandlers() returned %d handlers, want 1", len(handlers))
	}

	fn, ok := handlers["tool_x"]
	if !ok {
		t.Fatal("handler for 'tool_x' not found")
	}

	result, err := fn(context.Background(), nil)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.Content[0].Text != "ok" {
		t.Errorf("handler returned %q, want 'ok'", result.Content[0].Text)
	}
}

func TestToolRegistry_BuildHandlers_AggregatesMultipleProviders(t *testing.T) {
	r := &ToolRegistry{}
	handler1 := func(_ context.Context, _ map[string]interface{}) (*ToolResult, error) {
		return &ToolResult{Content: []ContentBlock{{Type: "text", Text: "from_p1"}}}, nil
	}
	handler2 := func(_ context.Context, _ map[string]interface{}) (*ToolResult, error) {
		return &ToolResult{Content: []ContentBlock{{Type: "text", Text: "from_p2"}}}, nil
	}

	p1 := stubProvider{
		tools:    []Tool{{Name: "alpha", Description: "A", InputSchema: InputSchema{Type: "object"}}},
		handlers: map[string]ToolHandler{"alpha": handler1},
	}
	p2 := stubProvider{
		tools:    []Tool{{Name: "beta", Description: "B", InputSchema: InputSchema{Type: "object"}}},
		handlers: map[string]ToolHandler{"beta": handler2},
	}

	r.mu.Lock()
	r.providers = append(r.providers, p1, p2)
	r.mu.Unlock()

	handlers := r.buildHandlers(nil)
	if len(handlers) != 2 {
		t.Fatalf("expected 2 handlers, got %d", len(handlers))
	}

	r1, _ := handlers["alpha"](context.Background(), nil)
	if r1.Content[0].Text != "from_p1" {
		t.Errorf("alpha handler returned %q, want 'from_p1'", r1.Content[0].Text)
	}
	r2, _ := handlers["beta"](context.Background(), nil)
	if r2.Content[0].Text != "from_p2" {
		t.Errorf("beta handler returned %q, want 'from_p2'", r2.Content[0].Text)
	}
}

func TestToolRegistry_BuildHandlers_DuplicatePanics(t *testing.T) {
	r := &ToolRegistry{}
	dummyHandler := func(_ context.Context, _ map[string]interface{}) (*ToolResult, error) {
		return nil, nil
	}

	p1 := stubProvider{
		tools:    []Tool{{Name: "dup_tool", Description: "D", InputSchema: InputSchema{Type: "object"}}},
		handlers: map[string]ToolHandler{"dup_tool": dummyHandler},
	}
	p2 := stubProvider{
		tools:    []Tool{{Name: "dup_tool", Description: "D2", InputSchema: InputSchema{Type: "object"}}},
		handlers: map[string]ToolHandler{"dup_tool": dummyHandler},
	}

	r.mu.Lock()
	r.providers = append(r.providers, p1, p2)
	r.mu.Unlock()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for duplicate tool key, got none")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic value is not a string: %v", r)
		}
		if msg != "mcp: duplicate tool key registered: dup_tool" {
			t.Errorf("unexpected panic message: %q", msg)
		}
	}()

	r.buildHandlers(nil)
}

func TestGlobalRegistry_IsNonNil(t *testing.T) {
	// The global registry should be initialized at package level.
	if globalRegistry == nil {
		t.Fatal("globalRegistry is nil")
	}
}

func TestGlobalRegistry_HasProviders(t *testing.T) {
	// After init(), all providers should have been registered.
	globalRegistry.mu.RLock()
	count := len(globalRegistry.providers)
	globalRegistry.mu.RUnlock()

	if count == 0 {
		t.Fatal("globalRegistry has no providers after init()")
	}
}

func TestGWXHandler_ListTools_MatchesGlobalRegistry(t *testing.T) {
	h := &GWXHandler{}
	fromHandler := h.ListTools()
	fromRegistry := globalRegistry.allTools()

	if len(fromHandler) != len(fromRegistry) {
		t.Errorf("GWXHandler.ListTools() = %d tools, globalRegistry.allTools() = %d tools",
			len(fromHandler), len(fromRegistry))
	}

	// Verify tool names match
	handlerNames := make(map[string]bool)
	for _, tool := range fromHandler {
		handlerNames[tool.Name] = true
	}
	for _, tool := range fromRegistry {
		if !handlerNames[tool.Name] {
			t.Errorf("registry tool %q missing from GWXHandler.ListTools()", tool.Name)
		}
	}
}
