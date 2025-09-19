package utils

import (
	"sync"
	"time"
)

// PostbackCircuitBreakers manages circuit breakers for different ad networks
type PostbackCircuitBreakers struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

var (
	postbackBreakers *PostbackCircuitBreakers
	postbackOnce     sync.Once
)

// GetPostbackBreakers returns singleton instance of postback circuit breakers
func GetPostbackBreakers() *PostbackCircuitBreakers {
	postbackOnce.Do(func() {
		postbackBreakers = &PostbackCircuitBreakers{
			breakers: make(map[string]*CircuitBreaker),
		}

		// Initialize circuit breakers for each ad network
		config := CircuitBreakerConfig{
			MaxFailures:  3,                // Open after 3 consecutive failures
			ResetTimeout: 30 * time.Second, // Try again after 30 seconds
		}

		postbackBreakers.breakers["propeller"] = NewCircuitBreaker("PropellerAds", config)
		postbackBreakers.breakers["galaksion"] = NewCircuitBreaker("Galaksion", config)
		postbackBreakers.breakers["popcash"] = NewCircuitBreaker("Popcash", config)
		postbackBreakers.breakers["clickadilla"] = NewCircuitBreaker("ClickAdilla", config)
	})
	return postbackBreakers
}

// Execute runs a postback function with circuit breaker protection
func (pb *PostbackCircuitBreakers) Execute(network string, fn func() error) error {
	pb.mu.RLock()
	breaker, exists := pb.breakers[network]
	pb.mu.RUnlock()

	if !exists {
		// If breaker doesn't exist, create it on the fly
		pb.mu.Lock()
		breaker = NewCircuitBreaker(network, CircuitBreakerConfig{
			MaxFailures:  3,
			ResetTimeout: 30 * time.Second,
		})
		pb.breakers[network] = breaker
		pb.mu.Unlock()
	}

	return breaker.Execute(fn)
}

// GetStats returns statistics for all circuit breakers
func (pb *PostbackCircuitBreakers) GetStats() map[string]interface{} {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, breaker := range pb.breakers {
		stats[name] = breaker.GetStats()
	}

	return stats
}

// Reset resets a specific circuit breaker
func (pb *PostbackCircuitBreakers) Reset(network string) bool {
	pb.mu.RLock()
	breaker, exists := pb.breakers[network]
	pb.mu.RUnlock()

	if exists {
		breaker.Reset()
		return true
	}
	return false
}

// ResetAll resets all circuit breakers
func (pb *PostbackCircuitBreakers) ResetAll() {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	for _, breaker := range pb.breakers {
		breaker.Reset()
	}
}
