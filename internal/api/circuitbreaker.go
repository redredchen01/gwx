package api

import (
	"fmt"
	"sync"
	"time"
)

const (
	cbFailureThreshold = 5
	cbResetAfter       = 30 * time.Second
)

// CircuitBreaker implements a simple binary circuit breaker.
// After cbFailureThreshold consecutive failures, it opens and rejects
// requests for cbResetAfter duration before auto-recovering.
type CircuitBreaker struct {
	mu          sync.Mutex
	failures    int
	lastFailure time.Time
	open        bool
	service     string
}

// NewCircuitBreaker creates a circuit breaker for the named service.
func NewCircuitBreaker(service string) *CircuitBreaker {
	return &CircuitBreaker{service: service}
}

// IsOpen returns true if the circuit is open (should reject requests).
// Auto-closes after the reset window elapses.
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.open {
		return false
	}
	if time.Since(cb.lastFailure) >= cbResetAfter {
		cb.open = false
		cb.failures = 0
		return false
	}
	return true
}

// RecordSuccess resets the failure counter and closes the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.open = false
}

// RecordFailure increments the failure counter and opens the circuit
// if the threshold is reached. Returns true if the circuit just opened.
func (cb *CircuitBreaker) RecordFailure() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cbFailureThreshold && !cb.open {
		cb.open = true
		return true
	}
	return false
}

// State returns "open" or "closed".
func (cb *CircuitBreaker) State() string {
	if cb.IsOpen() {
		return "open"
	}
	return "closed"
}

func (cb *CircuitBreaker) String() string {
	return fmt.Sprintf("CircuitBreaker[%s]: %s (failures=%d)", cb.service, cb.State(), cb.failures)
}
