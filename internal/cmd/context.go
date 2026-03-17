package cmd

import (
	"sync"
	"time"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// ContextCmd gathers all context related to a topic across Gmail, Drive, and Calendar.
type ContextCmd struct {
	Topic string `arg:"" help:"Topic or keyword to gather context for"`
	Days  int    `help:"How many days of calendar events to include" default:"7" short:"d"`
	Limit int    `help:"Max results per service" default:"5" short:"n"`
}

// ContextResult holds aggregated context from multiple services.
type ContextResult struct {
	Topic    string            `json:"topic"`
	Emails   *ContextEmails    `json:"emails"`
	Files    *ContextFiles     `json:"files"`
	Events   *ContextEvents    `json:"events,omitempty"`
	Summary  string            `json:"summary"`
}

type ContextEmails struct {
	Count    int                   `json:"count"`
	Messages []api.MessageSummary  `json:"messages"`
	Error    string                `json:"error,omitempty"`
}

type ContextFiles struct {
	Count int               `json:"count"`
	Files []api.FileSummary  `json:"files"`
	Error string            `json:"error,omitempty"`
}

type ContextEvents struct {
	Count  int                 `json:"count"`
	Events []api.EventSummary  `json:"events"`
	Error  string              `json:"error,omitempty"`
}

func (c *ContextCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "context"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}
	if err := EnsureAuth(rctx, []string{"gmail", "drive", "calendar"}); err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}
	if rctx.DryRun {
		rctx.Printer.Success(map[string]interface{}{"dry_run": "context", "topic": c.Topic})
		return nil
	}

	result := &ContextResult{Topic: c.Topic}
	var wg sync.WaitGroup

	// Gmail search
	wg.Add(1)
	go func() {
		defer wg.Done()
		svc := api.NewGmailService(rctx.APIClient)
		messages, _, err := svc.SearchMessages(rctx.Context, c.Topic, int64(c.Limit))
		if err != nil {
			result.Emails = &ContextEmails{Error: err.Error()}
			return
		}
		result.Emails = &ContextEmails{Count: len(messages), Messages: messages}
	}()

	// Drive search (name + fulltext)
	wg.Add(1)
	go func() {
		defer wg.Done()
		svc := api.NewDriveService(rctx.APIClient)
		query := "fullText contains '" + escapeDriveQuery(c.Topic) + "'"
		files, err := svc.SearchFiles(rctx.Context, query, int64(c.Limit))
		if err != nil {
			// Fallback to name search
			files, err = svc.SearchFiles(rctx.Context, "name contains '"+escapeDriveQuery(c.Topic)+"'", int64(c.Limit))
			if err != nil {
				result.Files = &ContextFiles{Error: err.Error()}
				return
			}
		}
		result.Files = &ContextFiles{Count: len(files), Files: files}
	}()

	// Calendar search (events in the next N days that mention the topic)
	wg.Add(1)
	go func() {
		defer wg.Done()
		svc := api.NewCalendarService(rctx.APIClient)
		now := time.Now()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 0, c.Days)

		events, err := svc.ListEvents(rctx.Context, "primary", start, end, 50)
		if err != nil {
			result.Events = &ContextEvents{Error: err.Error()}
			return
		}

		// Filter events that mention the topic in title or description
		topicLower := toLower(c.Topic)
		var matched []api.EventSummary
		for _, e := range events {
			if containsCI(e.Title, topicLower) || containsCI(e.Description, topicLower) {
				matched = append(matched, e)
			}
		}
		result.Events = &ContextEvents{Count: len(matched), Events: matched}
	}()

	wg.Wait()

	// Generate summary
	emailCount := 0
	fileCount := 0
	eventCount := 0
	if result.Emails != nil {
		emailCount = result.Emails.Count
	}
	if result.Files != nil {
		fileCount = result.Files.Count
	}
	if result.Events != nil {
		eventCount = result.Events.Count
	}
	result.Summary = formatContextSummary(c.Topic, emailCount, fileCount, eventCount)

	rctx.Printer.Success(result)
	return nil
}

func formatContextSummary(topic string, emails, files, events int) string {
	s := "Context for \"" + topic + "\": "
	s += intWord(emails, "email") + ", "
	s += intWord(files, "file") + ", "
	s += intWord(events, "upcoming event") + "."
	return s
}

func intWord(n int, singular string) string {
	if n == 1 {
		return "1 " + singular
	}
	return itoa(n) + " " + singular + "s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func containsCI(s, lowerSubstr string) bool {
	sl := toLower(s)
	if len(sl) < len(lowerSubstr) {
		return false
	}
	for i := 0; i <= len(sl)-len(lowerSubstr); i++ {
		if sl[i:i+len(lowerSubstr)] == lowerSubstr {
			return true
		}
	}
	return false
}
