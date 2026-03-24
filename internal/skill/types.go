package skill

// Skill represents a parsed YAML skill definition.
type Skill struct {
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version"`
	Description string            `yaml:"description"`
	Inputs      []Input           `yaml:"inputs"`
	Steps       []Step            `yaml:"steps"`
	Output      string            `yaml:"output"`
	Meta        map[string]string `yaml:"meta"`
}

// Input defines a skill parameter.
type Input struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"` // string, int, bool
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default"`
	Description string `yaml:"description"`
}

// Step is a single operation in a skill pipeline.
type Step struct {
	ID       string            `yaml:"id"`
	Tool     string            `yaml:"tool"`
	Args     map[string]string `yaml:"args"`
	Store    string            `yaml:"store"`
	OnFail   string            `yaml:"on_fail"` // skip, abort (default: abort)
	Parallel bool              `yaml:"parallel"` // run concurrently with adjacent parallel steps
	Each     string            `yaml:"each"`     // iterate over list expression, e.g. "{{.steps.contacts}}"
	If       string            `yaml:"if"`       // conditional: skip step when expression evaluates to falsy
}

// StepResult holds the output of a single executed step.
type StepResult struct {
	ID     string
	Output interface{}
	Err    error
}

// RunResult is the final output of a skill execution.
type RunResult struct {
	Skill   string       `json:"skill"`
	Success bool         `json:"success"`
	Steps   []StepReport `json:"steps"`
	Output  interface{}  `json:"output,omitempty"`
	Error   string       `json:"error,omitempty"`
}

// StepReport summarises a single step in the run result.
type StepReport struct {
	ID      string      `json:"id"`
	Tool    string      `json:"tool"`
	Success bool        `json:"success"`
	Output  interface{} `json:"output,omitempty"`
	Error   string      `json:"error,omitempty"`
}
