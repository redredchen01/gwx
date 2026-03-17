package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// mockTransport returns canned responses for testing.
type mockTransport struct {
	responses []*http.Response
	calls     int
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	idx := m.calls
	m.calls++
	if idx >= len(m.responses) {
		idx = len(m.responses) - 1
	}
	return m.responses[idx], nil
}

func newResponse(status int, headers map[string]string) *http.Response {
	resp := &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
	}
	for k, v := range headers {
		resp.Header.Set(k, v)
	}
	return resp
}

func TestRetryTransport_SuccessNoRetry(t *testing.T) {
	mock := &mockTransport{
		responses: []*http.Response{newResponse(200, nil)},
	}
	rt := &RetryTransport{Base: mock}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if mock.calls != 1 {
		t.Fatalf("expected 1 call, got %d", mock.calls)
	}
}

func TestRetryTransport_429Retry(t *testing.T) {
	mock := &mockTransport{
		responses: []*http.Response{
			newResponse(429, map[string]string{"Retry-After": "1"}),
			newResponse(429, map[string]string{"Retry-After": "1"}),
			newResponse(200, nil),
		},
	}
	rt := &RetryTransport{Base: mock}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 after retries, got %d", resp.StatusCode)
	}
	if mock.calls != 3 {
		t.Fatalf("expected 3 calls (1 original + 2 retries), got %d", mock.calls)
	}
}

func TestRetryTransport_429ExhaustedRetries(t *testing.T) {
	// All 429s — should exhaust retries and return 429
	responses := make([]*http.Response, maxRetry429+2)
	for i := range responses {
		responses[i] = newResponse(429, map[string]string{"Retry-After": "0"})
	}
	mock := &mockTransport{responses: responses}
	rt := &RetryTransport{Base: mock}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 429 {
		t.Fatalf("expected 429 after exhausted retries, got %d", resp.StatusCode)
	}
}

func TestRetryTransport_5xxRetry(t *testing.T) {
	mock := &mockTransport{
		responses: []*http.Response{
			newResponse(503, nil),
			newResponse(200, nil),
		},
	}
	rt := &RetryTransport{Base: mock}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 after 5xx retry, got %d", resp.StatusCode)
	}
	if mock.calls != 2 {
		t.Fatalf("expected 2 calls, got %d", mock.calls)
	}
}

func TestRetryTransport_4xxNoRetry(t *testing.T) {
	mock := &mockTransport{
		responses: []*http.Response{newResponse(404, nil)},
	}
	rt := &RetryTransport{Base: mock}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	if mock.calls != 1 {
		t.Fatalf("expected 1 call (no retry for 4xx), got %d", mock.calls)
	}
}

func TestRetryTransport_BodyReplay(t *testing.T) {
	mock := &mockTransport{
		responses: []*http.Response{
			newResponse(503, nil),
			newResponse(200, nil),
		},
	}
	rt := &RetryTransport{Base: mock}

	body := strings.NewReader(`{"key": "value"}`)
	req, _ := http.NewRequest("POST", "http://example.com", body)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if mock.calls != 2 {
		t.Fatalf("expected 2 calls (body should be replayed), got %d", mock.calls)
	}
}

func TestRetryTransport_CircuitBreakerIntegration(t *testing.T) {
	cb := NewCircuitBreaker("test")

	// Open the circuit
	for i := 0; i < cbFailureThreshold; i++ {
		cb.RecordFailure()
	}

	mock := &mockTransport{
		responses: []*http.Response{newResponse(200, nil)},
	}
	rt := &RetryTransport{Base: mock, CB: cb}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected circuit open error")
	}
	if _, ok := err.(*CircuitOpenError); !ok {
		t.Fatalf("expected CircuitOpenError, got %T: %v", err, err)
	}
	if mock.calls != 0 {
		t.Fatalf("should not have made any calls with open circuit, got %d", mock.calls)
	}
}

func TestRetryTransport_CircuitBreakerRecordsSuccess(t *testing.T) {
	cb := NewCircuitBreaker("test")
	// Add some failures (but not enough to open)
	for i := 0; i < cbFailureThreshold-2; i++ {
		cb.RecordFailure()
	}

	mock := &mockTransport{
		responses: []*http.Response{newResponse(200, nil)},
	}
	rt := &RetryTransport{Base: mock, CB: cb}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Success should have reset failures
	if cb.failures != 0 {
		t.Fatalf("expected failures=0 after success, got %d", cb.failures)
	}
}

func TestRetryTransport_ContextCancellation(t *testing.T) {
	mock := &mockTransport{
		responses: []*http.Response{
			newResponse(429, map[string]string{"Retry-After": "60"}),
			newResponse(200, nil),
		},
	}
	rt := &RetryTransport{Base: mock}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected context deadline error")
	}
}

func TestCalculateBackoff_RetryAfterSeconds(t *testing.T) {
	resp := newResponse(429, map[string]string{"Retry-After": "5"})
	d := calculateBackoff(resp, 0)
	if d != 5*time.Second {
		t.Fatalf("expected 5s, got %v", d)
	}
}

func TestCalculateBackoff_RetryAfterDate(t *testing.T) {
	future := time.Now().Add(3 * time.Second).UTC().Format(http.TimeFormat)
	resp := newResponse(429, map[string]string{"Retry-After": future})
	d := calculateBackoff(resp, 0)
	if d < 2*time.Second || d > 4*time.Second {
		t.Fatalf("expected ~3s, got %v", d)
	}
}

func TestCalculateBackoff_ExponentialFallback(t *testing.T) {
	resp := newResponse(429, nil) // no Retry-After header
	d0 := calculateBackoff(resp, 0)
	d1 := calculateBackoff(resp, 1)

	// attempt 0: ~1s, attempt 1: ~2s (with jitter)
	if d0 < 500*time.Millisecond || d0 > 2*time.Second {
		t.Fatalf("attempt 0 backoff out of range: %v", d0)
	}
	if d1 < 1*time.Second || d1 > 4*time.Second {
		t.Fatalf("attempt 1 backoff out of range: %v", d1)
	}
}

func TestCircuitOpenError_Message(t *testing.T) {
	err := &CircuitOpenError{Service: "gmail"}
	msg := err.Error()
	if !containsStr(msg, "gmail") {
		t.Fatalf("error message should contain service name, got %q", msg)
	}
}
