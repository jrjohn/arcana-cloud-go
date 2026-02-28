package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %v, want 3", cfg.MaxAttempts)
	}
	if cfg.InitialInterval != 100*time.Millisecond {
		t.Errorf("InitialInterval = %v, want 100ms", cfg.InitialInterval)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", cfg.Multiplier)
	}
}

func TestRetry_Success(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %v, want 1", attempts)
	}
}

func TestRetry_SucceedsAfterRetries(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temp error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %v, want 3", attempts)
	}
}

func TestRetry_AllFail(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	}

	testErr := errors.New("persistent error")
	attempts := 0

	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		return testErr
	})

	if err != testErr {
		t.Errorf("Retry() error = %v, want %v", err, testErr)
	}
	if attempts != 3 {
		t.Errorf("attempts = %v, want 3", attempts)
	}
}

func TestRetry_ContextCancelled(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     10,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     time.Second,
		Multiplier:      2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := Retry(ctx, cfg, func(ctx context.Context) error {
		return errors.New("error")
	})

	if err == nil {
		t.Error("Retry() should return error when context is cancelled")
	}
}

func TestRetry_ContextCancelledDuringSleep(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 500 * time.Millisecond,
		MaxInterval:     time.Second,
		Multiplier:      2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := Retry(ctx, cfg, func(ctx context.Context) error {
		return errors.New("error")
	})

	if err == nil {
		t.Error("Retry() should return error when context is cancelled during sleep")
	}
}

func TestRetry_RetryableErrors(t *testing.T) {
	retryableErr := errors.New("retryable")
	nonRetryableErr := errors.New("not retryable")

	cfg := &RetryConfig{
		MaxAttempts:     5,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
		RetryableErrors: []error{retryableErr},
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		if attempts == 1 {
			return nonRetryableErr
		}
		return nil
	})

	if err != nonRetryableErr {
		t.Errorf("Retry() error = %v, want %v", err, nonRetryableErr)
	}
	if attempts != 1 {
		t.Errorf("attempts = %v, want 1 (non-retryable error)", attempts)
	}
}

func TestRetry_RetryableError_Retried(t *testing.T) {
	retryableErr := errors.New("retryable")

	cfg := &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
		RetryableErrors: []error{retryableErr},
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return retryableErr
		}
		return nil
	})

	if err != nil {
		t.Errorf("Retry() error = %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %v, want 3", attempts)
	}
}

func TestRetryWithResult_Success(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	}

	result, err := RetryWithResult(context.Background(), cfg, func(ctx context.Context) (string, error) {
		return "hello", nil
	})

	if err != nil {
		t.Errorf("RetryWithResult() error = %v", err)
	}
	if result != "hello" {
		t.Errorf("result = %v, want hello", result)
	}
}

func TestRetryWithResult_SucceedsAfterRetries(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	}

	attempts := 0
	result, err := RetryWithResult(context.Background(), cfg, func(ctx context.Context) (int, error) {
		attempts++
		if attempts < 3 {
			return 0, errors.New("error")
		}
		return 42, nil
	})

	if err != nil {
		t.Errorf("RetryWithResult() error = %v", err)
	}
	if result != 42 {
		t.Errorf("result = %v, want 42", result)
	}
}

func TestRetryWithResult_AllFail(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     2,
		InitialInterval: time.Millisecond,
		MaxInterval:     10 * time.Millisecond,
		Multiplier:      2.0,
	}

	testErr := errors.New("persistent error")
	_, err := RetryWithResult(context.Background(), cfg, func(ctx context.Context) (int, error) {
		return 0, testErr
	})

	if err != testErr {
		t.Errorf("RetryWithResult() error = %v, want %v", err, testErr)
	}
}

func TestRetryWithResult_ContextCancelled(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     5,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     time.Second,
		Multiplier:      2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RetryWithResult(ctx, cfg, func(ctx context.Context) (int, error) {
		return 0, errors.New("error")
	})

	if err == nil {
		t.Error("RetryWithResult() should return error when context is cancelled")
	}
}

func TestRetryWithResult_ContextCancelledDuringSleep(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 500 * time.Millisecond,
		MaxInterval:     time.Second,
		Multiplier:      2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := RetryWithResult(ctx, cfg, func(ctx context.Context) (int, error) {
		return 0, errors.New("error")
	})

	if err == nil {
		t.Error("RetryWithResult() should return error when context cancelled during sleep")
	}
}

func TestExponentialBackoff(t *testing.T) {
	delay := ExponentialBackoff(1, 100*time.Millisecond, 10*time.Second)
	if delay < 100*time.Millisecond {
		t.Errorf("Backoff delay = %v, want >= 100ms", delay)
	}

	// Test max cap
	delay = ExponentialBackoff(20, 100*time.Millisecond, 500*time.Millisecond)
	if delay > 500*time.Millisecond+125*time.Millisecond { // 25% jitter
		t.Errorf("Backoff delay = %v, exceeds maxDelay with jitter", delay)
	}
}

func TestCalculateInterval_NoRandomization(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:         3,
		InitialInterval:     100 * time.Millisecond,
		MaxInterval:         time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0,
	}

	interval := calculateInterval(100*time.Millisecond, cfg)
	if interval != 100*time.Millisecond {
		t.Errorf("calculateInterval() = %v, want 100ms", interval)
	}
}

func TestCalculateInterval_WithRandomization(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:         3,
		InitialInterval:     100 * time.Millisecond,
		MaxInterval:         time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.5,
	}

	// Should be between 50ms and 150ms
	for i := 0; i < 10; i++ {
		interval := calculateInterval(100*time.Millisecond, cfg)
		if interval < 50*time.Millisecond || interval > 150*time.Millisecond {
			t.Errorf("calculateInterval() = %v, want between 50ms-150ms", interval)
		}
	}
}

func TestNextBackoffInterval(t *testing.T) {
	cfg := &RetryConfig{
		MaxAttempts:         3,
		InitialInterval:     100 * time.Millisecond,
		MaxInterval:         200 * time.Millisecond,
		Multiplier:          2.0,
		RandomizationFactor: 0,
	}

	_, next := nextBackoffInterval(100*time.Millisecond, cfg)
	if next != 200*time.Millisecond {
		t.Errorf("next interval = %v, want 200ms", next)
	}

	// Test max cap
	_, next = nextBackoffInterval(200*time.Millisecond, cfg)
	if next != 200*time.Millisecond {
		t.Errorf("next interval (capped) = %v, want 200ms", next)
	}
}

func TestIsRetryableError_EmptyList(t *testing.T) {
	err := errors.New("any error")
	if !isRetryableError(err, nil) {
		t.Error("isRetryableError() should return true for empty retryable list")
	}
}

func TestIsRetryableError_Matching(t *testing.T) {
	retryable := errors.New("retryable")
	other := errors.New("other")

	if !isRetryableError(retryable, []error{retryable}) {
		t.Error("isRetryableError() should return true for matching error")
	}
	if isRetryableError(other, []error{retryable}) {
		t.Error("isRetryableError() should return false for non-matching error")
	}
}
