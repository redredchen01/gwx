package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/redredchen01/gwx/internal/api"
)

// ParallelScheduleResult is the output of RunParallelSchedule.
type ParallelScheduleResult struct {
	Title     string            `json:"title"`
	Duration  string            `json:"duration"`
	Attendees []string          `json:"attendees"`
	Slots     []AttendeeSlots   `json:"slots"`
	Events    []EventPreview    `json:"events,omitempty"`
	Execute   *ExecuteResult    `json:"execute,omitempty"`
}

// AttendeeSlots holds available slots for one attendee.
type AttendeeSlots struct {
	Attendee string              `json:"attendee"`
	Slots    []map[string]string `json:"slots,omitempty"`
	Error    string              `json:"error,omitempty"`
}

// EventPreview previews an event to be created.
type EventPreview struct {
	Attendee string `json:"attendee"`
	Start    string `json:"start"`
	End      string `json:"end"`
}

// ParallelScheduleOpts configures the parallel-schedule workflow.
type ParallelScheduleOpts struct {
	Title     string
	Attendees []string
	Duration  string // e.g. "30m", "1h"
	DaysAhead int
	Execute   bool
	NoInput   bool
	IsMCP     bool
}

// RunParallelSchedule finds free slots and creates 1-on-1 review events.
func RunParallelSchedule(ctx context.Context, client *api.Client, opts ParallelScheduleOpts) (*ParallelScheduleResult, error) {
	dur, err := time.ParseDuration(opts.Duration)
	if err != nil {
		return nil, fmt.Errorf("invalid duration %q: %w", opts.Duration, err)
	}

	daysAhead := opts.DaysAhead
	if daysAhead <= 0 {
		daysAhead = 7
	}

	calSvc := api.NewCalendarService(client)

	result := &ParallelScheduleResult{
		Title:     opts.Title,
		Duration:  opts.Duration,
		Attendees: opts.Attendees,
	}

	// Find slots for each attendee in parallel
	fetchers := make([]Fetcher, len(opts.Attendees))
	for i, attendee := range opts.Attendees {
		att := attendee
		fetchers[i] = Fetcher{
			Name: att,
			Fn: func(ctx context.Context) (interface{}, error) {
				return calSvc.FindSlot(ctx, []string{att}, dur, daysAhead)
			},
		}
	}

	fetchResults := RunParallel(ctx, fetchers)

	var events []EventPreview
	for _, r := range fetchResults {
		as := AttendeeSlots{Attendee: r.Name}
		if r.Error != nil {
			as.Error = fmt.Sprintf("no available slots in the next %d days", daysAhead)
		} else if slots, ok := r.Value.([]map[string]string); ok {
			as.Slots = slots
			// Pick first slot for event preview
			if len(slots) > 0 {
				events = append(events, EventPreview{
					Attendee: r.Name,
					Start:    slots[0]["start"],
					End:      slots[0]["end"],
				})
			}
		}
		result.Slots = append(result.Slots, as)
	}
	result.Events = events

	// Execute: create events
	if opts.Execute && len(events) > 0 {
		var actions []Action
		for _, ev := range events {
			ev := ev
			actions = append(actions, Action{
				Name:        "create_event",
				Description: fmt.Sprintf("Create 1-on-1 with %s at %s", ev.Attendee, ev.Start),
				Fn: func(ctx context.Context) (interface{}, error) {
					return calSvc.CreateEvent(ctx, "primary", &api.EventInput{
						Title:     fmt.Sprintf("%s — %s", opts.Title, ev.Attendee),
						Start:     ev.Start,
						End:       ev.End,
						Attendees: []string{ev.Attendee},
					})
				},
			})
		}

		execResult, err := Dispatch(ctx, actions, ExecuteOpts{
			Execute: opts.Execute,
			NoInput: opts.NoInput,
			IsMCP:   opts.IsMCP,
		})
		if err != nil {
			return nil, err
		}
		result.Execute = execResult
	}

	return result, nil
}
