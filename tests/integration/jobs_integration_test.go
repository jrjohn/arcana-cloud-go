// +build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/lock"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/queue"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/scheduler"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/worker"
	"github.com/jrjohn/arcana-cloud-go/internal/testutil"
)

// TestIntegration_FullJobLifecycle tests the complete job lifecycle
func TestIntegration_FullJobLifecycle(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoRedis(t)

	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	// Setup components
	q := queue.NewRedisQueue(client)
	lm := lock.NewLockManager(client, lock.DefaultLockManagerConfig())

	poolConfig := worker.DefaultWorkerPoolConfig()
	poolConfig.Concurrency = 4
	poolConfig.PollInterval = 50 * time.Millisecond
	pool := worker.NewWorkerPool(q, logger, poolConfig)
	pool.SetLockManager(lm)

	sched := scheduler.NewScheduler(client, q, logger)
	service := jobs.NewJobService(q, pool, sched)

	// Track processed jobs
	var processedCount atomic.Int32
	var processedPayloads []string
	var mu sync.Mutex

	type TestPayload struct {
		Message string `json:"message"`
		Index   int    `json:"index"`
	}

	// Register handler
	pool.RegisterHandler("integration-test", func(ctx context.Context, payload []byte) error {
		var p TestPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}
		mu.Lock()
		processedPayloads = append(processedPayloads, p.Message)
		mu.Unlock()
		processedCount.Add(1)
		return nil
	})

	// Start pool
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(context.Background())

	// Enqueue jobs
	jobCount := 10
	jobIDs := make([]string, jobCount)
	for i := 0; i < jobCount; i++ {
		jobID, err := service.Enqueue(ctx, "integration-test", TestPayload{
			Message: "test-message",
			Index:   i,
		})
		if err != nil {
			t.Fatalf("Failed to enqueue job %d: %v", i, err)
		}
		jobIDs[i] = jobID
	}

	// Wait for all jobs to be processed
	testutil.WaitForCondition(t, 10*time.Second, func() bool {
		return processedCount.Load() >= int32(jobCount)
	}, "All jobs should be processed")

	// Verify all jobs completed
	for _, jobID := range jobIDs {
		job, err := service.GetJob(ctx, jobID)
		if err != nil {
			t.Errorf("Failed to get job %s: %v", jobID, err)
			continue
		}
		if job.Status != jobs.JobStatusCompleted {
			t.Errorf("Job %s status = %v, want completed", jobID, job.Status)
		}
	}

	// Check stats
	stats, err := service.GetQueueStats(ctx)
	if err != nil {
		t.Fatalf("GetQueueStats() error = %v", err)
	}
	if stats.Completed < int64(jobCount) {
		t.Errorf("Completed = %v, want >= %v", stats.Completed, jobCount)
	}
}

// TestIntegration_JobFailureAndRetry tests job failure and retry mechanism
func TestIntegration_JobFailureAndRetry(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoRedis(t)

	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	q := queue.NewRedisQueue(client)
	lm := lock.NewLockManager(client, lock.DefaultLockManagerConfig())

	poolConfig := worker.DefaultWorkerPoolConfig()
	poolConfig.Concurrency = 2
	poolConfig.PollInterval = 50 * time.Millisecond
	pool := worker.NewWorkerPool(q, logger, poolConfig)
	pool.SetLockManager(lm)

	var attemptCount atomic.Int32

	// Handler that fails on first 2 attempts
	pool.RegisterHandler("retry-test", func(ctx context.Context, payload []byte) error {
		count := attemptCount.Add(1)
		if count < 3 {
			return errors.New("intentional failure")
		}
		return nil
	})

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Enqueue job with retries
	job, _ := jobs.NewJobPayload("retry-test", nil,
		jobs.WithRetryPolicy(jobs.RetryPolicy{
			MaxRetries:   5,
			Strategy:     jobs.RetryStrategyFixed,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     time.Second,
		}),
	)
	q.Enqueue(ctx, job)

	// Wait for successful completion (after retries)
	testutil.WaitForCondition(t, 30*time.Second, func() bool {
		j, _ := q.GetJob(ctx, job.ID)
		return j != nil && j.Status == jobs.JobStatusCompleted
	}, "Job should complete after retries")

	if attemptCount.Load() < 3 {
		t.Errorf("Attempt count = %v, want >= 3", attemptCount.Load())
	}
}

