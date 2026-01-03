package jobs

import (
	"context"
	"time"
)

// Service defines the interface for job operations
type Service interface {
	// Enqueue adds a job to the queue
	Enqueue(ctx context.Context, jobType string, payload any, opts ...JobOption) (string, error)

	// EnqueueAt schedules a job for a specific time
	EnqueueAt(ctx context.Context, jobType string, payload any, scheduledAt time.Time, opts ...JobOption) (string, error)

	// EnqueueIn schedules a job after a delay
	EnqueueIn(ctx context.Context, jobType string, payload any, delay time.Duration, opts ...JobOption) (string, error)

	// GetJob retrieves a job by ID
	GetJob(ctx context.Context, jobID string) (*JobPayload, error)

	// CancelJob cancels a pending job
	CancelJob(ctx context.Context, jobID string) error

	// RetryJob retries a failed job
	RetryJob(ctx context.Context, jobID string) error

	// GetQueueStats returns queue statistics
	GetQueueStats(ctx context.Context) (*QueueStats, error)

	// GetDLQJobs returns jobs in the dead letter queue
	GetDLQJobs(ctx context.Context, limit int) ([]*JobPayload, error)

	// RetryDLQJob retries a job from the DLQ
	RetryDLQJob(ctx context.Context, jobID string) error

	// PurgeDLQ removes all jobs from the DLQ
	PurgeDLQ(ctx context.Context) error
}

// QueueStats contains queue statistics
type QueueStats struct {
	Pending        int64            `json:"pending"`
	Scheduled      int64            `json:"scheduled"`
	Running        int64            `json:"running"`
	Completed      int64            `json:"completed"`
	Failed         int64            `json:"failed"`
	Dead           int64            `json:"dead"`
	QueueSizes     map[string]int64 `json:"queue_sizes"`
	WorkerStats    WorkerStats      `json:"worker_stats"`
	SchedulerStats SchedulerStats   `json:"scheduler_stats"`
}

// WorkerStats contains worker pool statistics
type WorkerStats struct {
	Running       bool  `json:"running"`
	ActiveWorkers int64 `json:"active_workers"`
	Concurrency   int   `json:"concurrency"`
	ProcessedJobs int64 `json:"processed_jobs"`
	FailedJobs    int64 `json:"failed_jobs"`
}

// SchedulerStats contains scheduler statistics
type SchedulerStats struct {
	IsLeader          bool     `json:"is_leader"`
	RegisteredJobs    int      `json:"registered_jobs"`
	ScheduledJobNames []string `json:"scheduled_job_names"`
}
