package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redredchen01/gwx/internal/mcp"
)

// skillProvider implements mcp.ToolProvider for dynamically loaded skills.
type skillProvider struct {
	mu     sync.RWMutex
	skills []*Skill
}

var provider = &skillProvider{}

func init() {
	mcp.RegisterProvider(provider)
}

// Reload discovers skills from disk and refreshes the provider's skill list.
// Safe for concurrent use. Uses the skill cache to avoid redundant disk reads.
func Reload() error {
	InvalidateSkillCache()
	skills, err := CachedLoadAll()
	if err != nil {
		return err
	}
	provider.mu.Lock()
	provider.skills = skills
	provider.mu.Unlock()
	return nil
}

// RegisterSkills explicitly sets the loaded skills (used by CLI commands that
// have already loaded skills).
func RegisterSkills(skills []*Skill) {
	provider.mu.Lock()
	provider.skills = skills
	provider.mu.Unlock()
}

// Skills returns the currently loaded skills snapshot.
func Skills() []*Skill {
	provider.mu.RLock()
	defer provider.mu.RUnlock()
	out := make([]*Skill, len(provider.skills))
	copy(out, provider.skills)
	return out
}

// Tools returns MCP tool definitions for all loaded skills.
func (p *skillProvider) Tools() []mcp.Tool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tools := make([]mcp.Tool, 0, len(p.skills))
	for _, s := range p.skills {
		tools = append(tools, skillToTool(s))
	}
	return tools
}

// Handlers returns MCP tool handlers for all loaded skills.
func (p *skillProvider) Handlers(h *mcp.GWXHandler) map[string]mcp.ToolHandler {
	p.mu.RLock()
	defer p.mu.RUnlock()

	handlers := make(map[string]mcp.ToolHandler, len(p.skills))
	for _, s := range p.skills {
		sk := s // capture
		toolName := skillToolName(sk)
		handlers[toolName] = makeSkillHandler(sk, h)
	}
	return handlers
}

func skillToolName(s *Skill) string {
	return "skill_" + s.Name
}

func skillToTool(s *Skill) mcp.Tool {
	props := make(map[string]mcp.Property, len(s.Inputs))
	var required []string
	for _, inp := range s.Inputs {
		propType := "string"
		switch inp.Type {
		case "int":
			propType = "integer"
		case "bool":
			propType = "boolean"
		}
		prop := mcp.Property{
			Type:        propType,
			Description: inp.Description,
		}
		if inp.Default != "" {
			prop.Default = inp.Default
		}
		props[inp.Name] = prop
		if inp.Required {
			required = append(required, inp.Name)
		}
	}

	desc := s.Description
	if desc == "" {
		desc = fmt.Sprintf("Execute skill: %s", s.Name)
	}

	return mcp.Tool{
		Name:        skillToolName(s),
		Description: desc,
		InputSchema: mcp.InputSchema{
			Type:       "object",
			Properties: props,
			Required:   required,
		},
	}
}

// gwxHandlerCaller adapts *mcp.GWXHandler to the skill.ToolCaller interface.
type gwxHandlerCaller struct {
	h *mcp.GWXHandler
}

func (c *gwxHandlerCaller) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	result, err := c.h.CallTool(name, args)
	if err != nil {
		return nil, err
	}
	if result.IsError {
		if len(result.Content) > 0 {
			return nil, fmt.Errorf("tool %s error: %s", name, result.Content[0].Text)
		}
		return nil, fmt.Errorf("tool %s returned error", name)
	}
	// Parse the text content back to structured data.
	if len(result.Content) > 0 {
		var parsed interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &parsed); err == nil {
			return parsed, nil
		}
		return result.Content[0].Text, nil
	}
	return nil, nil
}

func makeSkillHandler(s *Skill, h *mcp.GWXHandler) mcp.ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*mcp.ToolResult, error) {
		// Build string input map from args.
		inputs := make(map[string]string, len(args))
		for k, v := range args {
			inputs[k] = fmt.Sprintf("%v", v)
		}

		caller := &gwxHandlerCaller{h: h}
		engine := NewEngine(caller)

		execCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		rr, err := engine.Run(execCtx, s, inputs)
		if err != nil {
			return nil, err
		}

		raw, err := json.MarshalIndent(rr, "", "  ")
		if err != nil {
			return nil, err
		}
		return &mcp.ToolResult{
			Content: []mcp.ContentBlock{{Type: "text", Text: string(raw)}},
			IsError: !rr.Success,
		}, nil
	}
}
