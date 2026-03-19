package workflow

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/redredchen01/gwx/internal/api"
)

// StandupResult is the output of RunStandup.
type StandupResult struct {
	Date        string           `json:"date"`
	EmailDigest *DigestSection   `json:"email_digest"`
	Calendar    *CalendarSection `json:"calendar"`
	Tasks       *TasksSection    `json:"tasks"`
	GitChanges  *GitSection      `json:"git_changes"`
	Execute     *ExecuteResult   `json:"execute,omitempty"`
}

// DigestSection holds email digest results.
type DigestSection struct {
	TotalMessages int              `json:"total_messages"`
	TotalUnread   int              `json:"total_unread"`
	Groups        []api.DigestGroup `json:"groups,omitempty"`
	Summary       string           `json:"summary"`
	Error         string           `json:"error,omitempty"`
}

// CalendarSection holds calendar event results.
type CalendarSection struct {
	Count  int                `json:"count"`
	Events []api.EventSummary `json:"events,omitempty"`
	Error  string             `json:"error,omitempty"`
}

// TasksSection holds task results.
type TasksSection struct {
	Count int            `json:"count"`
	Tasks []api.TaskItem `json:"tasks,omitempty"`
	Error string         `json:"error,omitempty"`
}

// GitSection holds git log results.
type GitSection struct {
	Commits []string `json:"commits"`
	Error   string   `json:"error,omitempty"`
}

// StandupOpts configures the standup workflow.
type StandupOpts struct {
	Days    int
	Execute bool
	NoInput bool
	IsMCP   bool
	Push    string // "chat:spaces/XXX" or "email:addr@example.com"
}

// RunStandup aggregates Git + Gmail + Calendar + Tasks data.
func RunStandup(ctx context.Context, client *api.Client, opts StandupOpts) (*StandupResult, error) {
	days := opts.Days
	if days <= 0 {
		days = 1
	}

	result := &StandupResult{
		Date: time.Now().Format("2006-01-02"),
	}

	// Parallel fetch
	fetchers := []Fetcher{
		{Name: "email_digest", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewGmailService(client)
			return svc.DigestMessages(ctx, 30, false)
		}},
		{Name: "calendar", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewCalendarService(client)
			return svc.Agenda(ctx, 1)
		}},
		{Name: "tasks", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewTasksService(client)
			lists, err := svc.ListTaskLists(ctx)
			if err != nil {
				return nil, err
			}
			var allTasks []api.TaskItem
			for _, l := range lists {
				tasks, err := svc.ListTasks(ctx, l.ID, false)
				if err != nil {
					continue
				}
				allTasks = append(allTasks, tasks...)
			}
			return allTasks, nil
		}},
		{Name: "git", Fn: func(ctx context.Context) (interface{}, error) {
			since := fmt.Sprintf("--since=%d days ago", days)
			cmd := exec.CommandContext(ctx, "git", "log", "--oneline", since)
			out, err := cmd.Output()
			if err != nil {
				return []string{}, nil // non-git repo is OK
			}
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(lines) == 1 && lines[0] == "" {
				return []string{}, nil
			}
			return lines, nil
		}},
	}

	fetchResults := RunParallel(ctx, fetchers)

	// Map results
	if r := FindResult(fetchResults, "email_digest"); r != nil {
		if r.Error != nil {
			result.EmailDigest = &DigestSection{Error: r.Error.Error()}
		} else if dr, ok := r.Value.(*api.DigestResult); ok {
			result.EmailDigest = &DigestSection{
				TotalMessages: dr.TotalMessages,
				TotalUnread:   dr.TotalUnread,
				Groups:        dr.Groups,
				Summary:       dr.Summary,
			}
		}
	}

	if r := FindResult(fetchResults, "calendar"); r != nil {
		if r.Error != nil {
			result.Calendar = &CalendarSection{Error: r.Error.Error()}
		} else if events, ok := r.Value.([]api.EventSummary); ok {
			result.Calendar = &CalendarSection{Count: len(events), Events: events}
		}
	}

	if r := FindResult(fetchResults, "tasks"); r != nil {
		if r.Error != nil {
			result.Tasks = &TasksSection{Error: r.Error.Error()}
		} else if tasks, ok := r.Value.([]api.TaskItem); ok {
			result.Tasks = &TasksSection{Count: len(tasks), Tasks: tasks}
		}
	}

	if r := FindResult(fetchResults, "git"); r != nil {
		if r.Error != nil {
			result.GitChanges = &GitSection{Error: r.Error.Error()}
		} else if commits, ok := r.Value.([]string); ok {
			result.GitChanges = &GitSection{Commits: commits}
		}
	}

	// Handle --execute
	if opts.Push != "" {
		var actions []Action
		if strings.HasPrefix(opts.Push, "chat:") {
			spaceName := strings.TrimPrefix(opts.Push, "chat:")
			actions = append(actions, Action{
				Name:        "send_chat",
				Description: fmt.Sprintf("Send standup to Chat space %s", spaceName),
				Fn: func(ctx context.Context) (interface{}, error) {
					svc := api.NewChatService(client)
					text := formatStandupText(result)
					return svc.SendMessage(ctx, spaceName, text)
				},
			})
		} else if strings.HasPrefix(opts.Push, "email:") {
			to := strings.TrimPrefix(opts.Push, "email:")
			actions = append(actions, Action{
				Name:        "send_email",
				Description: fmt.Sprintf("Send standup email to %s", to),
				Fn: func(ctx context.Context) (interface{}, error) {
					svc := api.NewGmailService(client)
					text := formatStandupText(result)
					return svc.SendMessage(ctx, &api.SendInput{
						To:      []string{to},
						Subject: fmt.Sprintf("Daily Standup — %s", result.Date),
						Body:    text,
					})
				},
			})
		}

		if len(actions) > 0 {
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
	}

	return result, nil
}

func formatStandupText(r *StandupResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Daily Standup — %s\n\n", r.Date))

	if r.GitChanges != nil && len(r.GitChanges.Commits) > 0 {
		sb.WriteString("## Git Activity\n")
		for _, c := range r.GitChanges.Commits {
			sb.WriteString(fmt.Sprintf("- %s\n", c))
		}
		sb.WriteString("\n")
	}

	if r.EmailDigest != nil && r.EmailDigest.Summary != "" {
		sb.WriteString(fmt.Sprintf("## Email\n%s\n\n", r.EmailDigest.Summary))
	}

	if r.Calendar != nil && r.Calendar.Count > 0 {
		sb.WriteString("## Meetings Today\n")
		for _, e := range r.Calendar.Events {
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", e.Title, e.Start))
		}
		sb.WriteString("\n")
	}

	if r.Tasks != nil && r.Tasks.Count > 0 {
		sb.WriteString("## Tasks\n")
		for _, t := range r.Tasks.Tasks {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", t.Status, t.Title))
		}
	}

	return sb.String()
}
