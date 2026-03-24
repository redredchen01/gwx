package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const githubAPIBase = "https://api.github.com"

// GitHubClient wraps the GitHub REST API using a personal access token.
// Uses raw net/http — no external SDK dependency.
type GitHubClient struct {
	token   string
	baseURL string
	http    *http.Client
}

// NewGitHubClient creates a GitHub API client with the given personal access token.
func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		token:   token,
		baseURL: githubAPIBase,
		http:    &http.Client{},
	}
}

// NewGitHubTestClient creates a GitHub client pointing at a test server.
func NewGitHubTestClient(token string, httpClient *http.Client, baseURL string) *GitHubClient {
	return &GitHubClient{
		token:   token,
		baseURL: baseURL,
		http:    httpClient,
	}
}

// do executes an authenticated GitHub API request and decodes the JSON response.
func (g *GitHubClient) do(ctx context.Context, method, path string, body io.Reader, result interface{}) error {
	url := g.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("github: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := g.http.Do(req)
	if err != nil {
		return fmt.Errorf("github: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("github: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return &GitHubAPIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("github: decode response: %w", err)
		}
	}
	return nil
}

// GitHubAPIError represents an error response from the GitHub API.
type GitHubAPIError struct {
	StatusCode int
	Body       string
}

func (e *GitHubAPIError) Error() string {
	return fmt.Sprintf("GitHub API error (HTTP %d): %s", e.StatusCode, e.Body)
}

// --- Repository operations ---

// ListRepos lists repositories for the authenticated user.
func (g *GitHubClient) ListRepos(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 30
	}
	path := "/user/repos?sort=updated&per_page=" + strconv.Itoa(limit)
	var repos []map[string]interface{}
	if err := g.do(ctx, "GET", path, nil, &repos); err != nil {
		return nil, err
	}
	// Return only essential fields to keep output manageable
	result := make([]map[string]interface{}, 0, len(repos))
	for _, r := range repos {
		result = append(result, map[string]interface{}{
			"full_name":   r["full_name"],
			"description": r["description"],
			"private":     r["private"],
			"html_url":    r["html_url"],
			"language":    r["language"],
			"updated_at":  r["updated_at"],
			"open_issues": r["open_issues_count"],
			"stars":       r["stargazers_count"],
		})
	}
	return result, nil
}

// GetRepo gets a single repository by owner and name.
func (g *GitHubClient) GetRepo(ctx context.Context, owner, repo string) (map[string]interface{}, error) {
	path := "/repos/" + owner + "/" + repo
	var result map[string]interface{}
	if err := g.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// --- Issues ---

// ListIssues lists issues for a repository.
func (g *GitHubClient) ListIssues(ctx context.Context, owner, repo, state string, limit int) ([]map[string]interface{}, error) {
	if state == "" {
		state = "open"
	}
	if limit <= 0 {
		limit = 30
	}
	path := "/repos/" + owner + "/" + repo + "/issues?state=" + state + "&per_page=" + strconv.Itoa(limit)
	var issues []map[string]interface{}
	if err := g.do(ctx, "GET", path, nil, &issues); err != nil {
		return nil, err
	}
	// Filter out pull requests (GitHub API returns PRs as issues)
	result := make([]map[string]interface{}, 0, len(issues))
	for _, issue := range issues {
		if _, hasPR := issue["pull_request"]; hasPR {
			continue
		}
		result = append(result, map[string]interface{}{
			"number":     issue["number"],
			"title":      issue["title"],
			"state":      issue["state"],
			"user":       extractUser(issue["user"]),
			"labels":     extractLabels(issue["labels"]),
			"created_at": issue["created_at"],
			"updated_at": issue["updated_at"],
			"html_url":   issue["html_url"],
			"comments":   issue["comments"],
		})
	}
	return result, nil
}

// CreateIssue creates a new issue in a repository.
func (g *GitHubClient) CreateIssue(ctx context.Context, owner, repo, title, body string, labels []string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"title": title,
		"body":  body,
	}
	if len(labels) > 0 {
		payload["labels"] = labels
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("github: marshal issue: %w", err)
	}

	path := "/repos/" + owner + "/" + repo + "/issues"
	var result map[string]interface{}
	if err := g.do(ctx, "POST", path, strings.NewReader(string(jsonBody)), &result); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"number":     result["number"],
		"title":      result["title"],
		"html_url":   result["html_url"],
		"created_at": result["created_at"],
	}, nil
}

