package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContactsService_SearchContacts_ErrorPropagation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "internal server error"}}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()

	svc := NewContactsService(client)

	_, err := svc.SearchContacts(context.Background(), "test", 10)
	if err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}

func TestContactsService_SearchContacts_CacheHit(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": [{"person": {"names": [{"displayName": "Alice"}]}}]}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()
	client.NoCache = false // Enable cache

	svc := NewContactsService(client)

	// First call should hit the API
	result1, err := svc.SearchContacts(context.Background(), "Alice", 10)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 API call for first SearchContacts, got %d", callCount)
	}
	if len(result1) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(result1))
	}

	// Second call with same params should use cache
	result2, err := svc.SearchContacts(context.Background(), "Alice", 10)
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

func TestContactsService_SearchContacts_CacheBypass(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"results": [{"person": {"names": [{"displayName": "Bob"}]}}]}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()
	client.NoCache = true // Disable cache

	svc := NewContactsService(client)

	// Both calls should hit the API since cache is disabled
	svc.SearchContacts(context.Background(), "Bob", 10)
	svc.SearchContacts(context.Background(), "Bob", 10)

	if callCount != 2 {
		t.Fatalf("expected 2 API calls with NoCache=true, got %d", callCount)
	}
}

func TestContactsService_ListContacts_ErrorPropagation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "unauthorized"}}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()

	svc := NewContactsService(client)

	_, err := svc.ListContacts(context.Background(), 10)
	if err == nil {
		t.Error("expected error for 401 response, got nil")
	}
}

func TestContactsService_ListContacts_CacheHit(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"connections": [{"names": [{"displayName": "Carol"}]}]}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()
	client.NoCache = false

	svc := NewContactsService(client)

	// First call
	svc.ListContacts(context.Background(), 10)
	if callCount != 1 {
		t.Fatalf("expected 1 API call for first ListContacts, got %d", callCount)
	}

	// Second call should use cache
	svc.ListContacts(context.Background(), 10)
	if callCount != 1 {
		t.Fatalf("expected 1 API call total (cached), got %d", callCount)
	}
}
