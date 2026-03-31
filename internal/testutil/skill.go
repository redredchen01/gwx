package testutil

import (
	"context"
	"fmt"
	"sync"

	"github.com/redredchen01/gwx/internal/skill"
)

// MockCaller implements skill.ToolCaller for testing.
type MockCaller struct {
	mu       sync.Mutex
	responses map[string]toolResponse
}

type toolResponse struct {
	output interface{}
	err    error
}

// NewMockCaller creates a new mock tool caller.
func NewMockCaller() *MockCaller {
	return &MockCaller{
		responses: make(map[string]toolResponse),
	}
}

// On registers a response for a given tool.
func (m *MockCaller) On(tool string, output interface{}, err error) *MockCaller {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[tool] = toolResponse{output, err}
	return m
}

// CallTool implements skill.ToolCaller.
func (m *MockCaller) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	resp, ok := m.responses[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return resp.output, resp.err
}

// SkillBuilder constructs a Skill for testing.
type SkillBuilder struct {
	name        string
	version     string
	description string
	inputs      []skill.Input
	steps       []skill.Step
	output      string
	meta        map[string]string
}

// NewSkillBuilder creates a builder for testing.
func NewSkillBuilder(name string) *SkillBuilder {
	return &SkillBuilder{
		name:    name,
		version: "1.0",
		inputs:  []skill.Input{},
		steps:   []skill.Step{},
		meta:    make(map[string]string),
	}
}

// WithStep adds a sequential step.
func (b *SkillBuilder) WithStep(id, tool string, args map[string]string) *SkillBuilder {
	if args == nil {
		args = make(map[string]string)
	}
	b.steps = append(b.steps, skill.Step{
		ID:     id,
		Tool:   tool,
		Args:   args,
		OnFail: "abort",
	})
	return b
}

// WithDAGStep adds a step with explicit dependencies.
func (b *SkillBuilder) WithDAGStep(id, tool string, args map[string]string, dependsOn []string) *SkillBuilder {
	if args == nil {
		args = make(map[string]string)
	}
	b.steps = append(b.steps, skill.Step{
		ID:        id,
		Tool:      tool,
		Args:      args,
		DependsOn: dependsOn,
		OnFail:    "abort",
	})
	return b
}

// WithParallelStep adds a step that runs in parallel.
func (b *SkillBuilder) WithParallelStep(id, tool string, args map[string]string) *SkillBuilder {
	if args == nil {
		args = make(map[string]string)
	}
	b.steps = append(b.steps, skill.Step{
		ID:       id,
		Tool:     tool,
		Args:     args,
		Parallel: true,
		OnFail:   "abort",
	})
	return b
}

// WithInput adds an input parameter.
func (b *SkillBuilder) WithInput(name, typ string, required bool) *SkillBuilder {
	b.inputs = append(b.inputs, skill.Input{
		Name:     name,
		Type:     typ,
		Required: required,
	})
	return b
}

// WithDescription sets the description.
func (b *SkillBuilder) WithDescription(desc string) *SkillBuilder {
	b.description = desc
	return b
}

// WithOutput sets the output template.
func (b *SkillBuilder) WithOutput(output string) *SkillBuilder {
	b.output = output
	return b
}

// Build creates the final Skill.
func (b *SkillBuilder) Build() *skill.Skill {
	if len(b.inputs) == 0 {
		b.inputs = []skill.Input{
			{
				Name:     "example",
				Type:     "string",
				Required: false,
			},
		}
	}
	if len(b.steps) == 0 {
		b.steps = []skill.Step{
			{
				ID:     "step_1",
				Tool:   "echo",
				Args:   map[string]string{},
				OnFail: "abort",
			},
		}
	}
	if b.output == "" {
		b.output = "{{.steps.step_1}}"
	}
	return &skill.Skill{
		Name:        b.name,
		Version:     b.version,
		Description: b.description,
		Inputs:      b.inputs,
		Steps:       b.steps,
		Output:      b.output,
		Meta:        b.meta,
	}
}
