package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----- Mock Queue -----

type mockQueue struct {
	enqueueFunc          func(ctx context.Context, job *JobPayload) error
	dequeueFunc          func(ctx context.Context, priorities ...Priority) (*JobPayload, error)
	getJobFunc           func(ctx context.Context, jobID string) (*JobPayload, error)
	updateJobFunc        func(ctx context.Context, job *JobPayload) error
	completeFunc         func(ctx context.Context, jobID string) error
	failFunc             func(ctx context.Context, jobID string, jobErr error) error
	processScheduledFunc func(ctx context.Context) (int, error)
	getDLQJobsFunc       func(ctx context.Context, limit int64) ([]*JobPayload, error)
	retryDLQJobFunc      func(ctx context.Context, jobID string) error
	deleteJobFunc        func(ctx context.Context, jobID string) error
	requeueJobFunc       func(ctx context.Context, jobID string, queueKey string) error
	getStatsFunc         func(ctx context.Context) (map[string]int64, error)
}

func newDefaultMockQueue() *mockQueue {
	return &mockQueue{
		enqueueFunc: func(_ context.Context, _ *JobPayload) error { return nil },
		dequeueFunc: func(_ context.Context, _ ...Priority) (*JobPayload, error) { return nil, ErrQueueEmpty },
		getJobFunc: func(_ context.Context, jobID string) (*JobPayload, error) {
			return &JobPayload{ID: jobID, Type: "test", Status: JobStatusPending}, nil
		},
		updateJobFunc:        func(_ context.Context, _ *JobPayload) error { return nil },
		completeFunc:         func(_ context.Context, _ string) error { return nil },
		failFunc:             func(_ context.Context, _ string, _ error) error { return nil },
		processScheduledFunc: func(_ context.Context) (int, error) { return 0, nil },
		getDLQJobsFunc:       func(_ context.Context, _ int64) ([]*JobPayload, error) { return []*JobPayload{}, nil },
		retryDLQJobFunc:      func(_ context.Context, _ string) error { return nil },
		deleteJobFunc:        func(_ context.Context, _ string) error { return nil },
		requeueJobFunc:       func(_ context.Context, _ string, _ string) error { return nil },
		getStatsFunc: func(_ context.Context) (map[string]int64, error) {
			return map[string]int64{
				"pending":          5,
				"scheduled":        2,
				"completed_total":  100,
				"failed_total":     10,
				"dlq":              3,
				"queue_critical":   1,
				"queue_high":       2,
				"queue_normal":     2,
				"queue_low":        0,
			}, nil
		},
	}
}

func (m *mockQueue) Enqueue(ctx context.Context, job *JobPayload) error {
	return m.enqueueFunc(ctx, job)
}
func (m *mockQueue) Dequeue(ctx context.Context, priorities ...Priority) (*JobPayload, error) {
	return m.dequeueFunc(ctx, priorities...)
}
func (m *mockQueue) GetJob(ctx context.Context, jobID string) (*JobPayload, error) {
	return m.getJobFunc(ctx, jobID)
}
func (m *mockQueue) UpdateJob(ctx context.Context, job *JobPayload) error {
	return m.updateJobFunc(ctx, job)
}
func (m *mockQueue) Complete(ctx context.Context, jobID string) error {
	return m.completeFunc(ctx, jobID)
}
func (m *mockQueue) Fail(ctx context.Context, jobID string, jobErr error) error {
	return m.failFunc(ctx, jobID, jobErr)
}
func (m *mockQueue) ProcessScheduled(ctx context.Context) (int, error) {
	return m.processScheduledFunc(ctx)
}
func (m *mockQueue) GetDLQJobs(ctx context.Context, limit int64) ([]*JobPayload, error) {
	return m.getDLQJobsFunc(ctx, limit)
}
func (m *mockQueue) RetryDLQJob(ctx context.Context, jobID string) error {
	return m.retryDLQJobFunc(ctx, jobID)
}
func (m *mockQueue) DeleteJob(ctx context.Context, jobID string) error {
	return m.deleteJobFunc(ctx, jobID)
}
func (m *mockQueue) RequeueJob(ctx context.Context, jobID string, queueKey string) error {
	return m.requeueJobFunc(ctx, jobID, queueKey)
}
func (m *mockQueue) GetStats(ctx context.Context) (map[string]int64, error) {
	return m.getStatsFunc(ctx)
}

