package api

import (
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

func TestClientHTTPClientReusesPerServiceClient(t *testing.T) {
	client := NewClient(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"}))

	first := client.HTTPClient("gmail")
	second := client.HTTPClient("gmail")

	if first != second {
		t.Fatal("expected HTTPClient to reuse the same client for the same service")
	}
}

func TestClientHTTPClientSeparatesServices(t *testing.T) {
	client := NewClient(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"}))

	gmailClient := client.HTTPClient("gmail")
	driveClient := client.HTTPClient("drive")

	if gmailClient == driveClient {
		t.Fatal("expected distinct HTTP clients for different services")
	}
}

func TestClientGetOrCreateServiceReusesValue(t *testing.T) {
	client := NewClient(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"}))

	calls := 0
	factory := func() (any, error) {
		calls++
		return &struct{ Name string }{Name: "svc"}, nil
	}

	first, err := client.GetOrCreateService("gmail:v1", factory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := client.GetOrCreateService("gmail:v1", factory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first != second {
		t.Fatal("expected GetOrCreateService to reuse the cached service instance")
	}
	if calls != 1 {
		t.Fatalf("expected factory to be called once, got %d", calls)
	}
}

func TestClientGetOrCreateServiceConcurrent(t *testing.T) {
	client := NewClient(oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"}))

	var mu sync.Mutex
	calls := 0
	factory := func() (any, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		return &struct{ Name string }{Name: "svc"}, nil
	}

	const workers = 8
	results := make(chan any, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			svc, err := client.GetOrCreateService("drive:v3", factory)
			if err != nil {
				errs <- err
				return
			}
			results <- svc
		}()
	}
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	var first any
	for svc := range results {
		if first == nil {
			first = svc
			continue
		}
		if svc != first {
			t.Fatal("expected all callers to receive the same cached service instance")
		}
	}
	if calls == 0 {
		t.Fatal("expected factory to be called at least once")
	}
}
