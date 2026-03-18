package api

import (
	"testing"
)

// ---------------------------------------------------------------------------
// indexByKey
// ---------------------------------------------------------------------------

func TestIndexByKey_Basic(t *testing.T) {
	rows := [][]interface{}{
		{"alice", "30"},
		{"bob", "25"},
		{"charlie", "40"},
	}
	m := indexByKey(rows)
	if len(m) != 3 {
		t.Errorf("indexByKey len = %d, want 3", len(m))
	}
	if row, ok := m["alice"]; !ok {
		t.Error("indexByKey: missing 'alice'")
	} else if row[1] != "30" {
		t.Errorf("indexByKey alice[1] = %v, want '30'", row[1])
	}
}

func TestIndexByKey_SkipsEmptyFirstCell(t *testing.T) {
	rows := [][]interface{}{
		{"", "ignored"},
		{"key1", "value1"},
	}
	m := indexByKey(rows)
	if len(m) != 1 {
		t.Errorf("indexByKey with empty first cell: len = %d, want 1", len(m))
	}
	if _, ok := m["key1"]; !ok {
		t.Error("indexByKey: missing 'key1'")
	}
}

func TestIndexByKey_EmptyRows(t *testing.T) {
	m := indexByKey([][]interface{}{})
	if len(m) != 0 {
		t.Errorf("indexByKey empty rows: len = %d, want 0", len(m))
	}
}

func TestIndexByKey_EmptyRow(t *testing.T) {
	rows := [][]interface{}{{}}
	m := indexByKey(rows)
	if len(m) != 0 {
		t.Errorf("indexByKey single empty row: len = %d, want 0", len(m))
	}
}

func TestIndexByKey_TrimsWhitespace(t *testing.T) {
	rows := [][]interface{}{{"  key  ", "val"}}
	m := indexByKey(rows)
	if _, ok := m["key"]; !ok {
		t.Error("indexByKey: should trim whitespace from key")
	}
}

// ---------------------------------------------------------------------------
// cellStr
// ---------------------------------------------------------------------------

func TestCellStr_WithinBounds(t *testing.T) {
	row := []interface{}{"hello", 42, true}
	got := cellStr(row, 0)
	if got != "hello" {
		t.Errorf("cellStr index 0 = %q, want 'hello'", got)
	}
}

func TestCellStr_OutOfBounds(t *testing.T) {
	row := []interface{}{"a", "b"}
	got := cellStr(row, 5)
	if got != "" {
		t.Errorf("cellStr out of bounds = %q, want ''", got)
	}
}

func TestCellStr_TrimsWhitespace(t *testing.T) {
	row := []interface{}{"  value  "}
	got := cellStr(row, 0)
	if got != "value" {
		t.Errorf("cellStr trim = %q, want 'value'", got)
	}
}

func TestCellStr_NonStringValue(t *testing.T) {
	row := []interface{}{123}
	got := cellStr(row, 0)
	if got != "123" {
		t.Errorf("cellStr non-string = %q, want '123'", got)
	}
}
