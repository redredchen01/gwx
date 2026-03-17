package api

import (
	"context"
	"net/http"
	"sync"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// Client manages Google API access with resilience layers.
type Client struct {
	tokenSource oauth2.TokenSource
	rateLimiter *ServiceRateLimiter
	breakers    map[string]*CircuitBreaker
	mu          sync.Mutex
}

// NewClient creates an API client with the given token source.
func NewClient(ts oauth2.TokenSource) *Client {
	return &Client{
		tokenSource: ts,
		rateLimiter: NewServiceRateLimiter(),
		breakers:    make(map[string]*CircuitBreaker),
	}
}

// breaker returns or creates a circuit breaker for the service.
func (c *Client) breaker(service string) *CircuitBreaker {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cb, ok := c.breakers[service]; ok {
		return cb
	}
	cb := NewCircuitBreaker(service)
	c.breakers[service] = cb
	return cb
}

// HTTPClient returns an *http.Client wired with the full transport chain:
// BaseTransport → OAuth2Transport → RetryTransport
func (c *Client) HTTPClient(service string) *http.Client {
	cb := c.breaker(service)
	base := NewBaseTransport()

	oauthTransport := &oauth2.Transport{
		Source: c.tokenSource,
		Base:   base,
	}

	retryTransport := &RetryTransport{
		Base: oauthTransport,
		CB:   cb,
	}

	return &http.Client{Transport: retryTransport}
}

// ClientOptions returns google API client options for the given service.
// Rate limiting is NOT done here — callers must use WaitRate() before each API call.
func (c *Client) ClientOptions(ctx context.Context, service string) ([]option.ClientOption, error) {
	httpClient := c.HTTPClient(service)
	return []option.ClientOption{
		option.WithHTTPClient(httpClient),
	}, nil
}

// WaitRate blocks until the rate limiter allows a request for the service.
func (c *Client) WaitRate(ctx context.Context, service string) error {
	return c.rateLimiter.Wait(ctx, service)
}

// IsCircuitOpen checks if the circuit breaker for a service is open.
func (c *Client) IsCircuitOpen(service string) bool {
	return c.breaker(service).IsOpen()
}
