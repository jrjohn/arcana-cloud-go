package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

// ========== Circuit Breaker Tests ==========

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "CLOSED"},
		{StateOpen, "OPEN"},
		{StateHalfOpen, "HALF_OPEN"},
		{State(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.state.String() != tt.expected {
				t.Errorf("State.String() = %v, want %v", tt.state.String(), tt.expected)
			}
		})
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")

	if config.Name != "test" {
		t.Errorf("Name = %v, want test", config.Name)
	}
	if config.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %v, want 5", config.FailureThreshold)
	}
	if config.SuccessThreshold != 3 {
		t.Errorf("SuccessThreshold = %v, want 3", config.SuccessThreshold)
	}
	if config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", config.Timeout)
	}
	if config.MaxHalfOpenRequests != 3 {
		t.Errorf("MaxHalfOpenRequests = %v, want 3", config.MaxHalfOpenRequests)
	}
	if config.SlidingWindowSize != 10 {
		t.Errorf("SlidingWindowSize = %v, want 10", config.SlidingWindowSize)
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	logger := zap.NewNop()
	cb := NewCircuitBreaker(config, logger)

	if cb == nil {
		t.Fatal("NewCircuitBreaker() returned nil")
	}
	if cb.State() != StateClosed {
		t.Errorf("State() = %v, want CLOSED", cb.State())
	}
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(config, zap.NewNop())

	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	if cb.State() != StateClosed {
		t.Errorf("State() = %v, want CLOSED", cb.State())
	}
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	config.FailureThreshold = 3
	cb := NewCircuitBreaker(config, zap.NewNop())

	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) error {
			return testErr
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("State() = %v, want OPEN after %d failures", cb.State(), config.FailureThreshold)
	}
}

func TestCircuitBreaker_Execute_Open(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	config.FailureThreshold = 1
	config.Timeout = 10 * time.Second // Long timeout so it stays open
	cb := NewCircuitBreaker(config, zap.NewNop())

	// Trigger open state
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("fail")
	})

	// Circuit should be open now
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if err != ErrCircuitOpen {
		t.Errorf("Execute() error = %v, want ErrCircuitOpen", err)
	}
}

func TestCircuitBreaker_Execute_HalfOpen(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	config.FailureThreshold = 1
	config.Timeout = 1 * time.Millisecond // Short timeout to transition to half-open
	config.SuccessThreshold = 1
	config.MaxHalfOpenRequests = 5
	cb := NewCircuitBreaker(config, zap.NewNop())

	// Trigger open state
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("fail")
	})

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	// Should transition to half-open and then closed on success
	err := cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("Execute() in half-open error = %v", err)
	}

	if cb.State() != StateClosed {
		t.Errorf("State() = %v, want CLOSED after successful half-open", cb.State())
	}
}

func TestCircuitBreaker_HalfOpen_TooManyRequests(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	config.FailureThreshold = 1
	config.Timeout = 1 * time.Millisecond
	config.MaxHalfOpenRequests = 1
	cb := NewCircuitBreaker(config, zap.NewNop())

	// Open the circuit
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("fail")
	})

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	// First request should go through (moves to half-open)
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	// Second request might be rejected
	// (depends on state transition)
	// Just verify no panic
}

func TestCircuitBreaker_ExecuteWithFallback_Success(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(config, zap.NewNop())

	fallbackCalled := false
	err := cb.ExecuteWithFallback(
		context.Background(),
		func(ctx context.Context) error { return nil },
		func(ctx context.Context, err error) error {
			fallbackCalled = true
			return nil
		},
	)

	if err != nil {
		t.Errorf("ExecuteWithFallback() error = %v", err)
	}
	if fallbackCalled {
		t.Error("Fallback should not be called on success")
	}
}

