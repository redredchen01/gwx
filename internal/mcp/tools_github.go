package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/redredchen01/gwx/internal/api"
	"github.com/redredchen01/gwx/internal/auth"
)

type githubProvider struct{}

func (githubProvider) Tools() []Tool {
	return []Tool{
		{
			Name:        "github_repos",
			Description: "List GitHub repositories for the authenticated user. Requires GitHub PAT (gwx github login).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit": {Type: "integer", Description: "Max repositories (default 30)"},
				},
			},
		},
		{
			Name:        "github_issues",
			Description: "List issues for a GitHub repository.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"repo":  {Type: "string", Description: "Repository in 'owner/repo' format"},
					"state": {Type: "string", Description: "Issue state: open, closed, all (default open)"},
					"limit": {Type: "integer", Description: "Max issues (default 30)"},
				},
				Required: []string{"repo"},
			},
		},
		{
			Name:        "github_create_issue",
			Description: "Create a new issue in a GitHub repository. CAUTION: Creates a real issue.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"repo":   {Type: "string", Description: "Repository in 'owner/repo' format"},
					"title":  {Type: "string", Description: "Issue title"},
					"body":   {Type: "string", Description: "Issue body (Markdown)"},
					"labels": {Type: "string", Description: "Labels, comma-separated"},
				},
				Required: []string{"repo", "title"},
			},
		},
		{
			Name:        "github_pulls",
			Description: "List pull requests for a GitHub repository.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"repo":  {Type: "string", Description: "Repository in 'owner/repo' format"},
					"state": {Type: "string", Description: "PR state: open, closed, all (default open)"},
					"limit": {Type: "integer", Description: "Max PRs (default 30)"},
				},
				Required: []string{"repo"},
			},
		},
		{
			Name:        "github_pull",
			Description: "Get details of a single pull request.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"repo":   {Type: "string", Description: "Repository in 'owner/repo' format"},
					"number": {Type: "integer", Description: "Pull request number"},
				},
				Required: []string{"repo", "number"},
			},
		},
		{
			Name:        "github_runs",
			Description: "List recent GitHub Actions workflow runs for a repository.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"repo":  {Type: "string", Description: "Repository in 'owner/repo' format"},
					"limit": {Type: "integer", Description: "Max runs (default 10)"},
				},
				Required: []string{"repo"},
			},
		},
		{
			Name:        "github_notifications",
			Description: "List GitHub notifications for the authenticated user.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit": {Type: "integer", Description: "Max notifications (default 30)"},
				},
			},
		},
	}
}

// loadGitHub loads the GitHub token from keyring and returns a client.
// This is independent of Google auth — GWXHandler.client is not used.
func loadGitHub() (*api.GitHubClient, error) {
	token, err := auth.LoadProviderToken("github", "default")
	if err != nil {
		return nil, fmt.Errorf("not authenticated with GitHub. Run 'gwx github login --token <PAT>' to save your token")
	}
	return api.NewGitHubClient(token), nil
}

// parseRepo splits "owner/repo" into (owner, repo).
func parseRepo(s string) (string, string, error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repository must be in 'owner/repo' format, got %q", s)
	}
	return parts[0], parts[1], nil
}

func (githubProvider) Handlers(h *GWXHandler) map[string]ToolHandler {
	return map[string]ToolHandler{
		"github_repos":         githubRepos,
		"github_issues":        githubIssues,
		"github_create_issue":  githubCreateIssue,
		"github_pulls":         githubPulls,
		"github_pull":          githubPull,
		"github_runs":          githubRuns,
		"github_notifications": githubNotifications,
	}
}

func githubRepos(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	gh, err := loadGitHub()
	if err != nil {
		return nil, err
	}
	repos, err := gh.ListRepos(ctx, intArg(args, "limit", 30))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"repos": repos, "count": len(repos)})
}

func githubIssues(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	gh, err := loadGitHub()
	if err != nil {
		return nil, err
	}
	owner, repo, err := parseRepo(strArg(args, "repo"))
	if err != nil {
		return nil, err
	}
	issues, err := gh.ListIssues(ctx, owner, repo, strArg(args, "state"), intArg(args, "limit", 30))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"repo": strArg(args, "repo"), "issues": issues, "count": len(issues)})
}

func githubCreateIssue(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	gh, err := loadGitHub()
	if err != nil {
		return nil, err
	}
	owner, repo, err := parseRepo(strArg(args, "repo"))
	if err != nil {
		return nil, err
	}
	var labels []string
	if l := strArg(args, "labels"); l != "" {
		for _, s := range strings.Split(l, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				labels = append(labels, s)
			}
		}
	}
	issue, err := gh.CreateIssue(ctx, owner, repo, strArg(args, "title"), strArg(args, "body"), labels)
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"created": true, "issue": issue})
}

func githubPulls(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	gh, err := loadGitHub()
	if err != nil {
		return nil, err
	}
	owner, repo, err := parseRepo(strArg(args, "repo"))
	if err != nil {
		return nil, err
	}
	pulls, err := gh.ListPulls(ctx, owner, repo, strArg(args, "state"), intArg(args, "limit", 30))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"repo": strArg(args, "repo"), "pulls": pulls, "count": len(pulls)})
}

func githubPull(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	gh, err := loadGitHub()
	if err != nil {
		return nil, err
	}
	owner, repo, err := parseRepo(strArg(args, "repo"))
	if err != nil {
		return nil, err
	}
	pr, err := gh.GetPull(ctx, owner, repo, intArg(args, "number", 0))
	if err != nil {
		return nil, err
	}
	return jsonResult(pr)
}

func githubRuns(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	gh, err := loadGitHub()
	if err != nil {
		return nil, err
	}
	owner, repo, err := parseRepo(strArg(args, "repo"))
	if err != nil {
		return nil, err
	}
	runs, err := gh.ListWorkflowRuns(ctx, owner, repo, intArg(args, "limit", 10))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"repo": strArg(args, "repo"), "runs": runs, "count": len(runs)})
}

func githubNotifications(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	gh, err := loadGitHub()
	if err != nil {
		return nil, err
	}
	notifications, err := gh.ListNotifications(ctx, intArg(args, "limit", 30))
	if err != nil {
		return nil, err
	}
	return jsonResult(map[string]interface{}{"notifications": notifications, "count": len(notifications)})
}

func init() { RegisterProvider(githubProvider{}) }
