package config

import "time"

// WorkerConfig holds worker-specific configuration
type WorkerConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	Concurrency     int           `mapstructure:"concurrency"`
	PollInterval    time.Duration `mapstructure:"poll_interval"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// SchedulerConfig holds scheduler-specific configuration
type SchedulerConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	LeaderLockTTL time.Duration `mapstructure:"leader_lock_ttl"`
}

// HTTPConfig holds HTTP server enable/disable configuration
type HTTPConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// DefaultWorkerConfig returns default worker configuration
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		Enabled:         true,
		Concurrency:     8,
		PollInterval:    100 * time.Millisecond,
		ShutdownTimeout: 30 * time.Second,
	}
}

// DefaultSchedulerConfig returns default scheduler configuration
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Enabled:       true,
		LeaderLockTTL: 30 * time.Second,
	}
}