// --- Pull Requests ---

// ListPulls lists pull requests for a repository.
func (g *GitHubClient) ListPulls(ctx context.Context, owner, repo, state string, limit int) ([]map[string]interface{}, error) {
	if state == "" {
		state = "open"
	}
	if limit <= 0 {
		limit = 30
	}
	path := "/repos/" + owner + "/" + repo + "/pulls?state=" + state + "&per_page=" + strconv.Itoa(limit)
	var pulls []map[string]interface{}
	if err := g.do(ctx, "GET", path, nil, &pulls); err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(pulls))
	for _, pr := range pulls {
		entry := map[string]interface{}{
			"number":     pr["number"],
			"title":      pr["title"],
			"state":      pr["state"],
			"user":       extractUser(pr["user"]),
			"created_at": pr["created_at"],
			"updated_at": pr["updated_at"],
			"html_url":   pr["html_url"],
			"draft":      pr["draft"],
			"merged":     pr["merged_at"] != nil,
		}
		if head, ok := pr["head"].(map[string]interface{}); ok {
			entry["head_ref"] = head["ref"]
		}
		if base, ok := pr["base"].(map[string]interface{}); ok {
			entry["base_ref"] = base["ref"]
		}
		result = append(result, entry)
	}
	return result, nil
}

// GetPull gets a single pull request by number.
func (g *GitHubClient) GetPull(ctx context.Context, owner, repo string, number int) (map[string]interface{}, error) {
	path := "/repos/" + owner + "/" + repo + "/pulls/" + strconv.Itoa(number)
	var result map[string]interface{}
	if err := g.do(ctx, "GET", path, nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// --- Actions / Workflow Runs ---

// ListWorkflowRuns lists recent workflow runs for a repository.
func (g *GitHubClient) ListWorkflowRuns(ctx context.Context, owner, repo string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 10
	}
	path := "/repos/" + owner + "/" + repo + "/actions/runs?per_page=" + strconv.Itoa(limit)
	var resp map[string]interface{}
	if err := g.do(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}

	runs, ok := resp["workflow_runs"].([]interface{})
	if !ok {
		return nil, nil
	}

	result := make([]map[string]interface{}, 0, len(runs))
	for _, r := range runs {
		run, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		entry := map[string]interface{}{
			"id":           run["id"],
			"name":         run["name"],
			"status":       run["status"],
			"conclusion":   run["conclusion"],
			"head_branch":  run["head_branch"],
			"event":        run["event"],
			"created_at":   run["created_at"],
			"updated_at":   run["updated_at"],
			"html_url":     run["html_url"],
			"run_number":   run["run_number"],
		}
		result = append(result, entry)
	}
	return result, nil
}

// --- Notifications ---

// ListNotifications lists notifications for the authenticated user.
func (g *GitHubClient) ListNotifications(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 30
	}
	path := "/notifications?per_page=" + strconv.Itoa(limit)
	var notifications []map[string]interface{}
	if err := g.do(ctx, "GET", path, nil, &notifications); err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(notifications))
	for _, n := range notifications {
		entry := map[string]interface{}{
			"id":         n["id"],
			"reason":     n["reason"],
			"unread":     n["unread"],
			"updated_at": n["updated_at"],
		}
		if subj, ok := n["subject"].(map[string]interface{}); ok {
			entry["title"] = subj["title"]
			entry["type"] = subj["type"]
		}
		if repo, ok := n["repository"].(map[string]interface{}); ok {
			entry["repo"] = repo["full_name"]
		}
		result = append(result, entry)
	}
	return result, nil
}

// --- Helpers ---

func extractUser(v interface{}) string {
	if m, ok := v.(map[string]interface{}); ok {
		if login, ok := m["login"].(string); ok {
			return login
		}
	}
	return ""
}

func extractLabels(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	labels := make([]string, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok {
				labels = append(labels, name)
			}
		}
	}
	return labels
}
