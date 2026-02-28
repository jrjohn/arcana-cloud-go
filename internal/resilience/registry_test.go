package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewCircuitBreakerRegistry(t *testing.T) {
	logger := newTestLogger()
	registry := NewCircuitBreakerRegistry(logger)
	if registry == nil {
		t.Error("NewCircuitBreakerRegistry() returned nil")
	}
}

func TestCircuitBreakerRegistry_Get_CreatesDefault(t *testing.T) {
	logger := newTestLogger()
	registry := NewCircuitBreakerRegistry(logger)

	cb := registry.Get("test-breaker")
	if cb == nil {
		t.Error("Get() returned nil")
	}

	// State should be closed by default
	if cb.State() != StateClosed {
		t.Errorf("State = %v, want CLOSED", cb.State())
	}
}

func TestCircuitBreakerRegistry_Get_ReturnsSame(t *testing.T) {
	logger := newTestLogger()
	registry := NewCircuitBreakerRegistry(logger)

	cb1 := registry.Get("test")
	cb2 := registry.Get("test")

	if cb1 != cb2 {
		t.Error("Get() should return same instance for same name")
	}
}

func TestCircuitBreakerRegistry_RegisterConfig(t *testing.T) {
	logger := newTestLogger()
	registry := NewCircuitBreakerRegistry(logger)

	customCfg := &CircuitBreakerConfig{
		Name:                      "custom",
		FailureThreshold:          10,
		SuccessThreshold:          5,
		Timeout:                   60 * time.Second,
		MaxHalfOpenRequests:       5,
		SlidingWindowSize:         20,
		SlidingWindowType:         "count",
		SlowCallDurationThreshold: time.Second,
		SlowCallRateThreshold:     0.3,
	}

	registry.RegisterConfig(customCfg)

	// Get should use our custom config
	cb := registry.Get("custom")
	if cb == nil {
		t.Error("Get() returned nil after RegisterConfig")
	}
}

func TestCircuitBreakerRegistry_GetAll(t *testing.T) {
	logger := newTestLogger()
	registry := NewCircuitBreakerRegistry(logger)

	registry.Get("breaker1")
	registry.Get("breaker2")
	registry.Get("breaker3")

	all := registry.GetAll()
	if len(all) != 3 {
		t.Errorf("GetAll() returned %d breakers, want 3", len(all))
	}
}

func TestCircuitBreakerRegistry_GetMetrics(t *testing.T) {
	logger := newTestLogger()
	registry := NewCircuitBreakerRegistry(logger)

	cb := registry.Get("test")
	ctx := context.Background()
	cb.Execute(ctx, func(ctx context.Context) error { return nil })
	cb.Execute(ctx, func(ctx context.Context) error { return errors.New("err") })

	metrics := registry.GetMetrics()
	if m, ok := metrics["test"]; !ok {
		t.Error("GetMetrics() should contain 'test' entry")
	} else {
		if m.TotalCalls != 2 {
			t.Errorf("TotalCalls = %v, want 2", m.TotalCalls)
		}
	}
}

func TestCircuitBreakerRegistry_Reset(t *testing.T) {
	logger := newTestLogger()
	registry := NewCircuitBreakerRegistry(logger)

	cfg := &CircuitBreakerConfig{
		Name:                      "reset-test",
		FailureThreshold:          2,
		SuccessThreshold:          2,
		Timeout:                   30 * time.Second,
		MaxHalfOpenRequests:       2,
		SlidingWindowSize:         10,
		SlidingWindowType:         "count",
		SlowCallDurationThreshold: 2 * time.Second,
		SlowCallRateThreshold:     0.5,
	}
	registry.RegisterConfig(cfg)

	cb := registry.Get("reset-test")
	ctx := context.Background()

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(ctx, func(ctx context.Context) error {
			return errors.New("err")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("State = %v, want OPEN", cb.State())
	}

	registry.Reset()

	if cb.State() != StateClosed {
		t.Errorf("After Reset(), State = %v, want CLOSED", cb.State())
	}
}

func TestCircuitBreakerRegistry_ConcurrentAccess(t *testing.T) {
	logger := newTestLogger()
	registry := NewCircuitBreakerRegistry(logger)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			registry.Get("concurrent-test")
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	all := registry.GetAll()
	if len(all) != 1 {
		t.Errorf("GetAll() returned %d breakers, want 1 (concurrent creates same)", len(all))
	}
}
