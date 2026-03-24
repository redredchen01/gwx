package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newGitHubTestServer creates an httptest.Server and a GitHubClient that
// points at it.  Caller must defer ts.Close().
func newGitHubTestServer(handler http.HandlerFunc) (*httptest.Server, *GitHubClient) {
	ts := httptest.NewServer(handler)
	client := NewGitHubTestClient("test-token-abc", ts.Client(), ts.URL)
	return ts, client
}

func TestGitHubClient_ListRepos(t *testing.T) {
	ts, client := newGitHubTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/repos" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("sort") != "updated" {
			t.Fatal("expected sort=updated")
		}
		resp := []map[string]interface{}{
			{
				"full_name":        "owner/repo1",
				"description":      "first repo",
				"private":          false,
				"html_url":         "https://github.com/owner/repo1",
				"language":         "Go",
				"updated_at":       "2026-01-01T00:00:00Z",
				"open_issues_count": float64(3),
				"stargazers_count": float64(42),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	repos, err := client.ListRepos(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if repos[0]["full_name"] != "owner/repo1" {
		t.Fatalf("expected full_name=owner/repo1, got %v", repos[0]["full_name"])
	}
	if repos[0]["stars"] != float64(42) {
		t.Fatalf("expected stars=42, got %v", repos[0]["stars"])
	}
}

func TestGitHubClient_GetRepo(t *testing.T) {
	ts, client := newGitHubTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/octocat/hello-world" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]interface{}{
			"full_name":   "octocat/hello-world",
			"description": "A test repo",
			"private":     false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	repo, err := client.GetRepo(context.Background(), "octocat", "hello-world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo["full_name"] != "octocat/hello-world" {
		t.Fatalf("expected full_name=octocat/hello-world, got %v", repo["full_name"])
	}
}

func TestGitHubClient_ListIssues(t *testing.T) {
	ts, client := newGitHubTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/issues" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("state") != "open" {
			t.Fatal("expected default state=open")
		}
		resp := []map[string]interface{}{
			{
				"number":     float64(1),
				"title":      "Bug report",
				"state":      "open",
				"user":       map[string]interface{}{"login": "alice"},
				"labels":     []interface{}{map[string]interface{}{"name": "bug"}},
				"created_at": "2026-01-01T00:00:00Z",
				"updated_at": "2026-01-02T00:00:00Z",
				"html_url":   "https://github.com/owner/repo/issues/1",
				"comments":   float64(5),
			},
			{
				// This entry has pull_request key — should be filtered out
				"number":       float64(2),
				"title":        "PR as issue",
				"state":        "open",
				"pull_request": map[string]interface{}{"url": "..."},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	issues, err := client.ListIssues(context.Background(), "owner", "repo", "", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The PR entry should be filtered out, leaving 1 real issue.
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue (PRs filtered), got %d", len(issues))
	}
	if issues[0]["title"] != "Bug report" {
		t.Fatalf("expected title=Bug report, got %v", issues[0]["title"])
	}
	if issues[0]["user"] != "alice" {
		t.Fatalf("expected user=alice, got %v", issues[0]["user"])
	}
}

func TestGitHubClient_CreateIssue(t *testing.T) {
	ts, client := newGitHubTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/repos/owner/repo/issues" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["title"] != "New issue" {
			t.Fatalf("expected title=New issue, got %v", body["title"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number":     float64(42),
			"title":      "New issue",
			"html_url":   "https://github.com/owner/repo/issues/42",
			"created_at": "2026-01-01T00:00:00Z",
		})
	})
	defer ts.Close()

	result, err := client.CreateIssue(context.Background(), "owner", "repo", "New issue", "body text", []string{"bug"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["number"] != float64(42) {
		t.Fatalf("expected number=42, got %v", result["number"])
	}
}

func TestGitHubClient_ListPulls(t *testing.T) {
	ts, client := newGitHubTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/pulls" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		resp := []map[string]interface{}{
			{
				"number":     float64(10),
				"title":      "Add feature",
				"state":      "open",
				"user":       map[string]interface{}{"login": "bob"},
				"created_at": "2026-01-01T00:00:00Z",
				"updated_at": "2026-01-02T00:00:00Z",
				"html_url":   "https://github.com/owner/repo/pull/10",
				"draft":      false,
				"merged_at":  nil,
				"head":       map[string]interface{}{"ref": "feature-branch"},
				"base":       map[string]interface{}{"ref": "main"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	pulls, err := client.ListPulls(context.Background(), "owner", "repo", "open", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pulls) != 1 {
		t.Fatalf("expected 1 pull, got %d", len(pulls))
	}
	if pulls[0]["title"] != "Add feature" {
		t.Fatalf("expected title=Add feature, got %v", pulls[0]["title"])
	}
	if pulls[0]["head_ref"] != "feature-branch" {
		t.Fatalf("expected head_ref=feature-branch, got %v", pulls[0]["head_ref"])
	}
	if pulls[0]["base_ref"] != "main" {
		t.Fatalf("expected base_ref=main, got %v", pulls[0]["base_ref"])
	}
	if pulls[0]["merged"] != false {
		t.Fatalf("expected merged=false, got %v", pulls[0]["merged"])
	}
}

func TestGitHubClient_GetPull(t *testing.T) {
	ts, client := newGitHubTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/pulls/7" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]interface{}{
			"number": float64(7),
			"title":  "Fix bug",
			"state":  "closed",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	pr, err := client.GetPull(context.Background(), "owner", "repo", 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pr["number"] != float64(7) {
		t.Fatalf("expected number=7, got %v", pr["number"])
	}
}

func TestGitHubClient_ListWorkflowRuns(t *testing.T) {
	ts, client := newGitHubTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/actions/runs" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]interface{}{
			"workflow_runs": []interface{}{
				map[string]interface{}{
					"id":          float64(123),
					"name":        "CI",
					"status":      "completed",
					"conclusion":  "success",
					"head_branch": "main",
					"event":       "push",
					"created_at":  "2026-01-01T00:00:00Z",
					"updated_at":  "2026-01-01T00:05:00Z",
					"html_url":    "https://github.com/owner/repo/actions/runs/123",
					"run_number":  float64(45),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	runs, err := client.ListWorkflowRuns(context.Background(), "owner", "repo", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0]["name"] != "CI" {
		t.Fatalf("expected name=CI, got %v", runs[0]["name"])
	}
	if runs[0]["conclusion"] != "success" {
		t.Fatalf("expected conclusion=success, got %v", runs[0]["conclusion"])
	}
}

func TestGitHubClient_ListNotifications(t *testing.T) {
	ts, client := newGitHubTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/notifications" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		resp := []map[string]interface{}{
			{
				"id":         "1",
				"reason":     "mention",
				"unread":     true,
				"updated_at": "2026-01-01T00:00:00Z",
				"subject": map[string]interface{}{
					"title": "You were mentioned",
					"type":  "Issue",
				},
				"repository": map[string]interface{}{
					"full_name": "owner/repo",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer ts.Close()

	notifs, err := client.ListNotifications(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifs) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifs))
	}
	if notifs[0]["title"] != "You were mentioned" {
		t.Fatalf("expected title from subject, got %v", notifs[0]["title"])
	}
	if notifs[0]["repo"] != "owner/repo" {
		t.Fatalf("expected repo=owner/repo, got %v", notifs[0]["repo"])
	}
	if notifs[0]["reason"] != "mention" {
		t.Fatalf("expected reason=mention, got %v", notifs[0]["reason"])
	}
}

func TestGitHubClient_AuthHeader(t *testing.T) {
	ts, client := newGitHubTestServer(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token-abc" {
			t.Fatalf("expected Bearer test-token-abc, got %q", auth)
		}
		accept := r.Header.Get("Accept")
		if accept != "application/vnd.github+json" {
			t.Fatalf("expected Accept=application/vnd.github+json, got %q", accept)
		}
		apiVer := r.Header.Get("X-GitHub-Api-Version")
		if apiVer != "2022-11-28" {
			t.Fatalf("expected X-GitHub-Api-Version=2022-11-28, got %q", apiVer)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{})
	})
	defer ts.Close()

	_, _ = client.GetRepo(context.Background(), "o", "r")
}

func TestGitHubClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"not found", 404, `{"message":"Not Found"}`},
		{"forbidden", 403, `{"message":"API rate limit exceeded"}`},
		{"server error", 500, `{"message":"Internal Server Error"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, client := newGitHubTestServer(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.body))
			})
			defer ts.Close()

			_, err := client.GetRepo(context.Background(), "owner", "repo")
			if err == nil {
				t.Fatal("expected error for non-2xx status")
			}

			var apiErr *GitHubAPIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected *GitHubAPIError, got %T: %v", err, err)
			}
			if apiErr.StatusCode != tc.statusCode {
				t.Fatalf("expected status %d, got %d", tc.statusCode, apiErr.StatusCode)
			}
		})
	}
}
