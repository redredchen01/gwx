package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/redredchen01/gwx/internal/exitcode"
	"github.com/redredchen01/gwx/internal/skill"
)

// SkillCmd groups skill DSL operations.
type SkillCmd struct {
	List     SkillListCmd     `cmd:"" help:"List all loaded skills"`
	Inspect  SkillInspectCmd  `cmd:"" help:"Show details of a skill"`
	Validate SkillValidateCmd `cmd:"" help:"Validate a skill YAML file"`
	Install  SkillInstallCmd  `cmd:"" help:"Install a skill from file or URL"`
	Remove   SkillRemoveCmd   `cmd:"" help:"Remove an installed skill"`
	Search   SkillSearchCmd   `cmd:"" help:"Search skills by keyword"`
	Browse   SkillBrowseCmd   `cmd:"" help:"Browse skills by tag"`
	New      SkillNewCmd      `cmd:"" help:"Scaffold a new skill YAML"`
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

// ---- skill search ----

// SkillSearchCmd searches skills by keyword.
type SkillSearchCmd struct {
	Query  string `arg:"" help:"Search query (name, description, or tag)"`
	Source string `help:"Source: local or remote" default:"local" enum:"local,remote"`
}

func (c *SkillSearchCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "skill.search"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "skill.search", "query": c.Query, "source": c.Source})
		return nil
	}

	var skills []*skill.Skill
	var err error

	if c.Source == "remote" {
		// Remote catalog requires a URL; for now, not implemented.
		return rctx.Printer.ErrExit(exitcode.GeneralError, "remote catalog search not yet implemented")
	}

	// Local search
	allSkills, err := skill.LoadAll()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("load skills: %s", err))
	}

	skills = skill.SearchSkills(allSkills, c.Query)

	items := make([]map[string]interface{}, 0, len(skills))
	for _, s := range skills {
		items = append(items, map[string]interface{}{
			"name":        s.Name,
			"version":     s.Version,
			"description": s.Description,
			"tags":        skill.SkillTags(s),
		})
	}

	rctx.Printer.Success(map[string]interface{}{
		"query":   c.Query,
		"results": items,
		"count":   len(items),
	})
	return nil
}

// ---- skill browse ----

// SkillBrowseCmd browses skills by tag.
type SkillBrowseCmd struct {
	Tag    string `help:"Filter by tag (e.g. gmail, github, calendar)" default:""`
	Source string `help:"Source: local or remote" default:"local" enum:"local,remote"`
}

func (c *SkillBrowseCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "skill.browse"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{"dry_run": "skill.browse", "tag": c.Tag, "source": c.Source})
		return nil
	}

	var skills []*skill.Skill
	var err error

	if c.Source == "remote" {
		// Remote catalog requires a URL; for now, not implemented.
		return rctx.Printer.ErrExit(exitcode.GeneralError, "remote catalog browse not yet implemented")
	}

	// Local browse
	allSkills, err := skill.LoadAll()
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("load skills: %s", err))
	}

	if c.Tag != "" {
		skills = skill.FilterByTag(allSkills, c.Tag)
	} else {
		skills = allSkills
	}

	// Stable order
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })

	items := make([]map[string]interface{}, 0, len(skills))
	for _, s := range skills {
		items = append(items, map[string]interface{}{
			"name":        s.Name,
			"version":     s.Version,
			"description": s.Description,
			"tags":        skill.SkillTags(s),
		})
	}

	result := map[string]interface{}{
		"results": items,
		"count":   len(items),
	}
	if c.Tag != "" {
		result["tag"] = c.Tag
	}

	rctx.Printer.Success(result)
	return nil
}

// ---- skill new ----

// SkillNewCmd scaffolds a new skill YAML template.
type SkillNewCmd struct {
	Name        string   `arg:"" help:"Skill name (e.g. gmail-daily-digest)"`
	Description string   `help:"Short description" short:"d" default:""`
	Tool        []string `help:"Tool(s) to include (e.g. gmail.list)" short:"t"`
	Output      string   `help:"Output path; '-' for stdout" short:"o" default:"-"`
}

func (c *SkillNewCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "skill.new"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]string{
			"dry_run":     "skill.new",
			"name":        c.Name,
			"description": c.Description,
			"tools":       fmt.Sprintf("%v", c.Tool),
			"output":      c.Output,
		})
		return nil
	}

	// Generate the skeleton YAML.
	skeleton := skill.GenerateSkeletonYAML(c.Name, c.Description, c.Tool)

	// Output to stdout or file.
	if c.Output == "-" {
		// Write to stdout.
		rctx.Printer.Success(map[string]interface{}{
			"generated": true,
			"name":      c.Name,
			"yaml":      skeleton,
		})
		return nil
	}

	// Write to file.
	if err := os.WriteFile(c.Output, []byte(skeleton), 0644); err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("write skill file: %s", err))
	}

	// Verify the written file by parsing it.
	s, err := skill.LoadFile(c.Output)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("generated skill failed validation: %s", err))
	}

	rctx.Printer.Success(map[string]interface{}{
		"generated":   true,
		"name":        s.Name,
		"version":     s.Version,
		"description": s.Description,
		"steps":       len(s.Steps),
		"path":        c.Output,
	})
	return nil
}
