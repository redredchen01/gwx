package workflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunParallelAllSuccess(t *testing.T) {
	fetchers := []Fetcher{
		{Name: "a", Fn: func(ctx context.Context) (interface{}, error) { return "va", nil }},
		{Name: "b", Fn: func(ctx context.Context) (interface{}, error) { return "vb", nil }},
		{Name: "c", Fn: func(ctx context.Context) (interface{}, error) { return "vc", nil }},
	}
	results := RunParallel(context.Background(), fetchers)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, r := range results {
		if r.Error != nil {
			t.Errorf("result[%d] unexpected error: %v", i, r.Error)
		}
		if r.Name != fetchers[i].Name {
			t.Errorf("result[%d] name mismatch: %s vs %s", i, r.Name, fetchers[i].Name)
		}
	}
}

func TestRunParallelPartialFailure(t *testing.T) {
	fetchers := []Fetcher{
		{Name: "ok", Fn: func(ctx context.Context) (interface{}, error) { return "good", nil }},
		{Name: "fail", Fn: func(ctx context.Context) (interface{}, error) { return nil, errors.New("boom") }},
		{Name: "ok2", Fn: func(ctx context.Context) (interface{}, error) { return "also good", nil }},
	}
	results := RunParallel(context.Background(), fetchers)
	if results[0].Error != nil {
		t.Errorf("result[0] should succeed")
	}
	if results[1].Error == nil {
		t.Errorf("result[1] should fail")
	}
	if results[2].Error != nil {
		t.Errorf("result[2] should succeed")
	}
}

func TestRunParallelContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	fetchers := []Fetcher{
		{Name: "slow", Fn: func(ctx context.Context) (interface{}, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				return "late", nil
			}
		}},
	}
	results := RunParallel(ctx, fetchers)
	if results[0].Error == nil {
		t.Errorf("expected context cancel error")
	}
}

func TestRunParallelPanicRecovery(t *testing.T) {
	fetchers := []Fetcher{
		{Name: "ok", Fn: func(ctx context.Context) (interface{}, error) { return "safe", nil }},
		{Name: "panic", Fn: func(ctx context.Context) (interface{}, error) { panic("oh no") }},
	}
	results := RunParallel(context.Background(), fetchers)
	if results[0].Error != nil {
		t.Errorf("result[0] should succeed")
	}
	if results[1].Error == nil {
		t.Errorf("result[1] should have panic error")
	}
	if results[1].Value != nil {
		t.Errorf("result[1] value should be nil")
	}
}

func TestRunParallelOrderPreserved(t *testing.T) {
	names := []string{"first", "second", "third", "fourth", "fifth"}
	fetchers := make([]Fetcher, len(names))
	for i, n := range names {
		n := n
		fetchers[i] = Fetcher{
			Name: n,
			Fn:   func(ctx context.Context) (interface{}, error) { return n, nil },
		}
	}
	results := RunParallel(context.Background(), fetchers)
	for i, r := range results {
		if r.Name != names[i] {
			t.Errorf("order mismatch at %d: got %s, want %s", i, r.Name, names[i])
		}
	}
}
