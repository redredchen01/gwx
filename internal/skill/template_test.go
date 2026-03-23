package skill

import (
	"fmt"
	"strings"
	"testing"
)

func TestRenderArgs_InputReference(t *testing.T) {
	raw := map[string]string{
		"to": "{{.input.email}}",
	}
	inputs := map[string]string{
		"email": "alice@example.com",
	}
	store := map[string]interface{}{}

	out, err := renderArgs(raw, inputs, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["to"] != "alice@example.com" {
		t.Errorf("to = %v, want %q", out["to"], "alice@example.com")
	}
}

func TestRenderArgs_StepReference(t *testing.T) {
	raw := map[string]string{
		"data": "{{.steps.fetch}}",
	}
	inputs := map[string]string{}
	store := map[string]interface{}{
		"fetch": map[string]interface{}{
			"messages": []interface{}{"a", "b"},
		},
	}

	out, err := renderArgs(raw, inputs, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := out["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data = %T, want map[string]interface{}", out["data"])
	}
	msgs, ok := m["messages"].([]interface{})
	if !ok || len(msgs) != 2 {
		t.Errorf("messages = %v, want [a b]", m["messages"])
	}
}

func TestRenderArgs_StepFieldDrill(t *testing.T) {
	raw := map[string]string{
		"subject": "{{.steps.fetch.result.title}}",
	}
	inputs := map[string]string{}
	store := map[string]interface{}{
		"fetch": map[string]interface{}{
			"result": map[string]interface{}{
				"title": "Hello World",
			},
		},
	}

	out, err := renderArgs(raw, inputs, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["subject"] != "Hello World" {
		t.Errorf("subject = %v, want %q", out["subject"], "Hello World")
	}
}

func TestRenderValue_NoTemplate(t *testing.T) {
	val, err := renderValue("plain text", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "plain text" {
		t.Errorf("val = %v, want %q", val, "plain text")
	}
}

func TestRenderValue_SingleTemplatePreservesType(t *testing.T) {
	// A single {{...}} expression should preserve the native type.
	store := map[string]interface{}{
		"count_step": 42,
	}
	val, err := renderValue("{{.steps.count_step}}", map[string]string{}, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	intVal, ok := val.(int)
	if !ok {
		t.Fatalf("val = %T (%v), want int", val, val)
	}
	if intVal != 42 {
		t.Errorf("val = %d, want 42", intVal)
	}
}

func TestRenderValue_SingleTemplatePreservesMap(t *testing.T) {
	store := map[string]interface{}{
		"data": map[string]interface{}{"key": "value"},
	}
	val, err := renderValue("{{.steps.data}}", map[string]string{}, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := val.(map[string]interface{})
	if !ok {
		t.Fatalf("val = %T, want map[string]interface{}", val)
	}
	if m["key"] != "value" {
		t.Errorf("key = %v, want %q", m["key"], "value")
	}
}

func TestRenderValue_MixedTemplateConvertsToString(t *testing.T) {
	inputs := map[string]string{
		"name": "Alice",
	}
	store := map[string]interface{}{
		"num": 99,
	}
	val, err := renderValue("Hello {{.input.name}}, you have {{.steps.num}} items", inputs, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s, ok := val.(string)
	if !ok {
		t.Fatalf("val = %T, want string", val)
	}
	if s != "Hello Alice, you have 99 items" {
		t.Errorf("val = %q", s)
	}
}

func TestRenderValue_UnknownInput(t *testing.T) {
	_, err := renderValue("{{.input.missing}}", map[string]string{}, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for unknown input")
	}
	if !strings.Contains(err.Error(), "unknown input") {
		t.Errorf("error = %q, want substring %q", err, "unknown input")
	}
}

func TestRenderValue_UnknownStep(t *testing.T) {
	_, err := renderValue("{{.steps.nope}}", map[string]string{}, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for unknown step")
	}
	if !strings.Contains(err.Error(), "no output from step") {
		t.Errorf("error = %q, want substring %q", err, "no output from step")
	}
}

func TestRenderValue_UnknownNamespace(t *testing.T) {
	_, err := renderValue("{{.env.HOME}}", map[string]string{}, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for unknown namespace")
	}
	if !strings.Contains(err.Error(), "unknown namespace") {
		t.Errorf("error = %q, want substring %q", err, "unknown namespace")
	}
}

func TestRenderValue_InvalidExpression(t *testing.T) {
	_, err := renderValue("{{nodot}}", map[string]string{}, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}
	if !strings.Contains(err.Error(), "invalid expression") {
		t.Errorf("error = %q, want substring %q", err, "invalid expression")
	}
}

func TestRenderValue_UnclosedTemplate(t *testing.T) {
	_, err := renderValue("hello {{.input.x world", map[string]string{}, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for unclosed template")
	}
	if !strings.Contains(err.Error(), "unclosed template") {
		t.Errorf("error = %q, want substring %q", err, "unclosed template")
	}
}

func TestRenderValue_DrillFieldNotFound(t *testing.T) {
	store := map[string]interface{}{
		"s1": map[string]interface{}{
			"a": "b",
		},
	}
	_, err := renderValue("{{.steps.s1.missing}}", map[string]string{}, store)
	if err == nil {
		t.Fatal("expected error for missing drill field")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want substring %q", err, "not found")
	}
}

func TestRenderValue_DrillNestedFields(t *testing.T) {
	store := map[string]interface{}{
		"s1": map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": "deep_value",
			},
		},
	}
	val, err := renderValue("{{.steps.s1.level1.level2}}", map[string]string{}, store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "deep_value" {
		t.Errorf("val = %v, want %q", val, "deep_value")
	}
}

func TestRenderValue_DrillIntoNonMap(t *testing.T) {
	store := map[string]interface{}{
		"s1": map[string]interface{}{
			"scalar": "just a string",
		},
	}
	_, err := renderValue("{{.steps.s1.scalar.deeper}}", map[string]string{}, store)
	if err == nil {
		t.Fatal("expected error for drilling into non-map")
	}
}

func TestRenderArgs_MultipleKeys(t *testing.T) {
	raw := map[string]string{
		"to":      "{{.input.email}}",
		"subject": "Report for {{.input.name}}",
		"plain":   "no template here",
	}
	inputs := map[string]string{
		"email": "bob@test.com",
		"name":  "Bob",
	}

	out, err := renderArgs(raw, inputs, map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out["to"] != "bob@test.com" {
		t.Errorf("to = %v", out["to"])
	}
	if out["subject"] != "Report for Bob" {
		t.Errorf("subject = %v", out["subject"])
	}
	if out["plain"] != "no template here" {
		t.Errorf("plain = %v", out["plain"])
	}
}

func TestRenderArgs_ErrorPropagates(t *testing.T) {
	raw := map[string]string{
		"ok":  "literal",
		"bad": "{{.input.missing}}",
	}
	_, err := renderArgs(raw, map[string]string{}, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error to propagate from renderArgs")
	}
}

func TestRenderValue_SpacesInsideBraces(t *testing.T) {
	// Template with extra whitespace inside braces.
	inputs := map[string]string{"x": "val"}
	val, err := renderValue("{{  .input.x  }}", inputs, map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "val" {
		t.Errorf("val = %v, want %q", val, "val")
	}
}

func TestDrillField_JSONRoundtrip(t *testing.T) {
	// drillField should handle struct-like data by JSON roundtrip.
	type nested struct {
		Foo string `json:"foo"`
	}
	data := nested{Foo: "bar"}
	val, err := drillField(data, "foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fmt.Sprintf("%v", val) != "bar" {
		t.Errorf("val = %v, want %q", val, "bar")
	}
}
