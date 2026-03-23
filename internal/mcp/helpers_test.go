package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStrArg(t *testing.T) {
	args := map[string]interface{}{"name": "alice", "count": 42}

	if got := strArg(args, "name"); got != "alice" {
		t.Errorf("strArg(name) = %q, want alice", got)
	}
	if got := strArg(args, "missing"); got != "" {
		t.Errorf("strArg(missing) = %q, want empty", got)
	}
	// Non-string value should return empty
	if got := strArg(args, "count"); got != "" {
		t.Errorf("strArg(count) = %q, want empty (not a string)", got)
	}
}

func TestStrArg_EmptyString(t *testing.T) {
	args := map[string]interface{}{"empty": ""}
	if got := strArg(args, "empty"); got != "" {
		t.Errorf("strArg(empty) = %q, want empty string", got)
	}
}

func TestStrArg_NilArgs(t *testing.T) {
	// nil map should not panic, just return ""
	if got := strArg(nil, "anything"); got != "" {
		t.Errorf("strArg(nil, anything) = %q, want empty", got)
	}
}

func TestIntArg(t *testing.T) {
	args := map[string]interface{}{"limit": float64(25), "exact": 10}

	if got := intArg(args, "limit", 5); got != 25 {
		t.Errorf("intArg(limit) = %d, want 25", got)
	}
	if got := intArg(args, "exact", 5); got != 10 {
		t.Errorf("intArg(exact) = %d, want 10", got)
	}
	if got := intArg(args, "missing", 99); got != 99 {
		t.Errorf("intArg(missing) = %d, want 99", got)
	}
}

func TestIntArg_NonNumberFallsToDefault(t *testing.T) {
	args := map[string]interface{}{"bad": "not a number"}
	if got := intArg(args, "bad", 42); got != 42 {
		t.Errorf("intArg(bad string) = %d, want default 42", got)
	}
}

func TestIntArg_ZeroFloat(t *testing.T) {
	args := map[string]interface{}{"zero": float64(0)}
	if got := intArg(args, "zero", 99); got != 0 {
		t.Errorf("intArg(zero float64) = %d, want 0", got)
	}
}

func TestIntArg_NegativeFloat(t *testing.T) {
	args := map[string]interface{}{"neg": float64(-5)}
	if got := intArg(args, "neg", 10); got != -5 {
		t.Errorf("intArg(negative) = %d, want -5", got)
	}
}

func TestIntArg_NilArgs(t *testing.T) {
	if got := intArg(nil, "x", 7); got != 7 {
		t.Errorf("intArg(nil, x, 7) = %d, want 7", got)
	}
}

func TestBoolArg(t *testing.T) {
	args := map[string]interface{}{"flag": true, "off": false}

	if got := boolArg(args, "flag"); !got {
		t.Error("boolArg(flag) = false, want true")
	}
	if got := boolArg(args, "off"); got {
		t.Error("boolArg(off) = true, want false")
	}
	if got := boolArg(args, "missing"); got {
		t.Error("boolArg(missing) = true, want false")
	}
}

func TestBoolArg_NonBoolFallsToFalse(t *testing.T) {
	args := map[string]interface{}{"bad": "true", "num": 1}
	if got := boolArg(args, "bad"); got {
		t.Error("boolArg(string 'true') should return false, not coerce")
	}
	if got := boolArg(args, "num"); got {
		t.Error("boolArg(int 1) should return false, not coerce")
	}
}

func TestBoolArg_NilArgs(t *testing.T) {
	if got := boolArg(nil, "x"); got {
		t.Error("boolArg(nil, x) = true, want false")
	}
}

