package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/redredchen01/gwx/internal/exitcode"
	"github.com/redredchen01/gwx/internal/mcp"
	"github.com/redredchen01/gwx/internal/skill"
)

// SkillCmd groups skill DSL operations.
type SkillCmd struct {
	List     SkillListCmd     `cmd:"" help:"List all loaded skills"`
	Inspect  SkillInspectCmd  `cmd:"" help:"Show details of a skill"`
	Validate SkillValidateCmd `cmd:"" help:"Validate a skill YAML file"`
	Install  SkillInstallCmd  `cmd:"" help:"Install a skill from file or URL"`
	Remove   SkillRemoveCmd   `cmd:"" help:"Remove an installed skill"`
	Run      SkillRunCmd      `cmd:"" help:"Run a skill by name"`
	Create   SkillCreateCmd   `cmd:"" help:"Create a new skill scaffold"`
	Test     SkillTestCmd     `cmd:"" help:"Test a skill with mock data"`
}

// ---- skill list ----

// SkillListCmd lists all discovered skills.
type SkillListCmd struct{}

func (c *SkillListCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "skill.list"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "skill.list"})
		return nil
	}

	skills, err := skill.LoadAll()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("load skills: %s", err))
	}

	// Stable order.
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })

	items := make([]map[string]interface{}, 0, len(skills))
	for _, s := range skills {
		items = append(items, map[string]interface{}{
			"name":        s.Name,
			"version":     s.Version,
			"description": s.Description,
			"inputs":      len(s.Inputs),
			"steps":       len(s.Steps),
		})
	}
	rctx.Printer.Success(map[string]interface{}{"skills": items, "count": len(items)})
	return nil
}

// ---- skill inspect ----

// SkillInspectCmd shows full details of a single skill.
type SkillInspectCmd struct {
	Name string `arg:"" help:"Skill name to inspect"`
}

func (c *SkillInspectCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "skill.inspect"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "skill.inspect", "name": c.Name})
		return nil
	}

	skills, err := skill.LoadAll()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("load skills: %s", err))
	}

	var found *skill.Skill
	for _, s := range skills {
		if s.Name == c.Name {
			found = s
			break
		}
	}
	if found == nil {
		return rctx.Printer.ErrExit(exitcode.NotFound, fmt.Sprintf("skill %q not found", c.Name))
	}

	inputs := make([]map[string]interface{}, 0, len(found.Inputs))
	for _, inp := range found.Inputs {
		m := map[string]interface{}{
			"name":     inp.Name,
			"type":     inp.Type,
			"required": inp.Required,
		}
		if inp.Default != "" {
			m["default"] = inp.Default
		}
		if inp.Description != "" {
			m["description"] = inp.Description
		}
		inputs = append(inputs, m)
	}

	steps := make([]map[string]interface{}, 0, len(found.Steps))
	for _, st := range found.Steps {
		m := map[string]interface{}{
			"id":   st.ID,
			"tool": st.Tool,
		}
		if len(st.Args) > 0 {
			m["args"] = st.Args
		}
		if st.Store != "" {
			m["store"] = st.Store
		}
		if st.OnFail != "abort" {
			m["on_fail"] = st.OnFail
		}
		steps = append(steps, m)
	}

	rctx.Printer.Success(map[string]interface{}{
		"name":        found.Name,
		"version":     found.Version,
		"description": found.Description,
		"inputs":      inputs,
		"steps":       steps,
		"output":      found.Output,
		"meta":        found.Meta,
	})
	return nil
}

// ---- skill validate ----

// SkillValidateCmd validates a YAML skill file without running it.
type SkillValidateCmd struct {
	File string `arg:"" help:"Path to skill YAML file"`
}

func (c *SkillValidateCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "skill.validate"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "skill.validate", "file": c.File})
		return nil
	}

	s, err := skill.LoadFile(c.File)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, fmt.Sprintf("validation failed: %s", err))
	}
	rctx.Printer.Success(map[string]interface{}{
		"valid":       true,
		"name":        s.Name,
		"version":     s.Version,
		"description": s.Description,
		"inputs":      len(s.Inputs),
		"steps":       len(s.Steps),
	})
	return nil
}

// ---- skill install ----

// SkillInstallCmd installs a skill from a local file or a remote URL.
type SkillInstallCmd struct {
	Source string `arg:"" help:"File path or URL to install from"`
}

func (c *SkillInstallCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "skill.install"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "skill.install", "source": c.Source})
		return nil
	}

	var dest string
	var err error

	if strings.HasPrefix(c.Source, "http://") || strings.HasPrefix(c.Source, "https://") {
		dest, err = skill.InstallFromURL(c.Source)
	} else {
		dest, err = skill.InstallFromFile(c.Source)
	}
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("install skill: %s", err))
	}

	// Re-load the installed file to report details.
	s, loadErr := skill.LoadFile(dest)
	if loadErr != nil {
		// File was written but re-parse failed — should not happen since we validated earlier.
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("installed but failed to re-load: %s", loadErr))
	}

	rctx.Printer.Success(map[string]interface{}{
		"installed":   true,
		"name":        s.Name,
		"version":     s.Version,
		"description": s.Description,
		"path":        dest,
		"source":      c.Source,
	})
	return nil
}

