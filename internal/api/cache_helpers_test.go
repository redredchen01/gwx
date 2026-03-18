package api

import (
	"strings"
	"testing"
)

func TestCacheKey_Format(t *testing.T) {
	key := cacheKey("sheets", "ReadRange", "spreadsheet-id", "Sheet1!A1:B10")
	// 格式必須是 "service:method:<16 hex chars>"
	parts := strings.Split(key, ":")
	if len(parts) != 3 {
		t.Fatalf("cacheKey format wrong: got %q (want 3 colon-separated parts)", key)
	}
	if parts[0] != "sheets" {
		t.Errorf("service part = %q, want 'sheets'", parts[0])
	}
	if parts[1] != "ReadRange" {
		t.Errorf("method part = %q, want 'ReadRange'", parts[1])
	}
	if len(parts[2]) != 16 {
		t.Errorf("hash part len = %d, want 16 hex chars", len(parts[2]))
	}
}

func TestCacheKey_Deterministic(t *testing.T) {
	k1 := cacheKey("drive", "ListFiles", "id-abc", 10)
	k2 := cacheKey("drive", "ListFiles", "id-abc", 10)
	if k1 != k2 {
		t.Errorf("same params produced different keys: %q vs %q", k1, k2)
	}
}

func TestCacheKey_DifferentParams(t *testing.T) {
	k1 := cacheKey("drive", "ListFiles", "id-abc", 10)
	k2 := cacheKey("drive", "ListFiles", "id-abc", 20)
	if k1 == k2 {
		t.Errorf("different params produced same key: %q", k1)
	}
}

func TestCacheKey_DifferentServices(t *testing.T) {
	k1 := cacheKey("sheets", "Get", "id")
	k2 := cacheKey("drive", "Get", "id")
	if k1 == k2 {
		t.Errorf("different services produced same key")
	}
}

func TestCacheKey_NoParams(t *testing.T) {
	key := cacheKey("gmail", "ListLabels")
	parts := strings.Split(key, ":")
	if len(parts) != 3 {
		t.Fatalf("no-param cacheKey format wrong: %q", key)
	}
	if len(parts[2]) != 16 {
		t.Errorf("hash len = %d, want 16", len(parts[2]))
	}
}
