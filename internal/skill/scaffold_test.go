package skill

import (
	"strings"
	"testing"
)

func TestGenerateSkeleton_MinimalName(t *testing.T) {
	yaml := GenerateSkeletonYAML("my-skill", "", []string{})
	if !strings.Contains(yaml, "name: my-skill") {
		t.Fatalf("expected name field, got: %s", yaml)
	}
	if !strings.Contains(yaml, "version:") {
		t.Fatal("expected version field")
	}
	if !strings.Contains(yaml, "steps:") {
		t.Fatal("expected steps section")
	}
}

func TestGenerateSkeleton_WithDescription(t *testing.T) {
	yaml := GenerateSkeletonYAML("my-skill", "Automate email tasks", []string{})
	if !strings.Contains(yaml, `description: "Automate email tasks"`) {
		t.Fatalf("expected description, got: %s", yaml)
	}
}

func TestGenerateSkeleton_WithTools(t *testing.T) {
	yaml := GenerateSkeletonYAML("test", "", []string{"gmail.list", "sheets.append"})

	if !strings.Contains(yaml, "tool: gmail.list") {
		t.Fatal("expected gmail.list tool")
	}
	if !strings.Contains(yaml, "tool: sheets.append") {
		t.Fatal("expected sheets.append tool")
	}
	if !strings.Contains(yaml, "step_1") {
		t.Fatal("expected step_1")
	}
	if !strings.Contains(yaml, "step_2") {
		t.Fatal("expected step_2")
	}
}

func TestGenerateSkeleton_ValidYAML(t *testing.T) {
	yaml := GenerateSkeletonYAML("test-skill", "Test description", []string{"echo"})

	// Parse the generated YAML to ensure it's valid.
	skill, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("generated YAML should be valid: %v\nYAML:\n%s", err, yaml)
	}

	if skill.Name != "test-skill" {
		t.Fatalf("expected name test-skill, got %s", skill.Name)
	}
	if len(skill.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(skill.Steps))
	}
}

func TestGenerateSkeleton_MultipleTools(t *testing.T) {
	tools := []string{"gmail.list", "sheets.append", "calendar.create"}
	yaml := GenerateSkeletonYAML("multi", "", tools)

	for _, tool := range tools {
		if !strings.Contains(yaml, "tool: "+tool) {
			t.Fatalf("expected tool %s in YAML", tool)
		}
	}
}

func TestSanitizeName_Lowercase(t *testing.T) {
	result := sanitizeName("MySkill")
	if result != "myskill" {
		t.Fatalf("expected lowercase, got %s", result)
	}
}

func TestSanitizeName_Spaces(t *testing.T) {
	result := sanitizeName("my skill")
	if result != "my-skill" {
		t.Fatalf("expected hyphens instead of spaces, got %s", result)
	}
}

func TestSanitizeName_InvalidChars(t *testing.T) {
	result := sanitizeName("my@skill!")
	if !strings.HasPrefix(result, "my") || !strings.HasSuffix(result, "skill") {
		t.Fatalf("expected invalid chars removed, got %s", result)
	}
}
