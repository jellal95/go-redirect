package utils

import (
	"errors"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	Closed CircuitBreakerState = iota
	Open
	HalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu              sync.RWMutex
	state           CircuitBreakerState
	failureCount    int
	successCount    int
	maxFailures     int
	resetTimeout    time.Duration
	lastFailureTime time.Time
	name            string
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	MaxFailures  int
	ResetTimeout time.Duration
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:        Closed,
		maxFailures:  config.MaxFailures,
		resetTimeout: config.ResetTimeout,
		name:         name,
	}
}

// Execute runs the function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if we should transition from Open to HalfOpen
	if cb.state == Open {
		if time.Since(cb.lastFailureTime) >= cb.resetTimeout {
			cb.state = HalfOpen
			cb.successCount = 0
			LogInfo(LogEntry{
				Type: "circuit_breaker_half_open",
				Extra: map[string]interface{}{
					"breaker_name": cb.name,
					"state":        "half_open",
				},
			})
		} else {
			return errors.New("circuit breaker is open")
		}
	}

	// Execute the function
	err := fn()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}

	return err
}

// onFailure handles failure cases
func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == HalfOpen || cb.failureCount >= cb.maxFailures {
		cb.state = Open
		LogInfo(LogEntry{
			Type: "circuit_breaker_open",
			Extra: map[string]interface{}{
				"breaker_name":  cb.name,
				"failure_count": cb.failureCount,
				"max_failures":  cb.maxFailures,
				"state":         "open",
			},
		})
	}
}

// onSuccess handles success cases
func (cb *CircuitBreaker) onSuccess() {
	cb.failureCount = 0

	if cb.state == HalfOpen {
		cb.successCount++
		// After one successful request in HalfOpen, transition to Closed
		if cb.successCount >= 1 {
			cb.state = Closed
			LogInfo(LogEntry{
				Type: "circuit_breaker_closed",
				Extra: map[string]interface{}{
					"breaker_name": cb.name,
					"state":        "closed",
				},
			})
		}
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	var stateStr string
	switch cb.state {
	case Closed:
		stateStr = "closed"
	case Open:
		stateStr = "open"
	case HalfOpen:
		stateStr = "half_open"
	}

	return map[string]interface{}{
		"name":              cb.name,
		"state":             stateStr,
		"failure_count":     cb.failureCount,
		"success_count":     cb.successCount,
		"max_failures":      cb.maxFailures,
		"reset_timeout_sec": cb.resetTimeout.Seconds(),
		"last_failure":      cb.lastFailureTime,
	}
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = Closed
	cb.failureCount = 0
	cb.successCount = 0

	LogInfo(LogEntry{
		Type: "circuit_breaker_reset",
		Extra: map[string]interface{}{
			"breaker_name": cb.name,
			"state":        "closed",
		},
	})
}
