package cmd

import (
	"fmt"
	"strings"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/auth"
	"github.com/redredchen01/gwx/internal/exitcode"
)

// GitHubCmd groups GitHub operations.
// GitHub commands use a personal access token stored separately from Google OAuth.
type GitHubCmd struct {
	Login  GitHubLoginCmd  `cmd:"" help:"Save GitHub personal access token"`
	Logout GitHubLogoutCmd `cmd:"" help:"Remove saved GitHub token"`
	Status GitHubStatusCmd `cmd:"" help:"Check GitHub auth status"`
	Repos  GitHubReposCmd  `cmd:"" help:"List your repositories"`
	Issues GitHubIssuesCmd `cmd:"" help:"List issues for a repository"`
	Pulls  GitHubPullsCmd  `cmd:"" help:"List pull requests for a repository"`
	Pull   GitHubPullCmd   `cmd:"" help:"Get a single pull request"`
	Runs   GitHubRunsCmd   `cmd:"" help:"List workflow runs for a repository"`
	Notify GitHubNotifyCmd `cmd:"" help:"List notifications"`
	Create GitHubCreateCmd `cmd:"" help:"Create GitHub resources"`
}

// GitHubCreateCmd groups creation commands.
type GitHubCreateCmd struct {
	Issue GitHubCreateIssueCmd `cmd:"" help:"Create an issue"`
}

// loadGitHubClient loads the GitHub token from keyring and creates a client.
// Does NOT use Google auth (EnsureAuth) — GitHub auth is independent.
func loadGitHubClient(rctx *RunContext) (*api.GitHubClient, error) {
	token, err := auth.LoadProviderToken("github", rctx.Account)
	if err != nil {
		return nil, fmt.Errorf("not authenticated with GitHub. Run 'gwx github login --token <PAT>' first")
	}
	return api.NewGitHubClient(token), nil
}

// parseOwnerRepo splits "owner/repo" into (owner, repo).
func parseOwnerRepo(s string) (string, string, error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repository must be in 'owner/repo' format, got %q", s)
	}
	return parts[0], parts[1], nil
}

// --- Login ---

// GitHubLoginCmd saves a GitHub personal access token.
type GitHubLoginCmd struct {
	Token string `help:"GitHub personal access token (classic or fine-grained)" required:""`
}

func (c *GitHubLoginCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "github.login"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	if err := auth.SaveProviderToken("github", rctx.Account, c.Token); err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, fmt.Sprintf("save token: %s", err))
	}

	rctx.Printer.Success(map[string]interface{}{
		"provider": "github",
		"status":   "authenticated",
	})
	return nil
}

// --- Logout ---

// GitHubLogoutCmd removes the saved GitHub token.
type GitHubLogoutCmd struct{}

func (c *GitHubLogoutCmd) Run(rctx *RunContext) error {
	if err := auth.DeleteProviderToken("github", rctx.Account); err != nil {
		return rctx.Printer.ErrExit(exitcode.NotFound, "no saved GitHub token")
	}
	rctx.Printer.Success(map[string]string{
		"provider": "github",
		"status":   "logged_out",
	})
	return nil
}

// --- Status ---

// GitHubStatusCmd checks GitHub authentication status.
type GitHubStatusCmd struct{}

func (c *GitHubStatusCmd) Run(rctx *RunContext) error {
	if auth.HasProviderToken("github", rctx.Account) {
		rctx.Printer.Success(map[string]string{
			"provider": "github",
			"status":   "authenticated",
		})
		return nil
	}
	return rctx.Printer.ErrExit(exitcode.AuthRequired, "not authenticated with GitHub. Run 'gwx github login --token <PAT>'")
}

// --- Repos ---

// GitHubReposCmd lists repositories.
type GitHubReposCmd struct {
	Limit int `help:"Max repositories to return" default:"30" short:"n"`
}

func (c *GitHubReposCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "github.repos"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	gh, err := loadGitHubClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	repos, err := gh.ListRepos(rctx.Context, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"repos": repos,
		"count": len(repos),
	})
	return nil
}

// --- Issues ---

