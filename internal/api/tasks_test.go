package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTasksService_ListTasks_ErrorPropagation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "internal server error"}}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()

	svc := NewTasksService(client)

	_, err := svc.ListTasks(context.Background(), "@default", false)
	if err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}

func TestTasksService_ListTasks_CacheHit(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items": [{"id": "task1", "title": "Buy milk"}]}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()
	client.NoCache = false // Enable cache

	svc := NewTasksService(client)

	// First call should hit the API
	result1, err := svc.ListTasks(context.Background(), "@default", false)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 API call for first ListTasks, got %d", callCount)
	}
	if len(result1) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result1))
	}

	// Second call with same params should use cache
	result2, err := svc.ListTasks(context.Background(), "@default", false)
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

func TestTasksService_ListTasks_CacheBypass(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items": [{"id": "task1", "title": "Call dentist"}]}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()
	client.NoCache = true // Disable cache

	svc := NewTasksService(client)

	// Both calls should hit the API since cache is disabled
	svc.ListTasks(context.Background(), "@default", false)
	svc.ListTasks(context.Background(), "@default", false)

	if callCount != 2 {
		t.Fatalf("expected 2 API calls with NoCache=true, got %d", callCount)
	}
}

func TestTasksService_ListTasks_DifferentParamsBypass(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items": []}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()
	client.NoCache = false

	svc := NewTasksService(client)

	// Different parameters should create separate cache entries
	svc.ListTasks(context.Background(), "@default", false)
	svc.ListTasks(context.Background(), "@default", true) // showCompleted changed
	svc.ListTasks(context.Background(), "tasklist2", false)

	// Each unique parameter set should cause an API call
	if callCount != 3 {
		t.Fatalf("expected 3 API calls for different parameters, got %d", callCount)
	}
}

func TestTasksService_CreateTask_ErrorPropagation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": {"message": "bad request"}}`))
	}))
	defer ts.Close()

	client := NewClient(nil)
	client.endpoint = ts.URL
	client.httpClient = ts.Client()

	svc := NewTasksService(client)

	_, err := svc.CreateTask(context.Background(), "@default", "Test", "notes", "2026-12-31")
	if err == nil {
		t.Error("expected error for 400 response, got nil")
	}
}
