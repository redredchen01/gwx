package api

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// isNumeric
// ---------------------------------------------------------------------------

func TestIsNumeric_ValidIntegers(t *testing.T) {
	cases := []string{"0", "1", "42", "999999", "-1", "-42"}
	for _, s := range cases {
		if !isNumeric(s) {
			t.Errorf("isNumeric(%q) = false, want true", s)
		}
	}
}

func TestIsNumeric_ValidFloats(t *testing.T) {
	// "1." is accepted by the implementation (trailing dot is allowed).
	cases := []string{"3.14", "0.5", "-1.5", "100.0", "1."}
	for _, s := range cases {
		if !isNumeric(s) {
			t.Errorf("isNumeric(%q) = false, want true", s)
		}
	}
}

func TestIsNumeric_Invalid(t *testing.T) {
	cases := []string{"", "abc", "1a", "1.2.3", "--1", "1-2"}
	for _, s := range cases {
		if isNumeric(s) {
			t.Errorf("isNumeric(%q) = true, want false", s)
		}
	}
}

// ---------------------------------------------------------------------------
// uniqueValues
// ---------------------------------------------------------------------------

func TestUniqueValues_NoDuplicates(t *testing.T) {
	input := []string{"a", "b", "c"}
	got := uniqueValues(input)
	if len(got) != 3 {
		t.Errorf("uniqueValues(%v) len = %d, want 3", input, len(got))
	}
}

func TestUniqueValues_WithDuplicates(t *testing.T) {
	input := []string{"x", "y", "x", "z", "y"}
	got := uniqueValues(input)
	if len(got) != 3 {
		t.Errorf("uniqueValues(%v) len = %d, want 3", input, len(got))
	}
}

func TestUniqueValues_Empty(t *testing.T) {
	got := uniqueValues([]string{})
	if len(got) != 0 {
		t.Errorf("uniqueValues([]) len = %d, want 0", len(got))
	}
}

func TestUniqueValues_PreservesOrder(t *testing.T) {
	input := []string{"c", "a", "b", "a"}
	got := uniqueValues(input)
	if len(got) != 3 || got[0] != "c" || got[1] != "a" || got[2] != "b" {
		t.Errorf("uniqueValues order wrong: %v", got)
	}
}

// ---------------------------------------------------------------------------
// truncateSlice
// ---------------------------------------------------------------------------

func TestTruncateSlice_BelowLimit(t *testing.T) {
	s := []string{"a", "b"}
	got := truncateSlice(s, 5)
	if len(got) != 2 {
		t.Errorf("truncateSlice below limit: got len %d, want 2", len(got))
	}
}

func TestTruncateSlice_ExactLimit(t *testing.T) {
	s := []string{"a", "b", "c"}
	got := truncateSlice(s, 3)
	if len(got) != 3 {
		t.Errorf("truncateSlice exact limit: got len %d, want 3", len(got))
	}
}

func TestTruncateSlice_AboveLimit(t *testing.T) {
	s := []string{"a", "b", "c", "d", "e"}
	got := truncateSlice(s, 3)
	if len(got) != 3 {
		t.Errorf("truncateSlice above limit: got len %d, want 3", len(got))
	}
	if got[0] != "a" || got[2] != "c" {
		t.Errorf("truncateSlice content wrong: %v", got)
	}
}

// ---------------------------------------------------------------------------
// validateCell
// ---------------------------------------------------------------------------

func TestValidateCell_RequiredEmpty(t *testing.T) {
	col := ColumnRule{Header: "Name", Type: "freetext", Required: true}
	msg := validateCell(col, "")
	if msg == "" {
		t.Error("validateCell required+empty: expected error, got empty string")
	}
	if !strings.Contains(msg, "required") {
		t.Errorf("validateCell required+empty: error should mention 'required', got %q", msg)
	}
}

func TestValidateCell_OptionalEmpty(t *testing.T) {
	col := ColumnRule{Header: "Note", Type: "freetext", Required: false}
	msg := validateCell(col, "")
	if msg != "" {
		t.Errorf("validateCell optional+empty: expected empty string, got %q", msg)
	}
}

func TestValidateCell_EnumValid(t *testing.T) {
	col := ColumnRule{Header: "Status", Type: "enum", EnumValues: []string{"Active", "Inactive"}, Required: true}
	msg := validateCell(col, "active") // case-insensitive
	if msg != "" {
		t.Errorf("validateCell enum valid (case-insensitive): got %q", msg)
	}
}

func TestValidateCell_EnumInvalid(t *testing.T) {
	col := ColumnRule{Header: "Status", Type: "enum", EnumValues: []string{"Active", "Inactive"}, Required: true}
	msg := validateCell(col, "Unknown")
	if msg == "" {
		t.Error("validateCell enum invalid: expected error")
	}
}

