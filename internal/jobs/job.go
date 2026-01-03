package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Common errors
var (
	ErrJobNotFound     = errors.New("job not found")
	ErrDuplicateJob    = errors.New("duplicate job with same unique key")
	ErrQueueEmpty      = errors.New("queue is empty")
	ErrJobAlreadyTaken = errors.New("job already taken by another worker")
)

// Priority represents job priority levels
type Priority int

const (
	PriorityLow      Priority = 0
	PriorityNormal   Priority = 1
	PriorityHigh     Priority = 2
	PriorityCritical Priority = 3
)

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// QueueName returns the Redis queue name for this priority
func (p Priority) QueueName() string {
	return "arcana:jobs:queue:" + p.String()
}

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusRetrying  JobStatus = "retrying"
	JobStatusDead      JobStatus = "dead" // moved to DLQ
)

// RetryStrategy defines how retries should be handled
type RetryStrategy string

const (
	RetryStrategyExponential RetryStrategy = "exponential"
	RetryStrategyLinear      RetryStrategy = "linear"
	RetryStrategyFixed       RetryStrategy = "fixed"
)

// RetryPolicy defines the retry behavior for a job
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	Strategy      RetryStrategy `json:"strategy"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	Multiplier    float64       `json:"multiplier"`
	JitterEnabled bool          `json:"jitter_enabled"`
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:    3,
		Strategy:      RetryStrategyExponential,
		InitialDelay:  time.Second,
		MaxDelay:      5 * time.Minute,
		Multiplier:    2.0,
		JitterEnabled: true,
	}
}

// CalculateDelay calculates the delay for a given attempt number
func (p RetryPolicy) CalculateDelay(attempt int) time.Duration {
	var delay time.Duration

	switch p.Strategy {
	case RetryStrategyExponential:
		delay = time.Duration(float64(p.InitialDelay) * pow(p.Multiplier, float64(attempt-1)))
	case RetryStrategyLinear:
		delay = p.InitialDelay * time.Duration(attempt)
	case RetryStrategyFixed:
		delay = p.InitialDelay
	default:
		delay = p.InitialDelay
	}

	if delay > p.MaxDelay {
		delay = p.MaxDelay
	}

	return delay
}

func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// Job is the interface that all jobs must implement
type Job interface {
	// Type returns the unique job type identifier
	Type() string

	// Execute runs the job logic
	Execute(ctx context.Context) error

	// RetryPolicy returns the retry policy for this job
	RetryPolicy() RetryPolicy

	// Timeout returns the maximum execution time
	Timeout() time.Duration
}

// JobPayload is the serializable job data stored in the queue
type JobPayload struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Payload       json.RawMessage `json:"payload"`
	Priority      Priority        `json:"priority"`
	Status        JobStatus       `json:"status"`
	Attempts      int             `json:"attempts"`
	MaxRetries    int             `json:"max_retries"`
	RetryPolicy   RetryPolicy     `json:"retry_policy"`
	Timeout       time.Duration   `json:"timeout"`
	ScheduledAt   *time.Time      `json:"scheduled_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	StartedAt     *time.Time      `json:"started_at,omitempty"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
	LastError     string          `json:"last_error,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	UniqueKey     string          `json:"unique_key,omitempty"`
	Tags          []string        `json:"tags,omitempty"`
}

// NewJobPayload creates a new job payload
func NewJobPayload(jobType string, payload any, opts ...JobOption) (*JobPayload, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	jp := &JobPayload{
		ID:          uuid.New().String(),
		Type:        jobType,
		Payload:     data,
		Priority:    PriorityNormal,
		Status:      JobStatusPending,
		Attempts:    0,
		MaxRetries:  3,
		RetryPolicy: DefaultRetryPolicy(),
		Timeout:     5 * time.Minute,
		CreatedAt:   time.Now(),
	}

	for _, opt := range opts {
		opt(jp)
	}

	return jp, nil
}

// JobOption is a functional option for configuring a job
type JobOption func(*JobPayload)

// WithPriority sets the job priority
func WithPriority(p Priority) JobOption {
	return func(jp *JobPayload) {
		jp.Priority = p
	}
}

// WithRetryPolicy sets a custom retry policy
func WithRetryPolicy(policy RetryPolicy) JobOption {
	return func(jp *JobPayload) {
		jp.RetryPolicy = policy
		jp.MaxRetries = policy.MaxRetries
	}
}

// WithTimeout sets the job timeout
func WithTimeout(d time.Duration) JobOption {
	return func(jp *JobPayload) {
		jp.Timeout = d
	}
}

// WithScheduledAt schedules the job for a specific time
func WithScheduledAt(t time.Time) JobOption {
	return func(jp *JobPayload) {
		jp.ScheduledAt = &t
	}
}

// WithDelay schedules the job after a delay
func WithDelay(d time.Duration) JobOption {
	return func(jp *JobPayload) {
		t := time.Now().Add(d)
		jp.ScheduledAt = &t
	}
}

// WithCorrelationID sets the correlation ID for tracing
func WithCorrelationID(id string) JobOption {
	return func(jp *JobPayload) {
		jp.CorrelationID = id
	}
}

// WithUniqueKey sets a unique key for deduplication
func WithUniqueKey(key string) JobOption {
	return func(jp *JobPayload) {
		jp.UniqueKey = key
	}
}

// WithTags adds tags to the job
func WithTags(tags ...string) JobOption {
	return func(jp *JobPayload) {
		jp.Tags = append(jp.Tags, tags...)
	}
}

// JobResult represents the result of a job execution
type JobResult struct {
	JobID       string        `json:"job_id"`
	Status      JobStatus     `json:"status"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	CompletedAt time.Time     `json:"completed_at"`
}

