package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/redredchen01/gwx/internal/api"
)

// --- computeTestStats edge cases ---

func TestComputeTestStats_HeaderOnly(t *testing.T) {
	data := &api.SheetData{
		Values: [][]interface{}{
			{"TC", "Desc", "Module", "Status"},
		},
	}
	stats := computeTestStats(data)
	if stats["total"] != 0 {
		t.Errorf("header-only should have total=0, got %d", stats["total"])
	}
}

func TestComputeTestStats_EmptyValues(t *testing.T) {
	data := &api.SheetData{Values: [][]interface{}{}}
	stats := computeTestStats(data)
	if stats["total"] != 0 {
		t.Errorf("empty values should have total=0, got %d", stats["total"])
	}
}

func TestComputeTestStats_CaseInsensitive(t *testing.T) {
	data := &api.SheetData{
		Values: [][]interface{}{
			{"TC", "Desc", "Module", "Status"},
			{"TC-001", "test1", "auth", "PASS"},
			{"TC-002", "test2", "api", "Fail"},
			{"TC-003", "test3", "auth", "Pass"},
		},
	}
	stats := computeTestStats(data)
	// These should count regardless of case (lowercase comparison in safeCol)
	if stats["total"] != 3 {
		t.Errorf("total = %d, want 3", stats["total"])
	}
}

func TestComputeTestStats_MissingStatusColumn(t *testing.T) {
	data := &api.SheetData{
		Values: [][]interface{}{
			{"TC", "Desc"},
			{"TC-001", "test1"},
		},
	}
	stats := computeTestStats(data)
	// Row has len(row)=2 which is not > 3, so it's skipped entirely
	if stats["total"] != 0 {
		t.Errorf("total = %d, want 0 (row too short to have status)", stats["total"])
	}
}

// --- computeSpecStats edge cases ---

func TestComputeSpecStats_Nil(t *testing.T) {
	stats := computeSpecStats(nil)
	if stats["total"] != 0 {
		t.Errorf("nil data should return total=0, got %d", stats["total"])
	}
}

func TestComputeSpecStats_EmptyValues(t *testing.T) {
	data := &api.SheetData{Values: [][]interface{}{}}
	stats := computeSpecStats(data)
	if stats["total"] != 0 {
		t.Errorf("empty values should return total=0, got %d", stats["total"])
	}
}

func TestComputeSpecStats_HeaderOnly(t *testing.T) {
	data := &api.SheetData{
		Values: [][]interface{}{
			{"Feature", "Date", "Agent", "Status"},
		},
	}
	stats := computeSpecStats(data)
	if stats["total"] != 0 {
		t.Errorf("header-only should have total=0, got %d", stats["total"])
	}
}

// --- computeSprintStats edge cases ---

func TestComputeSprintStats_Nil(t *testing.T) {
	stats := computeSprintStats(nil)
	if stats["total"] != 0 {
		t.Errorf("nil data should return total=0, got %d", stats["total"])
	}
}

func TestComputeSprintStats_EmptyValues(t *testing.T) {
	data := &api.SheetData{Values: [][]interface{}{}}
	stats := computeSprintStats(data)
	if stats["total"] != 0 {
		t.Errorf("empty values should return total=0, got %d", stats["total"])
	}
}

func TestComputeSprintStats_HeaderOnly(t *testing.T) {
	data := &api.SheetData{
		Values: [][]interface{}{
			{"Title", "Assignee", "Priority", "Status"},
		},
	}
	stats := computeSprintStats(data)
	if stats["total"] != 0 {
		t.Errorf("header-only should have total=0, got %d", stats["total"])
	}
}

// --- safeCol edge cases ---

func TestSafeCol_NilRow(t *testing.T) {
	got := safeCol(nil, 0)
	if got != "" {
		t.Errorf("safeCol(nil, 0) = %q, want empty", got)
	}
}

func TestSafeCol_EmptyRow(t *testing.T) {
	got := safeCol([]interface{}{}, 0)
	if got != "" {
		t.Errorf("safeCol([], 0) = %q, want empty", got)
	}
}

func TestSafeCol_NonStringValue(t *testing.T) {
	row := []interface{}{42, true, nil}
	got := safeCol(row, 0)
	if got != "42" {
		t.Errorf("safeCol(42) = %q, want '42'", got)
	}
	got = safeCol(row, 1)
	if got != "true" {
		t.Errorf("safeCol(true) = %q, want 'true'", got)
	}
	// fmt.Sprintf("%v", nil) produces "<nil>"
	got = safeCol(row, 2)
	if got != "<nil>" {
		t.Errorf("safeCol(nil) = %q, want '<nil>'", got)
	}
}

func TestSafeCol_LargeIndex(t *testing.T) {
	row := []interface{}{"a"}
	got := safeCol(row, 100)
	if got != "" {
		t.Errorf("safeCol(row, 100) = %q, want empty", got)
	}
}

// --- formatStandupText additional edge cases ---

func TestFormatStandupText_EmailDigestWithSummary(t *testing.T) {
	r := &StandupResult{
		Date:        "2026-03-25",
		EmailDigest: &DigestSection{Summary: "3 urgent emails from boss"},
	}
	text := formatStandupText(r)
	if !strContains(text, "3 urgent emails from boss") {
		t.Error("should contain email summary")
	}
	if !strContains(text, "Email") {
		t.Error("should contain Email section header")
	}
}

func TestFormatStandupText_EmailDigestEmptySummary(t *testing.T) {
	r := &StandupResult{
		Date:        "2026-03-25",
		EmailDigest: &DigestSection{Summary: ""},
	}
	text := formatStandupText(r)
	if strContains(text, "## Email") {
		t.Error("empty summary should not show Email section")
	}
}

