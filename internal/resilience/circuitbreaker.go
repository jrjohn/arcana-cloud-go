package resilience

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

var (
	ErrCircuitOpen     = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	Name                     string        `mapstructure:"name"`
	FailureThreshold         int           `mapstructure:"failure_threshold"`
	SuccessThreshold         int           `mapstructure:"success_threshold"`
	Timeout                  time.Duration `mapstructure:"timeout"`
	MaxHalfOpenRequests      int           `mapstructure:"max_half_open_requests"`
	SlidingWindowSize        int           `mapstructure:"sliding_window_size"`
	SlidingWindowType        string        `mapstructure:"sliding_window_type"` // "count" or "time"
	SlowCallDurationThreshold time.Duration `mapstructure:"slow_call_duration_threshold"`
	SlowCallRateThreshold    float64       `mapstructure:"slow_call_rate_threshold"`
}

// DefaultCircuitBreakerConfig returns default configuration
func DefaultCircuitBreakerConfig(name string) *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		Name:                     name,
		FailureThreshold:         5,
		SuccessThreshold:         3,
		Timeout:                  30 * time.Second,
		MaxHalfOpenRequests:      3,
		SlidingWindowSize:        10,
		SlidingWindowType:        "count",
		SlowCallDurationThreshold: 2 * time.Second,
		SlowCallRateThreshold:    0.5,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config           *CircuitBreakerConfig
	state            State
	failures         int
	successes        int
	halfOpenRequests int
	lastFailure      time.Time
	mutex            sync.RWMutex
	logger           *zap.Logger
	metrics          *CircuitBreakerMetrics
	slidingWindow    *SlidingWindow
}

// CircuitBreakerMetrics holds circuit breaker metrics
type CircuitBreakerMetrics struct {
	TotalCalls       int64
	SuccessfulCalls  int64
	FailedCalls      int64
	RejectedCalls    int64
	SlowCalls        int64
	StateTransitions int64
	mutex            sync.RWMutex
}

// SlidingWindow for tracking call outcomes
type SlidingWindow struct {
	size     int
	outcomes []bool // true = success, false = failure
	durations []time.Duration
	index    int
	count    int
	mutex    sync.RWMutex
}

// NewSlidingWindow creates a new sliding window
func NewSlidingWindow(size int) *SlidingWindow {
	return &SlidingWindow{
		size:      size,
		outcomes:  make([]bool, size),
		durations: make([]time.Duration, size),
	}
}

// Record records an outcome
func (sw *SlidingWindow) Record(success bool, duration time.Duration) {
	sw.mutex.Lock()
	defer sw.mutex.Unlock()

	sw.outcomes[sw.index] = success
	sw.durations[sw.index] = duration
	sw.index = (sw.index + 1) % sw.size
	if sw.count < sw.size {
		sw.count++
	}
}

// FailureRate returns the failure rate
func (sw *SlidingWindow) FailureRate() float64 {
	sw.mutex.RLock()
	defer sw.mutex.RUnlock()

	if sw.count == 0 {
		return 0
	}

	failures := 0
	for i := 0; i < sw.count; i++ {
		if !sw.outcomes[i] {
			failures++
		}
	}
	return float64(failures) / float64(sw.count)
}

// SlowCallRate returns the slow call rate
func (sw *SlidingWindow) SlowCallRate(threshold time.Duration) float64 {
	sw.mutex.RLock()
	defer sw.mutex.RUnlock()

	if sw.count == 0 {
		return 0
	}

	slowCalls := 0
	for i := 0; i < sw.count; i++ {
		if sw.durations[i] > threshold {
			slowCalls++
		}
	}
	return float64(slowCalls) / float64(sw.count)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig, logger *zap.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		config:        config,
		state:         StateClosed,
		logger:        logger.With(zap.String("circuit_breaker", config.Name)),
		metrics:       &CircuitBreakerMetrics{},
		slidingWindow: NewSlidingWindow(config.SlidingWindowSize),
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if err := cb.allowRequest(); err != nil {
		cb.recordRejection()
		return err
	}

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	cb.recordOutcome(err == nil, duration)

	return err
}

// ExecuteWithFallback executes with a fallback function
func (cb *CircuitBreaker) ExecuteWithFallback(
	ctx context.Context,
	fn func(context.Context) error,
	fallback func(context.Context, error) error,
) error {
	err := cb.Execute(ctx, fn)
	if err != nil {
		return fallback(ctx, err)
	}
	return nil
}

// allowRequest checks if a request is allowed
func (cb *CircuitBreaker) allowRequest() error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case StateClosed:
		return nil
	case StateOpen:
		if time.Since(cb.lastFailure) > cb.config.Timeout {
			cb.transitionTo(StateHalfOpen)
			cb.halfOpenRequests = 1
			return nil
		}
		return ErrCircuitOpen
	case StateHalfOpen:
		if cb.halfOpenRequests >= cb.config.MaxHalfOpenRequests {
			return ErrTooManyRequests
		}
		cb.halfOpenRequests++
		return nil
	}
	return nil
}

// recordOutcome records the outcome of a call
func (cb *CircuitBreaker) recordOutcome(success bool, duration time.Duration) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.slidingWindow.Record(success, duration)
	cb.metrics.mutex.Lock()
	cb.metrics.TotalCalls++
	if success {
		cb.metrics.SuccessfulCalls++
	} else {
		cb.metrics.FailedCalls++
	}
	if duration > cb.config.SlowCallDurationThreshold {
		cb.metrics.SlowCalls++
	}
	cb.metrics.mutex.Unlock()

	switch cb.state {
	case StateClosed:
		if success {
			cb.failures = 0
		} else {
			cb.failures++
			cb.lastFailure = time.Now()
			if cb.failures >= cb.config.FailureThreshold {
				cb.transitionTo(StateOpen)
			}
		}
	case StateHalfOpen:
		if success {
			cb.successes++
			if cb.successes >= cb.config.SuccessThreshold {
				cb.transitionTo(StateClosed)
			}
		} else {
			cb.transitionTo(StateOpen)
		}
	}
}

// recordRejection records a rejected call
func (cb *CircuitBreaker) recordRejection() {
	cb.metrics.mutex.Lock()
	cb.metrics.RejectedCalls++
	cb.metrics.mutex.Unlock()
}

// transitionTo transitions to a new state
func (cb *CircuitBreaker) transitionTo(newState State) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenRequests = 0

	cb.metrics.mutex.Lock()
	cb.metrics.StateTransitions++
	cb.metrics.mutex.Unlock()

	cb.logger.Info("Circuit breaker state transition",
		zap.String("from", oldState.String()),
		zap.String("to", newState.String()),
	)
}

// State returns the current state
func (cb *CircuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Metrics returns the current metrics
func (cb *CircuitBreaker) Metrics() CircuitBreakerMetrics {
	cb.metrics.mutex.RLock()
	defer cb.metrics.mutex.RUnlock()
	return *cb.metrics
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenRequests = 0
}
