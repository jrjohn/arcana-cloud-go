package jobs

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPriority_String tests priority string representation
func TestPriority_String(t *testing.T) {
	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityLow, "low"},
		{PriorityNormal, "normal"},
		{PriorityHigh, "high"},
		{PriorityCritical, "critical"},
		{Priority(99), "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.priority.String())
		})
	}
}

// TestPriority_QueueName tests queue name generation
func TestPriority_QueueName(t *testing.T) {
	assert.Equal(t, "arcana:jobs:queue:low", PriorityLow.QueueName())
	assert.Equal(t, "arcana:jobs:queue:normal", PriorityNormal.QueueName())
	assert.Equal(t, "arcana:jobs:queue:high", PriorityHigh.QueueName())
	assert.Equal(t, "arcana:jobs:queue:critical", PriorityCritical.QueueName())
}

// TestDefaultRetryPolicy verifies defaults
func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()
	assert.Equal(t, 3, policy.MaxRetries)
	assert.Equal(t, RetryStrategyExponential, policy.Strategy)
	assert.Equal(t, time.Second, policy.InitialDelay)
	assert.Equal(t, 5*time.Minute, policy.MaxDelay)
	assert.Equal(t, 2.0, policy.Multiplier)
	assert.True(t, policy.JitterEnabled)
}

// TestRetryPolicy_CalculateDelay_Exponential tests exponential backoff
func TestRetryPolicy_CalculateDelay_Exponential(t *testing.T) {
	policy := RetryPolicy{
		Strategy:     RetryStrategyExponential,
		InitialDelay: time.Second,
		MaxDelay:     10 * time.Minute,
		Multiplier:   2.0,
	}

	// attempt 1: 1s * 2^0 = 1s
	assert.Equal(t, time.Second, policy.CalculateDelay(1))
	// attempt 2: 1s * 2^1 = 2s
	assert.Equal(t, 2*time.Second, policy.CalculateDelay(2))
	// attempt 3: 1s * 2^2 = 4s
	assert.Equal(t, 4*time.Second, policy.CalculateDelay(3))
	// attempt 4: 1s * 2^3 = 8s
	assert.Equal(t, 8*time.Second, policy.CalculateDelay(4))
}

// TestRetryPolicy_CalculateDelay_Linear tests linear backoff
func TestRetryPolicy_CalculateDelay_Linear(t *testing.T) {
	policy := RetryPolicy{
		Strategy:     RetryStrategyLinear,
		InitialDelay: 5 * time.Second,
		MaxDelay:     10 * time.Minute,
	}

	assert.Equal(t, 5*time.Second, policy.CalculateDelay(1))
	assert.Equal(t, 10*time.Second, policy.CalculateDelay(2))
	assert.Equal(t, 15*time.Second, policy.CalculateDelay(3))
}

// TestRetryPolicy_CalculateDelay_Fixed tests fixed backoff
func TestRetryPolicy_CalculateDelay_Fixed(t *testing.T) {
	policy := RetryPolicy{
		Strategy:     RetryStrategyFixed,
		InitialDelay: 30 * time.Second,
		MaxDelay:     10 * time.Minute,
	}

	assert.Equal(t, 30*time.Second, policy.CalculateDelay(1))
	assert.Equal(t, 30*time.Second, policy.CalculateDelay(2))
	assert.Equal(t, 30*time.Second, policy.CalculateDelay(10))
}

// TestRetryPolicy_CalculateDelay_Default uses InitialDelay for unknown strategy
func TestRetryPolicy_CalculateDelay_Default(t *testing.T) {
	policy := RetryPolicy{
		Strategy:     RetryStrategy("unknown"),
		InitialDelay: 10 * time.Second,
		MaxDelay:     10 * time.Minute,
	}

	assert.Equal(t, 10*time.Second, policy.CalculateDelay(1))
}

// TestRetryPolicy_CalculateDelay_MaxDelayCapped ensures delay doesn't exceed max
func TestRetryPolicy_CalculateDelay_MaxDelayCapped(t *testing.T) {
	policy := RetryPolicy{
		Strategy:     RetryStrategyExponential,
		InitialDelay: time.Second,
		MaxDelay:     5 * time.Second,
		Multiplier:   10.0,
	}

	// Should be capped at MaxDelay
	delay := policy.CalculateDelay(5)
	assert.Equal(t, 5*time.Second, delay)
}