func TestFormatStandupText_TasksWithStatus(t *testing.T) {
	r := &StandupResult{
		Date: "2026-03-25",
		Tasks: &TasksSection{
			Count: 2,
			Tasks: []api.TaskItem{
				{Title: "Code review", Status: "needsAction"},
				{Title: "Write docs", Status: "completed"},
			},
		},
	}
	text := formatStandupText(r)
	if !strContains(text, "[needsAction] Code review") {
		t.Error("missing task with status")
	}
	if !strContains(text, "[completed] Write docs") {
		t.Error("missing completed task")
	}
}

// --- Dispatch additional edge cases ---

func TestDispatch_AllFail(t *testing.T) {
	actions := []Action{
		{Name: "fail1", Description: "will fail", Fn: func(ctx context.Context) (interface{}, error) { return nil, errors.New("err1") }},
		{Name: "fail2", Description: "will also fail", Fn: func(ctx context.Context) (interface{}, error) { return nil, errors.New("err2") }},
	}
	result, err := Dispatch(context.Background(), actions, ExecuteOpts{Execute: true, NoInput: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Executed {
		t.Fatal("should be executed")
	}
	for _, ar := range result.Actions {
		if ar.Success {
			t.Errorf("action %s should have failed", ar.Name)
		}
		if ar.Error == "" {
			t.Errorf("action %s should have error message", ar.Name)
		}
	}
}

func TestDispatch_AllSucceed(t *testing.T) {
	actions := []Action{
		{Name: "ok1", Description: "will succeed", Fn: func(ctx context.Context) (interface{}, error) { return "done1", nil }},
		{Name: "ok2", Description: "will also succeed", Fn: func(ctx context.Context) (interface{}, error) { return "done2", nil }},
	}
	result, err := Dispatch(context.Background(), actions, ExecuteOpts{Execute: true, NoInput: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Executed {
		t.Fatal("should be executed")
	}
	for _, ar := range result.Actions {
		if !ar.Success {
			t.Errorf("action %s should have succeeded", ar.Name)
		}
	}
}

// --- RunParallel stress test ---

func TestRunParallel_LargeCount(t *testing.T) {
	const n = 50
	fetchers := make([]Fetcher, n)
	for i := 0; i < n; i++ {
		idx := i
		fetchers[i] = Fetcher{
			Name: "f" + string(rune('0'+idx%10)),
			Fn: func(ctx context.Context) (interface{}, error) {
				return idx, nil
			},
		}
	}
	results := RunParallel(context.Background(), fetchers)
	if len(results) != n {
		t.Fatalf("expected %d results, got %d", n, len(results))
	}
	for i, r := range results {
		if r.Error != nil {
			t.Errorf("result[%d] unexpected error: %v", i, r.Error)
		}
	}
}

// --- FindResult with error values ---

func TestFindResult_WithError(t *testing.T) {
	results := []FetchResult{
		{Name: "ok", Value: "good", Error: nil},
		{Name: "bad", Value: nil, Error: errors.New("boom")},
	}
	r := FindResult(results, "bad")
	if r == nil {
		t.Fatal("expected to find 'bad'")
	}
	if r.Error == nil {
		t.Error("expected error in result")
	}
	if r.Value != nil {
		t.Error("expected nil value for error result")
	}
}

// --- WeeklyDigestResult structure tests ---

func TestWeeklyDigestResult_EmptySections(t *testing.T) {
	r := WeeklyDigestResult{
		Period: "2026-03-18 to 2026-03-25",
	}
	if r.Period == "" {
		t.Error("Period should not be empty")
	}
	if r.EmailStats != nil {
		t.Error("EmailStats should be nil by default")
	}
	if r.MeetingLoad != nil {
		t.Error("MeetingLoad should be nil by default")
	}
	if r.TasksDone != nil {
		t.Error("TasksDone should be nil by default")
	}
}

// --- StandupResult structure ---

func TestStandupResult_Sections(t *testing.T) {
	r := StandupResult{Date: "2026-03-25"}
	if r.Date != "2026-03-25" {
		t.Errorf("Date = %q, want 2026-03-25", r.Date)
	}
	if r.EmailDigest != nil || r.Calendar != nil || r.Tasks != nil || r.GitChanges != nil {
		t.Error("sections should be nil by default")
	}
}

// --- ActionResult structure ---

func TestActionResult_SuccessFields(t *testing.T) {
	ar := ActionResult{Name: "test", Success: true, Result: "done"}
	if ar.Name != "test" {
		t.Errorf("Name = %q", ar.Name)
	}
	if !ar.Success {
		t.Error("should be successful")
	}
	if ar.Error != "" {
		t.Error("error should be empty for success")
	}
}

func TestActionResult_FailureFields(t *testing.T) {
	ar := ActionResult{Name: "test", Success: false, Error: "boom"}
	if ar.Success {
		t.Error("should not be successful")
	}
	if ar.Error != "boom" {
		t.Errorf("Error = %q, want boom", ar.Error)
	}
	if ar.Result != nil {
		t.Error("result should be nil for failure")
	}
}

// --- ExecuteResult structure ---

func TestExecuteResult_CancelledFields(t *testing.T) {
	r := ExecuteResult{Cancelled: true, Reason: "user declined"}
	if !r.Cancelled {
		t.Error("should be cancelled")
	}
	if r.Executed {
		t.Error("should not be executed when cancelled")
	}
	if r.Reason != "user declined" {
		t.Errorf("Reason = %q", r.Reason)
	}
}
