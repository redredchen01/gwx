package mcp

import (
	"testing"
)

// TestAllTools_RequiredFieldsExistInProperties verifies that every tool's
// Required slice only references keys that actually exist in Properties.
// This catches typos like Required: []string{"spreasheet_id"} vs "spreadsheet_id".
func TestAllTools_RequiredFieldsExistInProperties(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()

	for _, tool := range tools {
		for _, req := range tool.InputSchema.Required {
			if _, ok := tool.InputSchema.Properties[req]; !ok {
				t.Errorf("tool %s: required field %q not found in Properties", tool.Name, req)
			}
		}
	}
}

// TestAllTools_RequiredNotEmpty verifies that if a tool has a Required slice,
// it is non-empty (i.e., no tool accidentally declares Required: []string{}).
func TestAllTools_RequiredNotEmpty(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()

	for _, tool := range tools {
		if tool.InputSchema.Required != nil && len(tool.InputSchema.Required) == 0 {
			t.Errorf("tool %s: Required is set but empty — use nil instead", tool.Name)
		}
	}
}

// --- Representative tool schema tests ---

// TestGmailReply_Schema verifies gmail_reply has message_id and body as required.
func TestGmailReply_Schema(t *testing.T) {
	var target *Tool
	for _, tool := range NewTools() {
		tool := tool // capture
		if tool.Name == "gmail_reply" {
			target = &tool
			break
		}
	}
	if target == nil {
		t.Fatal("gmail_reply not found in NewTools()")
	}

	wantRequired := map[string]bool{"message_id": true, "body": true}
	if len(target.InputSchema.Required) != len(wantRequired) {
		t.Errorf("gmail_reply: expected %d required fields, got %d: %v",
			len(wantRequired), len(target.InputSchema.Required), target.InputSchema.Required)
	}
	for _, r := range target.InputSchema.Required {
		if !wantRequired[r] {
			t.Errorf("gmail_reply: unexpected required field %q", r)
		}
	}
	// Both must also exist in Properties
	for _, r := range target.InputSchema.Required {
		if _, ok := target.InputSchema.Properties[r]; !ok {
			t.Errorf("gmail_reply: required field %q missing from Properties", r)
		}
	}
}

// TestDriveBatchUpload_Schema verifies drive_batch_upload has paths as required.
func TestDriveBatchUpload_Schema(t *testing.T) {
	var target *Tool
	for _, tool := range BatchTools() {
		tool := tool
		if tool.Name == "drive_batch_upload" {
			target = &tool
			break
		}
	}
	if target == nil {
		t.Fatal("drive_batch_upload not found in BatchTools()")
	}

	if len(target.InputSchema.Required) != 1 || target.InputSchema.Required[0] != "paths" {
		t.Errorf("drive_batch_upload: expected Required=[\"paths\"], got %v", target.InputSchema.Required)
	}
	if _, ok := target.InputSchema.Properties["paths"]; !ok {
		t.Error("drive_batch_upload: 'paths' missing from Properties")
	}
}

// TestSheetsCreate_Schema verifies sheets_create has title as the only required field.
func TestSheetsCreate_Schema(t *testing.T) {
	var target *Tool
	for _, tool := range NewTools() {
		tool := tool
		if tool.Name == "sheets_create" {
			target = &tool
			break
		}
	}
	if target == nil {
		t.Fatal("sheets_create not found in NewTools()")
	}

	if len(target.InputSchema.Required) != 1 || target.InputSchema.Required[0] != "title" {
		t.Errorf("sheets_create: expected Required=[\"title\"], got %v", target.InputSchema.Required)
	}
	if _, ok := target.InputSchema.Properties["title"]; !ok {
		t.Error("sheets_create: 'title' missing from Properties")
	}
}

// --- BatchTools schema tests ---