// TestIntegration_JobMoveToDLQ tests that failed jobs move to DLQ
func TestIntegration_JobMoveToDLQ(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoRedis(t)

	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	q := queue.NewRedisQueue(client)
	lm := lock.NewLockManager(client, lock.DefaultLockManagerConfig())

	poolConfig := worker.DefaultWorkerPoolConfig()
	poolConfig.Concurrency = 1
	poolConfig.PollInterval = 50 * time.Millisecond
	pool := worker.NewWorkerPool(q, logger, poolConfig)
	pool.SetLockManager(lm)

	// Handler that always fails
	pool.RegisterHandler("dlq-test", func(ctx context.Context, payload []byte) error {
		return errors.New("always fails")
	})

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Enqueue job with no retries
	job, _ := jobs.NewJobPayload("dlq-test", nil)
	job.MaxRetries = 0
	q.Enqueue(ctx, job)

	// Wait for job to move to DLQ
	testutil.WaitForCondition(t, 5*time.Second, func() bool {
		j, _ := q.GetJob(ctx, job.ID)
		return j != nil && j.Status == jobs.JobStatusDead
	}, "Job should be in DLQ")

	// Verify job is in DLQ
	dlqJobs, err := q.GetDLQJobs(ctx, 10)
	if err != nil {
		t.Fatalf("GetDLQJobs() error = %v", err)
	}

	found := false
	for _, j := range dlqJobs {
		if j.ID == job.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Job not found in DLQ")
	}
}

// TestIntegration_ScheduledJobs tests scheduled job execution
func TestIntegration_ScheduledJobs(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoRedis(t)

	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	q := queue.NewRedisQueue(client)
	lm := lock.NewLockManager(client, lock.DefaultLockManagerConfig())

	poolConfig := worker.DefaultWorkerPoolConfig()
	poolConfig.Concurrency = 2
	poolConfig.PollInterval = 50 * time.Millisecond
	pool := worker.NewWorkerPool(q, logger, poolConfig)
	pool.SetLockManager(lm)

	var processed atomic.Bool

	pool.RegisterHandler("scheduled-test", func(ctx context.Context, payload []byte) error {
		processed.Store(true)
		return nil
	})

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Schedule a job for now
	job, _ := jobs.NewJobPayload("scheduled-test", nil,
		jobs.WithScheduledAt(time.Now().Add(-time.Second)),
	)
	q.Enqueue(ctx, job)

	// Wait for processing
	testutil.WaitForCondition(t, 5*time.Second, func() bool {
		return processed.Load()
	}, "Scheduled job should be processed")
}

// TestIntegration_PriorityQueue tests priority ordering
func TestIntegration_PriorityQueue(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoRedis(t)

	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	q := queue.NewRedisQueue(client)
	lm := lock.NewLockManager(client, lock.DefaultLockManagerConfig())

	poolConfig := worker.DefaultWorkerPoolConfig()
	poolConfig.Concurrency = 1
	poolConfig.PollInterval = 50 * time.Millisecond
	pool := worker.NewWorkerPool(q, logger, poolConfig)
	pool.SetLockManager(lm)

	var processOrder []jobs.Priority
	var mu sync.Mutex

	type PriorityPayload struct {
		Priority jobs.Priority `json:"priority"`
	}

	pool.RegisterHandler("priority-test", func(ctx context.Context, payload []byte) error {
		var p PriorityPayload
		json.Unmarshal(payload, &p)
		mu.Lock()
		processOrder = append(processOrder, p.Priority)
		mu.Unlock()
		return nil
	})

	// Enqueue jobs in reverse priority order
	priorities := []jobs.Priority{
		jobs.PriorityLow,
		jobs.PriorityNormal,
		jobs.PriorityHigh,
		jobs.PriorityCritical,
	}

	for _, p := range priorities {
		job, _ := jobs.NewJobPayload("priority-test", PriorityPayload{Priority: p}, jobs.WithPriority(p))
		q.Enqueue(ctx, job)
	}

	// Start pool after all jobs are enqueued
	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Wait for all jobs (Redis BLPOP has 1s minimum timeout, so need longer wait)
	testutil.WaitForCondition(t, 15*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(processOrder) == 4
	}, "All jobs should be processed")

	// Verify order (critical > high > normal > low)
	expectedOrder := []jobs.Priority{
		jobs.PriorityCritical,
		jobs.PriorityHigh,
		jobs.PriorityNormal,
		jobs.PriorityLow,
	}

	mu.Lock()
	defer mu.Unlock()
	for i, p := range processOrder {
		if p != expectedOrder[i] {
			t.Errorf("Position %d: got %v, want %v", i, p, expectedOrder[i])
		}
	}
}

