package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/redredchen01/gwx/internal/exitcode"
)

// --- suggestedAction comprehensive tests ---

func TestSuggestedAction_AllCodes(t *testing.T) {
	tests := []struct {
		code       int
		msg        string
		wantEmpty  bool
		wantSubstr string
	}{
		{exitcode.AuthRequired, "no credentials", false, "gwx onboard"},
		{exitcode.AuthExpired, "token expired", false, "gwx auth login"},
		{exitcode.PermissionDenied, "not in allowlist", false, "GWX_ENABLE_COMMANDS"},
		{exitcode.PermissionDenied, "insufficient scope", false, "re-authorize"},
		{exitcode.NotFound, "resource not found", false, "list"},
		{exitcode.RateLimited, "too many requests", false, "Wait"},
		{exitcode.CircuitOpen, "circuit breaker open", false, "circuit breaker"},
		{exitcode.Conflict, "concurrent modification", false, "Retry"},
		{exitcode.InvalidInput, "bad parameter", false, "--help"},
		{exitcode.OK, "success", true, ""},
		{exitcode.GeneralError, "generic error", true, ""},
		{exitcode.UsageError, "bad usage", true, ""},
		{exitcode.DryRunSuccess, "dry run ok", true, ""},
		{99, "unknown code", true, ""},
	}

	for _, tt := range tests {
		got := suggestedAction(tt.code, tt.msg)
		if tt.wantEmpty {
			if got != "" {
				t.Errorf("suggestedAction(%d, %q) = %q, want empty", tt.code, tt.msg, got)
			}
		} else {
			if got == "" {
				t.Errorf("suggestedAction(%d, %q) = empty, want containing %q", tt.code, tt.msg, tt.wantSubstr)
			} else if !strings.Contains(got, tt.wantSubstr) {
				t.Errorf("suggestedAction(%d, %q) = %q, want containing %q", tt.code, tt.msg, got, tt.wantSubstr)
			}
		}
	}
}

// --- printTable various inputs ---

func TestPrintTable_VariousInputs(t *testing.T) {
	t.Run("map_with_nested_array", func(t *testing.T) {
		var buf bytes.Buffer
		p := &Printer{Format: FormatTable, Writer: &buf}
		data := map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"name": "alice", "age": float64(30)},
				map[string]interface{}{"name": "bob", "age": float64(25)},
			},
		}
		p.printTable(data)
		out := buf.String()
		if !strings.Contains(out, "alice") || !strings.Contains(out, "bob") {
			t.Errorf("table should contain data, got: %s", out)
		}
	})

	t.Run("map_without_array", func(t *testing.T) {
		var buf bytes.Buffer
		p := &Printer{Format: FormatTable, Writer: &buf}
		data := map[string]interface{}{
			"status": "active",
			"count":  42,
		}
		p.printTable(data)
		out := buf.String()
		if !strings.Contains(out, "status") || !strings.Contains(out, "count") {
			t.Errorf("key-value table should contain keys, got: %s", out)
		}
	})

	t.Run("scalar_data", func(t *testing.T) {
		var buf bytes.Buffer
		p := &Printer{Format: FormatTable, Writer: &buf}
		p.printTable("just a string")
		out := buf.String()
		if !strings.Contains(out, "just a string") {
			t.Errorf("scalar should be printed directly, got: %s", out)
		}
	})

	t.Run("empty_map", func(t *testing.T) {
		var buf bytes.Buffer
		p := &Printer{Format: FormatTable, Writer: &buf}
		p.printTable(map[string]interface{}{})
		// Should produce a table with no data rows, but with empty header row
		// (no panic is the main check)
		_ = buf.String()
	})

	t.Run("array_with_mixed_types", func(t *testing.T) {
		var buf bytes.Buffer
		p := &Printer{Format: FormatTable, Writer: &buf}
		arr := []interface{}{
			map[string]interface{}{"id": "1", "val": "a"},
			"plain_string",
		}
		raw, _ := json.Marshal(arr)
		var parsed interface{}
		json.Unmarshal(raw, &parsed)
		p.printTable(parsed)
		out := buf.String()
		if out == "" {
			t.Error("output should not be empty")
		}
	})

	t.Run("map_multiple_arrays_picks_largest", func(t *testing.T) {
		var buf bytes.Buffer
		p := &Printer{Format: FormatTable, Writer: &buf}
		data := map[string]interface{}{
			"small": []interface{}{"a"},
			"big": []interface{}{
				map[string]interface{}{"x": "1"},
				map[string]interface{}{"x": "2"},
				map[string]interface{}{"x": "3"},
			},
		}
		p.printTable(data)
		out := buf.String()
		// Should render the "big" array as table
		if !strings.Contains(out, "x") {
			t.Errorf("should render largest array as table, got: %s", out)
		}
	})
}