// GitHubIssuesCmd lists issues for a repository.
type GitHubIssuesCmd struct {
	Repo  string `arg:"" help:"Repository in 'owner/repo' format"`
	State string `help:"Issue state: open, closed, all" default:"open" enum:"open,closed,all"`
	Limit int    `help:"Max issues to return" default:"30" short:"n"`
}

func (c *GitHubIssuesCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "github.issues"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	owner, repo, err := parseOwnerRepo(c.Repo)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, err.Error())
	}

	gh, err := loadGitHubClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	issues, err := gh.ListIssues(rctx.Context, owner, repo, c.State, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"repo":   c.Repo,
		"state":  c.State,
		"issues": issues,
		"count":  len(issues),
	})
	return nil
}

// --- Pull Requests ---

// GitHubPullsCmd lists pull requests.
type GitHubPullsCmd struct {
	Repo  string `arg:"" help:"Repository in 'owner/repo' format"`
	State string `help:"PR state: open, closed, all" default:"open" enum:"open,closed,all"`
	Limit int    `help:"Max PRs to return" default:"30" short:"n"`
}

func (c *GitHubPullsCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "github.pulls"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	owner, repo, err := parseOwnerRepo(c.Repo)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, err.Error())
	}

	gh, err := loadGitHubClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	pulls, err := gh.ListPulls(rctx.Context, owner, repo, c.State, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"repo":  c.Repo,
		"state": c.State,
		"pulls": pulls,
		"count": len(pulls),
	})
	return nil
}

// GitHubPullCmd gets a single pull request.
type GitHubPullCmd struct {
	Repo   string `arg:"" help:"Repository in 'owner/repo' format"`
	Number int    `arg:"" help:"Pull request number"`
}

func (c *GitHubPullCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "github.pull"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	owner, repo, err := parseOwnerRepo(c.Repo)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, err.Error())
	}

	gh, err := loadGitHubClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	pr, err := gh.GetPull(rctx.Context, owner, repo, c.Number)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(pr)
	return nil
}

// --- Workflow Runs ---

// GitHubRunsCmd lists workflow runs.
type GitHubRunsCmd struct {
	Repo  string `arg:"" help:"Repository in 'owner/repo' format"`
	Limit int    `help:"Max runs to return" default:"10" short:"n"`
}

func (c *GitHubRunsCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "github.runs"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	owner, repo, err := parseOwnerRepo(c.Repo)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, err.Error())
	}

	gh, err := loadGitHubClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	runs, err := gh.ListWorkflowRuns(rctx.Context, owner, repo, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"repo":  c.Repo,
		"runs":  runs,
		"count": len(runs),
	})
	return nil
}

// --- Notifications ---

// GitHubNotifyCmd lists notifications.
type GitHubNotifyCmd struct {
	Limit int `help:"Max notifications to return" default:"30" short:"n"`
}

func (c *GitHubNotifyCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "github.notify"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	gh, err := loadGitHubClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	notifications, err := gh.ListNotifications(rctx.Context, c.Limit)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"notifications": notifications,
		"count":         len(notifications),
	})
	return nil
}

// --- Create Issue ---

// GitHubCreateIssueCmd creates an issue.
type GitHubCreateIssueCmd struct {
	Repo   string   `arg:"" help:"Repository in 'owner/repo' format"`
	Title  string   `help:"Issue title" required:"" short:"t"`
	Body   string   `help:"Issue body" short:"b"`
	Labels []string `help:"Labels to apply (comma-separated)" short:"l"`
}

func (c *GitHubCreateIssueCmd) Run(rctx *RunContext) error {
	if err := CheckAllowlist(rctx, "github.create.issue"); err != nil {
		return rctx.Printer.ErrExit(exitcode.PermissionDenied, err.Error())
	}

	owner, repo, err := parseOwnerRepo(c.Repo)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.InvalidInput, err.Error())
	}

	gh, err := loadGitHubClient(rctx)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.AuthRequired, err.Error())
	}

	issue, err := gh.CreateIssue(rctx.Context, owner, repo, c.Title, c.Body, c.Labels)
	if err != nil {
		return rctx.Printer.ErrExit(exitcode.GeneralError, err.Error())
	}

	rctx.Printer.Success(map[string]interface{}{
		"created": true,
		"issue":   issue,
	})
	return nil
}