// TestIntegration_DistributedLocking tests that only one worker processes a job
func TestIntegration_DistributedLocking(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoRedis(t)

	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	q := queue.NewRedisQueue(client)

	// Create two workers with separate lock managers
	lm1 := lock.NewLockManager(client, lock.DefaultLockManagerConfig())
	lm2 := lock.NewLockManager(client, lock.DefaultLockManagerConfig())

	poolConfig := worker.DefaultWorkerPoolConfig()
	poolConfig.Concurrency = 1
	poolConfig.PollInterval = 50 * time.Millisecond

	pool1 := worker.NewWorkerPool(q, logger, poolConfig)
	pool1.SetLockManager(lm1)

	pool2 := worker.NewWorkerPool(q, logger, poolConfig)
	pool2.SetLockManager(lm2)

	var processCount atomic.Int32

	handlerFunc := func(ctx context.Context, payload []byte) error {
		processCount.Add(1)
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	pool1.RegisterHandler("lock-test", handlerFunc)
	pool2.RegisterHandler("lock-test", handlerFunc)

	pool1.Start(ctx)
	pool2.Start(ctx)
	defer pool1.Stop(context.Background())
	defer pool2.Stop(context.Background())

	// Enqueue job
	job, _ := jobs.NewJobPayload("lock-test", nil)
	q.Enqueue(ctx, job)

	// Wait for processing (Redis BLPOP has 1s minimum timeout)
	testutil.WaitForCondition(t, 5*time.Second, func() bool {
		return processCount.Load() >= 1
	}, "Job should be processed")

	// Wait a bit more to ensure no duplicate processing
	time.Sleep(500 * time.Millisecond)

	// Job should be processed exactly once
	if processCount.Load() != 1 {
		t.Errorf("Process count = %v, want 1 (job processed by multiple workers)", processCount.Load())
	}
}

// TestIntegration_IdempotencyCheck tests that duplicate jobs are not processed
func TestIntegration_IdempotencyCheck(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoRedis(t)

	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	q := queue.NewRedisQueue(client)
	lm := lock.NewLockManager(client, lock.DefaultLockManagerConfig())

	poolConfig := worker.DefaultWorkerPoolConfig()
	poolConfig.Concurrency = 2
	poolConfig.PollInterval = 50 * time.Millisecond
	pool := worker.NewWorkerPool(q, logger, poolConfig)
	pool.SetLockManager(lm)

	var processCount atomic.Int32

	pool.RegisterHandler("idempotent-test", func(ctx context.Context, payload []byte) error {
		processCount.Add(1)
		return nil
	})

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	uniqueKey := testutil.GenerateTestID()

	// First job
	job1, _ := jobs.NewJobPayload("idempotent-test", nil, jobs.WithUniqueKey(uniqueKey))
	q.Enqueue(ctx, job1)

	// Wait for first job
	testutil.WaitForCondition(t, 5*time.Second, func() bool {
		return processCount.Load() >= 1
	}, "First job should be processed")

	// Ensure idempotency is marked
	lm.MarkCompleted(ctx, uniqueKey, job1.ID)

	// Second job with same unique key
	job2, _ := jobs.NewJobPayload("idempotent-test", nil, jobs.WithUniqueKey(uniqueKey))
	q.Enqueue(ctx, job2)

	// Wait a bit
	time.Sleep(500 * time.Millisecond)

	// Should still be 1 (second job skipped)
	if processCount.Load() != 1 {
		t.Errorf("Process count = %v, want 1 (duplicate job processed)", processCount.Load())
	}
}

// TestIntegration_GracefulShutdown tests graceful shutdown of workers
func TestIntegration_GracefulShutdown(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoRedis(t)

	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	q := queue.NewRedisQueue(client)
	lm := lock.NewLockManager(client, lock.DefaultLockManagerConfig())

	poolConfig := worker.DefaultWorkerPoolConfig()
	poolConfig.Concurrency = 2
	poolConfig.PollInterval = 50 * time.Millisecond
	poolConfig.ShutdownTimeout = 5 * time.Second
	pool := worker.NewWorkerPool(q, logger, poolConfig)
	pool.SetLockManager(lm)

	var jobStarted atomic.Bool
	var jobCompleted atomic.Bool

	pool.RegisterHandler("shutdown-test", func(ctx context.Context, payload []byte) error {
		jobStarted.Store(true)
		time.Sleep(500 * time.Millisecond)
		jobCompleted.Store(true)
		return nil
	})

	pool.Start(ctx)

	// Enqueue job
	job, _ := jobs.NewJobPayload("shutdown-test", nil)
	q.Enqueue(ctx, job)

	// Wait for job to start (Redis BLPOP has 1s minimum timeout)
	testutil.WaitForCondition(t, 5*time.Second, func() bool {
		return jobStarted.Load()
	}, "Job should start")

	// Shutdown while job is running
	pool.Stop(context.Background())

	// Job should complete (graceful shutdown)
	if !jobCompleted.Load() {
		t.Error("Job should complete during graceful shutdown")
	}
}

// TestIntegration_SchedulerLeaderElection tests scheduler leader election
func TestIntegration_SchedulerLeaderElection(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoRedis(t)

	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	ctx := context.Background()
	logger := testutil.NewTestLogger(t)

	q := queue.NewRedisQueue(client)

	sched1 := scheduler.NewScheduler(client, q, logger)
	sched2 := scheduler.NewScheduler(client, q, logger)

	sched1.Start(ctx)
	defer sched1.Stop(context.Background())

	// Wait for first scheduler to become leader
	time.Sleep(200 * time.Millisecond)

	sched2.Start(ctx)
	defer sched2.Stop(context.Background())

	// Wait for leader election to settle
	time.Sleep(200 * time.Millisecond)

	// Only one should be leader
	leader1 := sched1.IsLeader()
	leader2 := sched2.IsLeader()

	if leader1 && leader2 {
		t.Error("Both schedulers are leaders - should only be one")
	}
	if !leader1 && !leader2 {
		t.Error("No scheduler is leader - one should be")
	}
}