// --- mapSummary priority fields ---

func TestMapSummary_PriorityFields(t *testing.T) {
	t.Run("all_priority_fields", func(t *testing.T) {
		m := map[string]interface{}{
			"subject": "Test Subject",
			"title":   "Test Title",
			"name":    "Test Name",
			"id":      "123",
			"extra":   "ignored",
		}
		got := mapSummary(m)
		// Should pick first 3 priority fields found
		if !strings.Contains(got, "subject=Test Subject") {
			t.Errorf("should contain subject, got: %s", got)
		}
		// Only first 3 should be shown
		parts := strings.Split(got, "  ")
		if len(parts) > 3 {
			t.Errorf("should show at most 3 fields, got %d: %s", len(parts), got)
		}
	})

	t.Run("no_priority_fields_fallback", func(t *testing.T) {
		m := map[string]interface{}{
			"custom1": "val1",
			"custom2": "val2",
			"custom3": "val3",
			"custom4": "val4",
		}
		got := mapSummary(m)
		if got == "" {
			t.Error("fallback summary should not be empty")
		}
		// Should have at most 3 fields in fallback
		parts := strings.Split(got, "  ")
		if len(parts) > 3 {
			t.Errorf("fallback should show at most 3 fields, got %d: %s", len(parts), got)
		}
	})

	t.Run("nil_value_skipped", func(t *testing.T) {
		m := map[string]interface{}{
			"subject": nil,
			"name":    "valid",
		}
		got := mapSummary(m)
		if strings.Contains(got, "subject=") {
			t.Errorf("nil value should be skipped, got: %s", got)
		}
		if !strings.Contains(got, "name=valid") {
			t.Errorf("valid field should be present, got: %s", got)
		}
	})

	t.Run("email_and_status_fields", func(t *testing.T) {
		m := map[string]interface{}{
			"email":  "test@example.com",
			"status": "active",
		}
		got := mapSummary(m)
		if !strings.Contains(got, "email=test@example.com") {
			t.Errorf("should contain email, got: %s", got)
		}
		if !strings.Contains(got, "status=active") {
			t.Errorf("should contain status, got: %s", got)
		}
	})
}

// --- IsTTY test ---

func TestIsTTY_ReturnsBoolean(t *testing.T) {
	// In CI/test environments, IsTTY() should return false since stdout
	// is typically a pipe, not a terminal. The main goal is to verify
	// the function does not panic.
	result := IsTTY()
	// We can't assert the value since it depends on the environment,
	// but in test mode it's almost always false.
	_ = result
}

// --- renderArrayAsTable edge cases ---

func TestRenderArrayAsTable_NestedObjectsSkipped(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}
	arr := []interface{}{
		map[string]interface{}{
			"id":     "1",
			"nested": map[string]interface{}{"deep": "value"},
			"list":   []interface{}{"a", "b"},
		},
	}
	p.renderArrayAsTable(arr)
	out := buf.String()
	// Only "id" should appear as a column header (nested objects/arrays are skipped)
	if !strings.Contains(out, "id") {
		t.Errorf("simple field should be in headers, got: %s", out)
	}
}

func TestRenderArrayAsTable_NilValueInRow(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}
	arr := []interface{}{
		map[string]interface{}{"id": "1", "value": nil},
	}
	p.renderArrayAsTable(arr)
	out := buf.String()
	if !strings.Contains(out, "id") {
		t.Errorf("should render table, got: %s", out)
	}
}

// --- printPlain with map containing arrays of non-maps ---

func TestPrintPlain_MapWithScalarArray(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatPlain, Writer: &buf}
	data := map[string]interface{}{
		"tags": []interface{}{"go", "test", "mcp"},
	}
	p.printPlainMap(data, "")
	out := buf.String()
	if !strings.Contains(out, "(3 items)") {
		t.Errorf("should show item count, got: %s", out)
	}
	if !strings.Contains(out, "go") {
		t.Errorf("should list items, got: %s", out)
	}
}

