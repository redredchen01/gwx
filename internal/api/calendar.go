package api

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/calendar/v3"
)

func (cs *CalendarService) service(ctx context.Context) (*calendar.Service, error) {
	svc, err := cs.client.GetOrCreateService("calendar:v3", func() (any, error) {
		opts, err := cs.client.ClientOptions(ctx, "calendar")
		if err != nil {
			return nil, err
		}
		return calendar.NewService(ctx, opts...)
	})
	if err != nil {
		return nil, fmt.Errorf("create calendar service: %w", err)
	}
	return svc.(*calendar.Service), nil
}

// CalendarService wraps Calendar API operations.
type CalendarService struct {
	client *Client
}

// NewCalendarService creates a Calendar service wrapper.
func NewCalendarService(client *Client) *CalendarService {
	return &CalendarService{client: client}
}

// EventSummary is a simplified event for agent consumption.
type EventSummary struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Start       string   `json:"start"`
	End         string   `json:"end"`
	Location    string   `json:"location,omitempty"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"`
	Organizer   string   `json:"organizer,omitempty"`
	Attendees   []string `json:"attendees,omitempty"`
	HangoutLink string   `json:"hangout_link,omitempty"`
	HTMLLink    string   `json:"html_link,omitempty"`
	AllDay      bool     `json:"all_day"`
}

// EventInput holds parameters for creating/updating events.
type EventInput struct {
	Title       string   `json:"title"`
	Start       string   `json:"start"` // RFC3339 or date
	End         string   `json:"end"`   // RFC3339 or date
	Location    string   `json:"location,omitempty"`
	Description string   `json:"description,omitempty"`
	Attendees   []string `json:"attendees,omitempty"`
	TimeZone    string   `json:"timezone,omitempty"`
}