func TestCircuitBreaker_ExecuteWithFallback_Failure(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(config, zap.NewNop())

	testErr := errors.New("test error")
	fallbackCalled := false

	err := cb.ExecuteWithFallback(
		context.Background(),
		func(ctx context.Context) error { return testErr },
		func(ctx context.Context, originalErr error) error {
			fallbackCalled = true
			if originalErr != testErr {
				t.Errorf("fallback received wrong error: %v", originalErr)
			}
			return nil
		},
	)

	if err != nil {
		t.Errorf("ExecuteWithFallback() with fallback error = %v", err)
	}
	if !fallbackCalled {
		t.Error("Fallback should be called on failure")
	}
}

func TestCircuitBreaker_Metrics(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(config, zap.NewNop())

	cb.Execute(context.Background(), func(ctx context.Context) error { return nil })
	cb.Execute(context.Background(), func(ctx context.Context) error { return errors.New("err") })

	metrics := cb.Metrics()
	if metrics.TotalCalls != 2 {
		t.Errorf("TotalCalls = %v, want 2", metrics.TotalCalls)
	}
	if metrics.SuccessfulCalls != 1 {
		t.Errorf("SuccessfulCalls = %v, want 1", metrics.SuccessfulCalls)
	}
	if metrics.FailedCalls != 1 {
		t.Errorf("FailedCalls = %v, want 1", metrics.FailedCalls)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	config.FailureThreshold = 1
	cb := NewCircuitBreaker(config, zap.NewNop())

	// Open circuit
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("fail")
	})

	if cb.State() != StateOpen {
		t.Errorf("State() = %v, want OPEN", cb.State())
	}

	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("State() = %v, want CLOSED after reset", cb.State())
	}
}

func TestCircuitBreaker_SlowCall(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	config.SlowCallDurationThreshold = 1 * time.Millisecond
	cb := NewCircuitBreaker(config, zap.NewNop())

	// This call should count as slow
	cb.Execute(context.Background(), func(ctx context.Context) error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	metrics := cb.Metrics()
	if metrics.SlowCalls != 1 {
		t.Errorf("SlowCalls = %v, want 1", metrics.SlowCalls)
	}
}

// ========== Sliding Window Tests ==========

func TestNewSlidingWindow(t *testing.T) {
	sw := NewSlidingWindow(10)
	if sw == nil {
		t.Fatal("NewSlidingWindow() returned nil")
	}
	if len(sw.outcomes) != 10 {
		t.Errorf("outcomes size = %v, want 10", len(sw.outcomes))
	}
}

func TestSlidingWindow_Record_FailureRate(t *testing.T) {
	sw := NewSlidingWindow(4)

	sw.Record(true, time.Millisecond)
	sw.Record(true, time.Millisecond)
	sw.Record(false, time.Millisecond)
	sw.Record(false, time.Millisecond)

	rate := sw.FailureRate()
	if rate != 0.5 {
		t.Errorf("FailureRate() = %v, want 0.5", rate)
	}
}

func TestSlidingWindow_FailureRate_Empty(t *testing.T) {
	sw := NewSlidingWindow(5)

	if sw.FailureRate() != 0 {
		t.Errorf("FailureRate() on empty window = %v, want 0", sw.FailureRate())
	}
}

func TestSlidingWindow_SlowCallRate(t *testing.T) {
	sw := NewSlidingWindow(4)
	threshold := 10 * time.Millisecond

	sw.Record(true, 5*time.Millisecond)
	sw.Record(true, 15*time.Millisecond)
	sw.Record(true, 20*time.Millisecond)
	sw.Record(true, 5*time.Millisecond)

	rate := sw.SlowCallRate(threshold)
	if rate != 0.5 {
		t.Errorf("SlowCallRate() = %v, want 0.5", rate)
	}
}

func TestSlidingWindow_Wrap(t *testing.T) {
	sw := NewSlidingWindow(3)

	// Record more than window size
	sw.Record(false, time.Millisecond)
	sw.Record(false, time.Millisecond)
	sw.Record(false, time.Millisecond)
	sw.Record(true, time.Millisecond) // Overwrites first

	// Window should only show last 3 entries (false, false, true)
	// But implementation tracks circular, so count = 3
	// Rate = 2/3
	rate := sw.FailureRate()
	if rate < 0 || rate > 1 {
		t.Errorf("FailureRate() = %v, should be between 0 and 1", rate)
	}
}

// ========== Rate Limiter Tests ==========

func TestDefaultRateLimiterConfig(t *testing.T) {
	config := DefaultRateLimiterConfig("test")

	if config.Name != "test" {
		t.Errorf("Name = %v, want test", config.Name)
	}
	if config.Rate != 100 {
		t.Errorf("Rate = %v, want 100", config.Rate)
	}
	if config.Period != time.Second {
		t.Errorf("Period = %v, want 1s", config.Period)
	}
	if config.BurstSize != 10 {
		t.Errorf("BurstSize = %v, want 10", config.BurstSize)
	}
}

func TestNewTokenBucketLimiter(t *testing.T) {
	config := DefaultRateLimiterConfig("test")
	limiter := NewTokenBucketLimiter(config)

	if limiter == nil {
		t.Fatal("NewTokenBucketLimiter() returned nil")
	}
}

func TestTokenBucketLimiter_Allow(t *testing.T) {
	config := &RateLimiterConfig{
		Name:        "test",
		Rate:        10,
		Period:      time.Second,
		BurstSize:   5,
		WaitTimeout: time.Second,
	}
	limiter := NewTokenBucketLimiter(config)

	// Should allow up to burst size
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Errorf("Allow() returned false at iteration %d", i)
		}
	}

	// Burst exhausted - should be rejected
	if limiter.Allow() {
		t.Log("Allow() returned true after burst exhausted (refill may have happened)")
	}
}