// ----- Mock WorkerPool -----

type mockWorkerPool struct {
	stats WorkerPoolStats
}

func (m *mockWorkerPool) Start(_ context.Context) error { return nil }
func (m *mockWorkerPool) Stop(_ context.Context) error  { return nil }
func (m *mockWorkerPool) Stats() WorkerPoolStats        { return m.stats }

// ----- Mock Scheduler -----

type mockScheduler struct {
	isLeader bool
	jobs     []ScheduledJobInfo
}

func (m *mockScheduler) Start(_ context.Context) error     { return nil }
func (m *mockScheduler) Stop(_ context.Context) error      { return nil }
func (m *mockScheduler) IsLeader() bool                    { return m.isLeader }
func (m *mockScheduler) ListJobs() []ScheduledJobInfo      { return m.jobs }

// ----- Tests -----

func newTestJobService(q Queue, pool WorkerPool, sched Scheduler) Service {
	return NewJobService(q, pool, sched)
}

// TestNewJobService creates a service
func TestNewJobService(t *testing.T) {
	q := newDefaultMockQueue()
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)
	assert.NotNil(t, svc)
}

// TestJobService_Enqueue_Success enqueues a job
func TestJobService_Enqueue_Success(t *testing.T) {
	q := newDefaultMockQueue()
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	jobID, err := svc.Enqueue(context.Background(), "email-job", map[string]string{"to": "user@example.com"})
	require.NoError(t, err)
	assert.NotEmpty(t, jobID)
}

// TestJobService_Enqueue_WithOptions applies options
func TestJobService_Enqueue_WithOptions(t *testing.T) {
	q := newDefaultMockQueue()
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	jobID, err := svc.Enqueue(context.Background(), "email-job", nil,
		WithPriority(PriorityHigh),
		WithTimeout(60*time.Second),
		WithTags("email", "notification"),
	)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID)
}

// TestJobService_Enqueue_QueueError propagates error
func TestJobService_Enqueue_QueueError(t *testing.T) {
	q := newDefaultMockQueue()
	q.enqueueFunc = func(_ context.Context, _ *JobPayload) error {
		return errors.New("queue full")
	}
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	_, err := svc.Enqueue(context.Background(), "test-job", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue full")
}

// TestJobService_EnqueueAt schedules a job at a specific time
func TestJobService_EnqueueAt(t *testing.T) {
	q := newDefaultMockQueue()
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	scheduledAt := time.Now().Add(time.Hour)
	jobID, err := svc.EnqueueAt(context.Background(), "scheduled-job", nil, scheduledAt)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID)
}

// TestJobService_EnqueueIn schedules a job after a delay
func TestJobService_EnqueueIn(t *testing.T) {
	q := newDefaultMockQueue()
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	jobID, err := svc.EnqueueIn(context.Background(), "delayed-job", nil, 30*time.Second)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID)
}

// TestJobService_GetJob_Success returns the job
func TestJobService_GetJob_Success(t *testing.T) {
	q := newDefaultMockQueue()
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	job, err := svc.GetJob(context.Background(), "job-123")
	require.NoError(t, err)
	assert.Equal(t, "job-123", job.ID)
}

// TestJobService_GetJob_Error propagates error
func TestJobService_GetJob_Error(t *testing.T) {
	q := newDefaultMockQueue()
	q.getJobFunc = func(_ context.Context, _ string) (*JobPayload, error) {
		return nil, ErrJobNotFound
	}
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	_, err := svc.GetJob(context.Background(), "missing-job")
	assert.ErrorIs(t, err, ErrJobNotFound)
}

