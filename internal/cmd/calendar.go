package cmd

import (
	"fmt"
	"time"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// CalendarCmd groups Calendar operations.
type CalendarCmd struct {
	Agenda   CalendarAgendaCmd   `cmd:"" help:"Show upcoming events"`
	List     CalendarListCmd     `cmd:"" help:"List events in a date range"`
	Create   CalendarCreateCmd   `cmd:"" help:"Create an event"`
	Update   CalendarUpdateCmd   `cmd:"" help:"Update an event"`
	Delete   CalendarDeleteCmd   `cmd:"" help:"Delete an event"`
	FindSlot CalendarFindSlotCmd `cmd:"find-slot" help:"Find free time slots"`
}

// CalendarAgendaCmd shows today's (or N days) events.
type CalendarAgendaCmd struct {
	Days int `help:"Number of days to show" default:"1" short:"d"`
}

func (c *CalendarAgendaCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "calendar.agenda"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"calendar"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "calendar.agenda", "days": c.Days})
		return nil
	}

	calSvc := api.NewCalendarService(rctx.APIClient)
	events, err := calSvc.Agenda(rctx.Context, c.Days)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"events": events,
		"count":  len(events),
		"days":   c.Days,
	})
	return nil
}

// CalendarListCmd lists events in a date range.
type CalendarListCmd struct {
	From  string `help:"Start date (RFC3339 or YYYY-MM-DD)" required:""`
	To    string `help:"End date (RFC3339 or YYYY-MM-DD)" required:""`
	Limit int64  `help:"Max results" default:"25" short:"n"`
}

func (c *CalendarListCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "calendar.list"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"calendar"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	from, err := parseTime(c.From)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, fmt.Sprintf("invalid --from: %v", err))
	}
	to, err := parseTime(c.To)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, fmt.Sprintf("invalid --to: %v", err))
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "calendar.list", "from": c.From, "to": c.To})
		return nil
	}

	calSvc := api.NewCalendarService(rctx.APIClient)
	events, err := calSvc.ListEvents(rctx.Context, "primary", from, to, c.Limit)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"events": events,
		"count":  len(events),
		"from":   c.From,
		"to":     c.To,
	})
	return nil
}

// CalendarCreateCmd creates an event.
type CalendarCreateCmd struct {
	Title       string   `help:"Event title" required:""`
	Start       string   `help:"Start time (RFC3339 or YYYY-MM-DD)" required:""`
	End         string   `help:"End time (RFC3339 or YYYY-MM-DD)" required:""`
	Location    string   `help:"Location"`
	Description string   `help:"Description"`
	Attendees   []string `help:"Attendee emails"`
	Timezone    string   `help:"Timezone (e.g. Asia/Taipei)" name:"tz"`
}

func (c *CalendarCreateCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "calendar.create"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"calendar"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	input := &api.EventInput{
		Title:       c.Title,
		Start:       c.Start,
		End:         c.End,
		Location:    c.Location,
		Description: c.Description,
		Attendees:   c.Attendees,
		TimeZone:    c.Timezone,
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":   "calendar.create",
			"title":     input.Title,
			"start":     input.Start,
			"end":       input.End,
			"attendees": input.Attendees,
		})
		return nil
	}

	calSvc := api.NewCalendarService(rctx.APIClient)

	// Check for conflicts before creating
	conflicts, _ := calSvc.CheckConflicts(rctx.Context, "primary", input.Start, input.End)

	event, err := calSvc.CreateEvent(rctx.Context, "primary", input)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	result := map[string]interface{}{
		"created": true,
		"event":   event,
	}
	if len(conflicts) > 0 {
		result["conflicts"] = conflicts
		result["conflict_warning"] = fmt.Sprintf("%d conflicting event(s) in this time range", len(conflicts))
	}

	rctx.Printer.Success(result)
	return nil
}

// CalendarUpdateCmd updates an event.
type CalendarUpdateCmd struct {
	EventID     string   `arg:"" help:"Event ID to update"`
	Title       string   `help:"New title"`
	Start       string   `help:"New start time"`
	End         string   `help:"New end time"`
	Location    string   `help:"New location"`
	Description string   `help:"New description"`
	Attendees   []string `help:"New attendee list"`
	Timezone    string   `help:"Timezone" name:"tz"`
}

func (c *CalendarUpdateCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "calendar.update"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"calendar"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	input := &api.EventInput{
		Title:       c.Title,
		Start:       c.Start,
		End:         c.End,
		Location:    c.Location,
		Description: c.Description,
		Attendees:   c.Attendees,
		TimeZone:    c.Timezone,
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "calendar.update", "event_id": c.EventID})
		return nil
	}

	calSvc := api.NewCalendarService(rctx.APIClient)
	event, err := calSvc.UpdateEvent(rctx.Context, "primary", c.EventID, input)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"updated": true,
		"event":   event,
	})
	return nil
}

// CalendarDeleteCmd deletes an event.
type CalendarDeleteCmd struct {
	EventID string `arg:"" help:"Event ID to delete"`
}

func (c *CalendarDeleteCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "calendar.delete"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"calendar"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "calendar.delete", "event_id": c.EventID})
		return nil
	}

	calSvc := api.NewCalendarService(rctx.APIClient)
	if err := calSvc.DeleteEvent(rctx.Context, "primary", c.EventID); err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"deleted":  true,
		"event_id": c.EventID,
	})
	return nil
}

// CalendarFindSlotCmd finds free time slots.
type CalendarFindSlotCmd struct {
	Attendees []string `help:"Attendee emails to check" required:""`
	Duration  string   `help:"Meeting duration (e.g. 30m, 1h)" default:"30m" short:"d"`
	Days      int      `help:"Days ahead to search" default:"3"`
}

func (c *CalendarFindSlotCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "calendar.find-slot"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"calendar"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	duration, err := time.ParseDuration(c.Duration)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, fmt.Sprintf("invalid duration: %v", err))
	}

	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{
			"dry_run":   "calendar.find-slot",
			"attendees": c.Attendees,
			"duration":  c.Duration,
		})
		return nil
	}

	calSvc := api.NewCalendarService(rctx.APIClient)
	slots, err := calSvc.FindSlot(rctx.Context, c.Attendees, duration, c.Days)
	if err != nil {
		return handleAPIError(rctx, err)
	}

	rctx.Printer.Success(map[string]interface{}{
		"slots": slots,
		"count": len(slots),
	})
	return nil
}

func parseTime(s string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try date only
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	// Try relative: "today", "tomorrow"
	now := time.Now()
	switch s {
	case "today":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	case "tomorrow":
		t := now.AddDate(0, 0, 1)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), nil
	}
	return time.Time{}, fmt.Errorf("cannot parse %q (use RFC3339, YYYY-MM-DD, today, or tomorrow)", s)
}