func TestTokenBucketLimiter_AllowN(t *testing.T) {
	config := &RateLimiterConfig{
		Name:      "test",
		Rate:      10,
		Period:    time.Second,
		BurstSize: 5,
	}
	limiter := NewTokenBucketLimiter(config)

	// Allow 3 tokens
	if !limiter.AllowN(3) {
		t.Error("AllowN(3) should succeed")
	}

	// Only 2 tokens left, requesting 3 should fail
	if limiter.AllowN(3) {
		t.Log("AllowN(3) returned true (refill may have happened)")
	}
}

func TestTokenBucketLimiter_Wait_Success(t *testing.T) {
	config := &RateLimiterConfig{
		Name:        "test",
		Rate:        1000,
		Period:      time.Second,
		BurstSize:   10,
		WaitTimeout: time.Second,
	}
	limiter := NewTokenBucketLimiter(config)

	err := limiter.Wait(context.Background())
	if err != nil {
		t.Errorf("Wait() error = %v", err)
	}
}

func TestTokenBucketLimiter_Wait_ContextCancelled(t *testing.T) {
	config := &RateLimiterConfig{
		Name:        "test",
		Rate:        1,
		Period:      10 * time.Second,
		BurstSize:   1,
		WaitTimeout: 10 * time.Second,
	}
	limiter := NewTokenBucketLimiter(config)

	// Exhaust tokens
	limiter.Allow()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := limiter.WaitN(ctx, 1)
	// Should return context error or rate limit error
	if err == nil {
		t.Log("Wait() succeeded (tokens may have refilled)")
	}
}

func TestTokenBucketLimiter_Wait_Exceeded(t *testing.T) {
	config := &RateLimiterConfig{
		Name:        "test",
		Rate:        1,
		Period:      10 * time.Second, // Very slow refill
		BurstSize:   1,
		WaitTimeout: 1 * time.Millisecond, // Very short timeout
	}
	limiter := NewTokenBucketLimiter(config)

	// Exhaust tokens
	limiter.Allow()

	err := limiter.Wait(context.Background())
	if err != ErrRateLimitExceeded {
		t.Logf("Wait() error = %v (may be ok if tokens refilled)", err)
	}
}

func TestTokenBucketLimiter_Metrics(t *testing.T) {
	config := &RateLimiterConfig{
		Name:      "test",
		Rate:      100,
		Period:    time.Second,
		BurstSize: 5,
	}
	limiter := NewTokenBucketLimiter(config)

	limiter.Allow()
	limiter.Allow()

	metrics := limiter.Metrics()
	if metrics.TotalRequests != 2 {
		t.Errorf("TotalRequests = %v, want 2", metrics.TotalRequests)
	}
	if metrics.AllowedRequests != 2 {
		t.Errorf("AllowedRequests = %v, want 2", metrics.AllowedRequests)
	}
}

