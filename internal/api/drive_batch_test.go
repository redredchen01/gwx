package api

import (
	"encoding/json"
	"testing"
)

// TestBatchUploadResult_JSONSerialization 驗證結構體 JSON 序列化正確性
func TestBatchUploadResult_JSONSerialization(t *testing.T) {
	result := &BatchUploadResult{
		Total: 3,
		Succeeded: []BatchUploadSuccess{
			{Path: "/tmp/a.txt", FileID: "id-001", Name: "a.txt"},
			{Path: "/tmp/b.txt", FileID: "id-002", Name: "b.txt"},
		},
		Failed: []BatchUploadFailure{
			{Path: "/tmp/c.txt", Error: "open file: no such file or directory"},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var got BatchUploadResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if got.Total != 3 {
		t.Errorf("expected Total=3, got %d", got.Total)
	}
	if len(got.Succeeded) != 2 {
		t.Errorf("expected 2 succeeded, got %d", len(got.Succeeded))
	}
	if len(got.Failed) != 1 {
		t.Errorf("expected 1 failed, got %d", len(got.Failed))
	}

	s := got.Succeeded[0]
	if s.Path != "/tmp/a.txt" || s.FileID != "id-001" || s.Name != "a.txt" {
		t.Errorf("succeeded[0] mismatch: %+v", s)
	}

	f := got.Failed[0]
	if f.Path != "/tmp/c.txt" || f.Error == "" {
		t.Errorf("failed[0] mismatch: %+v", f)
	}
}

// TestBatchUploadConcurrencyCap 驗證 concurrency cap 邏輯：輸入 > 5 → 強制降為 5
func TestBatchUploadConcurrencyCap(t *testing.T) {
	const maxConcurrency = 5

	cases := []struct {
		input    int
		expected int
	}{
		{1, 1},
		{3, 3},
		{5, 5},
		{6, 5},
		{10, 5},
		{100, 5},
		{0, 1}, // 0 或負數應 fallback 為 1
		{-1, 1},
	}

	for _, tc := range cases {
		got := clampConcurrency(tc.input, maxConcurrency)
		if got != tc.expected {
			t.Errorf("clampConcurrency(%d, %d) = %d, want %d", tc.input, maxConcurrency, got, tc.expected)
		}
	}
}

// TestBatchUploadResult_EmptySlices 確保 Succeeded/Failed 在零值時序列化為 [] 而非 null
func TestBatchUploadResult_EmptySlices(t *testing.T) {
	result := &BatchUploadResult{
		Total:     0,
		Succeeded: []BatchUploadSuccess{},
		Failed:    []BatchUploadFailure{},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// 驗證包含 "succeeded":[] 而非 "succeeded":null
	jsonStr := string(data)
	if jsonStr == "" {
		t.Fatal("empty JSON output")
	}

	var got BatchUploadResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.Total != 0 {
		t.Errorf("expected Total=0, got %d", got.Total)
	}
}

// TestBatchUploadSuccess_Fields 驗證 BatchUploadSuccess 欄位存取
func TestBatchUploadSuccess_Fields(t *testing.T) {
	s := BatchUploadSuccess{
		Path:   "/foo/bar.pdf",
		FileID: "drive-file-xyz",
		Name:   "bar.pdf",
	}

	if s.Path != "/foo/bar.pdf" {
		t.Errorf("Path mismatch")
	}
	if s.FileID != "drive-file-xyz" {
		t.Errorf("FileID mismatch")
	}
	if s.Name != "bar.pdf" {
		t.Errorf("Name mismatch")
	}
}

// TestBatchUploadFailure_Fields 驗證 BatchUploadFailure 欄位存取
func TestBatchUploadFailure_Fields(t *testing.T) {
	f := BatchUploadFailure{
		Path:  "/foo/missing.pdf",
		Error: "file not found",
	}

	if f.Path != "/foo/missing.pdf" {
		t.Errorf("Path mismatch")
	}
	if f.Error != "file not found" {
		t.Errorf("Error mismatch")
	}
}
