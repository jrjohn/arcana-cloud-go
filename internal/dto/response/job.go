package response

import (
	"time"
)

// JobResponse represents a job in responses
type JobResponse struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	Priority      string     `json:"priority"`
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	MaxRetries    int        `json:"max_retries"`
	ScheduledAt   *time.Time `json:"scheduled_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	CorrelationID string     `json:"correlation_id,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
}

// QueueStatsResponse represents queue statistics
type QueueStatsResponse struct {
	Pending        int64             `json:"pending"`
	Scheduled      int64             `json:"scheduled"`
	Running        int64             `json:"running"`
	Completed      int64             `json:"completed"`
	Failed         int64             `json:"failed"`
	Dead           int64             `json:"dead"`
	QueueSizes     map[string]int64  `json:"queue_sizes"`
	WorkerStats    WorkerStatsResponse    `json:"worker_stats"`
	SchedulerStats SchedulerStatsResponse `json:"scheduler_stats"`
}

// WorkerStatsResponse represents worker pool statistics
type WorkerStatsResponse struct {
	Running       bool  `json:"running"`
	ActiveWorkers int64 `json:"active_workers"`
	Concurrency   int   `json:"concurrency"`
	ProcessedJobs int64 `json:"processed_jobs"`
	FailedJobs    int64 `json:"failed_jobs"`
}

// SchedulerStatsResponse represents scheduler statistics
type SchedulerStatsResponse struct {
	IsLeader          bool     `json:"is_leader"`
	RegisteredJobs    int      `json:"registered_jobs"`
	ScheduledJobNames []string `json:"scheduled_job_names"`
}

// ScheduledJobResponse represents a scheduled job
type ScheduledJobResponse struct {
	Name     string    `json:"name"`
	Schedule string    `json:"schedule"`
	JobType  string    `json:"job_type"`
	NextRun  time.Time `json:"next_run"`
	Priority string    `json:"priority"`
}

// JobEnqueueResponse represents the response after enqueuing a job
type JobEnqueueResponse struct {
	JobID   string `json:"job_id"`
	Message string `json:"message"`
}
