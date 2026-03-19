package workflow

import (
	"context"
	"strings"
	"time"

	"github.com/redredchen01/gwx/internal/api"
)

// ContextBoostResult is the output of RunContextBoost.
type ContextBoostResult struct {
	Topic    string              `json:"topic"`
	Emails   *RecentMailSection  `json:"emails"`
	Files    *RelatedDocsSection `json:"files"`
	Events   *CalendarSection    `json:"events"`
	Contacts *AttendeesSection   `json:"contacts"`
}

// ContextBoostOpts configures the context-boost workflow.
type ContextBoostOpts struct {
	Topic string
	Days  int
	Limit int
	IsMCP bool
}

// RunContextBoost gathers deep context for a topic across Gmail, Drive, Calendar, and Contacts.
func RunContextBoost(ctx context.Context, client *api.Client, opts ContextBoostOpts) (*ContextBoostResult, error) {
	days := opts.Days
	if days <= 0 {
		days = 14
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}

	result := &ContextBoostResult{Topic: opts.Topic}

	fetchers := []Fetcher{
		{Name: "emails", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewGmailService(client)
			msgs, _, err := svc.SearchMessages(ctx, opts.Topic, int64(limit))
			return msgs, err
		}},
		{Name: "files", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewDriveService(client)
			query := "fullText contains '" + strings.ReplaceAll(opts.Topic, "'", "\\'") + "'"
			return svc.SearchFiles(ctx, query, int64(limit))
		}},
		{Name: "events", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewCalendarService(client)
			now := time.Now()
			start := now.AddDate(0, 0, -days)
			end := now.AddDate(0, 0, days)
			return svc.ListEvents(ctx, "primary", start, end, 50)
		}},
		{Name: "contacts", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewContactsService(client)
			return svc.SearchContacts(ctx, opts.Topic, limit)
		}},
	}

	fetchResults := RunParallel(ctx, fetchers)

	if r := FindResult(fetchResults, "emails"); r != nil {
		if r.Error != nil {
			result.Emails = &RecentMailSection{Error: r.Error.Error()}
		} else if msgs, ok := r.Value.([]api.MessageSummary); ok {
			result.Emails = &RecentMailSection{Count: len(msgs), Messages: msgs}
		}
	}

	if r := FindResult(fetchResults, "files"); r != nil {
		if r.Error != nil {
			result.Files = &RelatedDocsSection{Error: r.Error.Error()}
		} else if files, ok := r.Value.([]api.FileSummary); ok {
			result.Files = &RelatedDocsSection{Count: len(files), Files: files}
		}
	}

	if r := FindResult(fetchResults, "events"); r != nil {
		if r.Error != nil {
			result.Events = &CalendarSection{Error: r.Error.Error()}
		} else if events, ok := r.Value.([]api.EventSummary); ok {
			result.Events = &CalendarSection{Count: len(events), Events: events}
		}
	}

	if r := FindResult(fetchResults, "contacts"); r != nil {
		if r.Error != nil {
			result.Contacts = &AttendeesSection{Error: r.Error.Error()}
		} else if contacts, ok := r.Value.([]api.ContactSummary); ok {
			result.Contacts = &AttendeesSection{Count: len(contacts), Contacts: contacts}
		}
	}

	return result, nil
}