// ---- skill remove ----

// SkillRemoveCmd removes an installed skill by name.
type SkillRemoveCmd struct {
	Name string `arg:"" help:"Skill name to remove"`
}

func (c *SkillRemoveCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "skill.remove"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "skill.remove", "name": c.Name})
		return nil
	}

	if err := skill.UninstallSkill(c.Name); err != nil {
		return rctx.Printer.ErrExit(exitcode.NotFound, fmt.Sprintf("remove skill: %s", err))
	}

	rctx.Printer.Success(map[string]interface{}{
		"removed": true,
		"name":    c.Name,
	})
	return nil
}

// ---- skill run ----

// SkillRunCmd executes a skill by name with optional parameters.
type SkillRunCmd struct {
	Name   string            `arg:"" help:"Skill name to run"`
	Params map[string]string `help:"Input parameters as key=value pairs" short:"p"`
}

// allGoogleServices returns all known Google service names for auth.
var allGoogleServices = []string{
	"gmail", "calendar", "drive", "docs", "sheets",
	"tasks", "people", "chat", "analytics", "searchconsole", "slides",
}

func (c *SkillRunCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "skill.run"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	skills, err := skill.LoadAll()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("load skills: %s", err))
	}

	var found *skill.Skill
	for _, s := range skills {
		if s.Name == c.Name {
			found = s
			break
		}
	}
	if found == nil {
		return rctx.Printer.ErrExit(exitcode.NotFound, fmt.Sprintf("skill %q not found", c.Name))
	}

	// Dry-run: preview steps and resolved params.
	if rctx.DryRun {
		stepPreviews := make([]map[string]interface{}, 0, len(found.Steps))
		for _, st := range found.Steps {
			sp := map[string]interface{}{
				"id":   st.ID,
				"tool": st.Tool,
			}
			if len(st.Args) > 0 {
				sp["args"] = st.Args
			}
			if st.If != "" {
				sp["if"] = st.If
			}
			if st.Each != "" {
				sp["each"] = st.Each
			}
			stepPreviews = append(stepPreviews, sp)
		}
		rctx.Printer.Success(map[string]interface{}{
			"dry_run": true,
			"skill":   found.Name,
			"params":  c.Params,
			"steps":   stepPreviews,
		})
		return nil
	}

	// Authenticate with all Google services (skill may use any of them).
	if err := EnsureAuth(rctx, allGoogleServices); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	// Build inputs map.
	inputs := make(map[string]string, len(c.Params))
	for k, v := range c.Params {
		inputs[k] = v
	}

	// Create ToolCaller via gwxHandlerCaller adapter (same pattern as registry.go).
	handler := mcp.NewGWXHandler(rctx.APIClient)
	caller := &gwxHandlerCaller{h: handler}
	engine := skill.NewEngine(caller)

	execCtx, cancel := context.WithTimeout(rctx.Context, 5*time.Minute)
	defer cancel()

	result, err := engine.Run(execCtx, found, inputs)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("run skill: %s", err))
	}

	rctx.Printer.Success(result)
	return nil
}

// gwxHandlerCaller adapts *mcp.GWXHandler to the skill.ToolCaller interface.
// This mirrors the same adapter in internal/skill/registry.go but is accessible
// from the cmd package without creating circular dependencies.
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
		if err := parseJSON(result.Content[0].Text, &parsed); err == nil {
			return parsed, nil
		}
		return result.Content[0].Text, nil
	}
	return nil, nil
}

// parseJSON is a helper to avoid importing encoding/json in multiple places.
func parseJSON(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}

// ---- skill create ----

// SkillCreateCmd scaffolds a new skill YAML file.
type SkillCreateCmd struct {
	Name string `arg:"" help:"Skill name (lowercase, hyphens allowed)"`
}

const skillTemplate = `name: %s
version: "1.0"
description: "TODO: describe what this skill does"

inputs:
  - name: example-input
    type: string
    required: false
    default: "hello"
    description: "An example input parameter"

steps:
  - id: step1
    tool: gmail_list
    args:
      limit: "10"

output: "{{.steps.step1}}"
`