// TestBatchTools_Schema verifies both batch tools have correct Required fields and Properties.
func TestBatchTools_Schema(t *testing.T) {
	cases := []struct {
		name     string
		required []string
	}{
		{"drive_batch_upload", []string{"paths"}},
		{"sheets_batch_append", []string{"spreadsheet_id", "entries"}},
	}

	toolMap := make(map[string]Tool)
	for _, tool := range BatchTools() {
		toolMap[tool.Name] = tool
	}

	for _, c := range cases {
		tool, ok := toolMap[c.name]
		if !ok {
			t.Errorf("batch tool %q not found", c.name)
			continue
		}
		if tool.InputSchema.Type != "object" {
			t.Errorf("%s: InputSchema.Type = %q, want \"object\"", c.name, tool.InputSchema.Type)
		}
		// Check required count
		if len(tool.InputSchema.Required) != len(c.required) {
			t.Errorf("%s: expected %d required fields, got %d: %v",
				c.name, len(c.required), len(tool.InputSchema.Required), tool.InputSchema.Required)
		}
		// Check each required field exists in Properties
		for _, r := range c.required {
			if _, ok := tool.InputSchema.Properties[r]; !ok {
				t.Errorf("%s: required field %q not in Properties", c.name, r)
			}
		}
	}
}

// TestNewTools_RequiredFieldsConsistency verifies every tool in NewTools()
// that has Required entries maps them to valid Properties keys.
func TestNewTools_RequiredFieldsConsistency(t *testing.T) {
	for _, tool := range NewTools() {
		for _, req := range tool.InputSchema.Required {
			if _, ok := tool.InputSchema.Properties[req]; !ok {
				t.Errorf("NewTool %s: required field %q not in Properties", tool.Name, req)
			}
		}
	}
}

// TestListTools_CountAtLeast59 is a looser bound that survives future additions.
// The exact-count test (TestListTools_Count) catches regressions; this test
// ensures no tools were accidentally dropped.
func TestListTools_CountAtLeast59(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()
	if len(tools) < 59 {
		t.Errorf("expected at least 59 tools, got %d", len(tools))
	}
}

// --- Comprehensive schema validity tests ---

// TestAllTools_NonEmptyName verifies every registered tool has a non-empty Name.
func TestAllTools_NonEmptyName(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()

	for i, tool := range tools {
		if tool.Name == "" {
			t.Errorf("tool at index %d has empty Name", i)
		}
	}
}

// TestAllTools_NoDuplicateNamesAcrossProviders verifies no two providers
// register tools with the same name. This is similar to TestListTools_NoDuplicateNames
// in tools_test.go but explicitly frames it as a cross-provider check.
func TestAllTools_NoDuplicateNamesAcrossProviders(t *testing.T) {
	globalRegistry.mu.RLock()
	providers := globalRegistry.providers
	globalRegistry.mu.RUnlock()

	seen := make(map[string]int) // tool name -> provider index
	for pi, p := range providers {
		for _, tool := range p.Tools() {
			if prevPI, exists := seen[tool.Name]; exists {
				t.Errorf("tool %q registered by provider %d and provider %d", tool.Name, prevPI, pi)
			}
			seen[tool.Name] = pi
		}
	}
}

// TestAllTools_PropertyTypesValid verifies all Property.Type values use
// standard JSON Schema types.
func TestAllTools_PropertyTypesValid(t *testing.T) {
	validTypes := map[string]bool{
		"string":  true,
		"integer": true,
		"number":  true,
		"boolean": true,
		"array":   true,
		"object":  true,
	}

	h := &GWXHandler{}
	tools := h.ListTools()

	for _, tool := range tools {
		for propName, prop := range tool.InputSchema.Properties {
			if !validTypes[prop.Type] {
				t.Errorf("tool %s: property %q has invalid type %q", tool.Name, propName, prop.Type)
			}
		}
	}
}

// TestAllTools_PropertiesHaveDescriptions verifies every property has a
// non-empty description (helps MCP clients provide good UX).
func TestAllTools_PropertiesHaveDescriptions(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()

	for _, tool := range tools {
		for propName, prop := range tool.InputSchema.Properties {
			if prop.Description == "" {
				t.Errorf("tool %s: property %q has empty description", tool.Name, propName)
			}
		}
	}
}

// TestAllTools_HandlersMatchTools verifies every tool listed by the registry
// has a corresponding handler entry, and vice versa.
func TestAllTools_HandlersMatchTools(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()
	handlers := globalRegistry.buildHandlers(h)

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	// Every handler must correspond to a registered tool
	for handlerName := range handlers {
		if !toolNames[handlerName] {
			t.Errorf("handler %q has no corresponding tool definition", handlerName)
		}
	}

	// Every tool must have a handler
	for _, tool := range tools {
		if _, ok := handlers[tool.Name]; !ok {
			t.Errorf("tool %q has no corresponding handler", tool.Name)
		}
	}
}