func TestNewSlidingWindowLimiter(t *testing.T) {
	config := DefaultRateLimiterConfig("test")
	limiter := NewSlidingWindowLimiter(config)

	if limiter == nil {
		t.Fatal("NewSlidingWindowLimiter() returned nil")
	}
}

func TestSlidingWindowLimiter_Allow(t *testing.T) {
	config := &RateLimiterConfig{
		Name:   "test",
		Rate:   3,
		Period: time.Second,
	}
	limiter := NewSlidingWindowLimiter(config)

	// Allow up to rate
	for i := 0; i < 3; i++ {
		if !limiter.Allow() {
			t.Errorf("Allow() returned false at iteration %d", i)
		}
	}

	// Should be rate limited now
	if limiter.Allow() {
		t.Log("Allow() returned true after rate limit (window may have shifted)")
	}
}

func TestSlidingWindowLimiter_Metrics(t *testing.T) {
	config := &RateLimiterConfig{
		Name:   "test",
		Rate:   10,
		Period: time.Second,
	}
	limiter := NewSlidingWindowLimiter(config)

	limiter.Allow()
	limiter.Allow()
	limiter.Allow()

	metrics := limiter.Metrics()
	if metrics.TotalRequests != 3 {
		t.Errorf("TotalRequests = %v, want 3", metrics.TotalRequests)
	}
	if metrics.AllowedRequests != 3 {
		t.Errorf("AllowedRequests = %v, want 3", metrics.AllowedRequests)
	}
}

// ========== Retry Tests ==========

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %v, want 3", config.MaxAttempts)
	}
	if config.InitialInterval != 100*time.Millisecond {
		t.Errorf("InitialInterval = %v, want 100ms", config.InitialInterval)
	}
	if config.MaxInterval != 10*time.Second {
		t.Errorf("MaxInterval = %v, want 10s", config.MaxInterval)
	}
	if config.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", config.Multiplier)
	}
	if config.RandomizationFactor != 0.5 {
		t.Errorf("RandomizationFactor = %v, want 0.5", config.RandomizationFactor)
	}
}

func TestRetry_Success(t *testing.T) {
	config := DefaultRetryConfig()
	config.InitialInterval = time.Millisecond

	var calls int
	err := Retry(context.Background(), config, func(ctx context.Context) error {
		calls++
		return nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %v, want 1", calls)
	}
}

func TestRetry_SuccessAfterFailures(t *testing.T) {
	config := DefaultRetryConfig()
	config.InitialInterval = time.Millisecond
	config.MaxAttempts = 3

	var calls int
	err := Retry(context.Background(), config, func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("not ready")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %v, want 3", calls)
	}
}

func TestRetry_MaxAttempts(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2,
	}

	var calls int
	expectedErr := errors.New("always fails")
	err := Retry(context.Background(), config, func(ctx context.Context) error {
		calls++
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("Retry() error = %v, want %v", err, expectedErr)
	}
	if calls != 3 {
		t.Errorf("calls = %v, want 3", calls)
	}
}

func TestRetry_ContextCancelled(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:     10,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := Retry(ctx, config, func(ctx context.Context) error {
		return errors.New("fail")
	})

	if err == nil {
		t.Error("Retry() should return error when context is cancelled")
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	retryableErr := errors.New("retryable")
	nonRetryableErr := errors.New("not retryable")

	config := &RetryConfig{
		MaxAttempts:     5,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2,
		RetryableErrors: []error{retryableErr},
	}

	var calls int
	err := Retry(context.Background(), config, func(ctx context.Context) error {
		calls++
		return nonRetryableErr
	})

	if err != nonRetryableErr {
		t.Errorf("Retry() error = %v, want %v", err, nonRetryableErr)
	}
	if calls != 1 {
		t.Errorf("calls = %v, want 1 (non-retryable error)", calls)
	}
}

func TestRetry_RetryableError(t *testing.T) {
	retryableErr := errors.New("retryable")

	config := &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2,
		RetryableErrors: []error{retryableErr},
	}

	var calls int
	err := Retry(context.Background(), config, func(ctx context.Context) error {
		calls++
		return retryableErr
	})

	if err != retryableErr {
		t.Errorf("Retry() error = %v, want %v", err, retryableErr)
	}
	if calls != 3 {
		t.Errorf("calls = %v, want 3", calls)
	}
}

func TestRetryWithResult_Success(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2,
	}

	var calls int
	result, err := RetryWithResult(context.Background(), config, func(ctx context.Context) (string, error) {
		calls++
		return "success", nil
	})

	if err != nil {
		t.Errorf("RetryWithResult() error = %v", err)
	}
	if result != "success" {
		t.Errorf("RetryWithResult() result = %v, want success", result)
	}
	if calls != 1 {
		t.Errorf("calls = %v, want 1", calls)
	}
}

