package config

import (
	"testing"
	"time"
)

func TestDefaultWorkerConfig(t *testing.T) {
	config := DefaultWorkerConfig()

	if !config.Enabled {
		t.Error("Enabled should be true by default")
	}
	if config.Concurrency != 8 {
		t.Errorf("Concurrency = %v, want 8", config.Concurrency)
	}
	if config.PollInterval != 100*time.Millisecond {
		t.Errorf("PollInterval = %v, want 100ms", config.PollInterval)
	}
	if config.ShutdownTimeout != 30*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 30s", config.ShutdownTimeout)
	}
}

func TestDefaultSchedulerConfig(t *testing.T) {
	config := DefaultSchedulerConfig()

	if !config.Enabled {
		t.Error("Enabled should be true by default")
	}
	if config.LeaderLockTTL != 30*time.Second {
		t.Errorf("LeaderLockTTL = %v, want 30s", config.LeaderLockTTL)
	}
}

func TestWorkerConfig_Struct(t *testing.T) {
	config := WorkerConfig{
		Enabled:         false,
		Concurrency:     4,
		PollInterval:    200 * time.Millisecond,
		ShutdownTimeout: 60 * time.Second,
	}

	if config.Enabled {
		t.Error("Enabled should be false")
	}
	if config.Concurrency != 4 {
		t.Errorf("Concurrency = %v, want 4", config.Concurrency)
	}
	if config.PollInterval != 200*time.Millisecond {
		t.Errorf("PollInterval = %v, want 200ms", config.PollInterval)
	}
}

func TestSchedulerConfig_Struct(t *testing.T) {
	config := SchedulerConfig{
		Enabled:       false,
		LeaderLockTTL: 60 * time.Second,
	}

	if config.Enabled {
		t.Error("Enabled should be false")
	}
	if config.LeaderLockTTL != 60*time.Second {
		t.Errorf("LeaderLockTTL = %v, want 60s", config.LeaderLockTTL)
	}
}

func TestHTTPConfig_Struct(t *testing.T) {
	enabled := HTTPConfig{Enabled: true}
	disabled := HTTPConfig{Enabled: false}

	if !enabled.Enabled {
		t.Error("Enabled should be true")
	}
	if disabled.Enabled {
		t.Error("Enabled should be false")
	}
}
