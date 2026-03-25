package api

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiter_KnownServiceThrottles(t *testing.T) {
	rl := NewServiceRateLimiter()
	ctx := context.Background()

	// Sheets has burst=2. Exhaust the burst bucket first.
	for i := 0; i < 2; i++ {
		if err := rl.Wait(ctx, "sheets"); err != nil {
			t.Fatalf("burst wait %d failed: %v", i, err)
		}
	}

	// Next call must wait for a token refill (~1200ms interval)
	start := time.Now()
	if err := rl.Wait(ctx, "sheets"); err != nil {
		t.Fatalf("throttled wait failed: %v", err)
	}
	elapsed := time.Since(start)

	// Should have waited at least some time (sheets has the slowest rate)
	if elapsed < 500*time.Millisecond {
		t.Fatalf("expected throttling for sheets, but elapsed only %v", elapsed)
	}
}

func TestRateLimiter_UnknownServicePassthrough(t *testing.T) {
	rl := NewServiceRateLimiter()
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 10; i++ {
		if err := rl.Wait(ctx, "unknown_service"); err != nil {
			t.Fatalf("unknown service wait failed: %v", err)
		}
	}
	elapsed := time.Since(start)

	// Unknown services should pass through instantly
	if elapsed > 100*time.Millisecond {
		t.Fatalf("unknown service should not throttle, but took %v", elapsed)
	}
}

func TestRateLimiter_ContextCancellation(t *testing.T) {
	rl := NewServiceRateLimiter()

	// Exhaust all burst tokens (sheets burst=2)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		rl.Wait(ctx, "sheets") //nolint:errcheck
	}

	// Cancel context before next call completes (must wait for refill)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx, "sheets")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestRateLimiter_DifferentServicesIndependent(t *testing.T) {
	rl := NewServiceRateLimiter()
	ctx := context.Background()

	// Exhaust gmail token
	rl.Wait(ctx, "gmail") //nolint:errcheck

	// Drive should still be available immediately
	start := time.Now()
	if err := rl.Wait(ctx, "drive"); err != nil {
		t.Fatalf("drive wait failed: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > 50*time.Millisecond {
		t.Fatalf("different services should be independent, drive took %v", elapsed)
	}
}

func TestRateLimiter_AllDefaultServicesRegistered(t *testing.T) {
	expected := []string{"gmail", "calendar", "drive", "sheets", "docs", "tasks", "people", "chat"}
	for _, svc := range expected {
		if _, ok := defaultRates[svc]; !ok {
			t.Errorf("missing default rate for service %q", svc)
		}
	}
}
