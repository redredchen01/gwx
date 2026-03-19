package api

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Conservative per-service rate limits.
// These are well below Google's actual quotas to avoid hitting walls.
var defaultRates = map[string]rate.Limit{
	"gmail":    rate.Every(250 * time.Millisecond), // 4 QPS (quota ~10)
	"calendar": rate.Every(250 * time.Millisecond), // 4 QPS
	"drive":    rate.Every(125 * time.Millisecond), // 8 QPS (quota ~20)
	"sheets":   rate.Every(1200 * time.Millisecond), // ~0.8 QPS (quota ~1)
	"docs":     rate.Every(500 * time.Millisecond),  // 2 QPS
	"tasks":    rate.Every(250 * time.Millisecond),  // 4 QPS
	"people":   rate.Every(250 * time.Millisecond),  // 4 QPS
	"chat":          rate.Every(250 * time.Millisecond),  // 4 QPS
	"analytics":     rate.Every(500 * time.Millisecond),  // 2 QPS (GA4 quota: 10 concurrent)
	"searchconsole": rate.Every(500 * time.Millisecond),  // 2 QPS (GSC quota: ~5 QPS)
}

// ServiceRateLimiter manages per-service token bucket rate limiters.
type ServiceRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

// NewServiceRateLimiter creates a rate limiter with default service limits.
func NewServiceRateLimiter() *ServiceRateLimiter {
	rl := &ServiceRateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
	for svc, r := range defaultRates {
		rl.limiters[svc] = rate.NewLimiter(r, 1) // burst=1, strictly conservative
	}
	return rl
}

// Wait blocks until the rate limiter allows a request for the given service.
// For unknown services, it passes through immediately.
func (rl *ServiceRateLimiter) Wait(ctx context.Context, service string) error {
	rl.mu.Lock()
	limiter, ok := rl.limiters[service]
	rl.mu.Unlock()
	if !ok {
		return nil
	}
	return limiter.Wait(ctx)
}
