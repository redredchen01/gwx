package workflow

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/redredchen01/gwx/internal/api"
)

// MeetingPrepResult is the output of RunMeetingPrep.
type MeetingPrepResult struct {
	Meeting     *api.EventSummary   `json:"meeting"`
	Attendees   *AttendeesSection   `json:"attendees"`
	RecentMail  *RecentMailSection  `json:"recent_mail"`
	RelatedDocs *RelatedDocsSection `json:"related_docs"`
	Execute     *ExecuteResult      `json:"execute,omitempty"`
}

// AttendeesSection holds contact info for attendees.
type AttendeesSection struct {
	Count    int                  `json:"count"`
	Contacts []api.ContactSummary `json:"contacts,omitempty"`
	Error    string               `json:"error,omitempty"`
}

// RecentMailSection holds recent email results.
type RecentMailSection struct {
	Count    int                  `json:"count"`
	Messages []api.MessageSummary `json:"messages,omitempty"`
	Error    string               `json:"error,omitempty"`
}

// RelatedDocsSection holds related Drive file results.
type RelatedDocsSection struct {
	Count int               `json:"count"`
	Files []api.FileSummary `json:"files,omitempty"`
	Error string            `json:"error,omitempty"`
}

// MeetingPrepOpts configures the meeting-prep workflow.
type MeetingPrepOpts struct {
	Meeting string
	Days    int
	Execute bool
	NoInput bool
	IsMCP   bool
}

// RunMeetingPrep finds a matching meeting and gathers context.
func RunMeetingPrep(ctx context.Context, client *api.Client, opts MeetingPrepOpts) (*MeetingPrepResult, error) {
	days := opts.Days
	if days <= 0 {
		days = 1
	}

	// Find matching meeting
	calSvc := api.NewCalendarService(client)
	events, err := calSvc.Agenda(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("calendar agenda: %w", err)
	}

	var match *api.EventSummary
	keyword := strings.ToLower(opts.Meeting)
	for i := range events {
		if strings.Contains(strings.ToLower(events[i].Title), keyword) {
			match = &events[i]
			break
		}
	}
	if match == nil {
		return nil, fmt.Errorf("no meeting matching %q found in the next %d day(s)", opts.Meeting, days)
	}

	result := &MeetingPrepResult{Meeting: match}

	// Parallel fetch attendee info, emails, docs
	fetchers := []Fetcher{
		{Name: "attendees", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewContactsService(client)
			contacts := make([]api.ContactSummary, len(match.Attendees))
			var wg sync.WaitGroup
			for i, email := range match.Attendees {
				wg.Add(1)
				go func(idx int, e string) {
					defer wg.Done()
					c, err := svc.SearchContacts(ctx, e, 1)
					if err == nil && len(c) > 0 {
						contacts[idx] = c[0]
					}
				}(i, email)
			}
			wg.Wait()
			// Filter out empty results
			var result []api.ContactSummary
			for _, c := range contacts {
				if len(c.Emails) > 0 || c.Name != "" {
					result = append(result, c)
				}
			}
			return result, nil
		}},
		{Name: "recent_mail", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewGmailService(client)
			query := match.Title
			messages, _, err := svc.SearchMessages(ctx, query, 5)
			return messages, err
		}},
		{Name: "related_docs", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewDriveService(client)
			query := "name contains '" + strings.ReplaceAll(match.Title, "'", "\\'") + "'"
			return svc.SearchFiles(ctx, query, 5)
		}},
	}

	fetchResults := RunParallel(ctx, fetchers)

	if r := FindResult(fetchResults, "attendees"); r != nil {
		if r.Error != nil {
			result.Attendees = &AttendeesSection{Error: r.Error.Error()}
		} else if contacts, ok := r.Value.([]api.ContactSummary); ok {
			result.Attendees = &AttendeesSection{Count: len(contacts), Contacts: contacts}
		}
	}

	if r := FindResult(fetchResults, "recent_mail"); r != nil {
		if r.Error != nil {
			result.RecentMail = &RecentMailSection{Error: r.Error.Error()}
		} else if msgs, ok := r.Value.([]api.MessageSummary); ok {
			result.RecentMail = &RecentMailSection{Count: len(msgs), Messages: msgs}
		}
	}

	if r := FindResult(fetchResults, "related_docs"); r != nil {
		if r.Error != nil {
			result.RelatedDocs = &RelatedDocsSection{Error: r.Error.Error()}
		} else if files, ok := r.Value.([]api.FileSummary); ok {
			result.RelatedDocs = &RelatedDocsSection{Count: len(files), Files: files}
		}
	}

	return result, nil
}
