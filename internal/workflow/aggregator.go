package workflow

import (
	"context"
	"fmt"
	"sync"
)

// FetchResult wraps a named result from a parallel fetch.
type FetchResult struct {
	Name  string
	Value interface{}
	Error error
}

// Fetcher is a named function that fetches data from a service.
type Fetcher struct {
	Name string
	Fn   func(ctx context.Context) (interface{}, error)
}

// RunParallel executes fetchers concurrently, returns results keyed by name.
// Partial failures are captured in FetchResult.Error, never abort the whole batch.
// Results are returned in the same order as the input fetchers.
func RunParallel(ctx context.Context, fetchers []Fetcher) []FetchResult {
	results := make([]FetchResult, len(fetchers))
	var wg sync.WaitGroup

	for i, f := range fetchers {
		wg.Add(1)
		go func(idx int, fetcher Fetcher) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					results[idx] = FetchResult{
						Name:  fetcher.Name,
						Error: fmt.Errorf("panic: %v", r),
					}
				}
			}()

			val, err := fetcher.Fn(ctx)
			results[idx] = FetchResult{
				Name:  fetcher.Name,
				Value: val,
				Error: err,
			}
		}(i, f)
	}

	wg.Wait()
	return results
}

// FindResult returns the FetchResult with the given name, or nil if not found.
func FindResult(results []FetchResult, name string) *FetchResult {
	for i := range results {
		if results[i].Name == name {
			return &results[i]
		}
	}
	return nil
}
