package utils

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds application performance metrics
type Metrics struct {
	mu                sync.RWMutex
	RequestCount      int64            `json:"request_count"`
	ErrorCount        int64            `json:"error_count"`
	RedirectCount     int64            `json:"redirect_count"`
	BlockedCount      int64            `json:"blocked_count"`
	PostbackCount     int64            `json:"postback_count"`
	TotalResponseTime int64            `json:"total_response_time_ms"`
	StatusCodes       map[int]int64    `json:"status_codes"`
	EndpointCounts    map[string]int64 `json:"endpoint_counts"`
	UserAgentCounts   map[string]int64 `json:"user_agent_counts"`
	RefererCounts     map[string]int64 `json:"referer_counts"`
	ProductSelections map[string]int64 `json:"product_selections"`
	LastResetTime     time.Time        `json:"last_reset_time"`
}

var globalMetrics = &Metrics{
	StatusCodes:       make(map[int]int64),
	EndpointCounts:    make(map[string]int64),
	UserAgentCounts:   make(map[string]int64),
	RefererCounts:     make(map[string]int64),
	ProductSelections: make(map[string]int64),
	LastResetTime:     time.Now(),
}

// GetMetrics returns current metrics snapshot
func GetMetrics() *Metrics {
	globalMetrics.mu.RLock()
	defer globalMetrics.mu.RUnlock()

	// Create a snapshot to avoid data races
	snapshot := &Metrics{
		RequestCount:      atomic.LoadInt64(&globalMetrics.RequestCount),
		ErrorCount:        atomic.LoadInt64(&globalMetrics.ErrorCount),
		RedirectCount:     atomic.LoadInt64(&globalMetrics.RedirectCount),
		BlockedCount:      atomic.LoadInt64(&globalMetrics.BlockedCount),
		PostbackCount:     atomic.LoadInt64(&globalMetrics.PostbackCount),
		TotalResponseTime: atomic.LoadInt64(&globalMetrics.TotalResponseTime),
		StatusCodes:       make(map[int]int64),
		EndpointCounts:    make(map[string]int64),
		UserAgentCounts:   make(map[string]int64),
		RefererCounts:     make(map[string]int64),
		ProductSelections: make(map[string]int64),
		LastResetTime:     globalMetrics.LastResetTime,
	}

	// Copy maps
	for k, v := range globalMetrics.StatusCodes {
		snapshot.StatusCodes[k] = v
	}
	for k, v := range globalMetrics.EndpointCounts {
		snapshot.EndpointCounts[k] = v
	}
	for k, v := range globalMetrics.UserAgentCounts {
		snapshot.UserAgentCounts[k] = v
	}
	for k, v := range globalMetrics.RefererCounts {
		snapshot.RefererCounts[k] = v
	}
	for k, v := range globalMetrics.ProductSelections {
		snapshot.ProductSelections[k] = v
	}

	return snapshot
}

// IncrementRequest increments request counter
func IncrementRequest() {
	atomic.AddInt64(&globalMetrics.RequestCount, 1)
}

// IncrementError increments error counter
func IncrementError() {
	atomic.AddInt64(&globalMetrics.ErrorCount, 1)
}

// IncrementRedirect increments redirect counter
func IncrementRedirect() {
	atomic.AddInt64(&globalMetrics.RedirectCount, 1)
}

// IncrementBlocked increments blocked request counter
func IncrementBlocked() {
	atomic.AddInt64(&globalMetrics.BlockedCount, 1)
}

// IncrementPostback increments postback counter
func IncrementPostback() {
	atomic.AddInt64(&globalMetrics.PostbackCount, 1)
}

// RecordResponseTime adds response time to total
func RecordResponseTime(ms int64) {
	atomic.AddInt64(&globalMetrics.TotalResponseTime, ms)
}

// RecordStatusCode increments status code counter
func RecordStatusCode(code int) {
	globalMetrics.mu.Lock()
	globalMetrics.StatusCodes[code]++
	globalMetrics.mu.Unlock()
}

// RecordEndpoint increments endpoint counter
func RecordEndpoint(endpoint string) {
	globalMetrics.mu.Lock()
	globalMetrics.EndpointCounts[endpoint]++
	globalMetrics.mu.Unlock()
}

// RecordUserAgent increments user agent counter (truncated)
func RecordUserAgent(ua string) {
	if len(ua) > 50 {
		ua = ua[:50] + "..."
	}
	globalMetrics.mu.Lock()
	globalMetrics.UserAgentCounts[ua]++
	globalMetrics.mu.Unlock()
}

// RecordReferer increments referer counter (truncated)
func RecordReferer(ref string) {
	if len(ref) > 50 {
		ref = ref[:50] + "..."
	}
	globalMetrics.mu.Lock()
	globalMetrics.RefererCounts[ref]++
	globalMetrics.mu.Unlock()
}

// RecordProductSelection increments product selection counter
func RecordProductSelection(productName string) {
	globalMetrics.mu.Lock()
	globalMetrics.ProductSelections[productName]++
	globalMetrics.mu.Unlock()
}

// ResetMetrics resets all counters
func ResetMetrics() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	atomic.StoreInt64(&globalMetrics.RequestCount, 0)
	atomic.StoreInt64(&globalMetrics.ErrorCount, 0)
	atomic.StoreInt64(&globalMetrics.RedirectCount, 0)
	atomic.StoreInt64(&globalMetrics.BlockedCount, 0)
	atomic.StoreInt64(&globalMetrics.PostbackCount, 0)
	atomic.StoreInt64(&globalMetrics.TotalResponseTime, 0)

	globalMetrics.StatusCodes = make(map[int]int64)
	globalMetrics.EndpointCounts = make(map[string]int64)
	globalMetrics.UserAgentCounts = make(map[string]int64)
	globalMetrics.RefererCounts = make(map[string]int64)
	globalMetrics.ProductSelections = make(map[string]int64)
	globalMetrics.LastResetTime = time.Now()
}

// AverageResponseTime calculates average response time
func (m *Metrics) AverageResponseTime() float64 {
	if m.RequestCount == 0 {
		return 0
	}
	return float64(m.TotalResponseTime) / float64(m.RequestCount)
}