func TestPrintPlain_MapWithIndent(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatPlain, Writer: &buf}
	data := map[string]interface{}{
		"key": "value",
	}
	p.printPlainMap(data, ">>")
	out := buf.String()
	if !strings.Contains(out, ">>key: value") {
		t.Errorf("should use indent prefix, got: %s", out)
	}
}

// --- Printer.Err with different formats ---

func TestPrinter_Err_PlainFormat(t *testing.T) {
	p := &Printer{Format: FormatPlain, Writer: &bytes.Buffer{}}
	code := p.Err(exitcode.InvalidInput, "bad argument")
	if code != exitcode.InvalidInput {
		t.Errorf("Err returned %d, want %d", code, exitcode.InvalidInput)
	}
}

func TestPrinter_Err_TableFormat(t *testing.T) {
	p := &Printer{Format: FormatTable, Writer: &bytes.Buffer{}}
	code := p.Err(exitcode.NotFound, "resource missing")
	if code != exitcode.NotFound {
		t.Errorf("Err returned %d, want %d", code, exitcode.NotFound)
	}
}

// --- Printer.ErrExit returns proper ExitError ---

func TestPrinter_ErrExit_AllCodes(t *testing.T) {
	codes := []int{
		exitcode.AuthRequired,
		exitcode.AuthExpired,
		exitcode.PermissionDenied,
		exitcode.NotFound,
		exitcode.RateLimited,
		exitcode.InvalidInput,
	}
	for _, code := range codes {
		p := &Printer{Format: FormatJSON, Writer: &bytes.Buffer{}}
		err := p.ErrExit(code, "test message")
		ee, ok := err.(*ExitError)
		if !ok {
			t.Errorf("code %d: expected *ExitError, got %T", code, err)
			continue
		}
		if ee.Code != code {
			t.Errorf("code %d: ExitError.Code = %d", code, ee.Code)
		}
		if ee.Error() != "test message" {
			t.Errorf("code %d: Error() = %q", code, ee.Error())
		}
	}
}

// --- Printer.Success with table format ---

func TestPrinter_SuccessTable_MapWithArray(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}
	data := map[string]interface{}{
		"results": []interface{}{
			map[string]interface{}{"name": "alpha", "score": float64(95)},
			map[string]interface{}{"name": "beta", "score": float64(87)},
		},
	}
	p.Success(data)
	out := buf.String()
	if !strings.Contains(out, "alpha") {
		t.Errorf("table output should contain data, got: %s", out)
	}
}

// --- filterFields additional tests ---

func TestFilterFields_Struct(t *testing.T) {
	type testStruct struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	data := testStruct{ID: "1", Name: "test", Age: 30}
	filtered := filterFields(data, []string{"id", "name"})
	m, ok := filtered.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", filtered)
	}
	if m["id"] != "1" {
		t.Error("id should be kept")
	}
	if m["name"] != "test" {
		t.Error("name should be kept")
	}
	if _, ok := m["age"]; ok {
		t.Error("age should be filtered out")
	}
}

func TestFilterFields_NonexistentField(t *testing.T) {
	data := map[string]interface{}{"id": "123"}
	filtered := filterFields(data, []string{"nonexistent"})
	m, ok := filtered.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", filtered)
	}
	if len(m) != 0 {
		t.Errorf("filtering by nonexistent field should produce empty map, got %v", m)
	}
}

// --- Response type tests ---

func TestResponse_JSONFields(t *testing.T) {
	resp := Response{
		Status: "ok",
		Data:   map[string]string{"key": "value"},
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"status":"ok"`) {
		t.Errorf("should contain status field, got: %s", raw)
	}
}

// --- Table with empty rows ---

func TestTable_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}
	p.Table([]string{"Col1", "Col2"}, [][]string{})
	out := buf.String()
	if !strings.Contains(out, "Col1") {
		t.Errorf("headers should be present even with no rows, got: %s", out)
	}
}

func TestTable_SingleRow(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}
	p.Table([]string{"Name"}, [][]string{{"Alice"}})
	out := buf.String()
	if !strings.Contains(out, "Alice") {
		t.Errorf("should contain row data, got: %s", out)
	}
}
