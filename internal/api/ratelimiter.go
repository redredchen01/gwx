package api

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// serviceRate pairs a token-bucket refill rate with a burst size.
// Burst > 1 allows short request spikes without waiting for the full interval.
type serviceRate struct {
	rate  rate.Limit
	burst int
}

// Conservative per-service rate limits.
// These are well below Google's actual quotas to avoid hitting walls.
// Burst values allow short bursts without per-request delay.
var defaultRates = map[string]serviceRate{
	"gmail":         {rate: rate.Every(250 * time.Millisecond), burst: 4},   // 4 QPS (quota ~10)
	"calendar":      {rate: rate.Every(250 * time.Millisecond), burst: 4},   // 4 QPS
	"drive":         {rate: rate.Every(125 * time.Millisecond), burst: 8},   // 8 QPS (quota ~20)
	"sheets":        {rate: rate.Every(1200 * time.Millisecond), burst: 2},  // ~0.8 QPS (quota ~1)
	"docs":          {rate: rate.Every(500 * time.Millisecond), burst: 3},   // 2 QPS
	"tasks":         {rate: rate.Every(250 * time.Millisecond), burst: 4},   // 4 QPS
	"people":        {rate: rate.Every(250 * time.Millisecond), burst: 4},   // 4 QPS
	"chat":          {rate: rate.Every(250 * time.Millisecond), burst: 4},   // 4 QPS
	"analytics":     {rate: rate.Every(500 * time.Millisecond), burst: 3},   // 2 QPS (GA4 quota: 10 concurrent)
	"searchconsole": {rate: rate.Every(500 * time.Millisecond), burst: 3},   // 2 QPS (GSC quota: ~5 QPS)
	"slides":        {rate: rate.Every(500 * time.Millisecond), burst: 3},   // 2 QPS (Slides quota: ~5 QPS)
	"forms":         {rate: rate.Every(500 * time.Millisecond), burst: 3},   // 2 QPS
	"bigquery":      {rate: rate.Every(500 * time.Millisecond), burst: 5},   // 2 QPS (BQ quota varies)
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
	for svc, sr := range defaultRates {
		rl.limiters[svc] = rate.NewLimiter(sr.rate, sr.burst)
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
