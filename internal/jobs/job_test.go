package jobs

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPriority_String(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		want     string
	}{
		{"Low", PriorityLow, "low"},
		{"Normal", PriorityNormal, "normal"},
		{"High", PriorityHigh, "high"},
		{"Critical", PriorityCritical, "critical"},
		{"Unknown", Priority(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.priority.String(); got != tt.want {
				t.Errorf("Priority.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriority_QueueName(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		want     string
	}{
		{"Low", PriorityLow, "arcana:jobs:queue:low"},
		{"Normal", PriorityNormal, "arcana:jobs:queue:normal"},
		{"High", PriorityHigh, "arcana:jobs:queue:high"},
		{"Critical", PriorityCritical, "arcana:jobs:queue:critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.priority.QueueName(); got != tt.want {
				t.Errorf("Priority.QueueName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobStatus_Constants(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
		want   string
	}{
		{"Pending", JobStatusPending, "pending"},
		{"Running", JobStatusRunning, "running"},
		{"Completed", JobStatusCompleted, "completed"},
		{"Failed", JobStatusFailed, "failed"},
		{"Retrying", JobStatusRetrying, "retrying"},
		{"Dead", JobStatusDead, "dead"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("JobStatus = %v, want %v", tt.status, tt.want)
			}
		})
	}
}

func TestRetryStrategy_Constants(t *testing.T) {
	tests := []struct {
		name     string
		strategy RetryStrategy
		want     string
	}{
		{"Exponential", RetryStrategyExponential, "exponential"},
		{"Linear", RetryStrategyLinear, "linear"},
		{"Fixed", RetryStrategyFixed, "fixed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.strategy) != tt.want {
				t.Errorf("RetryStrategy = %v, want %v", tt.strategy, tt.want)
			}
		})
	}
}

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	if policy.MaxRetries != 3 {
		t.Errorf("DefaultRetryPolicy().MaxRetries = %v, want 3", policy.MaxRetries)
	}
	if policy.Strategy != RetryStrategyExponential {
		t.Errorf("DefaultRetryPolicy().Strategy = %v, want exponential", policy.Strategy)
	}
	if policy.InitialDelay != time.Second {
		t.Errorf("DefaultRetryPolicy().InitialDelay = %v, want 1s", policy.InitialDelay)
	}
	if policy.MaxDelay != 5*time.Minute {
		t.Errorf("DefaultRetryPolicy().MaxDelay = %v, want 5m", policy.MaxDelay)
	}
	if policy.Multiplier != 2.0 {
		t.Errorf("DefaultRetryPolicy().Multiplier = %v, want 2.0", policy.Multiplier)
	}
	if !policy.JitterEnabled {
		t.Errorf("DefaultRetryPolicy().JitterEnabled = %v, want true", policy.JitterEnabled)
	}
}

