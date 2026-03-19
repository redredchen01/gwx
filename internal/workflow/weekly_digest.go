package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/redredchen01/gwx/internal/api"
)

// WeeklyDigestResult is the output of RunWeeklyDigest.
type WeeklyDigestResult struct {
	Period      string              `json:"period"`
	EmailStats  *EmailStatsSection  `json:"email_stats"`
	MeetingLoad *MeetingLoadSection `json:"meeting_load"`
	TasksDone   *TasksDoneSection   `json:"tasks_done"`
	Execute     *ExecuteResult      `json:"execute,omitempty"`
}

// EmailStatsSection holds weekly email statistics.
type EmailStatsSection struct {
	TotalMessages int    `json:"total_messages"`
	TotalUnread   int    `json:"total_unread"`
	Summary       string `json:"summary"`
	Error         string `json:"error,omitempty"`
}

// MeetingLoadSection holds weekly meeting statistics.
type MeetingLoadSection struct {
	Count int                `json:"count"`
	Events []api.EventSummary `json:"events,omitempty"`
	Error string              `json:"error,omitempty"`
}

// TasksDoneSection holds completed tasks.
type TasksDoneSection struct {
	Count int            `json:"count"`
	Tasks []api.TaskItem `json:"tasks,omitempty"`
	Error string         `json:"error,omitempty"`
}

// WeeklyDigestOpts configures the weekly-digest workflow.
type WeeklyDigestOpts struct {
	Weeks   int
	Execute bool
	NoInput bool
	IsMCP   bool
}

// RunWeeklyDigest aggregates weekly email, meeting, and task data.
func RunWeeklyDigest(ctx context.Context, client *api.Client, opts WeeklyDigestOpts) (*WeeklyDigestResult, error) {
	weeks := opts.Weeks
	if weeks <= 0 {
		weeks = 1
	}
	days := weeks * 7

	now := time.Now()
	start := now.AddDate(0, 0, -days)
	result := &WeeklyDigestResult{
		Period: fmt.Sprintf("%s to %s", start.Format("2006-01-02"), now.Format("2006-01-02")),
	}

	fetchers := []Fetcher{
		{Name: "email", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewGmailService(client)
			return svc.DigestMessages(ctx, 100, false)
		}},
		{Name: "meetings", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewCalendarService(client)
			end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
			return svc.ListEvents(ctx, "primary", start, end, 200)
		}},
		{Name: "tasks", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewTasksService(client)
			lists, err := svc.ListTaskLists(ctx)
			if err != nil {
				return nil, err
			}
			var done []api.TaskItem
			for _, l := range lists {
				tasks, err := svc.ListTasks(ctx, l.ID, true)
				if err != nil {
					continue
				}
				for _, t := range tasks {
					if t.Status == "completed" {
						done = append(done, t)
					}
				}
			}
			return done, nil
		}},
	}

	fetchResults := RunParallel(ctx, fetchers)

	if r := FindResult(fetchResults, "email"); r != nil {
		if r.Error != nil {
			result.EmailStats = &EmailStatsSection{Error: r.Error.Error()}
		} else if dr, ok := r.Value.(*api.DigestResult); ok {
			result.EmailStats = &EmailStatsSection{
				TotalMessages: dr.TotalMessages,
				TotalUnread:   dr.TotalUnread,
				Summary:       dr.Summary,
			}
		}
	}

	if r := FindResult(fetchResults, "meetings"); r != nil {
		if r.Error != nil {
			result.MeetingLoad = &MeetingLoadSection{Error: r.Error.Error()}
		} else if events, ok := r.Value.([]api.EventSummary); ok {
			result.MeetingLoad = &MeetingLoadSection{Count: len(events), Events: events}
		}
	}

	if r := FindResult(fetchResults, "tasks"); r != nil {
		if r.Error != nil {
			result.TasksDone = &TasksDoneSection{Error: r.Error.Error()}
		} else if tasks, ok := r.Value.([]api.TaskItem); ok {
			result.TasksDone = &TasksDoneSection{Count: len(tasks), Tasks: tasks}
		}
	}

	return result, nil
}