// TestNewJobPayload creates a job payload with defaults
func TestNewJobPayload(t *testing.T) {
	payload := map[string]string{"key": "value"}
	jp, err := NewJobPayload("test-job", payload)
	require.NoError(t, err)

	assert.NotEmpty(t, jp.ID)
	assert.Equal(t, "test-job", jp.Type)
	assert.Equal(t, PriorityNormal, jp.Priority)
	assert.Equal(t, JobStatusPending, jp.Status)
	assert.Equal(t, 0, jp.Attempts)
	assert.Equal(t, 3, jp.MaxRetries)
	assert.Equal(t, 5*time.Minute, jp.Timeout)
	assert.WithinDuration(t, time.Now(), jp.CreatedAt, time.Second)

	// Check payload was serialized
	var decoded map[string]string
	require.NoError(t, json.Unmarshal(jp.Payload, &decoded))
	assert.Equal(t, "value", decoded["key"])
}

// TestNewJobPayload_WithOptions applies options correctly
func TestNewJobPayload_WithOptions(t *testing.T) {
	scheduledAt := time.Now().Add(time.Hour)
	jp, err := NewJobPayload("typed-job", struct{ Name string }{"test"},
		WithPriority(PriorityHigh),
		WithTimeout(30*time.Second),
		WithScheduledAt(scheduledAt),
		WithCorrelationID("corr-123"),
		WithUniqueKey("unique-key"),
		WithTags("tag1", "tag2"),
	)
	require.NoError(t, err)

	assert.Equal(t, PriorityHigh, jp.Priority)
	assert.Equal(t, 30*time.Second, jp.Timeout)
	assert.NotNil(t, jp.ScheduledAt)
	assert.WithinDuration(t, scheduledAt, *jp.ScheduledAt, time.Second)
	assert.Equal(t, "corr-123", jp.CorrelationID)
	assert.Equal(t, "unique-key", jp.UniqueKey)
	assert.Equal(t, []string{"tag1", "tag2"}, jp.Tags)
}

// TestNewJobPayload_WithRetryPolicy overrides retry policy
func TestNewJobPayload_WithRetryPolicy(t *testing.T) {
	customPolicy := RetryPolicy{
		MaxRetries:   5,
		Strategy:     RetryStrategyLinear,
		InitialDelay: 2 * time.Second,
		MaxDelay:     30 * time.Second,
	}
	jp, err := NewJobPayload("retry-job", nil, WithRetryPolicy(customPolicy))
	require.NoError(t, err)

	assert.Equal(t, 5, jp.MaxRetries)
	assert.Equal(t, RetryStrategyLinear, jp.RetryPolicy.Strategy)
}

// TestNewJobPayload_WithDelay sets scheduled_at to now + delay
func TestNewJobPayload_WithDelay(t *testing.T) {
	before := time.Now()
	jp, err := NewJobPayload("delay-job", nil, WithDelay(5*time.Minute))
	require.NoError(t, err)
	after := time.Now()

	assert.NotNil(t, jp.ScheduledAt)
	assert.True(t, jp.ScheduledAt.After(before.Add(4*time.Minute+59*time.Second)))
	assert.True(t, jp.ScheduledAt.Before(after.Add(5*time.Minute+time.Second)))
}

// TestNewJobPayload_NilPayload handles nil payload
func TestNewJobPayload_NilPayload(t *testing.T) {
	jp, err := NewJobPayload("nil-job", nil)
	require.NoError(t, err)
	assert.NotNil(t, jp)
}

// TestJobStatus_Values checks job status constants
func TestJobStatus_Values(t *testing.T) {
	assert.Equal(t, JobStatus("pending"), JobStatusPending)
	assert.Equal(t, JobStatus("running"), JobStatusRunning)
	assert.Equal(t, JobStatus("completed"), JobStatusCompleted)
	assert.Equal(t, JobStatus("failed"), JobStatusFailed)
	assert.Equal(t, JobStatus("retrying"), JobStatusRetrying)
	assert.Equal(t, JobStatus("dead"), JobStatusDead)
}

// TestJobErrors checks error constants
func TestJobErrors(t *testing.T) {
	assert.NotNil(t, ErrJobNotFound)
	assert.NotNil(t, ErrDuplicateJob)
	assert.NotNil(t, ErrQueueEmpty)
	assert.NotNil(t, ErrJobAlreadyTaken)
}

// TestPow helper function
func TestPow(t *testing.T) {
	assert.Equal(t, 1.0, pow(2, 0))
	assert.Equal(t, 2.0, pow(2, 1))
	assert.Equal(t, 4.0, pow(2, 2))
	assert.Equal(t, 8.0, pow(2, 3))
	assert.Equal(t, 1.0, pow(10, 0))
	assert.Equal(t, 1000.0, pow(10, 3))
}

// TestWithPriority_AllLevels tests all priority options
func TestWithPriority_AllLevels(t *testing.T) {
	priorities := []Priority{PriorityLow, PriorityNormal, PriorityHigh, PriorityCritical}
	for _, p := range priorities {
		jp := &JobPayload{}
		WithPriority(p)(jp)
		assert.Equal(t, p, jp.Priority)
	}
}
