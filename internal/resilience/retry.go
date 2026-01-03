package resilience

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts       int           `mapstructure:"max_attempts"`
	InitialInterval   time.Duration `mapstructure:"initial_interval"`
	MaxInterval       time.Duration `mapstructure:"max_interval"`
	Multiplier        float64       `mapstructure:"multiplier"`
	RandomizationFactor float64     `mapstructure:"randomization_factor"`
	RetryableErrors   []error       `mapstructure:"-"`
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:       3,
		InitialInterval:   100 * time.Millisecond,
		MaxInterval:       10 * time.Second,
		Multiplier:        2.0,
		RandomizationFactor: 0.5,
	}
}

// Retry executes a function with retry logic
func Retry(ctx context.Context, config *RetryConfig, fn func(context.Context) error) error {
	var lastErr error
	interval := config.InitialInterval

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		// Check if error is retryable
		if len(config.RetryableErrors) > 0 {
			retryable := false
			for _, re := range config.RetryableErrors {
				if lastErr == re {
					retryable = true
					break
				}
			}
			if !retryable {
				return lastErr
			}
		}

		if attempt < config.MaxAttempts {
			// Calculate next interval with jitter
			nextInterval := calculateInterval(interval, config)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(nextInterval):
			}

			// Update interval for next iteration
			interval = time.Duration(float64(interval) * config.Multiplier)
			if interval > config.MaxInterval {
				interval = config.MaxInterval
			}
		}
	}

	return lastErr
}

// RetryWithResult executes a function with retry logic and returns a result
func RetryWithResult[T any](ctx context.Context, config *RetryConfig, fn func(context.Context) (T, error)) (T, error) {
	var lastErr error
	var result T
	interval := config.InitialInterval

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		result, lastErr = fn(ctx)
		if lastErr == nil {
			return result, nil
		}

		if attempt < config.MaxAttempts {
			nextInterval := calculateInterval(interval, config)

			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(nextInterval):
			}

			interval = time.Duration(float64(interval) * config.Multiplier)
			if interval > config.MaxInterval {
				interval = config.MaxInterval
			}
		}
	}

	return result, lastErr
}

func calculateInterval(base time.Duration, config *RetryConfig) time.Duration {
	if config.RandomizationFactor == 0 {
		return base
	}

	delta := config.RandomizationFactor * float64(base)
	minInterval := float64(base) - delta
	maxInterval := float64(base) + delta

	// Random value between minInterval and maxInterval
	return time.Duration(minInterval + (rand.Float64() * (maxInterval - minInterval)))
}

// ExponentialBackoff calculates exponential backoff duration
func ExponentialBackoff(attempt int, baseDelay time.Duration, maxDelay time.Duration) time.Duration {
	delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt-1)))
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter (0-25% of delay)
	jitter := time.Duration(rand.Float64() * 0.25 * float64(delay))
	return delay + jitter
}
