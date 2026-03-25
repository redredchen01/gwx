package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// Client manages Google API access with resilience layers.
type Client struct {
	tokenSource oauth2.TokenSource
	rateLimiter *ServiceRateLimiter
	breakers    map[string]*CircuitBreaker
	services    map[string]any
	cache       *Cache
	NoCache     bool
	endpoint    string       // override API endpoint for testing
	httpClient  *http.Client // override HTTP client for testing
	mu          sync.Mutex

	// Cached HTTP clients per service — avoids recreating transports on every call.
	httpClients   map[string]*http.Client
	baseTransport *http.Transport // shared connection pool across all services
}

// NewClient creates an API client with the given token source.
func NewClient(ts oauth2.TokenSource) *Client {
	return &Client{
		tokenSource: ts,
		rateLimiter: NewServiceRateLimiter(),
		breakers:    make(map[string]*CircuitBreaker),
		services:    make(map[string]any),
		httpClients: make(map[string]*http.Client),
		cache:       NewCache(CacheConfig{MaxEntries: 1024, DefaultTTL: 5 * time.Minute}),
	}
}

// breaker returns or creates a circuit breaker for the service.
func (c *Client) breaker(service string) *CircuitBreaker {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.breakerLocked(service)
}

// breakerLocked is the lock-free variant; caller must hold c.mu.
func (c *Client) breakerLocked(service string) *CircuitBreaker {
	if cb, ok := c.breakers[service]; ok {
		return cb
	}
	cb := NewCircuitBreaker(service)
	c.breakers[service] = cb
	return cb
}

// HTTPClient returns an *http.Client wired with the full transport chain:
// SharedTransport → OAuth2Transport → RetryTransport
// Clients are cached per service so connections are reused across calls.
// If the client was created with NewTestClient, returns the injected test client.
func (c *Client) HTTPClient(service string) *http.Client {
	if c.httpClient != nil {
		return c.httpClient
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if client, ok := c.httpClients[service]; ok {
		return client
	}

	cb := c.breakerLocked(service)
	client := &http.Client{
		Transport: &RetryTransport{
			Base: &oauth2.Transport{
				Source: c.tokenSource,
				Base:   c.sharedTransport(),
			},
			CB: cb,
		},
	}
	c.httpClients[service] = client
	return client
}

// sharedTransport returns a single optimized *http.Transport shared across all
// services. Must be called with c.mu held.
func (c *Client) sharedTransport() *http.Transport {
	if c.baseTransport == nil {
		c.baseTransport = newOptimizedTransport()
	}
	return c.baseTransport
}

// ClientOptions returns google API client options for the given service.
// Rate limiting is NOT done here — callers must use WaitRate() before each API call.
func (c *Client) ClientOptions(ctx context.Context, service string) ([]option.ClientOption, error) {
	httpClient := c.HTTPClient(service)
	opts := []option.ClientOption{
		option.WithHTTPClient(httpClient),
	}
	if c.endpoint != "" {
		opts = append(opts, option.WithEndpoint(c.endpoint))
	}
	return opts, nil
}

// NewTestClient creates an API client suitable for testing.
// It uses the provided HTTP client and endpoint, bypasses OAuth, rate limiting,
// circuit breakers, and caching so tests run fast and deterministically.
func NewTestClient(httpClient *http.Client, endpoint string) *Client {
	return &Client{
		rateLimiter: NewServiceRateLimiter(),
		breakers:    make(map[string]*CircuitBreaker),
		services:    make(map[string]any),
		httpClients: make(map[string]*http.Client),
		cache:       NewCache(CacheConfig{MaxEntries: 256, DefaultTTL: 5 * time.Minute}),
		NoCache:     true,
		endpoint:    endpoint,
		httpClient:  httpClient,
	}
}

// WaitRate blocks until the rate limiter allows a request for the service.
func (c *Client) WaitRate(ctx context.Context, service string) error {
	return c.rateLimiter.Wait(ctx, service)
}

// IsCircuitOpen checks if the circuit breaker for a service is open.
func (c *Client) IsCircuitOpen(service string) bool {
	return c.breaker(service).IsOpen()
}

// ServiceInit combines WaitRate + ClientOptions into a single call.
// Reduces the 3-line boilerplate in every API method to 1 line.
//
// Usage:
//
//	opts, err := ss.client.ServiceInit(ctx, "sheets")
//	if err != nil {
//	    return nil, err
//	}
//	svc, err := sheets.NewService(ctx, opts...)
func (c *Client) ServiceInit(ctx context.Context, service string) ([]option.ClientOption, error) {
	if err := c.WaitRate(ctx, service); err != nil {
		return nil, err
	}
	return c.ClientOptions(ctx, service)
}

// GetOrCreateService caches expensive Google API service objects per Client instance.
// It is safe for repeated sequential command use within the same process.
func (c *Client) GetOrCreateService(key string, factory func() (any, error)) (any, error) {
	c.mu.Lock()
	cached := c.services[key]
	c.mu.Unlock()
	if cached != nil {
		return cached, nil
	}

	created, err := factory()
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if cached = c.services[key]; cached != nil {
		return cached, nil
	}
	c.services[key] = created
	return created, nil
}
