package cmd

import (
	"testing"
)

// Tests for pure functions in the cmd package that don't require binary execution.

// --- parsePipeline ---

func TestParsePipeline_Single(t *testing.T) {
	stages := parsePipeline("gmail list")
	if len(stages) != 1 || stages[0] != "gmail list" {
		t.Fatalf("expected [gmail list], got %v", stages)
	}
}

func TestParsePipeline_Multi(t *testing.T) {
	stages := parsePipeline("gmail search invoice | sheets append ID A:C")
	if len(stages) != 2 {
		t.Fatalf("expected 2 stages, got %d: %v", len(stages), stages)
	}
	if stages[0] != "gmail search invoice" {
		t.Errorf("stage[0] = %q", stages[0])
	}
	if stages[1] != "sheets append ID A:C" {
		t.Errorf("stage[1] = %q", stages[1])
	}
}

func TestParsePipeline_Empty(t *testing.T) {
	stages := parsePipeline("")
	if len(stages) != 0 {
		t.Fatalf("expected 0 stages, got %d", len(stages))
	}
}

func TestParsePipeline_OnlyPipes(t *testing.T) {
	stages := parsePipeline("| | |")
	if len(stages) != 0 {
		t.Fatalf("expected 0 stages for only pipes, got %d", len(stages))
	}
}

func TestParsePipeline_TrimWhitespace(t *testing.T) {
	stages := parsePipeline("  gmail list  |  drive search report  ")
	if len(stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(stages))
	}
	if stages[0] != "gmail list" {
		t.Errorf("stage[0] not trimmed: %q", stages[0])
	}
	if stages[1] != "drive search report" {
		t.Errorf("stage[1] not trimmed: %q", stages[1])
	}
}

// --- splitArgs ---

func TestSplitArgs_Simple(t *testing.T) {
	args := splitArgs("gmail search hello")
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[0] != "gmail" || args[1] != "search" || args[2] != "hello" {
		t.Errorf("unexpected args: %v", args)
	}
}

func TestSplitArgs_DoubleQuoted(t *testing.T) {
	args := splitArgs(`gmail search "from:boss subject:urgent"`)
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[2] != "from:boss subject:urgent" {
		t.Errorf("quoted arg = %q", args[2])
	}
}

func TestSplitArgs_SingleQuoted(t *testing.T) {
	args := splitArgs("drive search 'name contains report'")
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
	if args[2] != "name contains report" {
		t.Errorf("quoted arg = %q", args[2])
	}
}

func TestSplitArgs_Empty(t *testing.T) {
	args := splitArgs("")
	if len(args) != 0 {
		t.Fatalf("expected 0 args, got %d", len(args))
	}
}

func TestSplitArgs_TabSeparated(t *testing.T) {
	args := splitArgs("a\tb\tc")
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
}

func TestSplitArgs_MultipleSpaces(t *testing.T) {
	args := splitArgs("a    b     c")
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}
}

// --- escapeDriveQuery ---

func TestEscapeDriveQuery_NoQuotes(t *testing.T) {
	got := escapeDriveQuery("hello world")
	if got != "hello world" {
		t.Errorf("expected no change, got %q", got)
	}
}

func TestEscapeDriveQuery_WithQuotes(t *testing.T) {
	got := escapeDriveQuery("it's a test")
	if got != "it\\'s a test" {
		t.Errorf("expected escaped quote, got %q", got)
	}
}

