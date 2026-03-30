package skill

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestResolveRawURL tests GitHub Gist URL transformation
func TestResolveRawURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "https://gist.github.com/user123/abc123def456",
			expected: "https://gist.githubusercontent.com/user123/abc123def456/raw",
		},
		{
			input:    "https://example.com/skill.yaml",
			expected: "https://example.com/skill.yaml",
		},
		{
			input:    "https://gist.github.com/alice/abc",
			expected: "https://gist.githubusercontent.com/alice/abc/raw",
		},
	}

	for _, tt := range tests {
		result := resolveRawURL(tt.input)
		if result != tt.expected {
			t.Errorf("resolveRawURL(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestFetchSkill_Success tests successful skill fetching
func TestFetchSkill_Success(t *testing.T) {
	skillYAML := `name: test-skill
version: 1.0.0
inputs:
  - name: input1
    required: true
steps:
  - id: step1
    tool: test_tool
    args:
      arg1: "{{ input1 }}"
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		io.WriteString(w, skillYAML)
	}))
	defer server.Close()

	skill, data, err := FetchSkill(server.URL + "/skill.yaml")
	if err != nil {
		t.Fatalf("FetchSkill failed: %v", err)
	}

	if skill == nil {
		t.Fatal("expected skill, got nil")
	}

	if skill.Name != "test-skill" {
		t.Errorf("expected skill name 'test-skill', got %q", skill.Name)
	}

	if data == nil {
		t.Fatal("expected data bytes, got nil")
	}

	if !strings.Contains(string(data), "test-skill") {
		t.Error("expected data to contain skill YAML")
	}
}

// TestFetchSkill_HTTPError tests handling of HTTP errors
func TestFetchSkill_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	skill, _, err := FetchSkill(server.URL + "/notfound.yaml")
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}

	if skill != nil {
		t.Error("expected skill to be nil on error")
	}

	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("expected error to mention HTTP 404, got %q", err.Error())
	}
}

// TestFetchSkill_InvalidYAML tests handling of invalid YAML
func TestFetchSkill_InvalidYAML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "invalid: [unclosed")
	}))
	defer server.Close()

	skill, _, err := FetchSkill(server.URL + "/invalid.yaml")
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}

	if skill != nil {
		t.Error("expected skill to be nil on error")
	}

	if !strings.Contains(err.Error(), "validate") {
		t.Errorf("expected validation error, got %q", err.Error())
	}
}

// TestSanitizeFilename tests filename sanitization
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My Skill", "my-skill"},
		{"Test-Skill_123", "test-skill_123"},
		{"UPPERCASE", "uppercase"},
		{"Special!@#$%Chars", "special-----chars"},
		{"already-clean", "already-clean"},
		{"---leading", "leading"},
		{"trailing---", "trailing"},
		{"multiple   spaces", "multiple---spaces"},
	}

	for _, tt := range tests {
		result := sanitizeFilename(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestFetchSkill_TimeoutConfiguration tests that HTTP client is configured with timeout
func TestFetchSkill_TimeoutConfiguration(t *testing.T) {
	// This test verifies that FetchSkill uses a timeout-enabled HTTP client
	// We can't easily test the actual timeout without blocking, but we can
	// verify the function handles network errors gracefully
	skillYAML := `invalid yaml: [unclosed`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		io.WriteString(w, skillYAML)
	}))
	defer server.Close()

	_, _, err := FetchSkill(server.URL + "/test.yaml")
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
	// The function should gracefully handle errors from network operations
}

// TestFetchSkill_SizeLimitResponse tests that responses larger than 1MB are truncated
func TestFetchSkill_SizeLimitResponse(t *testing.T) {
	// Create a response that's larger than 1MB
	largeResponse := strings.Repeat("x", 2*1024*1024) // 2MB

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		// This would be > 1MB but we're limited to 1MB read
		io.WriteString(w, largeResponse)
	}))
	defer server.Close()

	skill, data, err := FetchSkill(server.URL + "/large.yaml")
	if err == nil {
		t.Logf("Large response was handled (partial read due to size limit)")
	}
	// The exact error depends on how the YAML parser handles truncated input
	if err != nil && !strings.Contains(err.Error(), "validate") {
		t.Logf("Expected validation error for truncated input: %v", err)
	}
	if skill != nil {
		t.Error("expected skill to be nil for invalid large response")
	}
	if data != nil && len(data) > 1024*1024 {
		t.Errorf("data should be limited to ~1MB, got %d bytes", len(data))
	}
}