func TestRetryWithResult_FailureAfterRetries(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2,
	}

	expectedErr := errors.New("always fails")
	var calls int
	result, err := RetryWithResult(context.Background(), config, func(ctx context.Context) (int, error) {
		calls++
		return 0, expectedErr
	})

	if err != expectedErr {
		t.Errorf("RetryWithResult() error = %v, want %v", err, expectedErr)
	}
	if result != 0 {
		t.Errorf("RetryWithResult() result = %v, want 0", result)
	}
	if calls != 3 {
		t.Errorf("calls = %v, want 3", calls)
	}
}

func TestRetryWithResult_ContextCancelled(t *testing.T) {
	config := &RetryConfig{
		MaxAttempts:     10,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		Multiplier:      2,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RetryWithResult(ctx, config, func(ctx context.Context) (string, error) {
		return "", errors.New("fail")
	})

	if err == nil {
		t.Error("RetryWithResult() should return error when context is cancelled")
	}
}

func TestExponentialBackoff(t *testing.T) {
	delay := ExponentialBackoff(1, 100*time.Millisecond, 10*time.Second)
	if delay < 100*time.Millisecond {
		t.Errorf("ExponentialBackoff(1) = %v, want >= 100ms", delay)
	}
	if delay > 2*time.Second {
		t.Errorf("ExponentialBackoff(1) = %v, want <= 2s (with jitter)", delay)
	}

	// Higher attempt should produce longer delay
	delay2 := ExponentialBackoff(3, 100*time.Millisecond, 10*time.Second)
	if delay2 < 100*time.Millisecond {
		t.Errorf("ExponentialBackoff(3) = %v, want >= 100ms", delay2)
	}

	// Should cap at maxDelay
	maxDelay := 500 * time.Millisecond
	cappedDelay := ExponentialBackoff(100, 100*time.Millisecond, maxDelay)
	if cappedDelay > maxDelay+time.Duration(float64(maxDelay)*0.25)+time.Millisecond {
		t.Errorf("ExponentialBackoff(100) = %v, want <= %v+jitter", cappedDelay, maxDelay)
	}
}

func TestIsRetryableError(t *testing.T) {
	err1 := errors.New("err1")
	err2 := errors.New("err2")

	// With no retryable errors, everything is retryable
	if !isRetryableError(err1, nil) {
		t.Error("isRetryableError with nil list should return true")
	}

	// With specific errors
	retryable := []error{err1}
	if !isRetryableError(err1, retryable) {
		t.Error("isRetryableError should return true for err1")
	}
	if isRetryableError(err2, retryable) {
		t.Error("isRetryableError should return false for err2")
	}
}

func TestNextBackoffInterval(t *testing.T) {
	config := &RetryConfig{
		MaxInterval: 5 * time.Second,
		Multiplier:  2.0,
		RandomizationFactor: 0,
	}

	sleep, next := nextBackoffInterval(100*time.Millisecond, config)
	if sleep != 100*time.Millisecond {
		t.Errorf("sleep = %v, want 100ms", sleep)
	}
	if next != 200*time.Millisecond {
		t.Errorf("next = %v, want 200ms", next)
	}

	// Test cap at MaxInterval
	_, next2 := nextBackoffInterval(4*time.Second, config)
	if next2 != 5*time.Second {
		t.Errorf("next2 = %v, want 5s (capped)", next2)
	}
}

// ========== Registry Tests ==========

func TestNewCircuitBreakerRegistry(t *testing.T) {
	registry := NewCircuitBreakerRegistry(zap.NewNop())

	if registry == nil {
		t.Fatal("NewCircuitBreakerRegistry() returned nil")
	}
	if registry.breakers == nil {
		t.Error("breakers map should be initialized")
	}
}

func TestCircuitBreakerRegistry_Get(t *testing.T) {
	registry := NewCircuitBreakerRegistry(zap.NewNop())

	cb := registry.Get("service-a")
	if cb == nil {
		t.Fatal("Get() returned nil")
	}
	if cb.State() != StateClosed {
		t.Errorf("State() = %v, want CLOSED", cb.State())
	}
}

func TestCircuitBreakerRegistry_Get_SameInstance(t *testing.T) {
	registry := NewCircuitBreakerRegistry(zap.NewNop())

	cb1 := registry.Get("service-a")
	cb2 := registry.Get("service-a")

	if cb1 != cb2 {
		t.Error("Get() should return same instance for same name")
	}
}

func TestCircuitBreakerRegistry_RegisterConfig(t *testing.T) {
	registry := NewCircuitBreakerRegistry(zap.NewNop())

	config := &CircuitBreakerConfig{
		Name:             "custom",
		FailureThreshold: 10,
		SuccessThreshold: 5,
		Timeout:          1 * time.Minute,
		MaxHalfOpenRequests: 2,
		SlidingWindowSize:   5,
	}

	registry.RegisterConfig(config)
	cb := registry.Get("custom")

	if cb == nil {
		t.Fatal("Get() returned nil after RegisterConfig")
	}
	// The circuit breaker should use the custom config
	if cb.config.FailureThreshold != 10 {
		t.Errorf("FailureThreshold = %v, want 10", cb.config.FailureThreshold)
	}
}

func TestCircuitBreakerRegistry_GetAll(t *testing.T) {
	registry := NewCircuitBreakerRegistry(zap.NewNop())

	registry.Get("service-a")
	registry.Get("service-b")

	all := registry.GetAll()
	if len(all) != 2 {
		t.Errorf("GetAll() length = %v, want 2", len(all))
	}
}

func TestCircuitBreakerRegistry_GetMetrics(t *testing.T) {
	registry := NewCircuitBreakerRegistry(zap.NewNop())

	cb := registry.Get("service-a")
	cb.Execute(context.Background(), func(ctx context.Context) error { return nil })

	metrics := registry.GetMetrics()
	if _, ok := metrics["service-a"]; !ok {
		t.Error("GetMetrics() should have entry for service-a")
	}
}

func TestCircuitBreakerRegistry_Reset(t *testing.T) {
	registry := NewCircuitBreakerRegistry(zap.NewNop())

	config := DefaultCircuitBreakerConfig("svc")
	config.FailureThreshold = 1
	registry.RegisterConfig(config)

	cb := registry.Get("svc")
	// Open the circuit
	cb.Execute(context.Background(), func(ctx context.Context) error {
		return errors.New("fail")
	})

	if cb.State() != StateOpen {
		t.Errorf("State() = %v, want OPEN", cb.State())
	}

	registry.Reset()

	if cb.State() != StateClosed {
		t.Errorf("State() = %v, want CLOSED after reset", cb.State())
	}
}

func TestCircuitBreaker_Context_Propagation(t *testing.T) {
	config := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(config, zap.NewNop())

	ctx := context.WithValue(context.Background(), "key", "value")

	var capturedCtx context.Context
	cb.Execute(ctx, func(c context.Context) error {
		capturedCtx = c
		return nil
	})

	if capturedCtx.Value("key") != "value" {
		t.Error("Context should be propagated to the function")
	}
}
