package workflow

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/redredchen01/gwx/internal/api"
)

// BugIntakeResult is the output of RunBugIntake.
type BugIntakeResult struct {
	BugID       string              `json:"bug_id,omitempty"`
	RelatedMail *RecentMailSection  `json:"related_mail"`
	RelatedDocs *RelatedDocsSection `json:"related_docs"`
	GitHistory  *GitSection         `json:"git_history"`
	Execute     *ExecuteResult      `json:"execute,omitempty"`
}

// BugIntakeOpts configures the bug-intake workflow.
type BugIntakeOpts struct {
	BugID   string
	After   string // date filter like "2026/03/15"
	Execute bool
	NoInput bool
	IsMCP   bool
}

// RunBugIntake searches for bug-related emails, docs, and git history.
func RunBugIntake(ctx context.Context, client *api.Client, opts BugIntakeOpts) (*BugIntakeResult, error) {
	result := &BugIntakeResult{BugID: opts.BugID}

	// Build Gmail search query
	query := "subject:(bug OR error OR crash OR issue)"
	if opts.After != "" {
		query += fmt.Sprintf(" after:%s", opts.After)
	}
	if opts.BugID != "" {
		query += fmt.Sprintf(" %s", opts.BugID)
	}

	fetchers := []Fetcher{
		{Name: "related_mail", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewGmailService(client)
			msgs, _, err := svc.SearchMessages(ctx, query, 10)
			return msgs, err
		}},
		{Name: "related_docs", Fn: func(ctx context.Context) (interface{}, error) {
			svc := api.NewDriveService(client)
			searchTerm := opts.BugID
			if searchTerm == "" {
				searchTerm = "bug"
			}
			q := "name contains '" + strings.ReplaceAll(searchTerm, "'", "\\'") + "'"
			return svc.SearchFiles(ctx, q, 5)
		}},
		{Name: "git_history", Fn: func(ctx context.Context) (interface{}, error) {
			if opts.BugID == "" {
				return []string{}, nil
			}
			cmd := exec.CommandContext(ctx, "git", "log", "--all", "--oneline", fmt.Sprintf("--grep=%s", opts.BugID))
			out, err := cmd.Output()
			if err != nil {
				return []string{}, nil // non-git repo or no matches is OK
			}
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(lines) == 1 && lines[0] == "" {
				return []string{}, nil
			}
			return lines, nil
		}},
	}

	fetchResults := RunParallel(ctx, fetchers)

	if r := FindResult(fetchResults, "related_mail"); r != nil {
		if r.Error != nil {
			result.RelatedMail = &RecentMailSection{Error: r.Error.Error()}
		} else if msgs, ok := r.Value.([]api.MessageSummary); ok {
			result.RelatedMail = &RecentMailSection{Count: len(msgs), Messages: msgs}
		}
	}

	if r := FindResult(fetchResults, "related_docs"); r != nil {
		if r.Error != nil {
			result.RelatedDocs = &RelatedDocsSection{Error: r.Error.Error()}
		} else if files, ok := r.Value.([]api.FileSummary); ok {
			result.RelatedDocs = &RelatedDocsSection{Count: len(files), Files: files}
		}
	}

	if r := FindResult(fetchResults, "git_history"); r != nil {
		if r.Error != nil {
			result.GitHistory = &GitSection{Error: r.Error.Error()}
		} else if commits, ok := r.Value.([]string); ok {
			result.GitHistory = &GitSection{Commits: commits}
		}
	}

	return result, nil
}
