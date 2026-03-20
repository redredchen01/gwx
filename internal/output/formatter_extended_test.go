package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/redredchen01/gwx/internal/exitcode"
)

func TestSuggestedAction(t *testing.T) {
	tests := []struct {
		code int
		msg  string
		want string
	}{
		{exitcode.AuthRequired, "no token", "gwx onboard"},
		{exitcode.AuthExpired, "expired", "gwx auth login"},
		{exitcode.PermissionDenied, "not in allowlist", "GWX_ENABLE_COMMANDS"},
		{exitcode.PermissionDenied, "scope missing", "re-authorize"},
		{exitcode.NotFound, "not found", "list"},
		{exitcode.RateLimited, "429", "Wait"},
		{exitcode.CircuitOpen, "circuit open", "circuit breaker"},
		{exitcode.Conflict, "conflict", "Retry"},
		{exitcode.InvalidInput, "bad input", "--help"},
		{exitcode.GeneralError, "unknown", ""},
	}
	for _, tt := range tests {
		got := suggestedAction(tt.code, tt.msg)
		if tt.want == "" {
			if got != "" {
				t.Errorf("suggestedAction(%d, %q) = %q, want empty", tt.code, tt.msg, got)
			}
			continue
		}
		if !strings.Contains(got, tt.want) {
			t.Errorf("suggestedAction(%d, %q) = %q, want containing %q", tt.code, tt.msg, got, tt.want)
		}
	}
}

func TestPrintPlainMap(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatPlain, Writer: &buf}

	data := map[string]interface{}{
		"count": 2,
		"name":  "test",
	}
	p.printPlainMap(data, "")

	out := buf.String()
	// Keys should be sorted (count before name)
	countIdx := strings.Index(out, "count:")
	nameIdx := strings.Index(out, "name:")
	if countIdx < 0 || nameIdx < 0 {
		t.Fatalf("expected both keys in output, got: %s", out)
	}
	if countIdx > nameIdx {
		t.Errorf("keys should be sorted: count before name, got: %s", out)
	}
}

func TestPrintPlainMapNested(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatPlain, Writer: &buf}

	data := map[string]interface{}{
		"outer": map[string]interface{}{
			"inner": "value",
		},
	}
	p.printPlainMap(data, "")

	out := buf.String()
	if !strings.Contains(out, "outer:") || !strings.Contains(out, "inner: value") {
		t.Fatalf("nested map not rendered correctly: %s", out)
	}
}

func TestPrintPlainMapArray(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatPlain, Writer: &buf}

	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"title": "first", "id": "1"},
			map[string]interface{}{"title": "second", "id": "2"},
		},
	}
	p.printPlainMap(data, "")

	out := buf.String()
	if !strings.Contains(out, "(2 items)") {
		t.Fatalf("array count not shown: %s", out)
	}
	if !strings.Contains(out, "title=first") {
		t.Fatalf("array item summary missing: %s", out)
	}
}

func TestPrintTableFromMap(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}

	// Map with array — should render array as table
	data := map[string]interface{}{
		"count": 2,
		"messages": []interface{}{
			map[string]interface{}{"id": "1", "subject": "hello"},
			map[string]interface{}{"id": "2", "subject": "world"},
		},
	}
	p.printTable(data)

	out := buf.String()
	if !strings.Contains(out, "id") || !strings.Contains(out, "subject") {
		t.Fatalf("table headers missing: %s", out)
	}
	if !strings.Contains(out, "hello") || !strings.Contains(out, "world") {
		t.Fatalf("table data missing: %s", out)
	}
}

func TestPrintTableNoArray(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}

	// Map without array — should render as key-value table
	data := map[string]interface{}{
		"name":  "test",
		"count": 42,
	}
	p.printTable(data)

	out := buf.String()
	if !strings.Contains(out, "count") || !strings.Contains(out, "name") {
		t.Fatalf("key-value table missing keys: %s", out)
	}
}

func TestMapSummary(t *testing.T) {
	// Priority fields: subject, title, name, summary, id, email, status
	m := map[string]interface{}{
		"subject": "hello",
		"id":      "123",
		"extra":   "ignored",
	}
	got := mapSummary(m)
	if !strings.Contains(got, "subject=hello") {
		t.Errorf("mapSummary should contain subject, got: %s", got)
	}
	if !strings.Contains(got, "id=123") {
		t.Errorf("mapSummary should contain id, got: %s", got)
	}
}

func TestMapSummaryFallback(t *testing.T) {
	// No priority fields — should fallback to first 3
	m := map[string]interface{}{
		"foo": "bar",
		"baz": "qux",
	}
	got := mapSummary(m)
	if got == "" {
		t.Error("mapSummary fallback should not be empty")
	}
}

func TestRenderArrayAsTableSorted(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}

	arr := []interface{}{
		map[string]interface{}{"z_col": "1", "a_col": "2"},
	}
	p.renderArrayAsTable(arr)

	out := buf.String()
	aIdx := strings.Index(out, "a_col")
	zIdx := strings.Index(out, "z_col")
	if aIdx < 0 || zIdx < 0 {
		t.Fatalf("headers missing: %s", out)
	}
	if aIdx > zIdx {
		t.Errorf("headers should be sorted: a_col before z_col, got: %s", out)
	}
}
