package api

import (
	"context"
	"sync"
)

const maxDriveBatchConcurrency = 5

// BatchUploadResult summarises a batch upload operation.
type BatchUploadResult struct {
	Total     int                  `json:"total"`
	Succeeded []BatchUploadSuccess `json:"succeeded"`
	Failed    []BatchUploadFailure `json:"failed"`
}

// BatchUploadSuccess records a single successful upload within a batch.
type BatchUploadSuccess struct {
	Path   string `json:"path"`
	FileID string `json:"file_id"`
	Name   string `json:"name"`
}

// BatchUploadFailure records a single failed upload within a batch.
type BatchUploadFailure struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// batchResult is the internal per-goroutine result carrier.
type batchResult struct {
	success *BatchUploadSuccess
	failure *BatchUploadFailure
}

// clampConcurrency returns a concurrency value that is:
//   - at least 1
//   - at most max
func clampConcurrency(n, max int) int {
	if n < 1 {
		return 1
	}
	if n > max {
		return max
	}
	return n
}

// BatchUploadFiles uploads multiple local files to Drive concurrently.
//
// concurrency controls how many uploads run in parallel; values above 5 are
// capped at 5. A partial failure does not abort the batch — all paths are
// attempted and results are collected.
//
// Each upload calls UploadFile(ctx, path, folder, "") which already handles
// rate limiting internally via WaitRate.
func (ds *DriveService) BatchUploadFiles(ctx context.Context, paths []string, folder string, concurrency int) (*BatchUploadResult, error) {
	concurrency = clampConcurrency(concurrency, maxDriveBatchConcurrency)

	sem := make(chan struct{}, concurrency)
	results := make(chan batchResult, len(paths))

	var wg sync.WaitGroup

	for _, p := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			// Acquire semaphore slot.
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results <- batchResult{
					failure: &BatchUploadFailure{
						Path:  path,
						Error: ctx.Err().Error(),
					},
				}
				return
			}
			defer func() { <-sem }()

			summary, err := ds.UploadFile(ctx, path, folder, "")
			if err != nil {
				results <- batchResult{
					failure: &BatchUploadFailure{
						Path:  path,
						Error: err.Error(),
					},
				}
				return
			}

			results <- batchResult{
				success: &BatchUploadSuccess{
					Path:   path,
					FileID: summary.ID,
					Name:   summary.Name,
				},
			}
		}(p)
	}

	// Close results channel once all goroutines finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	out := &BatchUploadResult{
		Total:     len(paths),
		Succeeded: make([]BatchUploadSuccess, 0, len(paths)),
		Failed:    make([]BatchUploadFailure, 0),
	}

	for r := range results {
		if r.success != nil {
			out.Succeeded = append(out.Succeeded, *r.success)
		} else if r.failure != nil {
			out.Failed = append(out.Failed, *r.failure)
		}
	}

	return out, nil
}