func (c *SkillCreateCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "skill.create"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "skill.create", "name": c.Name})
		return nil
	}

	// Validate name: lowercase + hyphens only.
	for _, r := range c.Name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return rctx.Printer.ErrExit(exitcode.InvalidInput,
				fmt.Sprintf("skill name %q must be lowercase alphanumeric with hyphens only", c.Name))
		}
	}
	if c.Name == "" {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, "skill name cannot be empty")
	}

	dir := "skills"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("create skills directory: %s", err))
	}

	dest := filepath.Join(dir, c.Name+".yaml")

	// Check if file already exists.
	if _, err := os.Stat(dest); err == nil {
		return rctx.Printer.ErrExit(exitcode.Conflict, fmt.Sprintf("skill file %s already exists", dest))
	}

	content := fmt.Sprintf(skillTemplate, c.Name)
	if err := os.WriteFile(dest, []byte(content), 0644); err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("write skill file: %s", err))
	}

	rctx.Printer.Success(map[string]interface{}{
		"created": true,
		"name":    c.Name,
		"path":    dest,
	})
	return nil
}

// ---- skill test ----

// SkillTestCmd tests a skill with mock data (no real API calls).
type SkillTestCmd struct {
	Name string `arg:"" help:"Skill name to test"`
}

func (c *SkillTestCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "skill.test"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "skill.test", "name": c.Name})
		return nil
	}

	skills, err := skill.LoadAll()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("load skills: %s", err))
	}

	var found *skill.Skill
	for _, s := range skills {
		if s.Name == c.Name {
			found = s
			break
		}
	}
	if found == nil {
		return rctx.Printer.ErrExit(exitcode.NotFound, fmt.Sprintf("skill %q not found", c.Name))
	}

	// Build default inputs.
	inputs := make(map[string]string, len(found.Inputs))
	for _, inp := range found.Inputs {
		if inp.Default != "" {
			inputs[inp.Name] = inp.Default
		} else if inp.Required {
			// Provide a test value for required inputs without defaults.
			inputs[inp.Name] = "test-" + inp.Name
		}
	}

	// Run with mock caller.
	caller := &mockToolCaller{}
	engine := skill.NewEngine(caller)

	execCtx, cancel := context.WithTimeout(rctx.Context, 30*time.Second)
	defer cancel()

	result, err := engine.Run(execCtx, found, inputs)
	if err != nil {
		rctx.Printer.Success(map[string]interface{}{
			"test":    c.Name,
			"success": false,
			"error":   err.Error(),
			"inputs":  inputs,
		})
		return nil
	}

	// Build step summary.
	stepSummary := make([]map[string]interface{}, 0, len(result.Steps))
	for _, s := range result.Steps {
		entry := map[string]interface{}{
			"id":      s.ID,
			"tool":    s.Tool,
			"success": s.Success,
		}
		if s.Error != "" {
			entry["error"] = s.Error
		}
		stepSummary = append(stepSummary, entry)
	}

	rctx.Printer.Success(map[string]interface{}{
		"test":            c.Name,
		"success":         result.Success,
		"steps_executed":  len(result.Steps),
		"steps":           stepSummary,
		"inputs_used":     inputs,
		"output_rendered": result.Output != nil,
		"output":          result.Output,
	})
	return nil
}

// mockToolCaller returns realistic-looking sample data based on tool name prefix.
type mockToolCaller struct{}

func (m *mockToolCaller) CallTool(_ context.Context, name string, _ map[string]interface{}) (interface{}, error) {
	return mockDataForTool(name), nil
}

func mockDataForTool(toolName string) interface{} {
	switch {
	case strings.HasPrefix(toolName, "gmail_"):
		return map[string]interface{}{
			"messages": []interface{}{
				map[string]interface{}{"id": "mock1", "subject": "Test"},
			},
			"count": float64(1),
		}
	case strings.HasPrefix(toolName, "calendar_"):
		return map[string]interface{}{
			"events": []interface{}{
				map[string]interface{}{"id": "mock1", "title": "Test Meeting"},
			},
			"count": float64(1),
		}
	case strings.HasPrefix(toolName, "drive_"):
		return map[string]interface{}{
			"files": []interface{}{
				map[string]interface{}{"id": "mock1", "name": "Test.doc"},
			},
			"count": float64(1),
		}
	case strings.HasPrefix(toolName, "sheets_"):
		return map[string]interface{}{
			"values": []interface{}{
				[]interface{}{"A1", "B1"},
			},
			"range": "Sheet1!A:B",
		}
	case strings.HasPrefix(toolName, "contacts_"):
		return map[string]interface{}{
			"contacts": []interface{}{
				map[string]interface{}{"id": "mock1", "name": "Test Contact", "email": "test@example.com"},
			},
			"count": float64(1),
		}
	case strings.HasPrefix(toolName, "tasks_"):
		return map[string]interface{}{
			"tasks": []interface{}{
				map[string]interface{}{"id": "mock1", "title": "Test Task"},
			},
			"count": float64(1),
		}
	case strings.HasPrefix(toolName, "slides_"):
		return map[string]interface{}{
			"slides": []interface{}{
				map[string]interface{}{"id": "mock1", "title": "Test Slide"},
			},
			"count": float64(1),
		}
	default:
		return map[string]interface{}{
			"status": "ok",
			"tool":   toolName,
		}
	}
}
