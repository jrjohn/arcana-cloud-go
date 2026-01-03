package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrRateLimitExceeded = errors.New("rate limit exceeded")

// RateLimiterConfig holds rate limiter configuration
type RateLimiterConfig struct {
	Name            string        `mapstructure:"name"`
	Rate            int           `mapstructure:"rate"`              // requests per period
	Period          time.Duration `mapstructure:"period"`            // time period
	BurstSize       int           `mapstructure:"burst_size"`        // max burst
	WaitTimeout     time.Duration `mapstructure:"wait_timeout"`      // max wait time
	FairnessEnabled bool          `mapstructure:"fairness_enabled"`
}

// DefaultRateLimiterConfig returns default configuration
func DefaultRateLimiterConfig(name string) *RateLimiterConfig {
	return &RateLimiterConfig{
		Name:            name,
		Rate:            100,
		Period:          time.Second,
		BurstSize:       10,
		WaitTimeout:     time.Second,
		FairnessEnabled: true,
	}
}

// TokenBucketLimiter implements token bucket rate limiting
type TokenBucketLimiter struct {
	config      *RateLimiterConfig
	tokens      float64
	maxTokens   float64
	refillRate  float64 // tokens per nanosecond
	lastRefill  time.Time
	mutex       sync.Mutex
	metrics     *RateLimiterMetrics
}

// RateLimiterMetrics holds rate limiter metrics
type RateLimiterMetrics struct {
	TotalRequests   int64
	AllowedRequests int64
	RejectedRequests int64
	WaitedRequests  int64
	mutex           sync.RWMutex
}

// NewTokenBucketLimiter creates a new token bucket rate limiter
func NewTokenBucketLimiter(config *RateLimiterConfig) *TokenBucketLimiter {
	maxTokens := float64(config.BurstSize)
	refillRate := float64(config.Rate) / float64(config.Period.Nanoseconds())

	return &TokenBucketLimiter{
		config:     config,
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
		metrics:    &RateLimiterMetrics{},
	}
}

// Allow checks if a request is allowed
func (l *TokenBucketLimiter) Allow() bool {
	return l.AllowN(1)
}

// AllowN checks if N requests are allowed
func (l *TokenBucketLimiter) AllowN(n int) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.refill()

	l.metrics.mutex.Lock()
	l.metrics.TotalRequests++
	l.metrics.mutex.Unlock()

	if l.tokens >= float64(n) {
		l.tokens -= float64(n)
		l.metrics.mutex.Lock()
		l.metrics.AllowedRequests++
		l.metrics.mutex.Unlock()
		return true
	}

	l.metrics.mutex.Lock()
	l.metrics.RejectedRequests++
	l.metrics.mutex.Unlock()
	return false
}

// Wait waits until a request is allowed or context is done
func (l *TokenBucketLimiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}

// WaitN waits until N requests are allowed
func (l *TokenBucketLimiter) WaitN(ctx context.Context, n int) error {
	l.metrics.mutex.Lock()
	l.metrics.TotalRequests++
	l.metrics.mutex.Unlock()

	// Try immediate acquisition
	l.mutex.Lock()
	l.refill()
	if l.tokens >= float64(n) {
		l.tokens -= float64(n)
		l.mutex.Unlock()
		l.metrics.mutex.Lock()
		l.metrics.AllowedRequests++
		l.metrics.mutex.Unlock()
		return nil
	}

	// Calculate wait time
	tokensNeeded := float64(n) - l.tokens
	waitTime := time.Duration(tokensNeeded / l.refillRate)
	l.mutex.Unlock()

	if waitTime > l.config.WaitTimeout {
		l.metrics.mutex.Lock()
		l.metrics.RejectedRequests++
		l.metrics.mutex.Unlock()
		return ErrRateLimitExceeded
	}

	l.metrics.mutex.Lock()
	l.metrics.WaitedRequests++
	l.metrics.mutex.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		l.mutex.Lock()
		l.refill()
		l.tokens -= float64(n)
		l.mutex.Unlock()
		l.metrics.mutex.Lock()
		l.metrics.AllowedRequests++
		l.metrics.mutex.Unlock()
		return nil
	}
}

// refill adds tokens based on elapsed time (must be called with mutex held)
func (l *TokenBucketLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(l.lastRefill)
	l.lastRefill = now

	tokensToAdd := float64(elapsed.Nanoseconds()) * l.refillRate
	l.tokens += tokensToAdd
	if l.tokens > l.maxTokens {
		l.tokens = l.maxTokens
	}
}

// Metrics returns current metrics
func (l *TokenBucketLimiter) Metrics() RateLimiterMetrics {
	l.metrics.mutex.RLock()
	defer l.metrics.mutex.RUnlock()
	return *l.metrics
}

// SlidingWindowLimiter implements sliding window rate limiting
type SlidingWindowLimiter struct {
	config     *RateLimiterConfig
	timestamps []time.Time
	mutex      sync.Mutex
	metrics    *RateLimiterMetrics
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter
func NewSlidingWindowLimiter(config *RateLimiterConfig) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		config:     config,
		timestamps: make([]time.Time, 0, config.Rate),
		metrics:    &RateLimiterMetrics{},
	}
}

// Allow checks if a request is allowed
func (l *SlidingWindowLimiter) Allow() bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()
	windowStart := now.Add(-l.config.Period)

	// Remove timestamps outside the window
	newTimestamps := make([]time.Time, 0, len(l.timestamps))
	for _, ts := range l.timestamps {
		if ts.After(windowStart) {
			newTimestamps = append(newTimestamps, ts)
		}
	}
	l.timestamps = newTimestamps

	l.metrics.mutex.Lock()
	l.metrics.TotalRequests++
	l.metrics.mutex.Unlock()

	if len(l.timestamps) < l.config.Rate {
		l.timestamps = append(l.timestamps, now)
		l.metrics.mutex.Lock()
		l.metrics.AllowedRequests++
		l.metrics.mutex.Unlock()
		return true
	}

	l.metrics.mutex.Lock()
	l.metrics.RejectedRequests++
	l.metrics.mutex.Unlock()
	return false
}

// Metrics returns current metrics
func (l *SlidingWindowLimiter) Metrics() RateLimiterMetrics {
	l.metrics.mutex.RLock()
	defer l.metrics.mutex.RUnlock()
	return *l.metrics
}