// TestJobService_CancelJob_Success deletes the job
func TestJobService_CancelJob_Success(t *testing.T) {
	q := newDefaultMockQueue()
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	err := svc.CancelJob(context.Background(), "job-123")
	assert.NoError(t, err)
}

// TestJobService_CancelJob_Error propagates error
func TestJobService_CancelJob_Error(t *testing.T) {
	q := newDefaultMockQueue()
	q.deleteJobFunc = func(_ context.Context, _ string) error {
		return errors.New("delete failed")
	}
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	err := svc.CancelJob(context.Background(), "job-123")
	assert.Error(t, err)
}

// TestJobService_RetryJob_Success re-enqueues a failed job
func TestJobService_RetryJob_Success(t *testing.T) {
	q := newDefaultMockQueue()
	var enqueuedJob *JobPayload
	q.enqueueFunc = func(_ context.Context, job *JobPayload) error {
		enqueuedJob = job
		return nil
	}
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	err := svc.RetryJob(context.Background(), "job-123")
	require.NoError(t, err)
	require.NotNil(t, enqueuedJob)
	assert.Equal(t, JobStatusPending, enqueuedJob.Status)
	assert.Equal(t, 0, enqueuedJob.Attempts)
	assert.Empty(t, enqueuedJob.LastError)
	assert.Nil(t, enqueuedJob.ScheduledAt)
}

// TestJobService_RetryJob_GetJobError returns error if job not found
func TestJobService_RetryJob_GetJobError(t *testing.T) {
	q := newDefaultMockQueue()
	q.getJobFunc = func(_ context.Context, _ string) (*JobPayload, error) {
		return nil, ErrJobNotFound
	}
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	err := svc.RetryJob(context.Background(), "missing-job")
	assert.ErrorIs(t, err, ErrJobNotFound)
}

// TestJobService_GetQueueStats_Success returns stats
func TestJobService_GetQueueStats_Success(t *testing.T) {
	q := newDefaultMockQueue()
	pool := &mockWorkerPool{
		stats: WorkerPoolStats{
			Running:       true,
			ActiveWorkers: 2,
			Concurrency:   4,
			ProcessedJobs: 100,
			FailedJobs:    5,
		},
	}
	svc := newTestJobService(q, pool, nil)

	stats, err := svc.GetQueueStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(5), stats.Pending)
	assert.Equal(t, int64(2), stats.Scheduled)
	assert.Equal(t, int64(100), stats.Completed)
	assert.Equal(t, int64(10), stats.Failed)
	assert.Equal(t, int64(3), stats.Dead)
	assert.True(t, stats.WorkerStats.Running)
	assert.Equal(t, int64(2), stats.WorkerStats.ActiveWorkers)
	assert.Equal(t, 4, stats.WorkerStats.Concurrency)
}

// TestJobService_GetQueueStats_WithScheduler includes scheduler stats
func TestJobService_GetQueueStats_WithScheduler(t *testing.T) {
	q := newDefaultMockQueue()
	pool := &mockWorkerPool{}
	sched := &mockScheduler{
		isLeader: true,
		jobs: []ScheduledJobInfo{
			{Name: "cleanup-job", Schedule: "0 * * * *", JobType: "cleanup"},
			{Name: "report-job", Schedule: "0 9 * * *", JobType: "report"},
		},
	}
	svc := newTestJobService(q, pool, sched)

	stats, err := svc.GetQueueStats(context.Background())
	require.NoError(t, err)
	assert.True(t, stats.SchedulerStats.IsLeader)
	assert.Equal(t, 2, stats.SchedulerStats.RegisteredJobs)
	assert.Contains(t, stats.SchedulerStats.ScheduledJobNames, "cleanup-job")
	assert.Contains(t, stats.SchedulerStats.ScheduledJobNames, "report-job")
}