func TestSplitArg(t *testing.T) {
	tests := []struct {
		name string
		args map[string]interface{}
		key  string
		want []string
	}{
		{"empty", map[string]interface{}{}, "x", nil},
		{"single", map[string]interface{}{"x": "alice"}, "x", []string{"alice"}},
		{"multi", map[string]interface{}{"x": "a, b ,c"}, "x", []string{"a", "b", "c"}},
		{"trailing comma", map[string]interface{}{"x": "a,b,"}, "x", []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitArg(tt.args, tt.key)
			if len(got) != len(tt.want) {
				t.Fatalf("splitArg() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitArg()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSplitArg_EmptyStringReturnsNil(t *testing.T) {
	args := map[string]interface{}{"x": ""}
	got := splitArg(args, "x")
	if got != nil {
		t.Errorf("splitArg(empty string) = %v, want nil", got)
	}
}

func TestSplitArg_OnlyCommasReturnsNil(t *testing.T) {
	args := map[string]interface{}{"x": ",,,"}
	got := splitArg(args, "x")
	if got != nil {
		t.Errorf("splitArg(only commas) = %v, want nil", got)
	}
}

func TestSplitArg_WhitespaceAroundValues(t *testing.T) {
	args := map[string]interface{}{"x": "  alice , bob , carol  "}
	got := splitArg(args, "x")
	want := []string{"alice", "bob", "carol"}
	if len(got) != len(want) {
		t.Fatalf("splitArg(spaced) = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("splitArg(spaced)[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSplitArg_NonStringReturnsNil(t *testing.T) {
	args := map[string]interface{}{"x": 123}
	got := splitArg(args, "x")
	if got != nil {
		t.Errorf("splitArg(int) = %v, want nil", got)
	}
}

func TestJsonResult(t *testing.T) {
	result, err := jsonResult(map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("jsonResult error: %v", err)
	}
	if result.IsError {
		t.Error("jsonResult should not set IsError")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}
	if result.Content[0].Type != "text" {
		t.Errorf("expected type text, got %s", result.Content[0].Type)
	}
	if result.Content[0].Text == "" {
		t.Error("expected non-empty text")
	}
}

func TestJsonResult_ContentIsValidJSON(t *testing.T) {
	input := map[string]interface{}{"count": 42, "items": []string{"a", "b"}}
	result, err := jsonResult(input)
	if err != nil {
		t.Fatalf("jsonResult error: %v", err)
	}

	// The text content must be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &parsed); err != nil {
		t.Fatalf("jsonResult text is not valid JSON: %v", err)
	}

	// Verify the round-tripped data
	if parsed["count"] != float64(42) {
		t.Errorf("count = %v, want 42", parsed["count"])
	}
	items, ok := parsed["items"].([]interface{})
	if !ok || len(items) != 2 {
		t.Errorf("items = %v, want [a, b]", parsed["items"])
	}
}

func TestJsonResult_NestedStruct(t *testing.T) {
	type inner struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	input := struct {
		Items []inner `json:"items"`
	}{
		Items: []inner{{ID: 1, Name: "test"}},
	}
	result, err := jsonResult(input)
	if err != nil {
		t.Fatalf("jsonResult error: %v", err)
	}

	if !strings.Contains(result.Content[0].Text, `"name": "test"`) {
		t.Errorf("expected nested struct field in output, got: %s", result.Content[0].Text)
	}
}

func TestJsonResult_EmptyMap(t *testing.T) {
	result, err := jsonResult(map[string]interface{}{})
	if err != nil {
		t.Fatalf("jsonResult error: %v", err)
	}
	if result.Content[0].Text != "{}" {
		t.Errorf("expected '{}', got %q", result.Content[0].Text)
	}
}

func TestJsonResult_NilSlice(t *testing.T) {
	// nil slice serializes as "null" in JSON
	result, err := jsonResult(map[string]interface{}{"items": ([]string)(nil)})
	if err != nil {
		t.Fatalf("jsonResult error: %v", err)
	}
	if !strings.Contains(result.Content[0].Text, "null") {
		t.Errorf("expected null for nil slice, got: %s", result.Content[0].Text)
	}
}
