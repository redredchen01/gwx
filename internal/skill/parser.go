package skill

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ParseFile reads and validates a YAML skill file.
func ParseFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill file %s: %w", path, err)
	}
	return Parse(data)
}

// Parse unmarshals and validates YAML bytes into a Skill.
func Parse(data []byte) (*Skill, error) {
	var s Skill
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse skill YAML: %w", err)
	}
	if err := validate(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func validate(s *Skill) error {
	if s.Name == "" {
		return fmt.Errorf("skill: name is required")
	}
	if len(s.Steps) == 0 {
		return fmt.Errorf("skill %q: at least one step is required", s.Name)
	}
	ids := make(map[string]bool, len(s.Steps))
	for i, st := range s.Steps {
		if st.ID == "" {
			s.Steps[i].ID = fmt.Sprintf("step_%d", i+1)
		}
		if ids[s.Steps[i].ID] {
			return fmt.Errorf("skill %q: duplicate step id %q", s.Name, s.Steps[i].ID)
		}
		ids[s.Steps[i].ID] = true
		if st.Tool == "" {
			return fmt.Errorf("skill %q step %q: tool is required", s.Name, s.Steps[i].ID)
		}
		if st.OnFail == "" {
			s.Steps[i].OnFail = "abort"
		}
	}
	for _, inp := range s.Inputs {
		if inp.Name == "" {
			return fmt.Errorf("skill %q: input name is required", s.Name)
		}
		switch inp.Type {
		case "", "string", "int", "bool":
			// ok
		default:
			return fmt.Errorf("skill %q input %q: unsupported type %q", s.Name, inp.Name, inp.Type)
		}
	}
	return nil
}
