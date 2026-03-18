package mcp

import (
	"testing"
)

func TestListTools_Count(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()

	// Verify total tool count matches actual registration
	if len(tools) != 59 {
		t.Errorf("expected 59 tools, got %d", len(tools))
	}
}

func TestListTools_NoDuplicateNames(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()

	seen := make(map[string]bool)
	for _, tool := range tools {
		if seen[tool.Name] {
			t.Errorf("duplicate tool name: %s", tool.Name)
		}
		seen[tool.Name] = true
	}
}

func TestListTools_AllHaveDescription(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()

	for _, tool := range tools {
		if tool.Description == "" {
			t.Errorf("tool %s has empty description", tool.Name)
		}
	}
}

func TestListTools_AllHaveInputSchema(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()

	for _, tool := range tools {
		if tool.InputSchema.Type != "object" {
			t.Errorf("tool %s has InputSchema.Type=%q, want 'object'", tool.Name, tool.InputSchema.Type)
		}
	}
}

func TestCallTool_UnknownTool(t *testing.T) {
	h := &GWXHandler{} // nil client — should not reach API
	_, err := h.CallTool("nonexistent_tool_xyz", nil)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if err.Error() != "unknown tool: nonexistent_tool_xyz" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExtendedTools_Count(t *testing.T) {
	tools := ExtendedTools()
	if len(tools) != 17 {
		t.Errorf("expected 17 extended tools, got %d", len(tools))
	}
}

func TestNewTools_Count(t *testing.T) {
	tools := NewTools()
	if len(tools) != 18 {
		t.Errorf("expected 18 new tools, got %d", len(tools))
	}
}

func TestBatchTools_Count(t *testing.T) {
	tools := BatchTools()
	if len(tools) != 2 {
		t.Errorf("expected 2 batch tools, got %d", len(tools))
	}
}