func TestRetryPolicy_CalculateDelay(t *testing.T) {
	tests := []struct {
		name     string
		policy   RetryPolicy
		attempt  int
		wantMin  time.Duration
		wantMax  time.Duration
	}{
		{
			name: "Exponential_Attempt1",
			policy: RetryPolicy{
				Strategy:     RetryStrategyExponential,
				InitialDelay: time.Second,
				MaxDelay:     time.Minute,
				Multiplier:   2.0,
			},
			attempt: 1,
			wantMin: time.Second,
			wantMax: time.Second,
		},
		{
			name: "Exponential_Attempt2",
			policy: RetryPolicy{
				Strategy:     RetryStrategyExponential,
				InitialDelay: time.Second,
				MaxDelay:     time.Minute,
				Multiplier:   2.0,
			},
			attempt: 2,
			wantMin: 2 * time.Second,
			wantMax: 2 * time.Second,
		},
		{
			name: "Exponential_Attempt3",
			policy: RetryPolicy{
				Strategy:     RetryStrategyExponential,
				InitialDelay: time.Second,
				MaxDelay:     time.Minute,
				Multiplier:   2.0,
			},
			attempt: 3,
			wantMin: 4 * time.Second,
			wantMax: 4 * time.Second,
		},
		{
			name: "Exponential_CappedAtMax",
			policy: RetryPolicy{
				Strategy:     RetryStrategyExponential,
				InitialDelay: time.Second,
				MaxDelay:     5 * time.Second,
				Multiplier:   2.0,
			},
			attempt: 10,
			wantMin: 5 * time.Second,
			wantMax: 5 * time.Second,
		},
		{
			name: "Linear_Attempt1",
			policy: RetryPolicy{
				Strategy:     RetryStrategyLinear,
				InitialDelay: time.Second,
				MaxDelay:     time.Minute,
			},
			attempt: 1,
			wantMin: time.Second,
			wantMax: time.Second,
		},
		{
			name: "Linear_Attempt3",
			policy: RetryPolicy{
				Strategy:     RetryStrategyLinear,
				InitialDelay: time.Second,
				MaxDelay:     time.Minute,
			},
			attempt: 3,
			wantMin: 3 * time.Second,
			wantMax: 3 * time.Second,
		},
		{
			name: "Fixed_AnyAttempt",
			policy: RetryPolicy{
				Strategy:     RetryStrategyFixed,
				InitialDelay: 5 * time.Second,
				MaxDelay:     time.Minute,
			},
			attempt: 5,
			wantMin: 5 * time.Second,
			wantMax: 5 * time.Second,
		},
		{
			name: "Unknown_DefaultsToInitial",
			policy: RetryPolicy{
				Strategy:     RetryStrategy("unknown"),
				InitialDelay: 3 * time.Second,
				MaxDelay:     time.Minute,
			},
			attempt: 2,
			wantMin: 3 * time.Second,
			wantMax: 3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.CalculateDelay(tt.attempt)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("CalculateDelay(%d) = %v, want between %v and %v", tt.attempt, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestNewJobPayload(t *testing.T) {
	type testPayload struct {
		Message string `json:"message"`
		Count   int    `json:"count"`
	}

	payload := testPayload{Message: "test", Count: 42}
	job, err := NewJobPayload("test-job", payload)

	if err != nil {
		t.Fatalf("NewJobPayload() error = %v", err)
	}

	// Verify ID is set
	if job.ID == "" {
		t.Error("NewJobPayload().ID is empty")
	}

	// Verify type
	if job.Type != "test-job" {
		t.Errorf("NewJobPayload().Type = %v, want test-job", job.Type)
	}

	// Verify payload
	var parsedPayload testPayload
	if err := json.Unmarshal(job.Payload, &parsedPayload); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}
	if parsedPayload.Message != "test" || parsedPayload.Count != 42 {
		t.Errorf("NewJobPayload().Payload = %+v, want {Message:test Count:42}", parsedPayload)
	}

	// Verify defaults
	if job.Priority != PriorityNormal {
		t.Errorf("NewJobPayload().Priority = %v, want normal", job.Priority)
	}
	if job.Status != JobStatusPending {
		t.Errorf("NewJobPayload().Status = %v, want pending", job.Status)
	}
	if job.Attempts != 0 {
		t.Errorf("NewJobPayload().Attempts = %v, want 0", job.Attempts)
	}
	if job.MaxRetries != 3 {
		t.Errorf("NewJobPayload().MaxRetries = %v, want 3", job.MaxRetries)
	}
	if job.Timeout != 5*time.Minute {
		t.Errorf("NewJobPayload().Timeout = %v, want 5m", job.Timeout)
	}
	if job.CreatedAt.IsZero() {
		t.Error("NewJobPayload().CreatedAt is zero")
	}
}

func TestNewJobPayload_WithOptions(t *testing.T) {
	job, err := NewJobPayload("test-job", map[string]string{"key": "value"},
		WithPriority(PriorityHigh),
		WithTimeout(10*time.Minute),
		WithUniqueKey("unique-123"),
		WithCorrelationID("corr-456"),
		WithTags("tag1", "tag2"),
	)

	if err != nil {
		t.Fatalf("NewJobPayload() error = %v", err)
	}

	if job.Priority != PriorityHigh {
		t.Errorf("Priority = %v, want high", job.Priority)
	}
	if job.Timeout != 10*time.Minute {
		t.Errorf("Timeout = %v, want 10m", job.Timeout)
	}
	if job.UniqueKey != "unique-123" {
		t.Errorf("UniqueKey = %v, want unique-123", job.UniqueKey)
	}
	if job.CorrelationID != "corr-456" {
		t.Errorf("CorrelationID = %v, want corr-456", job.CorrelationID)
	}
	if len(job.Tags) != 2 || job.Tags[0] != "tag1" || job.Tags[1] != "tag2" {
		t.Errorf("Tags = %v, want [tag1, tag2]", job.Tags)
	}
}

func TestWithPriority(t *testing.T) {
	tests := []Priority{PriorityLow, PriorityNormal, PriorityHigh, PriorityCritical}

	for _, p := range tests {
		t.Run(p.String(), func(t *testing.T) {
			job, _ := NewJobPayload("test", nil, WithPriority(p))
			if job.Priority != p {
				t.Errorf("WithPriority(%v) resulted in %v", p, job.Priority)
			}
		})
	}
}

func TestWithRetryPolicy(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:   5,
		Strategy:     RetryStrategyLinear,
		InitialDelay: 2 * time.Second,
		MaxDelay:     10 * time.Minute,
		Multiplier:   1.5,
	}

	job, _ := NewJobPayload("test", nil, WithRetryPolicy(policy))

	if job.MaxRetries != 5 {
		t.Errorf("MaxRetries = %v, want 5", job.MaxRetries)
	}
	if job.RetryPolicy.Strategy != RetryStrategyLinear {
		t.Errorf("RetryPolicy.Strategy = %v, want linear", job.RetryPolicy.Strategy)
	}
	if job.RetryPolicy.InitialDelay != 2*time.Second {
		t.Errorf("RetryPolicy.InitialDelay = %v, want 2s", job.RetryPolicy.InitialDelay)
	}
}

func TestWithTimeout(t *testing.T) {
	job, _ := NewJobPayload("test", nil, WithTimeout(15*time.Minute))
	if job.Timeout != 15*time.Minute {
		t.Errorf("Timeout = %v, want 15m", job.Timeout)
	}
}

func TestWithScheduledAt(t *testing.T) {
	scheduledTime := time.Now().Add(time.Hour)
	job, _ := NewJobPayload("test", nil, WithScheduledAt(scheduledTime))

	if job.ScheduledAt == nil {
		t.Fatal("ScheduledAt is nil")
	}
	if !job.ScheduledAt.Equal(scheduledTime) {
		t.Errorf("ScheduledAt = %v, want %v", job.ScheduledAt, scheduledTime)
	}
}

func TestWithDelay(t *testing.T) {
	before := time.Now()
	job, _ := NewJobPayload("test", nil, WithDelay(time.Hour))
	after := time.Now()

	if job.ScheduledAt == nil {
		t.Fatal("ScheduledAt is nil")
	}

	expectedMin := before.Add(time.Hour)
	expectedMax := after.Add(time.Hour)

	if job.ScheduledAt.Before(expectedMin) || job.ScheduledAt.After(expectedMax) {
		t.Errorf("ScheduledAt = %v, expected between %v and %v", job.ScheduledAt, expectedMin, expectedMax)
	}
}

func TestWithCorrelationID(t *testing.T) {
	job, _ := NewJobPayload("test", nil, WithCorrelationID("corr-12345"))
	if job.CorrelationID != "corr-12345" {
		t.Errorf("CorrelationID = %v, want corr-12345", job.CorrelationID)
	}
}

func TestWithUniqueKey(t *testing.T) {
	job, _ := NewJobPayload("test", nil, WithUniqueKey("unique-key-abc"))
	if job.UniqueKey != "unique-key-abc" {
		t.Errorf("UniqueKey = %v, want unique-key-abc", job.UniqueKey)
	}
}

func TestWithTags(t *testing.T) {
	job, _ := NewJobPayload("test", nil, WithTags("tag1", "tag2", "tag3"))
	if len(job.Tags) != 3 {
		t.Errorf("len(Tags) = %v, want 3", len(job.Tags))
	}

	// Add more tags
	job, _ = NewJobPayload("test", nil, WithTags("a"), WithTags("b", "c"))
	if len(job.Tags) != 3 {
		t.Errorf("len(Tags) = %v, want 3", len(job.Tags))
	}
}

func TestNewJobPayload_InvalidPayload(t *testing.T) {
	// Create a value that can't be marshaled to JSON
	ch := make(chan int)
	_, err := NewJobPayload("test", ch)
	if err == nil {
		t.Error("Expected error for non-marshalable payload")
	}
}

func TestJobPayload_Serialization(t *testing.T) {
	original, _ := NewJobPayload("test-type", map[string]int{"count": 5},
		WithPriority(PriorityHigh),
		WithUniqueKey("key-123"),
		WithTags("tag1"),
	)

	// Serialize
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Deserialize
	var restored JobPayload
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if restored.ID != original.ID {
		t.Errorf("ID = %v, want %v", restored.ID, original.ID)
	}
	if restored.Type != original.Type {
		t.Errorf("Type = %v, want %v", restored.Type, original.Type)
	}
	if restored.Priority != original.Priority {
		t.Errorf("Priority = %v, want %v", restored.Priority, original.Priority)
	}
	if restored.UniqueKey != original.UniqueKey {
		t.Errorf("UniqueKey = %v, want %v", restored.UniqueKey, original.UniqueKey)
	}
}

func TestJobResult(t *testing.T) {
	result := JobResult{
		JobID:       "job-123",
		Status:      JobStatusCompleted,
		Duration:    5 * time.Second,
		CompletedAt: time.Now(),
	}

	if result.JobID != "job-123" {
		t.Errorf("JobID = %v, want job-123", result.JobID)
	}
	if result.Status != JobStatusCompleted {
		t.Errorf("Status = %v, want completed", result.Status)
	}
	if result.Duration != 5*time.Second {
		t.Errorf("Duration = %v, want 5s", result.Duration)
	}
	if result.Error != "" {
		t.Errorf("Error = %v, want empty", result.Error)
	}
}

func TestJobResult_WithError(t *testing.T) {
	result := JobResult{
		JobID:       "job-456",
		Status:      JobStatusFailed,
		Error:       "something went wrong",
		Duration:    2 * time.Second,
		CompletedAt: time.Now(),
	}

	if result.Status != JobStatusFailed {
		t.Errorf("Status = %v, want failed", result.Status)
	}
	if result.Error != "something went wrong" {
		t.Errorf("Error = %v, want 'something went wrong'", result.Error)
	}
}

// Benchmarks
func BenchmarkNewJobPayload(b *testing.B) {
	payload := map[string]string{"key": "value"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewJobPayload("test-job", payload)
	}
}

func BenchmarkNewJobPayload_WithOptions(b *testing.B) {
	payload := map[string]string{"key": "value"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewJobPayload("test-job", payload,
			WithPriority(PriorityHigh),
			WithTimeout(10*time.Minute),
			WithUniqueKey("key"),
			WithTags("tag1", "tag2"),
		)
	}
}

func BenchmarkRetryPolicy_CalculateDelay(b *testing.B) {
	policy := DefaultRetryPolicy()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		policy.CalculateDelay(i % 10)
	}
}