// TestJobService_GetQueueStats_Error propagates error
func TestJobService_GetQueueStats_Error(t *testing.T) {
	q := newDefaultMockQueue()
	q.getStatsFunc = func(_ context.Context) (map[string]int64, error) {
		return nil, errors.New("redis unavailable")
	}
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	_, err := svc.GetQueueStats(context.Background())
	assert.Error(t, err)
}

// TestJobService_GetDLQJobs_Success returns DLQ jobs
func TestJobService_GetDLQJobs_Success(t *testing.T) {
	q := newDefaultMockQueue()
	q.getDLQJobsFunc = func(_ context.Context, _ int64) ([]*JobPayload, error) {
		return []*JobPayload{
			{ID: "dlq-1", Type: "email", Status: JobStatusDead},
			{ID: "dlq-2", Type: "webhook", Status: JobStatusDead},
		}, nil
	}
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	jobs, err := svc.GetDLQJobs(context.Background(), 10)
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
	assert.Equal(t, "dlq-1", jobs[0].ID)
}

// TestJobService_GetDLQJobs_Error propagates error
func TestJobService_GetDLQJobs_Error(t *testing.T) {
	q := newDefaultMockQueue()
	q.getDLQJobsFunc = func(_ context.Context, _ int64) ([]*JobPayload, error) {
		return nil, errors.New("dlq unavailable")
	}
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	_, err := svc.GetDLQJobs(context.Background(), 10)
	assert.Error(t, err)
}

// TestJobService_RetryDLQJob_Success retries a DLQ job
func TestJobService_RetryDLQJob_Success(t *testing.T) {
	q := newDefaultMockQueue()
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	err := svc.RetryDLQJob(context.Background(), "dlq-job-1")
	assert.NoError(t, err)
}

// TestJobService_RetryDLQJob_Error propagates error
func TestJobService_RetryDLQJob_Error(t *testing.T) {
	q := newDefaultMockQueue()
	q.retryDLQJobFunc = func(_ context.Context, _ string) error {
		return errors.New("retry failed")
	}
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	err := svc.RetryDLQJob(context.Background(), "dlq-job-1")
	assert.Error(t, err)
}

// TestJobService_PurgeDLQ_Success removes all DLQ jobs
func TestJobService_PurgeDLQ_Success(t *testing.T) {
	deletedIDs := []string{}
	q := newDefaultMockQueue()
	q.getDLQJobsFunc = func(_ context.Context, _ int64) ([]*JobPayload, error) {
		return []*JobPayload{
			{ID: "dlq-1"},
			{ID: "dlq-2"},
			{ID: "dlq-3"},
		}, nil
	}
	q.deleteJobFunc = func(_ context.Context, jobID string) error {
		deletedIDs = append(deletedIDs, jobID)
		return nil
	}
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	err := svc.PurgeDLQ(context.Background())
	require.NoError(t, err)
	assert.Len(t, deletedIDs, 3)
	assert.Contains(t, deletedIDs, "dlq-1")
	assert.Contains(t, deletedIDs, "dlq-2")
	assert.Contains(t, deletedIDs, "dlq-3")
}

// TestJobService_PurgeDLQ_GetJobsError propagates error
func TestJobService_PurgeDLQ_GetJobsError(t *testing.T) {
	q := newDefaultMockQueue()
	q.getDLQJobsFunc = func(_ context.Context, _ int64) ([]*JobPayload, error) {
		return nil, errors.New("cannot fetch DLQ")
	}
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	err := svc.PurgeDLQ(context.Background())
	assert.Error(t, err)
}

// TestJobService_PurgeDLQ_Empty purges without error when DLQ is empty
func TestJobService_PurgeDLQ_Empty(t *testing.T) {
	q := newDefaultMockQueue()
	pool := &mockWorkerPool{}
	svc := newTestJobService(q, pool, nil)

	err := svc.PurgeDLQ(context.Background())
	assert.NoError(t, err)
}