// ListEvents lists events in a time range.
func (cs *CalendarService) ListEvents(ctx context.Context, calendarID string, timeMin, timeMax time.Time, maxResults int64) ([]EventSummary, error) {
	if !cs.client.NoCache {
		key := cacheKey("calendar", "ListEvents", calendarID, timeMin.Unix(), timeMax.Unix(), maxResults)
		if cached, ok := cs.client.cache.Get(key); ok {
			return cached.([]EventSummary), nil
		}
	}

	if err := cs.client.WaitRate(ctx, "calendar"); err != nil {
		return nil, err
	}

	svc, err := cs.service(ctx)
	if err != nil {
		return nil, err
	}

	if calendarID == "" {
		calendarID = "primary"
	}

	call := svc.Events.List(calendarID).
		TimeMin(timeMin.Format(time.RFC3339)).
		TimeMax(timeMax.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime")

	if maxResults > 0 {
		call = call.MaxResults(maxResults)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	var events []EventSummary
	for _, e := range resp.Items {
		events = append(events, eventToSummary(e))
	}
	if !cs.client.NoCache {
		key := cacheKey("calendar", "ListEvents", calendarID, timeMin.Unix(), timeMax.Unix(), maxResults)
		cs.client.cache.Set(key, events, 2*time.Minute)
	}
	return events, nil
}

// Agenda returns today's events (or N days ahead).
func (cs *CalendarService) Agenda(ctx context.Context, days int) ([]EventSummary, error) {
	if !cs.client.NoCache {
		key := cacheKey("calendar", "Agenda", days)
		if cached, ok := cs.client.cache.Get(key); ok {
			return cached.([]EventSummary), nil
		}
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, days)
	events, err := cs.ListEvents(ctx, "primary", start, end, 50)
	if err != nil {
		return nil, err
	}
	if !cs.client.NoCache {
		key := cacheKey("calendar", "Agenda", days)
		cs.client.cache.Set(key, events, 2*time.Minute)
	}
	return events, nil
}

// ConflictInfo describes a scheduling conflict.
type ConflictInfo struct {
	EventID string `json:"event_id"`
	Title   string `json:"title"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

// CheckConflicts checks if a proposed time range conflicts with existing events.
func (cs *CalendarService) CheckConflicts(ctx context.Context, calendarID, startStr, endStr string) ([]ConflictInfo, error) {
	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		// Try date-only (all-day event)
		if _, err2 := time.Parse("2006-01-02", startStr); err2 == nil {
			return nil, nil // Skip conflict check for all-day events
		}
		return nil, fmt.Errorf("parse start time: %w", err)
	}
	endTime, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		return nil, fmt.Errorf("parse end time: %w", err)
	}

	events, err := cs.ListEvents(ctx, calendarID, startTime.Add(-1*time.Hour), endTime.Add(1*time.Hour), 20)
	if err != nil {
		return nil, err
	}

	var conflicts []ConflictInfo
	for _, e := range events {
		eStart, err1 := time.Parse(time.RFC3339, e.Start)
		eEnd, err2 := time.Parse(time.RFC3339, e.End)
		if err1 != nil || err2 != nil {
			continue
		}
		// Overlap: event starts before proposed ends AND event ends after proposed starts
		if eStart.Before(endTime) && eEnd.After(startTime) {
			conflicts = append(conflicts, ConflictInfo{
				EventID: e.ID,
				Title:   e.Title,
				Start:   e.Start,
				End:     e.End,
			})
		}
	}
	return conflicts, nil
}

// CreateEvent creates a new calendar event.
func (cs *CalendarService) CreateEvent(ctx context.Context, calendarID string, input *EventInput) (*EventSummary, error) {
	if err := cs.client.WaitRate(ctx, "calendar"); err != nil {
		return nil, err
	}

	svc, err := cs.service(ctx)
	if err != nil {
		return nil, err
	}

	if calendarID == "" {
		calendarID = "primary"
	}

	event := &calendar.Event{
		Summary:     input.Title,
		Location:    input.Location,
		Description: input.Description,
		Start:       parseEventDateTime(input.Start, input.TimeZone),
		End:         parseEventDateTime(input.End, input.TimeZone),
	}

	for _, email := range input.Attendees {
		event.Attendees = append(event.Attendees, &calendar.EventAttendee{Email: email})
	}

	created, err := svc.Events.Insert(calendarID, event).Do()
	if err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}

	summary := eventToSummary(created)
	if !cs.client.NoCache {
		cs.client.cache.InvalidatePrefix("calendar:")
	}
	return &summary, nil
}

// UpdateEvent updates an existing event.
func (cs *CalendarService) UpdateEvent(ctx context.Context, calendarID, eventID string, input *EventInput) (*EventSummary, error) {
	if err := cs.client.WaitRate(ctx, "calendar"); err != nil {
		return nil, err
	}

	svc, err := cs.service(ctx)
	if err != nil {
		return nil, err
	}

	if calendarID == "" {
		calendarID = "primary"
	}

	// Fetch existing event
	existing, err := svc.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}

	if input.Title != "" {
		existing.Summary = input.Title
	}
	if input.Location != "" {
		existing.Location = input.Location
	}
	if input.Description != "" {
		existing.Description = input.Description
	}
	if input.Start != "" {
		existing.Start = parseEventDateTime(input.Start, input.TimeZone)
	}
	if input.End != "" {
		existing.End = parseEventDateTime(input.End, input.TimeZone)
	}
	if len(input.Attendees) > 0 {
		existing.Attendees = nil
		for _, email := range input.Attendees {
			existing.Attendees = append(existing.Attendees, &calendar.EventAttendee{Email: email})
		}
	}

	updated, err := svc.Events.Update(calendarID, eventID, existing).Do()
	if err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}

	summary := eventToSummary(updated)
	if !cs.client.NoCache {
		cs.client.cache.InvalidatePrefix("calendar:")
	}
	return &summary, nil
}

// DeleteEvent deletes an event.
func (cs *CalendarService) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	if err := cs.client.WaitRate(ctx, "calendar"); err != nil {
		return err
	}

	svc, err := cs.service(ctx)
	if err != nil {
		return err
	}

	if calendarID == "" {
		calendarID = "primary"
	}

	if err := svc.Events.Delete(calendarID, eventID).Do(); err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	if !cs.client.NoCache {
		cs.client.cache.InvalidatePrefix("calendar:")
	}
	return nil
}

// FindSlot finds free time slots for given attendees.
func (cs *CalendarService) FindSlot(ctx context.Context, attendees []string, duration time.Duration, daysAhead int) ([]map[string]string, error) {
	if err := cs.client.WaitRate(ctx, "calendar"); err != nil {
		return nil, err
	}

	svc, err := cs.service(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	start := now.Truncate(time.Hour).Add(time.Hour) // next full hour
	end := now.AddDate(0, 0, daysAhead)

	var items []*calendar.FreeBusyRequestItem
	for _, email := range attendees {
		items = append(items, &calendar.FreeBusyRequestItem{Id: email})
	}

	fbReq := &calendar.FreeBusyRequest{
		TimeMin: start.Format(time.RFC3339),
		TimeMax: end.Format(time.RFC3339),
		Items:   items,
	}

	resp, err := svc.Freebusy.Query(fbReq).Do()
	if err != nil {
		return nil, fmt.Errorf("freebusy query: %w", err)
	}

	// Collect all busy periods
	type period struct{ start, end time.Time }
	var busyPeriods []period

	for _, cal := range resp.Calendars {
		for _, busy := range cal.Busy {
			s, _ := time.Parse(time.RFC3339, busy.Start)
			e, _ := time.Parse(time.RFC3339, busy.End)
			busyPeriods = append(busyPeriods, period{s, e})
		}
	}

	// Find free slots during business hours (9-18)
	var slots []map[string]string
	cursor := start
	for cursor.Before(end) && len(slots) < 5 {
		hour := cursor.Hour()
		if hour < 9 || hour >= 18 || cursor.Weekday() == time.Saturday || cursor.Weekday() == time.Sunday {
			cursor = cursor.Add(time.Hour)
			continue
		}

		slotEnd := cursor.Add(duration)
		if slotEnd.Hour() > 18 {
			// Skip to next day 9 AM
			cursor = time.Date(cursor.Year(), cursor.Month(), cursor.Day()+1, 9, 0, 0, 0, cursor.Location())
			continue
		}

		free := true
		for _, bp := range busyPeriods {
			if cursor.Before(bp.end) && slotEnd.After(bp.start) {
				free = false
				cursor = bp.end
				break
			}
		}

		if free {
			slots = append(slots, map[string]string{
				"start": cursor.Format(time.RFC3339),
				"end":   slotEnd.Format(time.RFC3339),
			})
			cursor = slotEnd
		}
	}

	return slots, nil
}

func eventToSummary(e *calendar.Event) EventSummary {
	s := EventSummary{
		ID:          e.Id,
		Title:       e.Summary,
		Location:    e.Location,
		Description: e.Description,
		Status:      e.Status,
		HangoutLink: e.HangoutLink,
		HTMLLink:    e.HtmlLink,
	}

	if e.Organizer != nil {
		s.Organizer = e.Organizer.Email
	}

	if e.Start != nil {
		if e.Start.DateTime != "" {
			s.Start = e.Start.DateTime
		} else {
			s.Start = e.Start.Date
			s.AllDay = true
		}
	}
	if e.End != nil {
		if e.End.DateTime != "" {
			s.End = e.End.DateTime
		} else {
			s.End = e.End.Date
		}
	}

	for _, a := range e.Attendees {
		s.Attendees = append(s.Attendees, a.Email)
	}

	return s
}

func parseEventDateTime(input, tz string) *calendar.EventDateTime {
	edt := &calendar.EventDateTime{}

	// If it looks like a date only (YYYY-MM-DD), treat as all-day
	if len(input) == 10 {
		edt.Date = input
	} else {
		edt.DateTime = input
		if tz != "" {
			edt.TimeZone = tz
		}
	}
	return edt
}
