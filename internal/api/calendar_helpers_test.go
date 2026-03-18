package api

import (
	"testing"

	"google.golang.org/api/calendar/v3"
)

// ---------------------------------------------------------------------------
// parseEventDateTime
// ---------------------------------------------------------------------------

func TestParseEventDateTime_DateOnly(t *testing.T) {
	edt := parseEventDateTime("2026-03-18", "")
	if edt.Date != "2026-03-18" {
		t.Errorf("Date = %q, want '2026-03-18'", edt.Date)
	}
	if edt.DateTime != "" {
		t.Errorf("DateTime should be empty for date-only input, got %q", edt.DateTime)
	}
}

func TestParseEventDateTime_DateTimeWithTZ(t *testing.T) {
	edt := parseEventDateTime("2026-03-18T10:00:00+08:00", "Asia/Taipei")
	if edt.DateTime != "2026-03-18T10:00:00+08:00" {
		t.Errorf("DateTime = %q, want '2026-03-18T10:00:00+08:00'", edt.DateTime)
	}
	if edt.TimeZone != "Asia/Taipei" {
		t.Errorf("TimeZone = %q, want 'Asia/Taipei'", edt.TimeZone)
	}
	if edt.Date != "" {
		t.Errorf("Date should be empty for datetime input, got %q", edt.Date)
	}
}

func TestParseEventDateTime_DateTimeNoTZ(t *testing.T) {
	edt := parseEventDateTime("2026-03-18T10:00:00Z", "")
	if edt.DateTime != "2026-03-18T10:00:00Z" {
		t.Errorf("DateTime = %q, want '2026-03-18T10:00:00Z'", edt.DateTime)
	}
	if edt.TimeZone != "" {
		t.Errorf("TimeZone should be empty when tz is '', got %q", edt.TimeZone)
	}
}

// ---------------------------------------------------------------------------
// eventToSummary
// ---------------------------------------------------------------------------

func TestEventToSummary_BasicFields(t *testing.T) {
	e := &calendar.Event{
		Id:          "evt-001",
		Summary:     "Team Meeting",
		Location:    "Room A",
		Description: "Weekly sync",
		Status:      "confirmed",
		HangoutLink: "https://meet.google.com/abc",
		HtmlLink:    "https://calendar.google.com/abc",
	}
	s := eventToSummary(e)

	if s.ID != "evt-001" {
		t.Errorf("ID = %q, want 'evt-001'", s.ID)
	}
	if s.Title != "Team Meeting" {
		t.Errorf("Title = %q, want 'Team Meeting'", s.Title)
	}
	if s.Location != "Room A" {
		t.Errorf("Location = %q, want 'Room A'", s.Location)
	}
	if s.Status != "confirmed" {
		t.Errorf("Status = %q, want 'confirmed'", s.Status)
	}
}

func TestEventToSummary_AllDayEvent(t *testing.T) {
	e := &calendar.Event{
		Id:      "evt-002",
		Summary: "Holiday",
		Start:   &calendar.EventDateTime{Date: "2026-03-18"},
		End:     &calendar.EventDateTime{Date: "2026-03-19"},
	}
	s := eventToSummary(e)

	if !s.AllDay {
		t.Error("AllDay should be true for date-only event")
	}
	if s.Start != "2026-03-18" {
		t.Errorf("Start = %q, want '2026-03-18'", s.Start)
	}
	if s.End != "2026-03-19" {
		t.Errorf("End = %q, want '2026-03-19'", s.End)
	}
}

func TestEventToSummary_TimedEvent(t *testing.T) {
	e := &calendar.Event{
		Id:      "evt-003",
		Summary: "Call",
		Start:   &calendar.EventDateTime{DateTime: "2026-03-18T10:00:00+08:00"},
		End:     &calendar.EventDateTime{DateTime: "2026-03-18T11:00:00+08:00"},
	}
	s := eventToSummary(e)

	if s.AllDay {
		t.Error("AllDay should be false for timed event")
	}
	if s.Start != "2026-03-18T10:00:00+08:00" {
		t.Errorf("Start = %q", s.Start)
	}
}

func TestEventToSummary_WithOrganizer(t *testing.T) {
	e := &calendar.Event{
		Id:        "evt-004",
		Summary:   "Review",
		Organizer: &calendar.EventOrganizer{Email: "boss@example.com"},
	}
	s := eventToSummary(e)
	if s.Organizer != "boss@example.com" {
		t.Errorf("Organizer = %q, want 'boss@example.com'", s.Organizer)
	}
}

func TestEventToSummary_WithAttendees(t *testing.T) {
	e := &calendar.Event{
		Id:      "evt-005",
		Summary: "Standup",
		Attendees: []*calendar.EventAttendee{
			{Email: "a@example.com"},
			{Email: "b@example.com"},
		},
	}
	s := eventToSummary(e)
	if len(s.Attendees) != 2 {
		t.Errorf("Attendees len = %d, want 2", len(s.Attendees))
	}
	if s.Attendees[0] != "a@example.com" {
		t.Errorf("Attendees[0] = %q, want 'a@example.com'", s.Attendees[0])
	}
}

func TestEventToSummary_NilStartEnd(t *testing.T) {
	e := &calendar.Event{Id: "evt-006", Summary: "No time"}
	s := eventToSummary(e)
	if s.Start != "" || s.End != "" {
		t.Errorf("nil Start/End should produce empty strings, got start=%q end=%q", s.Start, s.End)
	}
}

func TestEventToSummary_NilOrganizer(t *testing.T) {
	e := &calendar.Event{Id: "evt-007", Summary: "No organizer"}
	s := eventToSummary(e)
	if s.Organizer != "" {
		t.Errorf("nil Organizer should produce empty string, got %q", s.Organizer)
	}
}