// TestAllTools_NoDuplicateRequired verifies that no tool has duplicate
// entries in its Required slice.
func TestAllTools_NoDuplicateRequired(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()

	for _, tool := range tools {
		seen := make(map[string]bool)
		for _, req := range tool.InputSchema.Required {
			if seen[req] {
				t.Errorf("tool %s: duplicate required field %q", tool.Name, req)
			}
			seen[req] = true
		}
	}
}

// TestCoreTools_GmailSend_Schema spot-checks a core tool's schema.
func TestCoreTools_GmailSend_Schema(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()

	var target *Tool
	for _, tool := range tools {
		tool := tool
		if tool.Name == "gmail_send" {
			target = &tool
			break
		}
	}
	if target == nil {
		t.Fatal("gmail_send not found")
	}

	// Must require to, subject, body
	wantRequired := map[string]bool{"to": true, "subject": true, "body": true}
	if len(target.InputSchema.Required) != len(wantRequired) {
		t.Errorf("gmail_send: expected %d required, got %d: %v",
			len(wantRequired), len(target.InputSchema.Required), target.InputSchema.Required)
	}
	for _, r := range target.InputSchema.Required {
		if !wantRequired[r] {
			t.Errorf("gmail_send: unexpected required field %q", r)
		}
	}

	// Must have cc as optional property
	if _, ok := target.InputSchema.Properties["cc"]; !ok {
		t.Error("gmail_send: missing optional 'cc' property")
	}
}

// TestCoreTools_CalendarCreate_Schema spot-checks calendar_create.
func TestCoreTools_CalendarCreate_Schema(t *testing.T) {
	h := &GWXHandler{}
	tools := h.ListTools()

	var target *Tool
	for _, tool := range tools {
		tool := tool
		if tool.Name == "calendar_create" {
			target = &tool
			break
		}
	}
	if target == nil {
		t.Fatal("calendar_create not found")
	}

	wantRequired := map[string]bool{"title": true, "start": true, "end": true}
	for _, r := range target.InputSchema.Required {
		if !wantRequired[r] {
			t.Errorf("calendar_create: unexpected required field %q", r)
		}
		delete(wantRequired, r)
	}
	for missing := range wantRequired {
		t.Errorf("calendar_create: missing required field %q", missing)
	}
}

// TestWorkflowTools_SchemaConsistency verifies all workflow tools have valid schemas.
func TestWorkflowTools_SchemaConsistency(t *testing.T) {
	for _, tool := range WorkflowTools() {
		if tool.Name == "" {
			t.Error("workflow tool with empty name")
		}
		if tool.Description == "" {
			t.Errorf("workflow tool %s has empty description", tool.Name)
		}
		if tool.InputSchema.Type != "object" {
			t.Errorf("workflow tool %s: InputSchema.Type = %q, want object", tool.Name, tool.InputSchema.Type)
		}
		for _, req := range tool.InputSchema.Required {
			if _, ok := tool.InputSchema.Properties[req]; !ok {
				t.Errorf("workflow tool %s: required field %q not in Properties", tool.Name, req)
			}
		}
	}
}

// TestExtendedTools_SchemaConsistency verifies all extended tools have valid schemas.
func TestExtendedTools_SchemaConsistency(t *testing.T) {
	for _, tool := range ExtendedTools() {
		if tool.Name == "" {
			t.Error("extended tool with empty name")
		}
		if tool.Description == "" {
			t.Errorf("extended tool %s has empty description", tool.Name)
		}
		if tool.InputSchema.Type != "object" {
			t.Errorf("extended tool %s: InputSchema.Type = %q, want object", tool.Name, tool.InputSchema.Type)
		}
		for _, req := range tool.InputSchema.Required {
			if _, ok := tool.InputSchema.Properties[req]; !ok {
				t.Errorf("extended tool %s: required field %q not in Properties", tool.Name, req)
			}
		}
	}
}
