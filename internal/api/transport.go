package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"time"
)

const (
	maxRetry429           = 3
	maxRetry5xx           = 2
	maxBodyRead           = 1 << 20 // 1 MB for draining
	maxRetryDelay         = 30 * time.Second
	defaultDialTimeout    = 10 * time.Second
	defaultTLSHandshake   = 10 * time.Second
	defaultIdleConnTTL    = 120 * time.Second
	defaultExpectContinue = 1 * time.Second
	defaultResponseHeader = 30 * time.Second
)

// RetryTransport wraps an http.RoundTripper with automatic retry for
// rate-limit (429) and server error (5xx) responses.
type RetryTransport struct {
	Base http.RoundTripper
	CB   *CircuitBreaker
}

func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Circuit breaker check
	if t.CB != nil && t.CB.IsOpen() {
		return nil, &CircuitOpenError{Service: t.CB.service}
	}

	// Buffer body for replay
	body, err := ensureReplayableBody(req)
	if err != nil {
		return nil, err
	}

	var resp *http.Response
	var lastErr error

	totalAttempts := 1 + maxRetry429 + maxRetry5xx
	retries429 := 0
	retries5xx := 0

	for attempt := 0; attempt < totalAttempts; attempt++ {
		if body != nil {
			req.Body = io.NopCloser(bytes.NewReader(body))
		}

		resp, lastErr = t.base().RoundTrip(req)
		if lastErr != nil {
			if t.CB != nil {
				t.CB.RecordFailure()
			}
			return nil, lastErr
		}

		switch {
		case resp.StatusCode < 400:
			if t.CB != nil {
				t.CB.RecordSuccess()
			}
			return resp, nil

		case resp.StatusCode == 429 && retries429 < maxRetry429:
			retries429++
			backoff := calculateBackoff(resp, attempt)
			drainBody(resp)
			if err := sleepWithContext(req.Context(), backoff); err != nil {
				return nil, err
			}

		case resp.StatusCode >= 500 && retries5xx < maxRetry5xx:
			retries5xx++
			backoff := time.Duration(attempt+1) * time.Second
			drainBody(resp)
			if err := sleepWithContext(req.Context(), backoff); err != nil {
				return nil, err
			}

		default:
			// 4xx (non-429) or exhausted retries
			if t.CB != nil && resp.StatusCode >= 500 {
				t.CB.RecordFailure()
			}
			return resp, nil
		}
	}

	if t.CB != nil {
		t.CB.RecordFailure()
	}
	if resp != nil {
		return resp, nil
	}
	return nil, lastErr
}

func (t *RetryTransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// calculateBackoff reads Retry-After header or falls back to exponential backoff with jitter.
func calculateBackoff(resp *http.Response, attempt int) time.Duration {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
			return clampDuration(time.Duration(secs) * time.Second)
		}
		if t, err := http.ParseTime(ra); err == nil {
			if d := time.Until(t); d > 0 {
				return clampDuration(d)
			}
		}
	}
	// Exponential backoff: 1s, 2s, 4s... + jitter up to 50%
	base := math.Pow(2, float64(attempt)) * float64(time.Second)
	jitter := rand.Float64() * base * 0.5 //nolint:gosec
	return clampDuration(time.Duration(base + jitter))
}

func ensureReplayableBody(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(data))
	return data, nil
}

func drainBody(resp *http.Response) {
	if resp.Body != nil {
		io.CopyN(io.Discard, resp.Body, maxBodyRead) //nolint:errcheck
		resp.Body.Close()
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// NewBaseTransport creates an http.Transport with TLS 1.2+ and reasonable timeouts.
// Deprecated: Prefer newOptimizedTransport() which tunes pool sizes for Google APIs.
// Kept for backward compatibility with tests and external callers.
func NewBaseTransport() *http.Transport {
	return newOptimizedTransport()
}

// newOptimizedTransport creates an http.Transport tuned for Google API workloads.
func newOptimizedTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	t.DialContext = (&net.Dialer{
		Timeout:   defaultDialTimeout,
		KeepAlive: 30 * time.Second,
	}).DialContext
	t.TLSHandshakeTimeout = defaultTLSHandshake
	t.ResponseHeaderTimeout = defaultResponseHeader
	t.ExpectContinueTimeout = defaultExpectContinue
	t.IdleConnTimeout = defaultIdleConnTTL
	t.MaxIdleConns = 200
	t.MaxIdleConnsPerHost = 20
	return t
}

func clampDuration(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	if d > maxRetryDelay {
		return maxRetryDelay
	}
	return d
}

// CircuitOpenError is returned when the circuit breaker is open.
type CircuitOpenError struct {
	Service string
}

func (e *CircuitOpenError) Error() string {
	return "circuit breaker open for service: " + e.Service + " (retry after 30s)"
}
