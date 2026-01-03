package jobs

import (
	"context"
	"time"
)

// WorkerPool is the interface for worker pool operations
type WorkerPool interface {
	// Start starts the worker pool
	Start(ctx context.Context) error
	// Stop stops the worker pool
	Stop(ctx context.Context) error
	// Stats returns worker pool statistics
	Stats() WorkerPoolStats
}

// WorkerPoolStats contains worker pool statistics
type WorkerPoolStats struct {
	Running       bool
	WorkerID      string
	ActiveWorkers int64
	ProcessedJobs int64
	FailedJobs    int64
	SkippedJobs   int64
	Concurrency   int
}

// jobService implements Service
type jobService struct {
	queue     Queue
	pool      WorkerPool
	scheduler Scheduler
}

// NewJobService creates a new job service
func NewJobService(q Queue, pool WorkerPool, sched Scheduler) Service {
	return &jobService{
		queue:     q,
		pool:      pool,
		scheduler: sched,
	}
}

func (s *jobService) Enqueue(ctx context.Context, jobType string, payload any, opts ...JobOption) (string, error) {
	job, err := NewJobPayload(jobType, payload, opts...)
	if err != nil {
		return "", err
	}

	if err := s.queue.Enqueue(ctx, job); err != nil {
		return "", err
	}

	return job.ID, nil
}

func (s *jobService) EnqueueAt(ctx context.Context, jobType string, payload any, scheduledAt time.Time, opts ...JobOption) (string, error) {
	opts = append(opts, WithScheduledAt(scheduledAt))
	return s.Enqueue(ctx, jobType, payload, opts...)
}

func (s *jobService) EnqueueIn(ctx context.Context, jobType string, payload any, delay time.Duration, opts ...JobOption) (string, error) {
	opts = append(opts, WithDelay(delay))
	return s.Enqueue(ctx, jobType, payload, opts...)
}

func (s *jobService) GetJob(ctx context.Context, jobID string) (*JobPayload, error) {
	return s.queue.GetJob(ctx, jobID)
}

func (s *jobService) CancelJob(ctx context.Context, jobID string) error {
	return s.queue.DeleteJob(ctx, jobID)
}

func (s *jobService) RetryJob(ctx context.Context, jobID string) error {
	job, err := s.queue.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Reset job state
	job.Status = JobStatusPending
	job.Attempts = 0
	job.LastError = ""
	job.ScheduledAt = nil

	return s.queue.Enqueue(ctx, job)
}

func (s *jobService) GetQueueStats(ctx context.Context) (*QueueStats, error) {
	queueStats, err := s.queue.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	poolStats := s.pool.Stats()

	var schedulerStats SchedulerStats
	if s.scheduler != nil {
		scheduledJobs := s.scheduler.ListJobs()
		jobNames := make([]string, len(scheduledJobs))
		for i, j := range scheduledJobs {
			jobNames[i] = j.Name
		}
		schedulerStats = SchedulerStats{
			IsLeader:          s.scheduler.IsLeader(),
			RegisteredJobs:    len(scheduledJobs),
			ScheduledJobNames: jobNames,
		}
	}

	return &QueueStats{
		Pending:   queueStats["pending"],
		Scheduled: queueStats["scheduled"],
		Completed: queueStats["completed_total"],
		Failed:    queueStats["failed_total"],
		Dead:      queueStats["dlq"],
		QueueSizes: map[string]int64{
			"critical": queueStats["queue_critical"],
			"high":     queueStats["queue_high"],
			"normal":   queueStats["queue_normal"],
			"low":      queueStats["queue_low"],
		},
		WorkerStats: WorkerStats{
			Running:       poolStats.Running,
			ActiveWorkers: poolStats.ActiveWorkers,
			Concurrency:   poolStats.Concurrency,
			ProcessedJobs: poolStats.ProcessedJobs,
			FailedJobs:    poolStats.FailedJobs,
		},
		SchedulerStats: schedulerStats,
	}, nil
}

func (s *jobService) GetDLQJobs(ctx context.Context, limit int) ([]*JobPayload, error) {
	return s.queue.GetDLQJobs(ctx, int64(limit))
}

func (s *jobService) RetryDLQJob(ctx context.Context, jobID string) error {
	return s.queue.RetryDLQJob(ctx, jobID)
}

func (s *jobService) PurgeDLQ(ctx context.Context) error {
	jobs, err := s.queue.GetDLQJobs(ctx, 10000)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		s.queue.DeleteJob(ctx, job.ID)
	}

	return nil
}
