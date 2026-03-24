package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// --- FilterFields tests ---

func TestFilterFields_KeepsSelected(t *testing.T) {
	data := map[string]interface{}{
		"id":      "123",
		"name":    "test",
		"secret":  "hidden",
		"details": "stuff",
	}
	filtered := filterFields(data, []string{"id", "name"})
	m, ok := filtered.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", filtered)
	}
	if m["id"] != "123" {
		t.Error("id should be kept")
	}
	if m["name"] != "test" {
		t.Error("name should be kept")
	}
	if _, ok := m["secret"]; ok {
		t.Error("secret should be filtered out")
	}
	if _, ok := m["details"]; ok {
		t.Error("details should be filtered out")
	}
}

func TestFilterFields_EmptyFields(t *testing.T) {
	data := map[string]interface{}{"id": "123"}
	filtered := filterFields(data, []string{})
	m, ok := filtered.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", filtered)
	}
	// Empty field list should return empty map
	if len(m) != 0 {
		t.Errorf("empty field list should produce empty result, got %d keys", len(m))
	}
}

func TestFilterFields_Array(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"id": "1", "name": "a", "extra": "x"},
		map[string]interface{}{"id": "2", "name": "b", "extra": "y"},
	}
	raw, _ := json.Marshal(data)
	var arr []interface{}
	json.Unmarshal(raw, &arr)

	filtered := filterFields(arr, []string{"id"})
	farr, ok := filtered.([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", filtered)
	}
	if len(farr) != 2 {
		t.Fatalf("expected 2 items, got %d", len(farr))
	}
}

func TestFilterFields_NonMap(t *testing.T) {
	// Scalar data should pass through unchanged
	filtered := filterFields("hello", []string{"id"})
	if filtered != "hello" {
		t.Errorf("scalar should pass through, got %v", filtered)
	}
}

// --- Printer with fields filter ---

func TestPrinter_SuccessWithFields(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatJSON, Writer: &buf, Fields: []string{"id"}}

	p.Success(map[string]interface{}{
		"id":     "123",
		"name":   "test",
		"secret": "hidden",
	})

	var resp Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	raw, _ := json.Marshal(resp.Data)
	var data map[string]interface{}
	json.Unmarshal(raw, &data)

	if data["id"] != "123" {
		t.Error("id should be present")
	}
	if _, ok := data["secret"]; ok {
		t.Error("secret should be filtered out")
	}
}

// --- NewPrinter ---

func TestNewPrinter_Defaults(t *testing.T) {
	p := NewPrinter(FormatJSON)
	if p == nil {
		t.Fatal("NewPrinter returned nil")
	}
	if p.Format != FormatJSON {
		t.Errorf("Format = %d, want FormatJSON", p.Format)
	}
	if p.Writer == nil {
		t.Error("Writer should not be nil")
	}
	if len(p.Fields) != 0 {
		t.Error("Fields should be empty by default")
	}
}

func TestNewPrinter_AllFormats(t *testing.T) {
	for _, f := range []Format{FormatJSON, FormatPlain, FormatTable} {
		p := NewPrinter(f)
		if p.Format != f {
			t.Errorf("NewPrinter(%d).Format = %d", f, p.Format)
		}
	}
}

// --- ExitError ---

func TestExitError_Interface(t *testing.T) {
	var err error = &ExitError{Code: 42, Msg: "test error"}
	if err.Error() != "test error" {
		t.Errorf("Error() = %q, want 'test error'", err.Error())
	}
}

// --- printTable with direct array ---

func TestPrintTable_DirectArray(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}

	data := []interface{}{
		map[string]interface{}{"id": "1", "name": "alice"},
		map[string]interface{}{"id": "2", "name": "bob"},
	}
	raw, _ := json.Marshal(data)
	var parsed interface{}
	json.Unmarshal(raw, &parsed)

	p.printTable(parsed)
	out := buf.String()
	if !strings.Contains(out, "alice") {
		t.Errorf("table should contain 'alice', got: %s", out)
	}
}

// --- renderArrayAsTable edge cases ---

func TestRenderArrayAsTable_Empty(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}
	p.renderArrayAsTable([]interface{}{})
	if buf.Len() != 0 {
		t.Error("empty array should produce no output")
	}
}

func TestRenderArrayAsTable_NonMapItems(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}
	p.renderArrayAsTable([]interface{}{"hello", "world"})
	out := buf.String()
	if !strings.Contains(out, "hello") {
		t.Errorf("non-map items should be printed directly, got: %s", out)
	}
}

func TestRenderArrayAsTable_TruncatesLongValues(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}

	longVal := strings.Repeat("x", 100)
	p.renderArrayAsTable([]interface{}{
		map[string]interface{}{"text": longVal},
	})
	out := buf.String()
	if strings.Contains(out, longVal) {
		t.Error("values over 60 chars should be truncated")
	}
	if !strings.Contains(out, "...") {
		t.Error("truncated values should end with '...'")
	}
}

// --- printPlain edge cases ---

func TestPrintPlain_Scalar(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatPlain, Writer: &buf}
	p.printPlain(42)
	if buf.String() != "42\n" {
		t.Errorf("scalar should print directly, got %q", buf.String())
	}
}

func TestPrintPlain_String(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatPlain, Writer: &buf}
	p.Success("hello world")
	if buf.String() != "hello world\n" {
		t.Errorf("string should print directly, got %q", buf.String())
	}
}

func TestPrintPlain_Array(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatPlain, Writer: &buf}
	p.printPlain([]interface{}{
		map[string]interface{}{"name": "a"},
		map[string]interface{}{"name": "b"},
	})
	out := buf.String()
	if !strings.Contains(out, "name: a") {
		t.Errorf("array of maps should print fields, got: %s", out)
	}
	if !strings.Contains(out, "---") {
		t.Errorf("array items should be separated by '---', got: %s", out)
	}
}

// --- MapSummary with empty map ---

func TestMapSummary_EmptyMap(t *testing.T) {
	got := mapSummary(map[string]interface{}{})
	if got != "" {
		t.Errorf("empty map summary should be empty, got %q", got)
	}
}
