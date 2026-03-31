package testutil

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// ---- MockCaller Tests ----

func TestMockCaller_OnCall(t *testing.T) {
	mc := NewMockCaller()
	mc.On("echo", "hello", nil)

	ctx := context.Background()
	result, err := mc.CallTool(ctx, "echo", map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "hello" {
		t.Fatalf("expected 'hello', got %v", result)
	}
}

func TestMockCaller_UnknownTool(t *testing.T) {
	mc := NewMockCaller()
	mc.On("echo", "response", nil)

	ctx := context.Background()
	_, err := mc.CallTool(ctx, "unknown", map[string]interface{}{})

	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if err.Error() != "unknown tool: unknown" {
		t.Fatalf("expected 'unknown tool: unknown', got %v", err)
	}
}

func TestMockCaller_MultipleTools(t *testing.T) {
	mc := NewMockCaller()
	mc.On("gmail.list", []string{"email1", "email2"}, nil)
	mc.On("sheets.append", "row_id_123", nil)

	ctx := context.Background()

	result1, err1 := mc.CallTool(ctx, "gmail.list", map[string]interface{}{})
	if err1 != nil {
		t.Fatalf("gmail.list failed: %v", err1)
	}
	emails, ok := result1.([]string)
	if !ok || len(emails) != 2 || emails[0] != "email1" || emails[1] != "email2" {
		t.Fatalf("gmail.list returned unexpected result: %v", result1)
	}

	result2, err2 := mc.CallTool(ctx, "sheets.append", map[string]interface{}{})
	if err2 != nil {
		t.Fatalf("sheets.append failed: %v", err2)
	}
	if result2 != "row_id_123" {
		t.Fatalf("sheets.append returned unexpected result: %v", result2)
	}
}

// ---- SkillBuilder Tests ----

func TestSkillBuilder_SimpleBuild(t *testing.T) {
	sb := NewSkillBuilder("test-skill")
	s := sb.Build()

	if s.Name != "test-skill" {
		t.Fatalf("expected name 'test-skill', got %s", s.Name)
	}
	if len(s.Inputs) == 0 {
		t.Fatal("expected inputs to be set")
	}
	if len(s.Steps) == 0 {
		t.Fatal("expected steps to be set")
	}
}

func TestSkillBuilder_WithDAGStep(t *testing.T) {
	sb := NewSkillBuilder("dag-test")
	sb.WithDAGStep("step1", "gmail.list", map[string]string{"query": "label:inbox"}, []string{})
	sb.WithDAGStep("step2", "sheets.append", map[string]string{"table": "Results"}, []string{"step1"})

	s := sb.Build()

	if len(s.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(s.Steps))
	}

	step2 := s.Steps[1]
	if step2.ID != "step2" {
		t.Fatalf("expected step ID 'step2', got %s", step2.ID)
	}
	if len(step2.DependsOn) != 1 || step2.DependsOn[0] != "step1" {
		t.Fatalf("expected DependsOn=['step1'], got %v", step2.DependsOn)
	}
}

func TestSkillBuilder_BuildValid(t *testing.T) {
	sb := NewSkillBuilder("validate-test")
	sb.WithInput("email", "string", true)
	sb.WithDescription("Test skill for validation")
	sb.WithStep("send_email", "gmail.send", map[string]string{"to": "user@example.com"})

	s := sb.Build()

	// Verify it matches the skill.Skill structure
	if s.Name == "" {
		t.Fatal("expected name to be set")
	}
	if s.Version == "" {
		t.Fatal("expected version to be set")
	}
	if len(s.Inputs) == 0 {
		t.Fatal("expected inputs")
	}
	if len(s.Steps) == 0 {
		t.Fatal("expected steps")
	}
	if s.Output == "" {
		t.Fatal("expected output template")
	}
}

// ---- Filesystem Tests ----

func TestSetSkillConfigHome_Isolation(t *testing.T) {
	skillDir := SetSkillConfigHome(t)

	// Verify the directory was created
	if stat, err := os.Stat(skillDir); err != nil || !stat.IsDir() {
		t.Fatalf("skill config dir not created: %s", skillDir)
	}

	// Verify GWX_SKILL_HOME is set
	home := os.Getenv("GWX_SKILL_HOME")
	if home != skillDir {
		t.Fatalf("expected GWX_SKILL_HOME=%s, got %s", skillDir, home)
	}

	// Create a test file in the skill dir
	testFile := filepath.Join(skillDir, "test.yaml")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if _, err := os.Stat(testFile); err != nil {
		t.Fatalf("test file not found: %v", err)
	}

	// After cleanup, verify env is restored (cleanup runs at end of test)
}

func TestTempSkillFile_Cleanup(t *testing.T) {
	filePath := TempSkillFile(t, "name: test\nversion: 1.0\n")

	// Verify file exists during test
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("temp skill file not created: %v", err)
	}

	// Read and verify content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}
	if string(content) != "name: test\nversion: 1.0\n" {
		t.Fatalf("unexpected file content: %s", content)
	}

	// t.Cleanup will delete the temp directory after this test
}

// ---- Integration Tests ----

func TestSkillBuilder_WithParallelStep(t *testing.T) {
	sb := NewSkillBuilder("parallel-test")
	sb.WithParallelStep("fetch_gmail", "gmail.list", map[string]string{})
	sb.WithParallelStep("fetch_sheets", "sheets.list", map[string]string{})
	sb.WithStep("combine", "merge", map[string]string{})

	s := sb.Build()

	if s.Steps[0].Parallel != true || s.Steps[1].Parallel != true {
		t.Fatal("expected parallel steps to have Parallel=true")
	}
	if s.Steps[2].Parallel != false {
		t.Fatal("expected non-parallel step to have Parallel=false")
	}
}

func TestMockCaller_WithError(t *testing.T) {
	mc := NewMockCaller()
	expectedErr := errors.New("tool failed")
	mc.On("failing_tool", nil, expectedErr)

	ctx := context.Background()
	_, err := mc.CallTool(ctx, "failing_tool", map[string]interface{}{})

	if err == nil {
		t.Fatal("expected error from mock tool")
	}
	if err.Error() != "tool failed" {
		t.Fatalf("expected 'tool failed', got %s", err.Error())
	}
}

// TestSkillBuilder_Output validates custom output template.
func TestSkillBuilder_CustomOutput(t *testing.T) {
	sb := NewSkillBuilder("custom-output-test")
	sb.WithStep("compute", "echo", map[string]string{"msg": "hello"})
	sb.WithOutput("Result: {{.steps.compute}}")

	s := sb.Build()

	if s.Output != "Result: {{.steps.compute}}" {
		t.Fatalf("expected custom output, got %s", s.Output)
	}
}

// ---- Builder Fluent Interface ----

func TestSkillBuilder_FluentChain(t *testing.T) {
	s := NewSkillBuilder("fluent-test").
		WithDescription("A test skill").
		WithInput("query", "string", true).
		WithStep("fetch", "api.get", map[string]string{"url": "https://example.com"}).
		Build()

	if s.Name != "fluent-test" {
		t.Fatalf("expected fluent chain to preserve name")
	}
	if s.Description != "A test skill" {
		t.Fatalf("expected description to be set")
	}
	if len(s.Inputs) < 1 {
		t.Fatal("expected input to be added")
	}
}
