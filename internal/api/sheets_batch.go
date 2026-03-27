package api

import (
	"context"
	"sync"
)

const maxSheetsBatchConcurrency = 5

// BatchAppendEntry holds a single range + values pair for batch append.
type BatchAppendEntry struct {
	Range  string          `json:"range"`
	Values [][]interface{} `json:"values"`
}

// BatchAppendResult aggregates results from a batch append operation.
type BatchAppendResult struct {
	Total     int                  `json:"total"`
	Succeeded []BatchAppendSuccess `json:"succeeded"`
	Failed    []BatchAppendFailure `json:"failed"`
}

// BatchAppendSuccess records a successful append for one entry.
type BatchAppendSuccess struct {
	Range       string `json:"range"`
	UpdatedRows int64  `json:"updated_rows"`
}

// BatchAppendFailure records a failed append for one entry.
type BatchAppendFailure struct {
	Range string `json:"range"`
	Error string `json:"error"`
}

// BatchAppendValues appends multiple ranges concurrently using goroutine fan-out.
// concurrency controls the semaphore size; it is capped at maxSheetsBatchConcurrency (5).
// Partial failures are collected rather than aborting the batch.
func (ss *SheetsService) BatchAppendValues(
	ctx context.Context,
	spreadsheetID string,
	entries []BatchAppendEntry,
	concurrency int,
) (*BatchAppendResult, error) {
	if len(entries) == 0 {
		return &BatchAppendResult{Total: 0}, nil
	}

	concurrency = clampConcurrency(concurrency, maxSheetsBatchConcurrency)

	type itemResult struct {
		success *BatchAppendSuccess
		failure *BatchAppendFailure
	}

	sem := make(chan struct{}, concurrency)
	results := make(chan itemResult, len(entries))

	var wg sync.WaitGroup
	for _, e := range entries {
		entry := e
		wg.Add(1)
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Done()
			results <- itemResult{failure: &BatchAppendFailure{
				Range: entry.Range,
				Error: ctx.Err().Error(),
			}}
			continue
		}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			res, err := ss.AppendValues(ctx, spreadsheetID, entry.Range, entry.Values)
			if err != nil {
				results <- itemResult{failure: &BatchAppendFailure{
					Range: entry.Range,
					Error: err.Error(),
				}}
				return
			}
			results <- itemResult{success: &BatchAppendSuccess{
				Range:       entry.Range,
				UpdatedRows: res.UpdatedRows,
			}}
		}()
	}

	// close results channel after all goroutines finish
	go func() {
		wg.Wait()
		close(results)
	}()

	out := &BatchAppendResult{Total: len(entries)}
	for r := range results {
		if r.success != nil {
			out.Succeeded = append(out.Succeeded, *r.success)
		} else {
			out.Failed = append(out.Failed, *r.failure)
		}
	}
	return out, nil
}
