package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPrinter_SuccessJSON(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatJSON, Writer: &buf}

	p.Success(map[string]string{"key": "value"})

	var resp Response
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	if resp.Status != "ok" {
		t.Fatalf("expected status 'ok', got %q", resp.Status)
	}
	if resp.Error != nil {
		t.Fatal("success response should not have error")
	}
}

func TestPrinter_SuccessPlain(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatPlain, Writer: &buf}

	p.Success("hello world")

	if buf.String() != "hello world\n" {
		t.Fatalf("expected 'hello world\\n', got %q", buf.String())
	}
}

func TestPrinter_ErrJSON(t *testing.T) {
	// Err writes to stderr, not the printer's Writer.
	// We test the return value.
	p := &Printer{Format: FormatJSON, Writer: &bytes.Buffer{}}
	code := p.Err(10, "auth required")
	if code != 10 {
		t.Fatalf("expected code 10, got %d", code)
	}
}

func TestPrinter_ErrExit(t *testing.T) {
	p := &Printer{Format: FormatJSON, Writer: &bytes.Buffer{}}
	err := p.ErrExit(30, "rate limited")

	if err == nil {
		t.Fatal("ErrExit should return error")
	}

	ee, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T", err)
	}
	if ee.Code != 30 {
		t.Fatalf("expected code 30, got %d", ee.Code)
	}
	if ee.Msg != "rate limited" {
		t.Fatalf("expected msg 'rate limited', got %q", ee.Msg)
	}
	if ee.Error() != "rate limited" {
		t.Fatalf("Error() should return msg")
	}
}

func TestPrinter_Table(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Format: FormatTable, Writer: &buf}

	headers := []string{"Name", "Value"}
	rows := [][]string{
		{"a", "1"},
		{"b", "2"},
	}
	p.Table(headers, rows)

	out := buf.String()
	if len(out) == 0 {
		t.Fatal("table output should not be empty")
	}
	// Should contain headers and data
	if !containsStr(out, "Name") || !containsStr(out, "Value") {
		t.Fatalf("table should contain headers, got: %s", out)
	}
	if !containsStr(out, "a") || !containsStr(out, "2") {
		t.Fatalf("table should contain data, got: %s", out)
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input string
		want  Format
	}{
		{"json", FormatJSON},
		{"JSON", FormatJSON},
		{"plain", FormatPlain},
		{"table", FormatTable},
		{"unknown", FormatJSON},
		{"", FormatJSON},
	}
	for _, tt := range tests {
		got := ParseFormat(tt.input)
		if got != tt.want {
			t.Errorf("ParseFormat(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
