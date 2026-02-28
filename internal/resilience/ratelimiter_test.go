package resilience

import (
	"context"
	"testing"
	"time"
)

func TestDefaultRateLimiterConfig(t *testing.T) {
	cfg := DefaultRateLimiterConfig("test")
	if cfg.Name != "test" {
		t.Errorf("Name = %v, want test", cfg.Name)
	}
	if cfg.Rate != 100 {
		t.Errorf("Rate = %v, want 100", cfg.Rate)
	}
	if cfg.BurstSize != 10 {
		t.Errorf("BurstSize = %v, want 10", cfg.BurstSize)
	}
	if cfg.Period != time.Second {
		t.Errorf("Period = %v, want 1s", cfg.Period)
	}
}

// TokenBucketLimiter tests
func TestTokenBucketLimiter_Allow(t *testing.T) {
	cfg := &RateLimiterConfig{
		Name:        "test",
		Rate:        10,
		Period:      time.Second,
		BurstSize:   5,
		WaitTimeout: time.Second,
	}
	limiter := NewTokenBucketLimiter(cfg)

	// Initial burst should be allowed
	allowed := 0
	for i := 0; i < 10; i++ {
		if limiter.Allow() {
			allowed++
		}
	}
	if allowed < 4 {
		t.Errorf("At least 4 requests should be allowed from burst, got %d", allowed)
	}
}

func TestTokenBucketLimiter_AllowN(t *testing.T) {
	cfg := &RateLimiterConfig{
		Name:        "test",
		Rate:        100,
		Period:      time.Second,
		BurstSize:   10,
		WaitTimeout: time.Second,
	}
	limiter := NewTokenBucketLimiter(cfg)

	// Should allow burst
	if !limiter.AllowN(5) {
		t.Error("AllowN(5) should succeed")
	}

	// Drain tokens
	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	// Now should fail
	if limiter.AllowN(100) {
		t.Error("AllowN(100) should fail when tokens exhausted")
	}
}

func TestTokenBucketLimiter_Metrics(t *testing.T) {
	cfg := &RateLimiterConfig{
		Name:        "test",
		Rate:        10,
		Period:      time.Second,
		BurstSize:   2,
		WaitTimeout: time.Millisecond,
	}
	limiter := NewTokenBucketLimiter(cfg)

	// Allow first requests
	limiter.Allow()
	limiter.Allow()

	// Drain tokens and check rejection
	for i := 0; i < 20; i++ {
		limiter.Allow()
	}

	metrics := limiter.Metrics()
	if metrics.TotalRequests == 0 {
		t.Error("TotalRequests should be > 0")
	}
	if metrics.AllowedRequests == 0 {
		t.Error("AllowedRequests should be > 0")
	}
}

func TestTokenBucketLimiter_Wait_Immediate(t *testing.T) {
	cfg := &RateLimiterConfig{
		Name:        "test",
		Rate:        100,
		Period:      time.Second,
		BurstSize:   10,
		WaitTimeout: time.Second,
	}
	limiter := NewTokenBucketLimiter(cfg)

	ctx := context.Background()
	err := limiter.Wait(ctx)
	if err != nil {
		t.Errorf("Wait() error = %v", err)
	}
}

func TestTokenBucketLimiter_WaitN_ExceedsTimeout(t *testing.T) {
	cfg := &RateLimiterConfig{
		Name:        "test",
		Rate:        1,
		Period:      time.Second,
		BurstSize:   1,
		WaitTimeout: time.Millisecond, // very short wait timeout
	}
	limiter := NewTokenBucketLimiter(cfg)

	// Drain the bucket
	limiter.Allow()

	ctx := context.Background()
	err := limiter.WaitN(ctx, 100) // Request way more than available
	if err != ErrRateLimitExceeded {
		t.Errorf("WaitN() error = %v, want ErrRateLimitExceeded", err)
	}
}

func TestTokenBucketLimiter_Wait_ContextCancelled(t *testing.T) {
	cfg := &RateLimiterConfig{
		Name:        "test",
		Rate:        1,
		Period:      time.Second,
		BurstSize:   1,
		WaitTimeout: 10 * time.Second,
	}
	limiter := NewTokenBucketLimiter(cfg)

	// Drain the bucket
	limiter.Allow()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := limiter.Wait(ctx)
	if err == nil {
		t.Error("Wait() should return error when context is cancelled")
	}
}

func TestTokenBucketLimiter_Refill(t *testing.T) {
	cfg := &RateLimiterConfig{
		Name:        "test",
		Rate:        100,
		Period:      time.Second,
		BurstSize:   2,
		WaitTimeout: 100 * time.Millisecond,
	}
	limiter := NewTokenBucketLimiter(cfg)

	// Use up tokens
	limiter.Allow()
	limiter.Allow()

	// Wait for refill
	time.Sleep(50 * time.Millisecond)

	// Should be able to make at least one request now
	if !limiter.Allow() {
		t.Error("After refill period, Allow() should succeed")
	}
}

// SlidingWindowLimiter tests
func TestSlidingWindowLimiter_Allow(t *testing.T) {
	cfg := &RateLimiterConfig{
		Name:   "test",
		Rate:   5,
		Period: time.Second,
	}
	limiter := NewSlidingWindowLimiter(cfg)

	// Should allow up to Rate requests
	allowed := 0
	for i := 0; i < 10; i++ {
		if limiter.Allow() {
			allowed++
		}
	}
	if allowed != 5 {
		t.Errorf("Allowed = %v, want 5", allowed)
	}
}

func TestSlidingWindowLimiter_Metrics(t *testing.T) {
	cfg := &RateLimiterConfig{
		Name:   "test",
		Rate:   3,
		Period: time.Second,
	}
	limiter := NewSlidingWindowLimiter(cfg)

	// Make 5 requests
	for i := 0; i < 5; i++ {
		limiter.Allow()
	}

	metrics := limiter.Metrics()
	if metrics.TotalRequests != 5 {
		t.Errorf("TotalRequests = %v, want 5", metrics.TotalRequests)
	}
	if metrics.AllowedRequests != 3 {
		t.Errorf("AllowedRequests = %v, want 3", metrics.AllowedRequests)
	}
	if metrics.RejectedRequests != 2 {
		t.Errorf("RejectedRequests = %v, want 2", metrics.RejectedRequests)
	}
}

func TestSlidingWindowLimiter_WindowExpires(t *testing.T) {
	cfg := &RateLimiterConfig{
		Name:   "test",
		Rate:   3,
		Period: 50 * time.Millisecond,
	}
	limiter := NewSlidingWindowLimiter(cfg)

	// Use up capacity
	for i := 0; i < 3; i++ {
		limiter.Allow()
	}

	// Should be rejected
	if limiter.Allow() {
		t.Error("Request should be rejected when rate limit reached")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow() {
		t.Error("Request should be allowed after window expires")
	}
}
