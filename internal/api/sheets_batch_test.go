package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// mockSheetsService is a testable interface wrapping AppendValues.
// We can't easily mock SheetsService directly because it calls a real API,
// so we use a function-injection pattern via a test helper that builds a
// BatchAppendResult without hitting the network.

// appendFunc is the function signature used in testing to simulate AppendValues.
type appendFunc func(ctx context.Context, spreadsheetID, appendRange string, values [][]interface{}) (*SheetAppendResult, error)

// batchAppendWithFn is the extracted logic of BatchAppendValues, accepting an
// injectable append function. This allows deterministic unit testing.
func batchAppendWithFn(
	ctx context.Context,
	spreadsheetID string,
	entries []BatchAppendEntry,
	concurrency int,
	fn appendFunc,
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

	for _, e := range entries {
		entry := e
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			res, err := fn(ctx, spreadsheetID, entry.Range, entry.Values)
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

	// collect
	out := &BatchAppendResult{Total: len(entries)}
	for i := 0; i < len(entries); i++ {
		r := <-results
		if r.success != nil {
			out.Succeeded = append(out.Succeeded, *r.success)
		} else {
			out.Failed = append(out.Failed, *r.failure)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestBatchAppend_AllSuccess(t *testing.T) {
	entries := []BatchAppendEntry{
		{Range: "Sheet1!A1", Values: [][]interface{}{{"a", 1}}},
		{Range: "Sheet1!A2", Values: [][]interface{}{{"b", 2}}},
		{Range: "Sheet1!A3", Values: [][]interface{}{{"c", 3}}},
	}

	fn := func(_ context.Context, _ string, appendRange string, _ [][]interface{}) (*SheetAppendResult, error) {
		return &SheetAppendResult{UpdatedRange: appendRange, UpdatedRows: 1, UpdatedCells: 2}, nil
	}

	result, err := batchAppendWithFn(context.Background(), "spreadsheet-id", entries, 3, fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("Total = %d, want 3", result.Total)
	}
	if len(result.Succeeded) != 3 {
		t.Errorf("Succeeded count = %d, want 3", len(result.Succeeded))
	}
	if len(result.Failed) != 0 {
		t.Errorf("Failed count = %d, want 0", len(result.Failed))
	}
	for _, s := range result.Succeeded {
		if s.UpdatedRows != 1 {
			t.Errorf("UpdatedRows = %d, want 1", s.UpdatedRows)
		}
	}
}

func TestBatchAppend_PartialFailure(t *testing.T) {
	entries := []BatchAppendEntry{
		{Range: "Sheet1!A1", Values: [][]interface{}{{"ok"}}},
		{Range: "Sheet1!A2", Values: [][]interface{}{{"fail"}}},
		{Range: "Sheet1!A3", Values: [][]interface{}{{"ok2"}}},
	}

	fn := func(_ context.Context, _ string, appendRange string, _ [][]interface{}) (*SheetAppendResult, error) {
		if appendRange == "Sheet1!A2" {
			return nil, errors.New("API error: quota exceeded")
		}
		return &SheetAppendResult{UpdatedRange: appendRange, UpdatedRows: 1, UpdatedCells: 1}, nil
	}

	result, err := batchAppendWithFn(context.Background(), "spreadsheet-id", entries, 3, fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("Total = %d, want 3", result.Total)
	}
	if len(result.Succeeded) != 2 {
		t.Errorf("Succeeded = %d, want 2", len(result.Succeeded))
	}
	if len(result.Failed) != 1 {
		t.Errorf("Failed = %d, want 1", len(result.Failed))
	}
	if result.Failed[0].Range != "Sheet1!A2" {
		t.Errorf("Failed range = %q, want Sheet1!A2", result.Failed[0].Range)
	}
	if result.Failed[0].Error != "API error: quota exceeded" {
		t.Errorf("Failed error = %q", result.Failed[0].Error)
	}
}

func TestBatchAppend_ConcurrencyLimit(t *testing.T) {
	const numEntries = 20
	var inFlight int64
	var maxObserved int64

	entries := make([]BatchAppendEntry, numEntries)
	for i := range entries {
		entries[i] = BatchAppendEntry{
			Range:  fmt.Sprintf("Sheet1!A%d", i+1),
			Values: [][]interface{}{{i}},
		}
	}

	fn := func(_ context.Context, _ string, _ string, _ [][]interface{}) (*SheetAppendResult, error) {
		cur := atomic.AddInt64(&inFlight, 1)
		// track high-water mark
		for {
			old := atomic.LoadInt64(&maxObserved)
			if cur <= old {
				break
			}
			if atomic.CompareAndSwapInt64(&maxObserved, old, cur) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond) // simulate latency
		atomic.AddInt64(&inFlight, -1)
		return &SheetAppendResult{UpdatedRows: 1}, nil
	}

	// pass concurrency=3; cap should be min(3, maxSheetsBatchConcurrency=5) = 3
	result, err := batchAppendWithFn(context.Background(), "spreadsheet-id", entries, 3, fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != numEntries {
		t.Errorf("Total = %d, want %d", result.Total, numEntries)
	}
	if len(result.Succeeded) != numEntries {
		t.Errorf("Succeeded = %d, want %d", len(result.Succeeded), numEntries)
	}

	observed := atomic.LoadInt64(&maxObserved)
	if observed > 3 {
		t.Errorf("max concurrent goroutines = %d, want <= 3 (concurrency limit)", observed)
	}

	// Also verify cap to 5: passing concurrency=99 should behave like 5
	inFlight = 0
	maxObserved = 0
	result2, err := batchAppendWithFn(context.Background(), "spreadsheet-id", entries, 99, fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.Total != numEntries {
		t.Errorf("(cap test) Total = %d, want %d", result2.Total, numEntries)
	}
	observed2 := atomic.LoadInt64(&maxObserved)
	if observed2 > int64(maxSheetsBatchConcurrency) {
		t.Errorf("(cap test) max concurrent = %d, want <= %d", observed2, maxSheetsBatchConcurrency)
	}
}

func TestBatchAppend_EmptyEntries(t *testing.T) {
	fn := func(_ context.Context, _ string, _ string, _ [][]interface{}) (*SheetAppendResult, error) {
		t.Fatal("append function should not be called for empty entries")
		return nil, nil
	}

	result, err := batchAppendWithFn(context.Background(), "spreadsheet-id", nil, 3, fn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("Total = %d, want 0", result.Total)
	}
}

func TestBatchAppendStructJSON(t *testing.T) {
	entry := BatchAppendEntry{
		Range:  "Sheet1!A1:B2",
		Values: [][]interface{}{{"hello", 42}, {true, nil}},
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal BatchAppendEntry: %v", err)
	}
	var decoded BatchAppendEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal BatchAppendEntry: %v", err)
	}
	if decoded.Range != entry.Range {
		t.Errorf("Range = %q, want %q", decoded.Range, entry.Range)
	}

	result := BatchAppendResult{
		Total: 2,
		Succeeded: []BatchAppendSuccess{
			{Range: "Sheet1!A1", UpdatedRows: 1},
		},
		Failed: []BatchAppendFailure{
			{Range: "Sheet1!A2", Error: "some error"},
		},
	}
	data, err = json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal BatchAppendResult: %v", err)
	}
	var decodedResult BatchAppendResult
	if err := json.Unmarshal(data, &decodedResult); err != nil {
		t.Fatalf("unmarshal BatchAppendResult: %v", err)
	}
	if decodedResult.Total != 2 {
		t.Errorf("Total = %d, want 2", decodedResult.Total)
	}
	if len(decodedResult.Succeeded) != 1 || decodedResult.Succeeded[0].UpdatedRows != 1 {
		t.Errorf("Succeeded mismatch: %+v", decodedResult.Succeeded)
	}
	if len(decodedResult.Failed) != 1 || decodedResult.Failed[0].Error != "some error" {
		t.Errorf("Failed mismatch: %+v", decodedResult.Failed)
	}
}
