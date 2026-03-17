package api

import (
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_StartsClose(t *testing.T) {
	cb := NewCircuitBreaker("test")
	if cb.IsOpen() {
		t.Fatal("new circuit breaker should be closed")
	}
	if cb.State() != "closed" {
		t.Fatalf("expected state 'closed', got %q", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker("test")

	// 4 failures: still closed
	for i := 0; i < cbFailureThreshold-1; i++ {
		opened := cb.RecordFailure()
		if opened {
			t.Fatalf("should not open after %d failures", i+1)
		}
	}
	if cb.IsOpen() {
		t.Fatal("should still be closed before threshold")
	}

	// 5th failure: opens
	opened := cb.RecordFailure()
	if !opened {
		t.Fatal("should have opened on threshold failure")
	}
	if !cb.IsOpen() {
		t.Fatal("should be open after threshold")
	}
	if cb.State() != "open" {
		t.Fatalf("expected state 'open', got %q", cb.State())
	}
}

func TestCircuitBreaker_SuccessResets(t *testing.T) {
	cb := NewCircuitBreaker("test")

	// Get close to threshold
	for i := 0; i < cbFailureThreshold-1; i++ {
		cb.RecordFailure()
	}

	// Success resets
	cb.RecordSuccess()

	// Now need full threshold again
	for i := 0; i < cbFailureThreshold-1; i++ {
		cb.RecordFailure()
	}
	if cb.IsOpen() {
		t.Fatal("should still be closed after reset + threshold-1 failures")
	}
}

func TestCircuitBreaker_AutoRecovery(t *testing.T) {
	// Use a breaker with very short reset time for testing
	cb := &CircuitBreaker{service: "test"}

	// Open the circuit
	for i := 0; i < cbFailureThreshold; i++ {
		cb.RecordFailure()
	}
	if !cb.IsOpen() {
		t.Fatal("should be open")
	}

	// Manipulate lastFailure to simulate time passing
	cb.mu.Lock()
	cb.lastFailure = time.Now().Add(-cbResetAfter - time.Second)
	cb.mu.Unlock()

	// Should auto-recover
	if cb.IsOpen() {
		t.Fatal("should have auto-recovered after reset window")
	}
	if cb.failures != 0 {
		t.Fatal("failures should be reset after recovery")
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker("concurrent")
	var wg sync.WaitGroup

	// Hammer it from multiple goroutines
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			cb.RecordFailure()
		}()
		go func() {
			defer wg.Done()
			cb.RecordSuccess()
		}()
		go func() {
			defer wg.Done()
			cb.IsOpen()
		}()
	}
	wg.Wait()
	// No race condition = pass (run with -race flag)
}

func TestCircuitBreaker_StaysOpenUntilReset(t *testing.T) {
	cb := NewCircuitBreaker("test")

	for i := 0; i < cbFailureThreshold; i++ {
		cb.RecordFailure()
	}

	// Additional failures don't re-trigger "just opened" return
	opened := cb.RecordFailure()
	if opened {
		t.Fatal("should not report 'just opened' when already open")
	}
	if !cb.IsOpen() {
		t.Fatal("should remain open")
	}
}

func TestCircuitBreaker_String(t *testing.T) {
	cb := NewCircuitBreaker("gmail")
	s := cb.String()
	if s == "" {
		t.Fatal("String() should not be empty")
	}
	// Should contain service name
	if !containsStr(s, "gmail") {
		t.Fatalf("String() should contain service name, got %q", s)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
