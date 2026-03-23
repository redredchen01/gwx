package skill

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockCaller implements ToolCaller for testing.
type mockCaller struct {
	calls   []mockCall
	results map[string]mockResult
}

type mockCall struct {
	Name string
	Args map[string]interface{}
}

type mockResult struct {
	Output interface{}
	Err    error
}

func newMockCaller() *mockCaller {
	return &mockCaller{
		results: make(map[string]mockResult),
	}
}

func (m *mockCaller) on(tool string, output interface{}, err error) {
	m.results[tool] = mockResult{Output: output, Err: err}
}

func (m *mockCaller) CallTool(_ context.Context, name string, args map[string]interface{}) (interface{}, error) {
	m.calls = append(m.calls, mockCall{Name: name, Args: args})
	r, ok := m.results[name]
	if !ok {
		return nil, fmt.Errorf("unexpected tool call: %s", name)
	}
	return r.Output, r.Err
}

func TestEngine_SingleStep(t *testing.T) {
	caller := newMockCaller()
	caller.on("gmail_list", map[string]interface{}{
		"messages": []interface{}{"msg1", "msg2"},
	}, nil)

	s := &Skill{
		Name: "one-step",
		Steps: []Step{
			{ID: "fetch", Tool: "gmail_list", Args: map[string]string{"query": "is:unread"}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Skill != "one-step" {
		t.Errorf("skill = %q", result.Skill)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(result.Steps))
	}
	if !result.Steps[0].Success {
		t.Errorf("step[0].success = false")
	}
	if result.Steps[0].ID != "fetch" {
		t.Errorf("step[0].id = %q", result.Steps[0].ID)
	}
	if result.Steps[0].Tool != "gmail_list" {
		t.Errorf("step[0].tool = %q", result.Steps[0].Tool)
	}
	// Default output = last step's output
	if result.Output == nil {
		t.Error("output is nil, want last step output")
	}

	// Verify the tool was called with correct args.
	if len(caller.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(caller.calls))
	}
	if caller.calls[0].Name != "gmail_list" {
		t.Errorf("call name = %q", caller.calls[0].Name)
	}
	if caller.calls[0].Args["query"] != "is:unread" {
		t.Errorf("call arg query = %v", caller.calls[0].Args["query"])
	}
}

func TestEngine_MultiStepDataFlow(t *testing.T) {
	caller := newMockCaller()
	caller.on("gmail_list", map[string]interface{}{
		"count": float64(3),
		"ids":   []interface{}{"a", "b", "c"},
	}, nil)
	caller.on("gmail_send", map[string]interface{}{
		"status": "sent",
	}, nil)

	s := &Skill{
		Name: "multi",
		Steps: []Step{
			{ID: "fetch", Tool: "gmail_list", Args: map[string]string{}, OnFail: "abort"},
			{ID: "send", Tool: "gmail_send", Args: map[string]string{
				"body": "Found {{.steps.fetch.count}} emails",
			}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(result.Steps))
	}

	// Verify second call got interpolated arg.
	if len(caller.calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(caller.calls))
	}
	body, ok := caller.calls[1].Args["body"].(string)
	if !ok {
		t.Fatalf("body arg = %T, want string", caller.calls[1].Args["body"])
	}
	if body != "Found 3 emails" {
		t.Errorf("body = %q, want %q", body, "Found 3 emails")
	}
}

func TestEngine_RequiredInputMissing(t *testing.T) {
	caller := newMockCaller()
	s := &Skill{
		Name: "needs-input",
		Inputs: []Input{
			{Name: "email", Required: true},
		},
		Steps: []Step{
			{ID: "s1", Tool: "test", OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	_, err := engine.Run(context.Background(), s, map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing required input")
	}
	if !strings.Contains(err.Error(), "missing required input") {
		t.Errorf("error = %q, want substring %q", err, "missing required input")
	}
	// No tool calls should have been made.
	if len(caller.calls) != 0 {
		t.Errorf("calls = %d, want 0", len(caller.calls))
	}
}

func TestEngine_DefaultInputApplied(t *testing.T) {
	caller := newMockCaller()
	caller.on("test_tool", "ok", nil)

	s := &Skill{
		Name: "defaults",
		Inputs: []Input{
			{Name: "color", Default: "blue"},
		},
		Steps: []Step{
			{ID: "s1", Tool: "test_tool", Args: map[string]string{
				"c": "{{.input.color}}",
			}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	// Verify default was applied.
	if len(caller.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(caller.calls))
	}
	if caller.calls[0].Args["c"] != "blue" {
		t.Errorf("arg c = %v, want %q", caller.calls[0].Args["c"], "blue")
	}
}

func TestEngine_DefaultInputNotOverridden(t *testing.T) {
	caller := newMockCaller()
	caller.on("test_tool", "ok", nil)

	s := &Skill{
		Name: "no-override",
		Inputs: []Input{
			{Name: "color", Default: "blue"},
		},
		Steps: []Step{
			{ID: "s1", Tool: "test_tool", Args: map[string]string{
				"c": "{{.input.color}}",
			}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	// Supply an explicit value — should NOT be overridden by default.
	result, err := engine.Run(context.Background(), s, map[string]string{"color": "red"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if caller.calls[0].Args["c"] != "red" {
		t.Errorf("arg c = %v, want %q (explicit input should win)", caller.calls[0].Args["c"], "red")
	}
}

func TestEngine_OnFailSkip(t *testing.T) {
	caller := newMockCaller()
	caller.on("bad_tool", nil, fmt.Errorf("connection refused"))
	caller.on("good_tool", map[string]interface{}{"ok": true}, nil)

	s := &Skill{
		Name: "skip-on-fail",
		Steps: []Step{
			{ID: "risky", Tool: "bad_tool", Args: map[string]string{}, OnFail: "skip"},
			{ID: "safe", Tool: "good_tool", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Overall should still be success because the failed step was skippable.
	if !result.Success {
		t.Errorf("expected success (on_fail=skip), got error: %s", result.Error)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(result.Steps))
	}
	if result.Steps[0].Success {
		t.Error("step[0] should have failed")
	}
	if result.Steps[0].Error == "" {
		t.Error("step[0] should have error message")
	}
	if !result.Steps[1].Success {
		t.Error("step[1] should have succeeded")
	}
	// Both tools should have been called.
	if len(caller.calls) != 2 {
		t.Errorf("calls = %d, want 2", len(caller.calls))
	}
}

func TestEngine_OnFailAbort(t *testing.T) {
	caller := newMockCaller()
	caller.on("bad_tool", nil, fmt.Errorf("timeout"))
	caller.on("never_called", "nope", nil)

	s := &Skill{
		Name: "abort-on-fail",
		Steps: []Step{
			{ID: "fail", Tool: "bad_tool", Args: map[string]string{}, OnFail: "abort"},
			{ID: "skip", Tool: "never_called", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure")
	}
	if result.Error == "" {
		t.Error("expected error message")
	}
	// Only one step should be in the report.
	if len(result.Steps) != 1 {
		t.Fatalf("steps = %d, want 1 (aborted after first)", len(result.Steps))
	}
	// Only the first tool should have been called.
	if len(caller.calls) != 1 {
		t.Errorf("calls = %d, want 1 (should not call after abort)", len(caller.calls))
	}
}

func TestEngine_OutputTemplateRendered(t *testing.T) {
	caller := newMockCaller()
	caller.on("gmail_list", map[string]interface{}{
		"total": float64(5),
	}, nil)

	s := &Skill{
		Name:   "with-output",
		Output: "Total emails: {{.steps.fetch.total}}",
		Steps: []Step{
			{ID: "fetch", Tool: "gmail_list", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	outputStr, ok := result.Output.(string)
	if !ok {
		t.Fatalf("output = %T, want string", result.Output)
	}
	if outputStr != "Total emails: 5" {
		t.Errorf("output = %q, want %q", outputStr, "Total emails: 5")
	}
}

func TestEngine_OutputTemplateNative(t *testing.T) {
	caller := newMockCaller()
	caller.on("gmail_list", map[string]interface{}{
		"messages": []interface{}{"a", "b"},
	}, nil)

	s := &Skill{
		Name:   "native-output",
		Output: "{{.steps.fetch}}",
		Steps: []Step{
			{ID: "fetch", Tool: "gmail_list", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("output = %T, want map", result.Output)
	}
	if m["messages"] == nil {
		t.Error("output missing 'messages' key")
	}
}

func TestEngine_NoOutputUsesLastStep(t *testing.T) {
	caller := newMockCaller()
	caller.on("step_a", map[string]interface{}{"a": true}, nil)
	caller.on("step_b", map[string]interface{}{"b": true}, nil)

	s := &Skill{
		Name: "no-output-template",
		Steps: []Step{
			{ID: "first", Tool: "step_a", Args: map[string]string{}, OnFail: "abort"},
			{ID: "second", Tool: "step_b", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("output = %T, want map", result.Output)
	}
	// Should be step_b's output, not step_a's.
	if m["b"] != true {
		t.Errorf("output = %v, want step_b output", result.Output)
	}
}

func TestEngine_StoreKeyOverride(t *testing.T) {
	caller := newMockCaller()
	caller.on("gmail_list", map[string]interface{}{"data": "yes"}, nil)
	caller.on("gmail_send", "sent", nil)

	s := &Skill{
		Name: "store-key",
		Steps: []Step{
			{ID: "s1", Tool: "gmail_list", Args: map[string]string{}, Store: "emails", OnFail: "abort"},
			{ID: "s2", Tool: "gmail_send", Args: map[string]string{
				"ref": "{{.steps.emails.data}}",
			}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	// The second step should have resolved .steps.emails.data via the store key.
	if caller.calls[1].Args["ref"] != "yes" {
		t.Errorf("ref arg = %v, want %q", caller.calls[1].Args["ref"], "yes")
	}
}

func TestEngine_ToolCallerErrorPropagation(t *testing.T) {
	caller := newMockCaller()
	caller.on("broken", nil, fmt.Errorf("API rate limit exceeded"))

	s := &Skill{
		Name: "error-prop",
		Steps: []Step{
			{ID: "s1", Tool: "broken", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("Run itself should not error, result should carry it: %v", err)
	}
	if result.Success {
		t.Error("expected failure")
	}
	if !strings.Contains(result.Error, "API rate limit exceeded") {
		t.Errorf("error = %q, want to contain tool error message", result.Error)
	}
}

func TestEngine_RenderArgsErrorAborts(t *testing.T) {
	caller := newMockCaller()
	// No tool result needed — should fail at render stage.

	s := &Skill{
		Name: "render-fail",
		Steps: []Step{
			{ID: "s1", Tool: "test", Args: map[string]string{
				"x": "{{.input.nonexistent}}",
			}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("Run itself should not error: %v", err)
	}
	if result.Success {
		t.Error("expected failure on render error")
	}
	if !strings.Contains(result.Error, "render args") {
		t.Errorf("error = %q, want substring %q", result.Error, "render args")
	}
	// Tool should NOT have been called.
	if len(caller.calls) != 0 {
		t.Errorf("calls = %d, want 0", len(caller.calls))
	}
}

func TestEngine_RenderArgsErrorSkip(t *testing.T) {
	caller := newMockCaller()
	caller.on("good_tool", "ok", nil)

	s := &Skill{
		Name: "render-fail-skip",
		Steps: []Step{
			{ID: "s1", Tool: "test", Args: map[string]string{
				"x": "{{.input.nonexistent}}",
			}, OnFail: "skip"},
			{ID: "s2", Tool: "good_tool", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success (render error was skippable), got error: %s", result.Error)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(result.Steps))
	}
	if result.Steps[0].Success {
		t.Error("step[0] should have failed (render error)")
	}
	if !result.Steps[1].Success {
		t.Error("step[1] should have succeeded")
	}
}

func TestEngine_RequiredInputWithDefault(t *testing.T) {
	caller := newMockCaller()
	caller.on("test_tool", "ok", nil)

	s := &Skill{
		Name: "req-with-default",
		Inputs: []Input{
			{Name: "x", Required: true, Default: "fallback"},
		},
		Steps: []Step{
			{ID: "s1", Tool: "test_tool", Args: map[string]string{
				"v": "{{.input.x}}",
			}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success (required input has default), got error: %s", result.Error)
	}
	if caller.calls[0].Args["v"] != "fallback" {
		t.Errorf("arg v = %v, want %q", caller.calls[0].Args["v"], "fallback")
	}
}

func TestEngine_NormaliseOutputJSONString(t *testing.T) {
	caller := newMockCaller()
	// Return a JSON string — should be normalised to map.
	caller.on("json_tool", `{"key":"value"}`, nil)

	s := &Skill{
		Name:   "json-normalise",
		Output: "{{.steps.s1.key}}",
		Steps: []Step{
			{ID: "s1", Tool: "json_tool", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Output != "value" {
		t.Errorf("output = %v, want %q", result.Output, "value")
	}
}

func TestEngine_OutputTemplateFallsBackToStore(t *testing.T) {
	caller := newMockCaller()
	caller.on("test_tool", map[string]interface{}{"x": 1}, nil)

	s := &Skill{
		Name:   "bad-output-tmpl",
		Output: "{{.steps.nonexistent.field}}",
		Steps: []Step{
			{ID: "s1", Tool: "test_tool", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success despite bad output template")
	}
	// When output template fails, engine falls back to store.
	if result.Output == nil {
		t.Error("output should fall back to store, not nil")
	}
}

func TestEngine_EmptySkillSteps(t *testing.T) {
	// Edge case: a skill that somehow has zero steps shouldn't panic.
	// (Normally caught by validate, but engine should be safe regardless.)
	caller := newMockCaller()
	s := &Skill{
		Name:  "empty",
		Steps: []Step{},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("empty skill should succeed vacuously")
	}
	if len(caller.calls) != 0 {
		t.Error("no tools should be called")
	}
}
