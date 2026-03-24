package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// notionRedirectTransport rewrites requests aimed at api.notion.com/v1 so
// they hit the local httptest server instead.
type notionRedirectTransport struct {
	target  *httptest.Server
	lastReq *http.Request // captured for header inspection
}

func (t *notionRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.lastReq = req
	const prefix = "https://api.notion.com/v1"
	raw := req.URL.String()
	if strings.HasPrefix(raw, prefix) {
		suffix := strings.TrimPrefix(raw, prefix)
		newURL := t.target.URL + suffix
		newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
		if err != nil {
			return nil, err
		}
		// Copy headers so the test server can inspect them.
		newReq.Header = req.Header
		req = newReq
	}
	return t.target.Client().Transport.RoundTrip(req)
}

// newNotionTestServer creates an httptest.Server and a NotionClient that
// redirects all Notion API calls to it.
func newNotionTestServer(handler http.HandlerFunc) (*httptest.Server, *NotionClient, *notionRedirectTransport) {
	ts := httptest.NewServer(handler)
	transport := &notionRedirectTransport{target: ts}
	client := NewNotionClient("ntn_test_token_abc")
	client.http = &http.Client{Transport: transport}
	return ts, client, transport
}

func TestNotionClient_SearchPages(t *testing.T) {
	ts, client, _ := newNotionTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []interface{}{
				map[string]interface{}{
					"id":     "page-1",
					"object": "page",
				},
				map[string]interface{}{
					"id":     "page-2",
					"object": "page",
				},
			},
		})
	})
	defer ts.Close()

	pages, err := client.SearchPages(context.Background(), "test", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	if pages[0]["id"] != "page-1" {
		t.Fatalf("expected id=page-1, got %v", pages[0]["id"])
	}
}

func TestNotionClient_GetPage(t *testing.T) {
	ts, client, _ := newNotionTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/pages/abc-123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "abc-123",
			"object": "page",
			"properties": map[string]interface{}{
				"title": "My Page",
			},
		})
	})
	defer ts.Close()

	page, err := client.GetPage(context.Background(), "abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page["id"] != "abc-123" {
		t.Fatalf("expected id=abc-123, got %v", page["id"])
	}
}

func TestNotionClient_CreatePage(t *testing.T) {
	ts, client, _ := newNotionTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/pages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		parent, ok := body["parent"].(map[string]interface{})
		if !ok {
			t.Fatal("expected parent in body")
		}
		if parent["database_id"] != "db-123" {
			t.Fatalf("expected parent database_id=db-123, got %v", parent["database_id"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "new-page-id",
			"object": "page",
		})
	})
	defer ts.Close()

	page, err := client.CreatePage(context.Background(), "db-123", "Test Page", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page["id"] != "new-page-id" {
		t.Fatalf("expected id=new-page-id, got %v", page["id"])
	}
}

func TestNotionClient_QueryDatabase(t *testing.T) {
	ts, client, _ := newNotionTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/databases/db-456/query" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []interface{}{
				map[string]interface{}{
					"id":     "row-1",
					"object": "page",
				},
			},
		})
	})
	defer ts.Close()

	rows, err := client.QueryDatabase(context.Background(), "db-456", nil, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0]["id"] != "row-1" {
		t.Fatalf("expected id=row-1, got %v", rows[0]["id"])
	}
}

func TestNotionClient_ListDatabases(t *testing.T) {
	ts, client, _ := newNotionTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		filter, ok := body["filter"].(map[string]interface{})
		if !ok {
			t.Fatal("expected filter in body")
		}
		if filter["value"] != "database" {
			t.Fatalf("expected filter value=database, got %v", filter["value"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []interface{}{
				map[string]interface{}{
					"id":     "db-1",
					"object": "database",
					"title":  []interface{}{map[string]interface{}{"plain_text": "My DB"}},
				},
			},
		})
	})
	defer ts.Close()

	dbs, err := client.ListDatabases(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dbs) != 1 {
		t.Fatalf("expected 1 database, got %d", len(dbs))
	}
	if dbs[0]["id"] != "db-1" {
		t.Fatalf("expected id=db-1, got %v", dbs[0]["id"])
	}
}

func TestNotionClient_GetDatabase(t *testing.T) {
	ts, client, _ := newNotionTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/databases/db-789" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "db-789",
			"object": "database",
		})
	})
	defer ts.Close()

	db, err := client.GetDatabase(context.Background(), "db-789")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db["id"] != "db-789" {
		t.Fatalf("expected id=db-789, got %v", db["id"])
	}
}

func TestNotionClient_Headers(t *testing.T) {
	transport := &notionRedirectTransport{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers on the request that arrived at the server.
		auth := r.Header.Get("Authorization")
		if auth != "Bearer ntn_test_token_abc" {
			t.Fatalf("expected Bearer ntn_test_token_abc, got %q", auth)
		}
		ver := r.Header.Get("Notion-Version")
		if ver != "2022-06-28" {
			t.Fatalf("expected Notion-Version=2022-06-28, got %q", ver)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "page-x",
			"object": "page",
		})
	}))
	defer ts.Close()

	transport.target = ts
	client := NewNotionClient("ntn_test_token_abc")
	client.http = &http.Client{Transport: transport}

	_, err := client.GetPage(context.Background(), "page-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNotionClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       map[string]interface{}
		wantErr    string
	}{
		{
			name:       "not found",
			statusCode: 404,
			body: map[string]interface{}{
				"code":    "object_not_found",
				"message": "Could not find page",
			},
			wantErr: "object_not_found",
		},
		{
			name:       "unauthorized",
			statusCode: 401,
			body: map[string]interface{}{
				"code":    "unauthorized",
				"message": "API token is invalid",
			},
			wantErr: "unauthorized",
		},
		{
			name:       "rate limited",
			statusCode: 429,
			body: map[string]interface{}{
				"code":    "rate_limited",
				"message": "Rate limited",
			},
			wantErr: "rate_limited",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, client, _ := newNotionTestServer(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				json.NewEncoder(w).Encode(tc.body)
			})
			defer ts.Close()

			_, err := client.GetPage(context.Background(), "nonexistent")
			if err == nil {
				t.Fatal("expected error for non-2xx status")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error to contain %q, got %v", tc.wantErr, err)
			}
		})
	}
}
