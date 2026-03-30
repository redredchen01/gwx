package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatService_ListSpaces_ErrorPropagation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "internal server error"}}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()

	svc := NewChatService(client)

	_, err := svc.ListSpaces(context.Background(), 0)
	if err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}

func TestChatService_ListSpaces_CacheHit(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"spaces": [{"name": "spaces/abc123", "displayName": "General"}]}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()
	client.NoCache = false // Enable cache

	svc := NewChatService(client)

	// First call should hit the API
	result1, err := svc.ListSpaces(context.Background(), 10)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 API call for first ListSpaces, got %d", callCount)
	}
	if len(result1) != 1 {
		t.Fatalf("expected 1 space, got %d", len(result1))
	}

	// Second call with same params should use cache
	result2, err := svc.ListSpaces(context.Background(), 10)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 API call total (cached), got %d", callCount)
	}

	if len(result2) != len(result1) {
		t.Fatalf("cached result differs: expected %d, got %d", len(result1), len(result2))
	}
}

func TestChatService_ListSpaces_CacheBypass(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"spaces": [{"name": "spaces/xyz789", "displayName": "Engineering"}]}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()
	client.NoCache = true // Disable cache

	svc := NewChatService(client)

	// Both calls should hit the API since cache is disabled
	svc.ListSpaces(context.Background(), 10)
	svc.ListSpaces(context.Background(), 10)

	if callCount != 2 {
		t.Fatalf("expected 2 API calls with NoCache=true, got %d", callCount)
	}
}

func TestChatService_ListSpaces_DifferentMaxResultsValues(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"spaces": []}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()
	client.NoCache = false

	svc := NewChatService(client)

	// Different maxResults values should create separate cache entries
	svc.ListSpaces(context.Background(), 10)
	svc.ListSpaces(context.Background(), 20)
	svc.ListSpaces(context.Background(), 0)

	// Each unique parameter set should cause an API call
	if callCount != 3 {
		t.Fatalf("expected 3 API calls for different maxResults, got %d", callCount)
	}
}

func TestChatService_SendMessage_ErrorPropagation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"message": "bad request"}}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()

	svc := NewChatService(client)

	_, err := svc.SendMessage(context.Background(), "spaces/abc123", "test message")
	if err == nil {
		t.Error("expected error for 400 response, got nil")
	}
}

func TestChatService_ListMessages_ErrorPropagation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": {"message": "forbidden"}}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()

	svc := NewChatService(client)

	_, err := svc.ListMessages(context.Background(), "spaces/abc123", 10)
	if err == nil {
		t.Error("expected error for 403 response, got nil")
	}
}
