package skill

import (
	"context"
	"encoding/json"
	"fmt"
)

// ToolCaller abstracts MCP tool invocation so the engine does not depend on the
// mcp package directly.
type ToolCaller interface {
	CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error)
}

// Engine executes parsed skills against a ToolCaller.
type Engine struct {
	caller ToolCaller
}

// NewEngine creates an engine that delegates tool calls to caller.
func NewEngine(caller ToolCaller) *Engine {
	return &Engine{caller: caller}
}

// Run executes a skill with the given input values and returns a structured result.
func (e *Engine) Run(ctx context.Context, s *Skill, inputs map[string]string) (*RunResult, error) {
	// Validate required inputs.
	for _, inp := range s.Inputs {
		if _, supplied := inputs[inp.Name]; !supplied {
			if inp.Required && inp.Default == "" {
				return nil, fmt.Errorf("missing required input %q", inp.Name)
			}
			if inp.Default != "" {
				inputs[inp.Name] = inp.Default
			}
		}
	}

	store := make(map[string]interface{}, len(s.Steps))
	result := &RunResult{
		Skill:   s.Name,
		Success: true,
		Steps:   make([]StepReport, 0, len(s.Steps)),
	}

	for _, step := range s.Steps {
		report := StepReport{ID: step.ID, Tool: step.Tool}

		args, err := renderArgs(step.Args, inputs, store)
		if err != nil {
			report.Success = false
			report.Error = fmt.Sprintf("render args: %s", err)
			result.Steps = append(result.Steps, report)
			if step.OnFail == "skip" {
				continue
			}
			result.Success = false
			result.Error = report.Error
			return result, nil
		}

		out, err := e.caller.CallTool(ctx, step.Tool, args)
		if err != nil {
			report.Success = false
			report.Error = err.Error()
			result.Steps = append(result.Steps, report)
			if step.OnFail == "skip" {
				continue
			}
			result.Success = false
			result.Error = report.Error
			return result, nil
		}

		// Normalise output to map[string]interface{} for template drilling.
		normalised := normaliseOutput(out)

		report.Success = true
		report.Output = normalised

		storeKey := step.Store
		if storeKey == "" {
			storeKey = step.ID
		}
		store[storeKey] = normalised

		result.Steps = append(result.Steps, report)
	}

	// Build final output.
	if s.Output != "" {
		rendered, err := renderValue(s.Output, inputs, store)
		if err != nil {
			result.Output = store
		} else {
			result.Output = rendered
		}
	} else {
		// Default: return last step's output.
		if len(result.Steps) > 0 {
			result.Output = result.Steps[len(result.Steps)-1].Output
		}
	}

	return result, nil
}

// normaliseOutput tries to unwrap the output into a map for template drilling.
// If the output is a JSON string, it is parsed. Otherwise it's left as-is.
func normaliseOutput(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return val
	case string:
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(val), &m); err == nil {
			return m
		}
		return val
	default:
		// JSON roundtrip to get map[string]interface{}.
		raw, err := json.Marshal(val)
		if err != nil {
			return val
		}
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			return val
		}
		return m
	}
}
