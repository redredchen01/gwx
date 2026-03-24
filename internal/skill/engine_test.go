package skill

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// mockCaller implements ToolCaller for testing.
type mockCaller struct {
	mu      sync.Mutex
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
	m.mu.Lock()
	m.calls = append(m.calls, mockCall{Name: name, Args: args})
	m.mu.Unlock()
	r, ok := m.results[name]
	if !ok {
		return nil, fmt.Errorf("unexpected tool call: %s", name)
	}
	return r.Output, r.Err
}

func (m *mockCaller) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockCaller) getCalls() []mockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]mockCall, len(m.calls))
	copy(cp, m.calls)
	return cp
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
	calls := caller.getCalls()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	if calls[0].Name != "gmail_list" {
		t.Errorf("call name = %q", calls[0].Name)
	}
	if calls[0].Args["query"] != "is:unread" {
		t.Errorf("call arg query = %v", calls[0].Args["query"])
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
	calls := caller.getCalls()
	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(calls))
	}
	body, ok := calls[1].Args["body"].(string)
	if !ok {
		t.Fatalf("body arg = %T, want string", calls[1].Args["body"])
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
	if caller.callCount() != 0 {
		t.Errorf("calls = %d, want 0", caller.callCount())
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
	calls := caller.getCalls()
	if len(calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(calls))
	}
	if calls[0].Args["c"] != "blue" {
		t.Errorf("arg c = %v, want %q", calls[0].Args["c"], "blue")
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
	calls := caller.getCalls()
	if calls[0].Args["c"] != "red" {
		t.Errorf("arg c = %v, want %q (explicit input should win)", calls[0].Args["c"], "red")
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
	if caller.callCount() != 2 {
		t.Errorf("calls = %d, want 2", caller.callCount())
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
	if caller.callCount() != 1 {
		t.Errorf("calls = %d, want 1 (should not call after abort)", caller.callCount())
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
	calls := caller.getCalls()
	if calls[1].Args["ref"] != "yes" {
		t.Errorf("ref arg = %v, want %q", calls[1].Args["ref"], "yes")
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
	if caller.callCount() != 0 {
		t.Errorf("calls = %d, want 0", caller.callCount())
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
	calls := caller.getCalls()
	if calls[0].Args["v"] != "fallback" {
		t.Errorf("arg v = %v, want %q", calls[0].Args["v"], "fallback")
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
	if caller.callCount() != 0 {
		t.Error("no tools should be called")
	}
}

// ── Feature 1: Parallel Steps ──────────────────────────────────────────────

func TestEngine_ParallelSteps(t *testing.T) {
	caller := newMockCaller()
	caller.on("gmail_search", map[string]interface{}{
		"results": []interface{}{"email1", "email2"},
	}, nil)
	caller.on("drive_search", map[string]interface{}{
		"files": []interface{}{"file1"},
	}, nil)
	caller.on("sheets_append", map[string]interface{}{"status": "ok"}, nil)

	s := &Skill{
		Name: "parallel-test",
		Steps: []Step{
			{ID: "emails", Tool: "gmail_search", Args: map[string]string{"q": "invoice"}, OnFail: "abort", Parallel: true},
			{ID: "files", Tool: "drive_search", Args: map[string]string{"q": "invoice"}, OnFail: "abort", Parallel: true},
			{ID: "combine", Tool: "sheets_append", Args: map[string]string{
				"data": "{{.steps.emails}}",
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
	if len(result.Steps) != 3 {
		t.Fatalf("steps = %d, want 3", len(result.Steps))
	}
	// Both parallel steps should have succeeded.
	if !result.Steps[0].Success {
		t.Errorf("step[0] (emails) failed: %s", result.Steps[0].Error)
	}
	if !result.Steps[1].Success {
		t.Errorf("step[1] (files) failed: %s", result.Steps[1].Error)
	}
	// The sequential step should reference parallel output.
	if !result.Steps[2].Success {
		t.Errorf("step[2] (combine) failed: %s", result.Steps[2].Error)
	}
	// All three tools should have been called.
	if caller.callCount() != 3 {
		t.Errorf("calls = %d, want 3", caller.callCount())
	}
}

func TestEngine_ParallelSteps_OneFailsSkip(t *testing.T) {
	caller := newMockCaller()
	caller.on("gmail_search", nil, fmt.Errorf("gmail timeout"))
	caller.on("drive_search", map[string]interface{}{"files": []interface{}{"f1"}}, nil)

	s := &Skill{
		Name: "parallel-skip",
		Steps: []Step{
			{ID: "emails", Tool: "gmail_search", Args: map[string]string{}, OnFail: "skip", Parallel: true},
			{ID: "files", Tool: "drive_search", Args: map[string]string{}, OnFail: "abort", Parallel: true},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should succeed overall: the failed step is on_fail=skip.
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(result.Steps))
	}
}

func TestEngine_ParallelSteps_OneFailsAbort(t *testing.T) {
	caller := newMockCaller()
	caller.on("gmail_search", nil, fmt.Errorf("gmail timeout"))
	caller.on("drive_search", map[string]interface{}{"files": []interface{}{"f1"}}, nil)

	s := &Skill{
		Name: "parallel-abort",
		Steps: []Step{
			{ID: "emails", Tool: "gmail_search", Args: map[string]string{}, OnFail: "abort", Parallel: true},
			{ID: "files", Tool: "drive_search", Args: map[string]string{}, OnFail: "abort", Parallel: true},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fail: one step has on_fail=abort and failed.
	if result.Success {
		t.Error("expected failure")
	}
}

func TestEngine_ParallelStepsMergeToStore(t *testing.T) {
	caller := newMockCaller()
	caller.on("tool_a", map[string]interface{}{"a": "alpha"}, nil)
	caller.on("tool_b", map[string]interface{}{"b": "beta"}, nil)
	caller.on("tool_c", "done", nil)

	s := &Skill{
		Name: "parallel-merge",
		Steps: []Step{
			{ID: "s1", Tool: "tool_a", Args: map[string]string{}, OnFail: "abort", Parallel: true},
			{ID: "s2", Tool: "tool_b", Args: map[string]string{}, OnFail: "abort", Parallel: true},
			{ID: "s3", Tool: "tool_c", Args: map[string]string{
				"ref_a": "{{.steps.s1.a}}",
				"ref_b": "{{.steps.s2.b}}",
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
	// Verify s3 received merged outputs from both parallel steps.
	calls := caller.getCalls()
	var s3Call *mockCall
	for i := range calls {
		if calls[i].Name == "tool_c" {
			s3Call = &calls[i]
			break
		}
	}
	if s3Call == nil {
		t.Fatal("tool_c was not called")
	}
	if s3Call.Args["ref_a"] != "alpha" {
		t.Errorf("ref_a = %v, want %q", s3Call.Args["ref_a"], "alpha")
	}
	if s3Call.Args["ref_b"] != "beta" {
		t.Errorf("ref_b = %v, want %q", s3Call.Args["ref_b"], "beta")
	}
}

// ── Feature 2: Transform Pseudo-Tool ───────────────────────────────────────

func TestEngine_TransformPick(t *testing.T) {
	caller := newMockCaller()
	caller.on("gmail_list", []interface{}{
		map[string]interface{}{"from": "alice", "subject": "hi", "body": "long text", "date": "2026-01-01"},
		map[string]interface{}{"from": "bob", "subject": "re", "body": "reply", "date": "2026-01-02"},
	}, nil)

	s := &Skill{
		Name: "transform-pick",
		Steps: []Step{
			{ID: "fetch", Tool: "gmail_list", Args: map[string]string{}, OnFail: "abort"},
			{ID: "extract", Tool: "transform", Args: map[string]string{
				"input": "{{.steps.fetch}}",
				"pick":  "from,subject",
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
	// Transform should NOT call the tool caller.
	if caller.callCount() != 1 {
		t.Errorf("calls = %d, want 1 (transform should not call MCP)", caller.callCount())
	}
	// Output should be a list with only from and subject.
	items, ok := result.Output.([]interface{})
	if !ok {
		t.Fatalf("output = %T, want []interface{}", result.Output)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	first, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatalf("items[0] = %T, want map", items[0])
	}
	if first["from"] != "alice" {
		t.Errorf("first.from = %v", first["from"])
	}
	if first["subject"] != "hi" {
		t.Errorf("first.subject = %v", first["subject"])
	}
	if _, has := first["body"]; has {
		t.Error("body should have been removed by pick")
	}
	if _, has := first["date"]; has {
		t.Error("date should have been removed by pick")
	}
}

func TestEngine_TransformCount(t *testing.T) {
	caller := newMockCaller()
	caller.on("gmail_list", []interface{}{"a", "b", "c"}, nil)

	s := &Skill{
		Name: "transform-count",
		Steps: []Step{
			{ID: "fetch", Tool: "gmail_list", Args: map[string]string{}, OnFail: "abort"},
			{ID: "cnt", Tool: "transform", Args: map[string]string{
				"input": "{{.steps.fetch}}",
				"count": "true",
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
	count, ok := result.Output.(float64)
	if !ok {
		t.Fatalf("output = %T (%v), want float64", result.Output, result.Output)
	}
	if count != 3 {
		t.Errorf("count = %v, want 3", count)
	}
}

func TestEngine_TransformFlatten(t *testing.T) {
	caller := newMockCaller()

	s := &Skill{
		Name: "transform-flatten",
		Steps: []Step{
			{ID: "flat", Tool: "transform", Args: map[string]string{
				"input":   "{{.steps.data}}",
				"flatten": "true",
			}, OnFail: "abort"},
		},
	}

	// Pre-populate store via a manual hack: use a step that provides nested arrays.
	caller.on("source", []interface{}{
		[]interface{}{"a", "b"},
		[]interface{}{"c"},
	}, nil)
	s.Steps = []Step{
		{ID: "data", Tool: "source", Args: map[string]string{}, OnFail: "abort"},
		{ID: "flat", Tool: "transform", Args: map[string]string{
			"input":   "{{.steps.data}}",
			"flatten": "true",
		}, OnFail: "abort"},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	items, ok := result.Output.([]interface{})
	if !ok {
		t.Fatalf("output = %T, want []interface{}", result.Output)
	}
	if len(items) != 3 {
		t.Errorf("len = %d, want 3 (flattened)", len(items))
	}
}

func TestEngine_TransformSortBy(t *testing.T) {
	caller := newMockCaller()
	caller.on("source", []interface{}{
		map[string]interface{}{"name": "Charlie", "score": float64(3)},
		map[string]interface{}{"name": "Alice", "score": float64(1)},
		map[string]interface{}{"name": "Bob", "score": float64(2)},
	}, nil)

	s := &Skill{
		Name: "transform-sort",
		Steps: []Step{
			{ID: "data", Tool: "source", Args: map[string]string{}, OnFail: "abort"},
			{ID: "sorted", Tool: "transform", Args: map[string]string{
				"input":   "{{.steps.data}}",
				"sort_by": "name",
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
	items, ok := result.Output.([]interface{})
	if !ok {
		t.Fatalf("output = %T, want []interface{}", result.Output)
	}
	if len(items) != 3 {
		t.Fatalf("len = %d, want 3", len(items))
	}
	first := items[0].(map[string]interface{})
	if first["name"] != "Alice" {
		t.Errorf("first name = %v, want Alice", first["name"])
	}
	last := items[2].(map[string]interface{})
	if last["name"] != "Charlie" {
		t.Errorf("last name = %v, want Charlie", last["name"])
	}
}

func TestEngine_TransformLimit(t *testing.T) {
	caller := newMockCaller()
	caller.on("source", []interface{}{"a", "b", "c", "d", "e"}, nil)

	s := &Skill{
		Name: "transform-limit",
		Steps: []Step{
			{ID: "data", Tool: "source", Args: map[string]string{}, OnFail: "abort"},
			{ID: "limited", Tool: "transform", Args: map[string]string{
				"input": "{{.steps.data}}",
				"limit": "2",
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
	items, ok := result.Output.([]interface{})
	if !ok {
		t.Fatalf("output = %T, want []interface{}", result.Output)
	}
	if len(items) != 2 {
		t.Errorf("len = %d, want 2", len(items))
	}
}

func TestEngine_TransformNoInput(t *testing.T) {
	caller := newMockCaller()
	s := &Skill{
		Name: "transform-no-input",
		Steps: []Step{
			{ID: "bad", Tool: "transform", Args: map[string]string{
				"count": "true",
			}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for transform without input")
	}
	if !strings.Contains(result.Error, "input") {
		t.Errorf("error = %q, want mention of input", result.Error)
	}
}

func TestEngine_TransformPickSingleMap(t *testing.T) {
	caller := newMockCaller()
	caller.on("source", map[string]interface{}{
		"name": "Alice", "age": float64(30), "secret": "xxx",
	}, nil)

	s := &Skill{
		Name: "transform-pick-map",
		Steps: []Step{
			{ID: "data", Tool: "source", Args: map[string]string{}, OnFail: "abort"},
			{ID: "picked", Tool: "transform", Args: map[string]string{
				"input": "{{.steps.data}}",
				"pick":  "name,age",
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
	m, ok := result.Output.(map[string]interface{})
	if !ok {
		t.Fatalf("output = %T, want map", result.Output)
	}
	if m["name"] != "Alice" {
		t.Errorf("name = %v", m["name"])
	}
	if _, has := m["secret"]; has {
		t.Error("secret should have been removed by pick")
	}
}

// ── Feature 3: Each Loop ──────────────────────────────────────────────────

func TestEngine_EachLoop(t *testing.T) {
	caller := newMockCaller()
	caller.on("contacts_list", []interface{}{
		map[string]interface{}{"email": "alice@test.com", "name": "Alice"},
		map[string]interface{}{"email": "bob@test.com", "name": "Bob"},
	}, nil)
	caller.on("gmail_send", map[string]interface{}{"status": "sent"}, nil)

	s := &Skill{
		Name: "each-test",
		Steps: []Step{
			{ID: "contacts", Tool: "contacts_list", Args: map[string]string{}, OnFail: "abort"},
			{ID: "notify", Tool: "gmail_send", Each: "{{.steps.contacts}}", Args: map[string]string{
				"to":      "{{.item.email}}",
				"subject": "Hello {{.item.name}}",
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
	// contacts_list called once, gmail_send called twice (once per contact).
	if caller.callCount() != 3 {
		t.Errorf("calls = %d, want 3 (1 list + 2 sends)", caller.callCount())
	}
	// Verify the gmail_send calls had the right arguments.
	calls := caller.getCalls()
	sendCalls := []mockCall{}
	for _, c := range calls {
		if c.Name == "gmail_send" {
			sendCalls = append(sendCalls, c)
		}
	}
	if len(sendCalls) != 2 {
		t.Fatalf("gmail_send calls = %d, want 2", len(sendCalls))
	}
	// Check that .item was resolved.
	foundAlice := false
	foundBob := false
	for _, c := range sendCalls {
		to, _ := c.Args["to"].(string)
		subj, _ := c.Args["subject"].(string)
		if to == "alice@test.com" && subj == "Hello Alice" {
			foundAlice = true
		}
		if to == "bob@test.com" && subj == "Hello Bob" {
			foundBob = true
		}
	}
	if !foundAlice {
		t.Error("missing gmail_send call for Alice")
	}
	if !foundBob {
		t.Error("missing gmail_send call for Bob")
	}
}

func TestEngine_EachLoopOnFailSkip(t *testing.T) {
	caller := newMockCaller()
	caller.on("contacts_list", []interface{}{
		map[string]interface{}{"email": "a@test.com", "name": "A"},
		map[string]interface{}{"email": "b@test.com", "name": "B"},
	}, nil)
	// gmail_send always fails.
	caller.on("gmail_send", nil, fmt.Errorf("send failed"))

	s := &Skill{
		Name: "each-skip",
		Steps: []Step{
			{ID: "contacts", Tool: "contacts_list", Args: map[string]string{}, OnFail: "abort"},
			{ID: "notify", Tool: "gmail_send", Each: "{{.steps.contacts}}", Args: map[string]string{
				"to": "{{.item.email}}",
			}, OnFail: "skip"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), s, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// on_fail=skip means the each loop should succeed overall.
	if !result.Success {
		t.Errorf("expected success (on_fail=skip), got error: %s", result.Error)
	}
}

func TestEngine_EachLoopOnFailAbort(t *testing.T) {
	caller := newMockCaller()
	caller.on("contacts_list", []interface{}{
		map[string]interface{}{"email": "a@test.com"},
	}, nil)
	caller.on("gmail_send", nil, fmt.Errorf("send failed"))

	s := &Skill{
		Name: "each-abort",
		Steps: []Step{
			{ID: "contacts", Tool: "contacts_list", Args: map[string]string{}, OnFail: "abort"},
			{ID: "notify", Tool: "gmail_send", Each: "{{.steps.contacts}}", Args: map[string]string{
				"to": "{{.item.email}}",
			}, OnFail: "abort"},
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
	if !strings.Contains(result.Error, "send failed") {
		t.Errorf("error = %q, want mention of send failure", result.Error)
	}
}

func TestEngine_EachLoopCollectsResults(t *testing.T) {
	caller := newMockCaller()
	caller.on("source", []interface{}{
		map[string]interface{}{"id": float64(1)},
		map[string]interface{}{"id": float64(2)},
	}, nil)
	caller.on("process", map[string]interface{}{"done": true}, nil)

	s := &Skill{
		Name:   "each-collect",
		Output: "{{.steps.processed}}",
		Steps: []Step{
			{ID: "items", Tool: "source", Args: map[string]string{}, OnFail: "abort"},
			{ID: "processed", Tool: "process", Each: "{{.steps.items}}", Args: map[string]string{
				"item_id": "{{.item.id}}",
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
	// Output should be the collected array from the each loop.
	items, ok := result.Output.([]interface{})
	if !ok {
		t.Fatalf("output = %T, want []interface{}", result.Output)
	}
	if len(items) != 2 {
		t.Errorf("collected items = %d, want 2", len(items))
	}
}

func TestEngine_EachWithTransform(t *testing.T) {
	// Each loop feeding into transform.
	caller := newMockCaller()
	caller.on("source", []interface{}{
		map[string]interface{}{"name": "Alice", "score": float64(90)},
		map[string]interface{}{"name": "Bob", "score": float64(80)},
	}, nil)
	caller.on("process", map[string]interface{}{"result": "ok"}, nil)

	s := &Skill{
		Name: "each-then-transform",
		Steps: []Step{
			{ID: "data", Tool: "source", Args: map[string]string{}, OnFail: "abort"},
			{ID: "work", Tool: "process", Each: "{{.steps.data}}", Args: map[string]string{
				"n": "{{.item.name}}",
			}, OnFail: "abort"},
			{ID: "cnt", Tool: "transform", Args: map[string]string{
				"input": "{{.steps.work}}",
				"count": "true",
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
	count, ok := result.Output.(float64)
	if !ok {
		t.Fatalf("output = %T, want float64", result.Output)
	}
	if count != 2 {
		t.Errorf("count = %v, want 2", count)
	}
}

// ── Feature: Skill Composition ─────────────────────────────────────────────

func TestEngine_SkillComposition_Basic(t *testing.T) {
	caller := newMockCaller()
	caller.on("gmail_list", map[string]interface{}{
		"messages": []interface{}{"msg1", "msg2"},
		"count":    float64(2),
	}, nil)

	// The sub-skill that will be called via skill:inner.
	innerSkill := &Skill{
		Name: "inner",
		Inputs: []Input{
			{Name: "limit", Default: "5"},
		},
		Steps: []Step{
			{ID: "fetch", Tool: "gmail_list", Args: map[string]string{
				"limit": "{{.input.limit}}",
			}, OnFail: "abort"},
		},
	}

	// The outer skill that references skill:inner.
	outerSkill := &Skill{
		Name: "outer",
		Steps: []Step{
			{ID: "brief", Tool: "skill:inner", Args: map[string]string{
				"limit": "10",
			}, OnFail: "abort"},
		},
	}

	// Use a custom loader that returns our test skills.
	loader := func() ([]*Skill, error) {
		return []*Skill{innerSkill, outerSkill}, nil
	}

	engine := NewEngine(caller).WithSkillLoader(loader)
	result, err := engine.Run(context.Background(), outerSkill, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	// The outer skill should have one step report.
	if len(result.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(result.Steps))
	}
	if !result.Steps[0].Success {
		t.Errorf("step[0] failed: %s", result.Steps[0].Error)
	}
	// The MCP caller should have been called once (by the inner skill's step).
	if caller.callCount() != 1 {
		t.Errorf("calls = %d, want 1", caller.callCount())
	}
	calls := caller.getCalls()
	if calls[0].Args["limit"] != "10" {
		t.Errorf("inner skill got limit = %v, want %q", calls[0].Args["limit"], "10")
	}
}

func TestEngine_SkillComposition_NotFound(t *testing.T) {
	caller := newMockCaller()

	outerSkill := &Skill{
		Name: "outer",
		Steps: []Step{
			{ID: "call", Tool: "skill:nonexistent", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	loader := func() ([]*Skill, error) {
		return []*Skill{outerSkill}, nil
	}

	engine := NewEngine(caller).WithSkillLoader(loader)
	result, err := engine.Run(context.Background(), outerSkill, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure when sub-skill is not found")
	}
	if !strings.Contains(result.Error, "not found") {
		t.Errorf("error = %q, want substring %q", result.Error, "not found")
	}
}

func TestEngine_SkillComposition_DepthLimit(t *testing.T) {
	caller := newMockCaller()

	// A skill that calls itself — infinite recursion.
	recursive := &Skill{
		Name: "recursive",
		Steps: []Step{
			{ID: "loop", Tool: "skill:recursive", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	loader := func() ([]*Skill, error) {
		return []*Skill{recursive}, nil
	}

	engine := NewEngine(caller).WithSkillLoader(loader)
	result, err := engine.Run(context.Background(), recursive, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure due to depth limit")
	}
	if !strings.Contains(result.Error, "depth limit") {
		t.Errorf("error = %q, want substring %q", result.Error, "depth limit")
	}
	// Should NOT have called MCP at all.
	if caller.callCount() != 0 {
		t.Errorf("calls = %d, want 0 (depth limit should prevent execution)", caller.callCount())
	}
}

func TestEngine_SkillComposition_OnFailSkip(t *testing.T) {
	caller := newMockCaller()
	caller.on("good_tool", map[string]interface{}{"ok": true}, nil)

	// Inner skill that will fail.
	failSkill := &Skill{
		Name: "fail-inner",
		Steps: []Step{
			{ID: "boom", Tool: "explode", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	outerSkill := &Skill{
		Name: "outer",
		Steps: []Step{
			{ID: "try", Tool: "skill:fail-inner", Args: map[string]string{}, OnFail: "skip"},
			{ID: "fallback", Tool: "good_tool", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	loader := func() ([]*Skill, error) {
		return []*Skill{failSkill, outerSkill}, nil
	}

	engine := NewEngine(caller).WithSkillLoader(loader)
	result, err := engine.Run(context.Background(), outerSkill, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should succeed overall because the failed sub-skill step is on_fail=skip.
	if !result.Success {
		t.Errorf("expected success (on_fail=skip), got error: %s", result.Error)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(result.Steps))
	}
	if result.Steps[0].Success {
		t.Error("step[0] should have failed (sub-skill failed)")
	}
	if !result.Steps[1].Success {
		t.Error("step[1] should have succeeded")
	}
}

func TestEngine_SkillComposition_ChainedDepth(t *testing.T) {
	caller := newMockCaller()
	caller.on("leaf_tool", map[string]interface{}{"value": "leaf"}, nil)

	// depth 0: A calls skill:B
	// depth 1: B calls skill:C
	// depth 2: C calls leaf_tool
	// Total depth = 2, within limit of 5.
	skillC := &Skill{
		Name: "C",
		Steps: []Step{
			{ID: "leaf", Tool: "leaf_tool", Args: map[string]string{}, OnFail: "abort"},
		},
	}
	skillB := &Skill{
		Name: "B",
		Steps: []Step{
			{ID: "call_c", Tool: "skill:C", Args: map[string]string{}, OnFail: "abort"},
		},
	}
	skillA := &Skill{
		Name: "A",
		Steps: []Step{
			{ID: "call_b", Tool: "skill:B", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	loader := func() ([]*Skill, error) {
		return []*Skill{skillA, skillB, skillC}, nil
	}

	engine := NewEngine(caller).WithSkillLoader(loader)
	result, err := engine.Run(context.Background(), skillA, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success for chained composition, got error: %s", result.Error)
	}
	if caller.callCount() != 1 {
		t.Errorf("calls = %d, want 1 (only leaf_tool)", caller.callCount())
	}
}

func TestEngine_SkillComposition_EmptyName(t *testing.T) {
	caller := newMockCaller()

	outerSkill := &Skill{
		Name: "outer",
		Steps: []Step{
			{ID: "bad", Tool: "skill:", Args: map[string]string{}, OnFail: "abort"},
		},
	}

	engine := NewEngine(caller)
	result, err := engine.Run(context.Background(), outerSkill, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected failure for empty skill name")
	}
	if !strings.Contains(result.Error, "empty skill name") {
		t.Errorf("error = %q, want substring %q", result.Error, "empty skill name")
	}
}

func TestArgsToInputs(t *testing.T) {
	args := map[string]interface{}{
		"str":  "hello",
		"num":  float64(42),
		"bool": true,
	}
	inputs := argsToInputs(args)

	if inputs["str"] != "hello" {
		t.Errorf("str = %q, want %q", inputs["str"], "hello")
	}
	if inputs["num"] != "42" {
		t.Errorf("num = %q, want %q", inputs["num"], "42")
	}
	if inputs["bool"] != "true" {
		t.Errorf("bool = %q, want %q", inputs["bool"], "true")
	}
}
