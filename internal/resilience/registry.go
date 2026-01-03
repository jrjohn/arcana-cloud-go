package resilience

import (
	"sync"

	"go.uber.org/zap"
)

// CircuitBreakerRegistry manages multiple circuit breakers
type CircuitBreakerRegistry struct {
	breakers map[string]*CircuitBreaker
	configs  map[string]*CircuitBreakerConfig
	logger   *zap.Logger
	mutex    sync.RWMutex
}

// NewCircuitBreakerRegistry creates a new registry
func NewCircuitBreakerRegistry(logger *zap.Logger) *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
		configs:  make(map[string]*CircuitBreakerConfig),
		logger:   logger,
	}
}

// RegisterConfig registers a circuit breaker configuration
func (r *CircuitBreakerRegistry) RegisterConfig(config *CircuitBreakerConfig) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.configs[config.Name] = config
}

// Get returns a circuit breaker by name, creating one if it doesn't exist
func (r *CircuitBreakerRegistry) Get(name string) *CircuitBreaker {
	r.mutex.RLock()
	if cb, ok := r.breakers[name]; ok {
		r.mutex.RUnlock()
		return cb
	}
	r.mutex.RUnlock()

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Double-check after acquiring write lock
	if cb, ok := r.breakers[name]; ok {
		return cb
	}

	// Get config or use default
	config, ok := r.configs[name]
	if !ok {
		config = DefaultCircuitBreakerConfig(name)
	}

	cb := NewCircuitBreaker(config, r.logger)
	r.breakers[name] = cb

	r.logger.Info("Created circuit breaker", zap.String("name", name))
	return cb
}

// GetAll returns all circuit breakers
func (r *CircuitBreakerRegistry) GetAll() map[string]*CircuitBreaker {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]*CircuitBreaker)
	for k, v := range r.breakers {
		result[k] = v
	}
	return result
}

// GetMetrics returns metrics for all circuit breakers
func (r *CircuitBreakerRegistry) GetMetrics() map[string]CircuitBreakerMetrics {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]CircuitBreakerMetrics)
	for name, cb := range r.breakers {
		result[name] = cb.Metrics()
	}
	return result
}

// Reset resets all circuit breakers
func (r *CircuitBreakerRegistry) Reset() {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, cb := range r.breakers {
		cb.Reset()
	}
}
