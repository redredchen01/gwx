package workflow

import (
	"strings"
	"testing"

	"github.com/redredchen01/gwx/internal/api"
)

func TestComputeTestStats(t *testing.T) {
	data := &api.SheetData{
		Values: [][]interface{}{
			{"TC", "Desc", "Module", "Status"}, // header
			{"TC-001", "test1", "auth", "pass"},
			{"TC-002", "test2", "api", "fail"},
			{"TC-003", "test3", "auth", "pass"},
			{"TC-004", "test4", "api", "skip"},
		},
	}
	stats := computeTestStats(data)
	if stats["pass"] != 2 {
		t.Errorf("pass = %d, want 2", stats["pass"])
	}
	if stats["fail"] != 1 {
		t.Errorf("fail = %d, want 1", stats["fail"])
	}
	if stats["skip"] != 1 {
		t.Errorf("skip = %d, want 1", stats["skip"])
	}
	if stats["total"] != 4 {
		t.Errorf("total = %d, want 4", stats["total"])
	}
}

func TestComputeTestStats_Nil(t *testing.T) {
	stats := computeTestStats(nil)
	if stats["total"] != 0 {
		t.Errorf("nil data should return total=0, got %d", stats["total"])
	}
}

func TestComputeSpecStats(t *testing.T) {
	data := &api.SheetData{
		Values: [][]interface{}{
			{"Feature", "Date", "Agent", "Status"},
			{"auth", "2026-03-20", "claude", "completed"},
			{"api", "2026-03-20", "claude", "in_progress"},
			{"ui", "2026-03-20", "claude", "completed"},
		},
	}
	stats := computeSpecStats(data)
	if stats["completed"] != 2 {
		t.Errorf("completed = %d, want 2", stats["completed"])
	}
	if stats["in_progress"] != 1 {
		t.Errorf("in_progress = %d, want 1", stats["in_progress"])
	}
	if stats["total"] != 3 {
		t.Errorf("total = %d, want 3", stats["total"])
	}
}

func TestComputeSprintStats(t *testing.T) {
	data := &api.SheetData{
		Values: [][]interface{}{
			{"Title", "Assignee", "Priority", "Status"},
			{"Task 1", "alice", "P0", "done"},
			{"Task 2", "bob", "P1", "todo"},
			{"Task 3", "alice", "P2", "in-progress"},
			{"Task 4", "bob", "P0", "done"},
		},
	}
	stats := computeSprintStats(data)
	if stats["done"] != 2 {
		t.Errorf("done = %d, want 2", stats["done"])
	}
	if stats["todo"] != 1 {
		t.Errorf("todo = %d, want 1", stats["todo"])
	}
	if stats["in-progress"] != 1 {
		t.Errorf("in-progress = %d, want 1", stats["in-progress"])
	}
	if stats["total"] != 4 {
		t.Errorf("total = %d, want 4", stats["total"])
	}
}

func TestSafeCol(t *testing.T) {
	row := []interface{}{"a", "b", "c"}
	tests := []struct {
		idx  int
		want string
	}{
		{0, "a"},
		{1, "b"},
		{2, "c"},
		{3, ""},  // out of bounds
		{-1, ""}, // negative
	}
	for _, tt := range tests {
		got := safeCol(row, tt.idx)
		if got != tt.want {
			t.Errorf("safeCol(row, %d) = %q, want %q", tt.idx, got, tt.want)
		}
	}
}

func TestFormatStandupText(t *testing.T) {
	r := &StandupResult{
		Date: "2026-03-20",
		GitChanges: &GitSection{
			Commits: []string{"feat: add login", "fix: auth bug"},
		},
		Calendar: &CalendarSection{
			Count:  1,
			Events: []api.EventSummary{{Title: "Standup", Start: "09:00"}},
		},
		Tasks: &TasksSection{
			Count: 1,
			Tasks: []api.TaskItem{{Title: "Buy milk", Status: "needsAction"}},
		},
	}

	text := formatStandupText(r)
	if !strings.Contains(text, "# Daily Standup — 2026-03-20") {
		t.Error("missing header")
	}
	if !strings.Contains(text, "feat: add login") {
		t.Error("missing git commit")
	}
	if !strings.Contains(text, "Standup (09:00)") {
		t.Error("missing calendar event")
	}
	if !strings.Contains(text, "Buy milk") {
		t.Error("missing task")
	}
}

func TestFormatStandupText_Empty(t *testing.T) {
	r := &StandupResult{Date: "2026-03-20"}
	text := formatStandupText(r)
	if !strings.Contains(text, "2026-03-20") {
		t.Error("empty standup should still have date")
	}
	if strings.Contains(text, "##") {
		t.Error("empty standup should not have section headers")
	}
}
