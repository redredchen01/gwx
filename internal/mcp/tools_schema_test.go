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
