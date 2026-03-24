package workflow

import (
	"context"
	"testing"

	"github.com/redredchen01/gwx/internal/api"
)

// --- FindResult ---

func TestFindResult_Found(t *testing.T) {
	results := []FetchResult{
		{Name: "a", Value: "va"},
		{Name: "b", Value: "vb"},
		{Name: "c", Value: "vc"},
	}
	r := FindResult(results, "b")
	if r == nil {
		t.Fatal("expected to find result 'b'")
	}
	if r.Value != "vb" {
		t.Errorf("value = %v, want vb", r.Value)
	}
}

func TestFindResult_NotFound(t *testing.T) {
	results := []FetchResult{
		{Name: "a", Value: "va"},
	}
	r := FindResult(results, "nonexistent")
	if r != nil {
		t.Errorf("expected nil for non-existent name, got %v", r)
	}
}

func TestFindResult_EmptySlice(t *testing.T) {
	r := FindResult(nil, "x")
	if r != nil {
		t.Errorf("expected nil for empty slice, got %v", r)
	}
}

func TestFindResult_DuplicateNames(t *testing.T) {
	results := []FetchResult{
		{Name: "dup", Value: "first"},
		{Name: "dup", Value: "second"},
	}
	r := FindResult(results, "dup")
	if r == nil || r.Value != "first" {
		t.Errorf("expected first occurrence, got %v", r)
	}
}

// --- RunParallel with empty fetchers ---

func TestRunParallel_EmptyFetchers(t *testing.T) {
	results := RunParallel(context.Background(), nil)
	if len(results) != 0 {
		t.Errorf("expected 0 results for nil fetchers, got %d", len(results))
	}
}

func TestRunParallel_EmptySlice(t *testing.T) {
	results := RunParallel(context.Background(), []Fetcher{})
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty fetchers, got %d", len(results))
	}
}

func TestRunParallel_SingleFetcher(t *testing.T) {
	fetchers := []Fetcher{
		{Name: "only", Fn: func(ctx context.Context) (interface{}, error) {
			return "single", nil
		}},
	}
	results := RunParallel(context.Background(), fetchers)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "only" || results[0].Value != "single" {
		t.Errorf("unexpected result: %v", results[0])
	}
}

// --- Dispatch edge cases ---

func TestDispatch_EmptyActions(t *testing.T) {
	result, err := Dispatch(context.Background(), nil, ExecuteOpts{Execute: true, NoInput: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Executed {
		t.Error("empty actions with execute=true should still mark as executed")
	}
	if len(result.Actions) != 0 {
		t.Error("no actions to execute")
	}
}

func TestDispatch_MCPOverridesExecute(t *testing.T) {
	called := false
	actions := []Action{{Name: "test", Description: "test", Fn: func(ctx context.Context) (interface{}, error) {
		called = true
		return nil, nil
	}}}
	result, err := Dispatch(context.Background(), actions, ExecuteOpts{Execute: true, IsMCP: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Executed {
		t.Error("MCP mode should prevent execution")
	}
	if called {
		t.Error("action should not be called in MCP mode")
	}
}

// --- ExecuteResult / ActionResult types ---

func TestExecuteResult_Fields(t *testing.T) {
	r := ExecuteResult{
		Executed:  true,
		Cancelled: false,
		Reason:    "",
		Actions: []ActionResult{
			{Name: "test", Success: true, Result: "done"},
		},
	}
	if !r.Executed {
		t.Error("Executed should be true")
	}
	if r.Cancelled {
		t.Error("Cancelled should be false")
	}
	if len(r.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(r.Actions))
	}
	if r.Actions[0].Name != "test" || !r.Actions[0].Success {
		t.Error("action should be named 'test' and successful")
	}
}

// --- formatStandupText edge cases ---

func TestFormatStandupText_OnlyGit(t *testing.T) {
	r := &StandupResult{
		Date:       "2026-03-24",
		GitChanges: &GitSection{Commits: []string{"fix: typo"}},
	}
	text := formatStandupText(r)
	if text == "" {
		t.Fatal("should not be empty")
	}
	if !strContains(text, "fix: typo") {
		t.Error("missing git commit")
	}
	if strContains(text, "Meetings Today") {
		t.Error("should not have meetings section")
	}
	if strContains(text, "## Tasks") {
		t.Error("should not have tasks section")
	}
}

func TestFormatStandupText_AllSections(t *testing.T) {
	r := &StandupResult{
		Date:        "2026-03-24",
		GitChanges:  &GitSection{Commits: []string{"feat: auth"}},
		EmailDigest: &DigestSection{Summary: "5 unread emails"},
		Calendar: &CalendarSection{
			Count: 2,
			Events: []api.EventSummary{
				{Title: "Standup", Start: "09:00"},
				{Title: "Review", Start: "14:00"},
			},
		},
		Tasks: &TasksSection{
			Count: 1,
			Tasks: []api.TaskItem{
				{Title: "Deploy", Status: "needsAction"},
			},
		},
	}
	text := formatStandupText(r)
	if !strContains(text, "Git Activity") {
		t.Error("missing Git Activity section")
	}
	if !strContains(text, "Email") {
		t.Error("missing Email section")
	}
	if !strContains(text, "Meetings Today") {
		t.Error("missing Meetings section")
	}
	if !strContains(text, "Tasks") {
		t.Error("missing Tasks section")
	}
}

func TestFormatStandupText_EmptyGitCommits(t *testing.T) {
	r := &StandupResult{
		Date:       "2026-03-24",
		GitChanges: &GitSection{Commits: []string{}},
	}
	text := formatStandupText(r)
	if strContains(text, "Git Activity") {
		t.Error("empty git commits should not show Git Activity section")
	}
}

func TestFormatStandupText_CalendarZeroCount(t *testing.T) {
	r := &StandupResult{
		Date:     "2026-03-24",
		Calendar: &CalendarSection{Count: 0},
	}
	text := formatStandupText(r)
	if strContains(text, "Meetings Today") {
		t.Error("zero count calendar should not show meetings section")
	}
}

// --- WeeklyDigestResult structure ---

func TestWeeklyDigestResult_Fields(t *testing.T) {
	r := WeeklyDigestResult{
		Period:      "2026-03-17 to 2026-03-24",
		EmailStats:  &EmailStatsSection{TotalMessages: 42, TotalUnread: 5, Summary: "test"},
		MeetingLoad: &MeetingLoadSection{Count: 3},
		TasksDone:   &TasksDoneSection{Count: 2},
	}
	if r.Period == "" {
		t.Error("Period should not be empty")
	}
	if r.EmailStats.TotalMessages != 42 {
		t.Errorf("TotalMessages = %d, want 42", r.EmailStats.TotalMessages)
	}
	if r.MeetingLoad.Count != 3 {
		t.Errorf("MeetingLoad.Count = %d, want 3", r.MeetingLoad.Count)
	}
	if r.TasksDone.Count != 2 {
		t.Errorf("TasksDone.Count = %d, want 2", r.TasksDone.Count)
	}
}

// strContains is a simple contains without importing strings.
func strContains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
