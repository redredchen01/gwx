package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// slackRedirectTransport rewrites requests aimed at slack.com/api/ so they
// hit the local httptest server instead.
type slackRedirectTransport struct {
	target *httptest.Server
}

func (t *slackRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL: keep the path after "slack.com/api/" and send it
	// to the test server.
	const prefix = "https://slack.com/api/"
	raw := req.URL.String()
	if strings.HasPrefix(raw, prefix) {
		newURL := t.target.URL + "/" + strings.TrimPrefix(raw, prefix)
		origHeaders := req.Header
		var err error
		req, err = http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
		if err != nil {
			return nil, err
		}
		req.Header = origHeaders
	}
	return t.target.Client().Transport.RoundTrip(req)
}

// newSlackTestServer creates an httptest.Server and a SlackClient that
// redirects all Slack API calls to it.
func newSlackTestServer(handler http.HandlerFunc) (*httptest.Server, *SlackClient) {
	ts := httptest.NewServer(handler)
	client := NewSlackClient("xoxb-test-token")
	client.http = &http.Client{
		Transport: &slackRedirectTransport{target: ts},
	}
	return ts, client
}

func TestSlackClient_ListChannels(t *testing.T) {
	ts, client := newSlackTestServer(func(w http.ResponseWriter, r *http.Request) {
		// Path should contain conversations.list
		if !strings.Contains(r.URL.Path, "conversations.list") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []interface{}{
				map[string]interface{}{
					"id":   "C123",
					"name": "general",
				},
				map[string]interface{}{
					"id":   "C456",
					"name": "random",
				},
			},
		})
	})
	defer ts.Close()

	channels, err := client.ListChannels(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}
	if channels[0]["name"] != "general" {
		t.Fatalf("expected first channel=general, got %v", channels[0]["name"])
	}
}

func TestSlackClient_SendMessage(t *testing.T) {
	ts, client := newSlackTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "chat.postMessage") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["channel"] != "C123" {
			t.Fatalf("expected channel=C123, got %v", body["channel"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"channel": "C123",
			"ts":      "1234567890.123456",
			"message": map[string]interface{}{
				"text": "hello",
			},
		})
	})
	defer ts.Close()

	result, err := client.SendMessage(context.Background(), "C123", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["channel"] != "C123" {
		t.Fatalf("expected channel=C123, got %v", result["channel"])
	}
	if result["ts"] != "1234567890.123456" {
		t.Fatalf("expected ts, got %v", result["ts"])
	}
}

func TestSlackClient_ListMessages(t *testing.T) {
	ts, client := newSlackTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "conversations.history") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"messages": []interface{}{
				map[string]interface{}{
					"text": "Hello world",
					"user": "U123",
					"ts":   "1234567890.000001",
				},
			},
		})
	})
	defer ts.Close()

	msgs, err := client.ListMessages(context.Background(), "C123", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0]["text"] != "Hello world" {
		t.Fatalf("expected text=Hello world, got %v", msgs[0]["text"])
	}
}

func TestSlackClient_SearchMessages(t *testing.T) {
	ts, client := newSlackTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "search.messages") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"messages": map[string]interface{}{
				"matches": []interface{}{
					map[string]interface{}{
						"text":    "found it",
						"channel": map[string]interface{}{"name": "general"},
					},
				},
			},
		})
	})
	defer ts.Close()

	matches, err := client.SearchMessages(context.Background(), "search term", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0]["text"] != "found it" {
		t.Fatalf("expected text=found it, got %v", matches[0]["text"])
	}
}

func TestSlackClient_ListUsers(t *testing.T) {
	ts, client := newSlackTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "users.list") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"members": []interface{}{
				map[string]interface{}{
					"id":   "U001",
					"name": "alice",
				},
				map[string]interface{}{
					"id":   "U002",
					"name": "bob",
				},
			},
		})
	})
	defer ts.Close()

	users, err := client.ListUsers(context.Background(), 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[1]["name"] != "bob" {
		t.Fatalf("expected second user=bob, got %v", users[1]["name"])
	}
}

func TestSlackClient_GetUser(t *testing.T) {
	ts, client := newSlackTestServer(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "users.info") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":        "U001",
				"name":      "alice",
				"real_name": "Alice Smith",
			},
		})
	})
	defer ts.Close()

	user, err := client.GetUser(context.Background(), "U001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user["name"] != "alice" {
		t.Fatalf("expected name=alice, got %v", user["name"])
	}
	if user["real_name"] != "Alice Smith" {
		t.Fatalf("expected real_name=Alice Smith, got %v", user["real_name"])
	}
}

func TestSlackClient_AuthHeader(t *testing.T) {
	ts, client := newSlackTestServer(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer xoxb-test-token" {
			t.Fatalf("expected Bearer xoxb-test-token, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"members": []interface{}{},
		})
	})
	defer ts.Close()

	_, _ = client.ListUsers(context.Background(), 1)
}

func TestSlackClient_SlackError(t *testing.T) {
	ts, client := newSlackTestServer(func(w http.ResponseWriter, r *http.Request) {
		// Slack returns HTTP 200 but ok=false with an error string.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "channel_not_found",
		})
	})
	defer ts.Close()

	_, err := client.ListMessages(context.Background(), "C999", 10)
	if err == nil {
		t.Fatal("expected error for ok=false response")
	}
	if !strings.Contains(err.Error(), "channel_not_found") {
		t.Fatalf("expected error to contain channel_not_found, got %v", err)
	}
}