// Queue is the interface for job queue operations
type Queue interface {
	// Enqueue adds a job to the queue
	Enqueue(ctx context.Context, job *JobPayload) error
	// Dequeue retrieves the next job from the queue
	Dequeue(ctx context.Context, priorities ...Priority) (*JobPayload, error)
	// GetJob retrieves a job by ID
	GetJob(ctx context.Context, jobID string) (*JobPayload, error)
	// UpdateJob updates a job's data
	UpdateJob(ctx context.Context, job *JobPayload) error
	// Complete marks a job as completed
	Complete(ctx context.Context, jobID string) error
	// Fail marks a job as failed and handles retry logic
	Fail(ctx context.Context, jobID string, jobErr error) error
	// ProcessScheduled moves scheduled jobs that are due to their queues
	ProcessScheduled(ctx context.Context) (int, error)
	// GetDLQJobs retrieves jobs from the dead letter queue
	GetDLQJobs(ctx context.Context, limit int64) ([]*JobPayload, error)
	// RetryDLQJob moves a job from DLQ back to the queue
	RetryDLQJob(ctx context.Context, jobID string) error
	// DeleteJob removes a job completely
	DeleteJob(ctx context.Context, jobID string) error
	// RequeueJob adds a job back to the queue
	RequeueJob(ctx context.Context, jobID string, queueKey string) error
	// GetStats returns queue statistics
	GetStats(ctx context.Context) (map[string]int64, error)
}

// ScheduledJobInfo represents information about a scheduled job
type ScheduledJobInfo struct {
	Name      string    `json:"name"`
	Schedule  string    `json:"schedule"`
	JobType   string    `json:"job_type"`
	NextRun   time.Time `json:"next_run"`
	Priority  string    `json:"priority"`
	Singleton bool      `json:"singleton"`
}

// Scheduler is the interface for job scheduler operations
type Scheduler interface {
	// Start starts the scheduler
	Start(ctx context.Context) error
	// Stop stops the scheduler
	Stop(ctx context.Context) error
	// IsLeader returns whether this instance is the leader
	IsLeader() bool
	// ListJobs returns all registered scheduled jobs
	ListJobs() []ScheduledJobInfo
}