func TestEscapeDriveQuery_Empty(t *testing.T) {
	got := escapeDriveQuery("")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// --- formatContextSummary ---

func TestFormatContextSummary(t *testing.T) {
	s := formatContextSummary("budget", 3, 1, 0)
	if s != `Context for "budget": 3 emails, 1 file, 0 upcoming events.` {
		t.Errorf("unexpected summary: %q", s)
	}
}

func TestFormatContextSummary_Singular(t *testing.T) {
	s := formatContextSummary("topic", 1, 1, 1)
	if s != `Context for "topic": 1 email, 1 file, 1 upcoming event.` {
		t.Errorf("unexpected summary: %q", s)
	}
}

// --- intWord ---

func TestIntWord_Zero(t *testing.T) {
	if got := intWord(0, "item"); got != "0 items" {
		t.Errorf("intWord(0, item) = %q", got)
	}
}

func TestIntWord_One(t *testing.T) {
	if got := intWord(1, "item"); got != "1 item" {
		t.Errorf("intWord(1, item) = %q", got)
	}
}

func TestIntWord_Many(t *testing.T) {
	if got := intWord(5, "item"); got != "5 items" {
		t.Errorf("intWord(5, item) = %q", got)
	}
}

// --- toLower ---

func TestToLower(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Hello", "hello"},
		{"UPPER", "upper"},
		{"already", "already"},
		{"MiXeD", "mixed"},
		{"", ""},
		{"123ABC", "123abc"},
	}
	for _, tt := range tests {
		if got := toLower(tt.in); got != tt.want {
			t.Errorf("toLower(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// --- containsCI ---

func TestContainsCI(t *testing.T) {
	tests := []struct {
		s, sub string
		want   bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello", "hello world", false},
		{"", "a", false},
		{"abc", "", true},
		{"BUDGET Meeting", "budget", true},
	}
	for _, tt := range tests {
		got := containsCI(tt.s, toLower(tt.sub))
		if got != tt.want {
			t.Errorf("containsCI(%q, %q) = %v, want %v", tt.s, tt.sub, got, tt.want)
		}
	}
}

// --- itoa ---

func TestItoa(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{100, "100"},
		{-5, "-5"},
	}
	for _, tt := range tests {
		if got := itoa(tt.n); got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

// --- Preflight / CheckAllowlist ---

func TestCheckAllowlist_NilAllowlist(t *testing.T) {
	rctx := &RunContext{Allowlist: nil}
	if err := CheckAllowlist(rctx, "any.command"); err != nil {
		t.Errorf("nil allowlist should allow all commands, got error: %v", err)
	}
}

// --- parseOwnerRepo ---

func TestParseOwnerRepo_Valid(t *testing.T) {
	owner, repo, err := parseOwnerRepo("octocat/hello-world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "octocat" {
		t.Errorf("owner = %q, want octocat", owner)
	}
	if repo != "hello-world" {
		t.Errorf("repo = %q, want hello-world", repo)
	}
}

func TestParseOwnerRepo_Invalid_NoSlash(t *testing.T) {
	_, _, err := parseOwnerRepo("just-a-name")
	if err == nil {
		t.Fatal("expected error for no slash")
	}
}

func TestParseOwnerRepo_Invalid_EmptyOwner(t *testing.T) {
	_, _, err := parseOwnerRepo("/repo")
	if err == nil {
		t.Fatal("expected error for empty owner")
	}
}

func TestParseOwnerRepo_Invalid_EmptyRepo(t *testing.T) {
	_, _, err := parseOwnerRepo("owner/")
	if err == nil {
		t.Fatal("expected error for empty repo")
	}
}

func TestParseOwnerRepo_WithSubpath(t *testing.T) {
	owner, repo, err := parseOwnerRepo("org/my-repo/extra")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "org" {
		t.Errorf("owner = %q, want org", owner)
	}
	// SplitN with 2 means "my-repo/extra"
	if repo != "my-repo/extra" {
		t.Errorf("repo = %q, want my-repo/extra", repo)
	}
}

// --- buildGwxArgs ---

func TestBuildGwxArgs_Default(t *testing.T) {
	rctx := &RunContext{Account: "default"}
	args := buildGwxArgs("gmail list --limit 10", rctx)
	// Should contain the original args plus --format json
	if len(args) < 4 {
		t.Fatalf("expected at least 4 args, got %d: %v", len(args), args)
	}
	found := false
	for i, a := range args {
		if a == "--format" && i+1 < len(args) && args[i+1] == "json" {
			found = true
		}
	}
	if !found {
		t.Errorf("args should contain --format json: %v", args)
	}
}

func TestBuildGwxArgs_NonDefaultAccount(t *testing.T) {
	rctx := &RunContext{Account: "work@company.com"}
	args := buildGwxArgs("gmail list", rctx)
	found := false
	for i, a := range args {
		if a == "--account" && i+1 < len(args) && args[i+1] == "work@company.com" {
			found = true
		}
	}
	if !found {
		t.Errorf("non-default account should add --account flag: %v", args)
	}
}

// --- parseJSON ---

func TestParseJSON_Valid(t *testing.T) {
	var result map[string]string
	if err := parseJSON(`{"key": "value"}`, &result); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("key = %q, want value", result["key"])
	}
}

func TestParseJSON_Invalid(t *testing.T) {
	var result map[string]string
	if err := parseJSON("not json", &result); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseJSON_Empty(t *testing.T) {
	var result map[string]string
	if err := parseJSON("", &result); err == nil {
		t.Fatal("expected error for empty string")
	}
}
