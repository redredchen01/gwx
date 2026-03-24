package api

import (
	"net/http"
	"testing"
	"time"
)

// --- Cache additional tests (not duplicating existing cache_test.go) ---

func TestCache_InvalidatePrefix_NoMatch(t *testing.T) {
	c := NewCache(CacheConfig{MaxEntries: 100, DefaultTTL: time.Minute})
	c.Set("gmail:list:abc", "v1", 0)
	c.InvalidatePrefix("sheets:")
	if c.Len() != 1 {
		t.Errorf("no match prefix should not remove entries, got %d", c.Len())
	}
}

func TestCache_InvalidatePrefix_Empty(t *testing.T) {
	c := NewCache(CacheConfig{MaxEntries: 100, DefaultTTL: time.Minute})
	c.Set("gmail:list:abc", "v1", 0)
	c.Set("drive:list:def", "v2", 0)
	// Empty prefix matches everything
	c.InvalidatePrefix("")
	if c.Len() != 0 {
		t.Errorf("empty prefix should invalidate all entries, got %d", c.Len())
	}
}

func TestCache_Invalidate_NotPresent(t *testing.T) {
	c := NewCache(CacheConfig{MaxEntries: 100, DefaultTTL: time.Minute})
	c.Set("key1", "val", 0)
	c.Invalidate("nonexistent")
	if c.Len() != 1 {
		t.Errorf("invalidating non-existent key should not change length, got %d", c.Len())
	}
}

func TestCache_Update_ExistingKey(t *testing.T) {
	c := NewCache(CacheConfig{MaxEntries: 100, DefaultTTL: time.Minute})
	c.Set("key", "v1", 0)
	c.Set("key", "v2", 0)
	if c.Len() != 1 {
		t.Errorf("updating existing key should not increase length, got %d", c.Len())
	}
	val, ok := c.Get("key")
	if !ok || val != "v2" {
		t.Errorf("updated key should return new value, got %v", val)
	}
}

func TestCache_DefaultConfig(t *testing.T) {
	c := NewCache(CacheConfig{})
	if c.config.MaxEntries != 256 {
		t.Errorf("default MaxEntries = %d, want 256", c.config.MaxEntries)
	}
	if c.config.DefaultTTL != 5*time.Minute {
		t.Errorf("default TTL = %v, want 5m", c.config.DefaultTTL)
	}
}

func TestCache_NegativeMaxEntries(t *testing.T) {
	c := NewCache(CacheConfig{MaxEntries: -1})
	if c.config.MaxEntries != 256 {
		t.Errorf("negative MaxEntries should default to 256, got %d", c.config.MaxEntries)
	}
}

func TestCache_NegativeTTL(t *testing.T) {
	c := NewCache(CacheConfig{DefaultTTL: -1})
	if c.config.DefaultTTL != 5*time.Minute {
		t.Errorf("negative DefaultTTL should default to 5m, got %v", c.config.DefaultTTL)
	}
}

func TestCache_EmptyLen(t *testing.T) {
	c := NewCache(CacheConfig{})
	if c.Len() != 0 {
		t.Errorf("new cache Len() = %d, want 0", c.Len())
	}
}

// --- cacheKey additional tests ---

func TestCacheKey_ComplexParams(t *testing.T) {
	k1 := cacheKey("sheets", "ReadRange", "id123", "Sheet1!A1:B10")
	k2 := cacheKey("sheets", "ReadRange", "id123", "Sheet1!A1:C10")
	if k1 == k2 {
		t.Error("different range params should produce different keys")
	}
}

func TestCacheKey_EmptyParams(t *testing.T) {
	key := cacheKey("gmail", "List")
	if key == "" {
		t.Error("cacheKey should not return empty string")
	}
}

func TestCacheKey_EmptyService(t *testing.T) {
	key := cacheKey("", "Method")
	if key == "" {
		t.Error("cacheKey should not return empty string")
	}
}

// --- CircuitBreaker additional tests (not duplicating circuitbreaker_test.go) ---

func TestCircuitBreaker_RecordSuccess_Resets_AfterPartialFailures(t *testing.T) {
	cb := NewCircuitBreaker("test_svc")
	// Record some failures (not enough to open)
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	cb.RecordSuccess()
	if cb.State() != "closed" {
		t.Error("circuit should be closed after success")
	}
	// Failures should be reset, so threshold-1 more failures should not open
	for i := 0; i < cbFailureThreshold-1; i++ {
		cb.RecordFailure()
	}
	if cb.State() != "closed" {
		t.Error("circuit should still be closed (threshold-1 failures after reset)")
	}
}

func TestCircuitBreaker_NewNotNil(t *testing.T) {
	cb := NewCircuitBreaker("svc")
	if cb == nil {
		t.Fatal("NewCircuitBreaker returned nil")
	}
}

// --- NewBaseTransport ---

func TestNewBaseTransport(t *testing.T) {
	tr := NewBaseTransport()
	if tr == nil {
		t.Fatal("NewBaseTransport returned nil")
	}
	if tr.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig should not be nil")
	}
	if tr.TLSClientConfig.MinVersion == 0 {
		t.Error("MinVersion should be set")
	}
	if tr.ResponseHeaderTimeout == 0 {
		t.Error("ResponseHeaderTimeout should be set")
	}
}

// --- calculateBackoff additional (not duplicating transport_test.go) ---

func TestCalculateBackoff_NoHeader_PositiveBackoff(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	d := calculateBackoff(resp, 0)
	if d <= 0 {
		t.Errorf("expected positive backoff, got %v", d)
	}
}

func TestCalculateBackoff_ExponentialGrowth(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	d0 := calculateBackoff(resp, 0)
	d2 := calculateBackoff(resp, 2)
	// attempt 2 should be roughly 4x attempt 0 (with jitter)
	if d2 < d0 {
		t.Errorf("backoff should grow with attempt: d0=%v, d2=%v", d0, d2)
	}
}

func TestCalculateBackoff_InvalidRetryAfter(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Set("Retry-After", "not-a-number")
	d := calculateBackoff(resp, 0)
	if d <= 0 {
		t.Errorf("invalid Retry-After should fall back to exponential, got %v", d)
	}
}

func TestCalculateBackoff_ZeroRetryAfter(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	resp.Header.Set("Retry-After", "0")
	d := calculateBackoff(resp, 0)
	if d <= 0 {
		t.Errorf("Retry-After: 0 should fall back to exponential, got %v", d)
	}
}

// --- CircuitOpenError additional ---

func TestCircuitOpenError_ContainsRetryHint(t *testing.T) {
	err := &CircuitOpenError{Service: "sheets"}
	msg := err.Error()
	if !containsSubstr(msg, "retry") || !containsSubstr(msg, "30s") {
		t.Errorf("error should mention retry and 30s, got: %s", msg)
	}
}

// --- RetryTransport base() ---

func TestRetryTransport_BaseFallback(t *testing.T) {
	rt := &RetryTransport{Base: nil}
	got := rt.base()
	if got != http.DefaultTransport {
		t.Error("nil Base should fall back to http.DefaultTransport")
	}
}

func TestRetryTransport_BaseSet(t *testing.T) {
	custom := &http.Transport{}
	rt := &RetryTransport{Base: custom}
	got := rt.base()
	if got != custom {
		t.Error("set Base should be returned")
	}
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
