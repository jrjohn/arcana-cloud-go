package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

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
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig("test")
	if cfg.Name != "test" {
		t.Errorf("Name = %v, want test", cfg.Name)
	}
	if cfg.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %v, want 5", cfg.FailureThreshold)
	}
	if cfg.SuccessThreshold != 3 {
		t.Errorf("SuccessThreshold = %v, want 3", cfg.SuccessThreshold)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
	if cfg.MaxHalfOpenRequests != 3 {
		t.Errorf("MaxHalfOpenRequests = %v, want 3", cfg.MaxHalfOpenRequests)
	}
}

func TestCircuitBreaker_InitialState(t *testing.T) {
	logger := newTestLogger()
	cfg := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(cfg, logger)

	if cb.State() != StateClosed {
		t.Errorf("Initial state = %v, want CLOSED", cb.State())
	}
}

func TestCircuitBreaker_SuccessfulCalls(t *testing.T) {
	logger := newTestLogger()
	cfg := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(cfg, logger)

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		err := cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
	}

	if cb.State() != StateClosed {
		t.Errorf("State after successes = %v, want CLOSED", cb.State())
	}

	metrics := cb.Metrics()
	if metrics.SuccessfulCalls != 3 {
		t.Errorf("SuccessfulCalls = %v, want 3", metrics.SuccessfulCalls)
	}
}

func TestCircuitBreaker_OpenAfterFailureThreshold(t *testing.T) {
	logger := newTestLogger()
	cfg := &CircuitBreakerConfig{
		Name:                      "test",
		FailureThreshold:          3,
		SuccessThreshold:          2,
		Timeout:                   100 * time.Millisecond,
		MaxHalfOpenRequests:       2,
		SlidingWindowSize:         10,
		SlidingWindowType:         "count",
		SlowCallDurationThreshold: 2 * time.Second,
		SlowCallRateThreshold:     0.5,
	}
	cb := NewCircuitBreaker(cfg, logger)

	ctx := context.Background()
	testErr := errors.New("test error")

	// Trigger failure threshold
	for i := 0; i < 3; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("State after failures = %v, want OPEN", cb.State())
	}

	// Next call should be rejected
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}

	metrics := cb.Metrics()
	if metrics.RejectedCalls != 1 {
		t.Errorf("RejectedCalls = %v, want 1", metrics.RejectedCalls)
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	logger := newTestLogger()
	cfg := &CircuitBreakerConfig{
		Name:                      "test",
		FailureThreshold:          2,
		SuccessThreshold:          2,
		Timeout:                   50 * time.Millisecond,
		MaxHalfOpenRequests:       2,
		SlidingWindowSize:         10,
		SlidingWindowType:         "count",
		SlowCallDurationThreshold: 2 * time.Second,
		SlowCallRateThreshold:     0.5,
	}
	cb := NewCircuitBreaker(cfg, logger)

	ctx := context.Background()
	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("State = %v, want OPEN", cb.State())
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Should transition to HalfOpen on next call
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Errorf("Execute() in half-open state error = %v", err)
	}

	// State should be half-open or closed (after success)
	state := cb.State()
	if state != StateHalfOpen && state != StateClosed {
		t.Errorf("State = %v, want HALF_OPEN or CLOSED", state)
	}
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	logger := newTestLogger()
	cfg := &CircuitBreakerConfig{
		Name:                      "test",
		FailureThreshold:          2,
		SuccessThreshold:          2,
		Timeout:                   50 * time.Millisecond,
		MaxHalfOpenRequests:       5,
		SlidingWindowSize:         10,
		SlidingWindowType:         "count",
		SlowCallDurationThreshold: 2 * time.Second,
		SlowCallRateThreshold:     0.5,
	}
	cb := NewCircuitBreaker(cfg, logger)

	ctx := context.Background()
	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Make enough successful calls to close
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}

	if cb.State() != StateClosed {
		t.Errorf("State = %v, want CLOSED", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	logger := newTestLogger()
	cfg := &CircuitBreakerConfig{
		Name:                      "test",
		FailureThreshold:          2,
		SuccessThreshold:          5,
		Timeout:                   50 * time.Millisecond,
		MaxHalfOpenRequests:       5,
		SlidingWindowSize:         10,
		SlidingWindowType:         "count",
		SlowCallDurationThreshold: 2 * time.Second,
		SlowCallRateThreshold:     0.5,
	}
	cb := NewCircuitBreaker(cfg, logger)

	ctx := context.Background()
	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Fail in half-open to reopen
	cb.Execute(ctx, func(ctx context.Context) error {
		return testErr
	})

	if cb.State() != StateOpen {
		t.Errorf("State = %v, want OPEN", cb.State())
	}
}

func TestCircuitBreaker_MaxHalfOpenRequests(t *testing.T) {
	logger := newTestLogger()
	cfg := &CircuitBreakerConfig{
		Name:                      "test",
		FailureThreshold:          2,
		SuccessThreshold:          10,
		Timeout:                   50 * time.Millisecond,
		MaxHalfOpenRequests:       2,
		SlidingWindowSize:         10,
		SlidingWindowType:         "count",
		SlowCallDurationThreshold: 2 * time.Second,
		SlowCallRateThreshold:     0.5,
	}
	cb := NewCircuitBreaker(cfg, logger)

	ctx := context.Background()
	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Use up max half-open requests
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return nil
		})
	}

	// Next call should get ErrTooManyRequests
	err := cb.Execute(ctx, func(ctx context.Context) error {
		return nil
	})
	if !errors.Is(err, ErrTooManyRequests) {
		t.Errorf("Expected ErrTooManyRequests, got %v", err)
	}
}

