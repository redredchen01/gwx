package mcp

import (
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