func TestValidateCell_URLValid(t *testing.T) {
	col := ColumnRule{Header: "Link", Type: "url", Required: true}
	for _, url := range []string{"https://example.com", "http://foo.bar"} {
		if msg := validateCell(col, url); msg != "" {
			t.Errorf("validateCell URL valid (%q): got %q", url, msg)
		}
	}
}

func TestValidateCell_URLInvalid(t *testing.T) {
	col := ColumnRule{Header: "Link", Type: "url", Required: true}
	msg := validateCell(col, "not-a-url")
	if msg == "" {
		t.Error("validateCell URL invalid: expected error")
	}
}

func TestValidateCell_NumberValid(t *testing.T) {
	col := ColumnRule{Header: "Age", Type: "number", Required: true}
	for _, n := range []string{"0", "42", "3.14", "-7"} {
		if msg := validateCell(col, n); msg != "" {
			t.Errorf("validateCell number valid (%q): got %q", n, msg)
		}
	}
}

func TestValidateCell_NumberInvalid(t *testing.T) {
	col := ColumnRule{Header: "Age", Type: "number", Required: true}
	msg := validateCell(col, "abc")
	if msg == "" {
		t.Error("validateCell number invalid: expected error")
	}
}

// ---------------------------------------------------------------------------
// analyzeColumn
// ---------------------------------------------------------------------------

func TestAnalyzeColumn_Empty(t *testing.T) {
	rows := [][]interface{}{{"", ""}, {"", ""}}
	col := analyzeColumn(0, "Notes", rows)
	if col.Type != "empty" {
		t.Errorf("analyzeColumn all-empty: type = %q, want 'empty'", col.Type)
	}
	if col.FillRate != "0/2" {
		t.Errorf("analyzeColumn all-empty: FillRate = %q, want '0/2'", col.FillRate)
	}
}

func TestAnalyzeColumn_NumberType(t *testing.T) {
	rows := [][]interface{}{{"10"}, {"20"}, {"30"}}
	col := analyzeColumn(0, "Score", rows)
	if col.Type != "number" {
		t.Errorf("analyzeColumn numbers: type = %q, want 'number'", col.Type)
	}
	if !col.Required {
		t.Error("analyzeColumn all-filled: Required should be true")
	}
}

func TestAnalyzeColumn_URLType(t *testing.T) {
	rows := [][]interface{}{
		{"https://a.com"},
		{"https://b.com"},
		{"https://c.com"},
	}
	col := analyzeColumn(0, "Link", rows)
	if col.Type != "url" {
		t.Errorf("analyzeColumn URLs: type = %q, want 'url'", col.Type)
	}
}

func TestAnalyzeColumn_EnumType(t *testing.T) {
	rows := [][]interface{}{
		{"Active"}, {"Inactive"}, {"Active"}, {"Active"}, {"Inactive"},
	}
	col := analyzeColumn(0, "Status", rows)
	if col.Type != "enum" {
		t.Errorf("analyzeColumn enum: type = %q, want 'enum'", col.Type)
	}
	if len(col.EnumValues) != 2 {
		t.Errorf("analyzeColumn enum: EnumValues len = %d, want 2", len(col.EnumValues))
	}
}

func TestAnalyzeColumn_FreetextType(t *testing.T) {
	rows := [][]interface{}{
		{"Alice"}, {"Bob"}, {"Charlie"}, {"Dave"}, {"Eve"},
		{"Frank"}, {"Grace"}, {"Heidi"},
	}
	col := analyzeColumn(0, "Name", rows)
	if col.Type != "freetext" {
		t.Errorf("analyzeColumn freetext: type = %q, want 'freetext'", col.Type)
	}
}

func TestAnalyzeColumn_OptionalHint(t *testing.T) {
	rows := [][]interface{}{{"hello"}, {"world"}}
	col := analyzeColumn(0, "备注(optional)", rows)
	if col.Required {
		t.Error("analyzeColumn optional header: Required should be false")
	}
}

// ---------------------------------------------------------------------------
// generateFillInstructions
// ---------------------------------------------------------------------------

func TestGenerateFillInstructions_ContainsHeaders(t *testing.T) {
	cols := []ColumnRule{
		{Index: 0, Header: "Name", Type: "freetext", Required: true, Description: "Free text."},
		{Index: 1, Header: "Status", Type: "enum", Required: false, EnumValues: []string{"A", "B"}, Description: "Select from: A / B."},
	}
	out := generateFillInstructions(cols)
	if !strings.Contains(out, "Name") {
		t.Error("generateFillInstructions: missing 'Name'")
	}
	if !strings.Contains(out, "Status") {
		t.Error("generateFillInstructions: missing 'Status'")
	}
	if !strings.Contains(out, "REQUIRED") {
		t.Error("generateFillInstructions: missing 'REQUIRED' keyword")
	}
	if !strings.Contains(out, "optional") {
		t.Error("generateFillInstructions: missing 'optional' keyword")
	}
}
