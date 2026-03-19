package workflow

import (
	"context"
	"errors"
	"testing"
)

func TestDispatchMCPMode(t *testing.T) {
	actions := []Action{{Name: "test", Description: "test action", Fn: func(ctx context.Context) (interface{}, error) {
		t.Fatal("should not execute in MCP mode")
		return nil, nil
	}}}
	result, err := Dispatch(context.Background(), actions, ExecuteOpts{IsMCP: true, Execute: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Executed {
		t.Fatal("should not be executed")
	}
	if result.Reason != "mcp_read_only" {
		t.Fatalf("wrong reason: %s", result.Reason)
	}
}

func TestDispatchNoExecuteFlag(t *testing.T) {
	actions := []Action{{Name: "test", Description: "test action", Fn: func(ctx context.Context) (interface{}, error) {
		t.Fatal("should not execute without --execute")
		return nil, nil
	}}}
	result, err := Dispatch(context.Background(), actions, ExecuteOpts{Execute: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Executed {
		t.Fatal("should not be executed")
	}
	if result.Reason != "no_execute_flag" {
		t.Fatalf("wrong reason: %s", result.Reason)
	}
}

func TestDispatchNoInputAutoExecute(t *testing.T) {
	called := false
	actions := []Action{{Name: "auto", Description: "auto action", Fn: func(ctx context.Context) (interface{}, error) {
		called = true
		return map[string]string{"status": "done"}, nil
	}}}
	result, err := Dispatch(context.Background(), actions, ExecuteOpts{Execute: true, NoInput: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Executed {
		t.Fatal("should be executed in NoInput mode")
	}
	if !called {
		t.Fatal("action was not called")
	}
	if len(result.Actions) != 1 || !result.Actions[0].Success {
		t.Fatal("action result should be success")
	}
}

func TestDispatchActionPartialFailure(t *testing.T) {
	actions := []Action{
		{Name: "ok", Description: "will succeed", Fn: func(ctx context.Context) (interface{}, error) { return "done", nil }},
		{Name: "fail", Description: "will fail", Fn: func(ctx context.Context) (interface{}, error) { return nil, errors.New("boom") }},
		{Name: "ok2", Description: "will also succeed", Fn: func(ctx context.Context) (interface{}, error) { return "also done", nil }},
	}
	result, err := Dispatch(context.Background(), actions, ExecuteOpts{Execute: true, NoInput: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Executed {
		t.Fatal("should be executed")
	}
	if len(result.Actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(result.Actions))
	}
	if !result.Actions[0].Success {
		t.Error("action[0] should succeed")
	}
	if result.Actions[1].Success {
		t.Error("action[1] should fail")
	}
	if result.Actions[1].Error != "boom" {
		t.Errorf("action[1] error should be 'boom', got %q", result.Actions[1].Error)
	}
	if !result.Actions[2].Success {
		t.Error("action[2] should succeed")
	}
}