func TestCircuitBreaker_ExecuteWithFallback(t *testing.T) {
	logger := newTestLogger()
	cfg := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(cfg, logger)

	ctx := context.Background()
	testErr := errors.New("test error")
	fallbackCalled := false

	err := cb.ExecuteWithFallback(
		ctx,
		func(ctx context.Context) error {
			return testErr
		},
		func(ctx context.Context, err error) error {
			fallbackCalled = true
			return nil
		},
	)

	if err != nil {
		t.Errorf("ExecuteWithFallback() error = %v", err)
	}
	if !fallbackCalled {
		t.Error("Fallback was not called")
	}
}

func TestCircuitBreaker_ExecuteWithFallback_Success(t *testing.T) {
	logger := newTestLogger()
	cfg := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(cfg, logger)

	ctx := context.Background()
	fallbackCalled := false

	err := cb.ExecuteWithFallback(
		ctx,
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context, err error) error {
			fallbackCalled = true
			return nil
		},
	)

	if err != nil {
		t.Errorf("ExecuteWithFallback() error = %v", err)
	}
	if fallbackCalled {
		t.Error("Fallback should not have been called on success")
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	logger := newTestLogger()
	cfg := &CircuitBreakerConfig{
		Name:                      "test",
		FailureThreshold:          2,
		SuccessThreshold:          2,
		Timeout:                   30 * time.Second,
		MaxHalfOpenRequests:       2,
		SlidingWindowSize:         10,
		SlidingWindowType:         "count",
		SlowCallDurationThreshold: 2 * time.Second,
		SlowCallRateThreshold:     0.5,
	}
	cb := NewCircuitBreaker(cfg, logger)

	ctx := context.Background()
	testErr := errors.New("test error")

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return testErr
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("State = %v, want OPEN", cb.State())
	}

	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("After Reset(), State = %v, want CLOSED", cb.State())
	}
}

func TestCircuitBreaker_Metrics(t *testing.T) {
	logger := newTestLogger()
	cfg := DefaultCircuitBreakerConfig("test")
	cb := NewCircuitBreaker(cfg, logger)

	ctx := context.Background()
	testErr := errors.New("test error")

	// One success
	cb.Execute(ctx, func(ctx context.Context) error { return nil })
	// One failure
	cb.Execute(ctx, func(ctx context.Context) error { return testErr })

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

func TestCircuitBreaker_SlowCalls(t *testing.T) {
	logger := newTestLogger()
	cfg := &CircuitBreakerConfig{
		Name:                      "test",
		FailureThreshold:          5,
		SuccessThreshold:          3,
		Timeout:                   30 * time.Second,
		MaxHalfOpenRequests:       3,
		SlidingWindowSize:         10,
		SlidingWindowType:         "count",
		SlowCallDurationThreshold: 10 * time.Millisecond,
		SlowCallRateThreshold:     0.5,
	}
	cb := NewCircuitBreaker(cfg, logger)

	ctx := context.Background()
	// Execute a "slow" call (simulated)
	cb.Execute(ctx, func(ctx context.Context) error {
		time.Sleep(20 * time.Millisecond)
		return nil
	})

	metrics := cb.Metrics()
	if metrics.SlowCalls != 1 {
		t.Errorf("SlowCalls = %v, want 1", metrics.SlowCalls)
	}
}

// SlidingWindow tests
func TestSlidingWindow_FailureRate(t *testing.T) {
	sw := NewSlidingWindow(10)

	if sw.FailureRate() != 0 {
		t.Errorf("Empty window FailureRate = %v, want 0", sw.FailureRate())
	}

	// Record 3 successes and 2 failures
	sw.Record(true, time.Millisecond)
	sw.Record(true, time.Millisecond)
	sw.Record(true, time.Millisecond)
	sw.Record(false, time.Millisecond)
	sw.Record(false, time.Millisecond)

	rate := sw.FailureRate()
	if rate != 0.4 {
		t.Errorf("FailureRate = %v, want 0.4", rate)
	}
}

func TestSlidingWindow_SlowCallRate(t *testing.T) {
	sw := NewSlidingWindow(10)

	threshold := 50 * time.Millisecond

	// Record 3 fast and 2 slow calls
	sw.Record(true, 10*time.Millisecond)
	sw.Record(true, 10*time.Millisecond)
	sw.Record(true, 10*time.Millisecond)
	sw.Record(true, 100*time.Millisecond)
	sw.Record(true, 100*time.Millisecond)

	rate := sw.SlowCallRate(threshold)
	if rate != 0.4 {
		t.Errorf("SlowCallRate = %v, want 0.4", rate)
	}
}

func TestSlidingWindow_SlowCallRate_Empty(t *testing.T) {
	sw := NewSlidingWindow(10)
	if sw.SlowCallRate(50*time.Millisecond) != 0 {
		t.Error("Empty window SlowCallRate should be 0")
	}
}

func TestSlidingWindow_Wrap(t *testing.T) {
	sw := NewSlidingWindow(3)

	// Fill and wrap around
	sw.Record(false, time.Millisecond)
	sw.Record(false, time.Millisecond)
	sw.Record(false, time.Millisecond)
	// Now overwrite with success
	sw.Record(true, time.Millisecond)
	sw.Record(true, time.Millisecond)
	sw.Record(true, time.Millisecond)

	rate := sw.FailureRate()
	if rate != 0.0 {
		t.Errorf("After wrap, FailureRate = %v, want 0.0", rate)
	}
}
