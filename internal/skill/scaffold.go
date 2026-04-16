package skill

import (
	"fmt"
	"strings"
)

// GenerateSkeletonYAML creates a minimal, valid skill YAML template.
func GenerateSkeletonYAML(name, description string, tools []string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("name: %s\n", sanitizeName(name)))
	sb.WriteString("version: \"1.0\"\n")

	if description != "" {
		sb.WriteString(fmt.Sprintf("description: %q\n", description))
	} else {
		sb.WriteString(fmt.Sprintf("description: \"TODO: Describe %s\"\n", name))
	}

	sb.WriteString("\ninputs:\n")
	sb.WriteString("  - name: example_input\n")
	sb.WriteString("    type: string\n")
	sb.WriteString("    required: false\n")
	sb.WriteString("    default: \"\"\n")
	sb.WriteString("    description: \"Example input parameter\"\n")

	sb.WriteString("\nsteps:\n")
	if len(tools) == 0 {
		tools = []string{"echo"}
	}
	for i, tool := range tools {
		sb.WriteString(scaffoldStep(i, tool))
	}

	sb.WriteString("\noutput: |\n")
	sb.WriteString("  {{.steps.step_1}}\n")

	sb.WriteString("\nmeta:\n")
	sb.WriteString("  author: \"\"\n")
	sb.WriteString("  tags: \"\"\n")

	return sb.String()
}

// scaffoldStep generates YAML for a single step.
func scaffoldStep(idx int, tool string) string {
	stepID := fmt.Sprintf("step_%d", idx+1)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  - id: %s\n", stepID))
	sb.WriteString(fmt.Sprintf("    tool: %s\n", tool))
	sb.WriteString("    args:\n")

	// Generate placeholder args based on tool naming convention.
	if strings.Contains(tool, ".") {
		parts := strings.Split(tool, ".")
		service := parts[0]
		sb.WriteString(fmt.Sprintf("      query: \"{{.inputs.example_input}}\"\n"))
		if service == "gmail" {
			sb.WriteString("      # Additional gmail args here\n")
		} else if service == "sheets" {
			sb.WriteString("      # Additional sheets args here\n")
		}
	} else {
		sb.WriteString("      input: \"{{.inputs.example_input}}\"\n")
	}

	return sb.String()
}

// sanitizeName ensures the skill name is a valid identifier.
// Converts to lowercase, replaces spaces with hyphens, removes invalid chars.
func sanitizeName(name string) string {
	name = strings.ToLower(name)
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			sb.WriteRune(r)
		} else if r == ' ' {
			sb.WriteRune('-')
		}
	}
	return strings.Trim(sb.String(), "-")
}
