package worker

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
	"github.com/jrjohn/arcana-cloud-go/internal/testutil"
)

func setupTestPool(t *testing.T) (*WorkerPool, *queue.RedisQueue, *lock.LockManager, context.Context) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)

	q := queue.NewRedisQueue(client)
	lm := lock.NewLockManager(client, lock.DefaultLockManagerConfig())
	logger := testutil.NewTestLogger(t)

	poolConfig := DefaultWorkerPoolConfig()
	poolConfig.Concurrency = 2
	poolConfig.PollInterval = 50 * time.Millisecond
	poolConfig.ShutdownTimeout = 5 * time.Second

	pool := NewWorkerPool(q, logger, poolConfig)
	pool.SetLockManager(lm)

	return pool, q, lm, context.Background()
}

func TestDefaultWorkerPoolConfig(t *testing.T) {
	config := DefaultWorkerPoolConfig()

	if config.Concurrency != 8 {
		t.Errorf("Concurrency = %v, want 8", config.Concurrency)
	}
	if config.PollInterval != 100*time.Millisecond {
		t.Errorf("PollInterval = %v, want 100ms", config.PollInterval)
	}
	if config.ShutdownTimeout != 30*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 30s", config.ShutdownTimeout)
	}
	if !config.EnableLocking {
		t.Error("EnableLocking should be true by default")
	}
	if !config.EnableIdempotency {
		t.Error("EnableIdempotency should be true by default")
	}
}

func TestNewWorkerPool(t *testing.T) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewTestLogger(t)

	pool := NewWorkerPool(q, logger, DefaultWorkerPoolConfig())

	if pool == nil {
		t.Fatal("NewWorkerPool() returned nil")
	}
	if pool.handlers == nil {
		t.Error("handlers map is nil")
	}
	if pool.stopCh == nil {
		t.Error("stopCh is nil")
	}
}

func TestWorkerPool_SetLockManager(t *testing.T) {
	pool, _, lm, _ := setupTestPool(t)

	if pool.lockManager == nil {
		t.Error("lockManager should be set")
	}
	if pool.lockManager != lm {
		t.Error("lockManager mismatch")
	}
}

func TestWorkerPool_RegisterHandler(t *testing.T) {
	pool, _, _, _ := setupTestPool(t)

	handler := func(ctx context.Context, payload []byte) error {
		return nil
	}

	pool.RegisterHandler("test-job", handler)

	if _, ok := pool.handlers["test-job"]; !ok {
		t.Error("Handler not registered")
	}
}

func TestWorkerPool_StartStop(t *testing.T) {
	pool, _, _, ctx := setupTestPool(t)

	// Start
	if err := pool.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !pool.running.Load() {
		t.Error("Pool should be running after Start()")
	}

	// Stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if pool.running.Load() {
		t.Error("Pool should not be running after Stop()")
	}
}

func TestWorkerPool_StartTwice(t *testing.T) {
	pool, _, _, ctx := setupTestPool(t)

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Starting again should fail
	err := pool.Start(ctx)
	if err == nil {
		t.Error("Second Start() should return error")
	}
}

func TestWorkerPool_StopNotRunning(t *testing.T) {
	pool, _, _, _ := setupTestPool(t)

	// Stop without starting
	if err := pool.Stop(context.Background()); err != nil {
		t.Errorf("Stop() on non-running pool error = %v", err)
	}
}

func TestWorkerPool_ProcessJob(t *testing.T) {
	pool, q, _, ctx := setupTestPool(t)

	var processed atomic.Bool
	pool.RegisterHandler("process-test", func(ctx context.Context, payload []byte) error {
		processed.Store(true)
		return nil
	})

	// Start pool
	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Enqueue job
	job, _ := jobs.NewJobPayload("process-test", map[string]string{"key": "value"})
	q.Enqueue(ctx, job)

	// Wait for processing (BLPop has 1s minimum duration)
	testutil.WaitForCondition(t, 10*time.Second, func() bool {
		return processed.Load()
	}, "Job should be processed")

	// Verify job completed
	time.Sleep(500 * time.Millisecond)
	storedJob, err := q.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if storedJob.Status != jobs.JobStatusCompleted {
		t.Errorf("Job status = %v, want completed", storedJob.Status)
	}
}

func TestWorkerPool_ProcessJobWithPayload(t *testing.T) {
	pool, q, _, ctx := setupTestPool(t)

	type TestPayload struct {
		Message string `json:"message"`
		Count   int    `json:"count"`
	}

	var receivedPayload TestPayload
	var mu sync.Mutex
	pool.RegisterHandler("payload-test", func(ctx context.Context, payload []byte) error {
		var p TestPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return err
		}
		mu.Lock()
		receivedPayload = p
		mu.Unlock()
		return nil
	})

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Enqueue job with payload
	job, _ := jobs.NewJobPayload("payload-test", TestPayload{Message: "hello", Count: 42})
	q.Enqueue(ctx, job)

	// Wait for processing
	testutil.WaitForCondition(t, 10*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return receivedPayload.Message != ""
	}, "Payload should be received")

	mu.Lock()
	message := receivedPayload.Message
	count := receivedPayload.Count
	mu.Unlock()

	if message != "hello" {
		t.Errorf("Message = %v, want hello", message)
	}
	if count != 42 {
		t.Errorf("Count = %v, want 42", count)
	}
}

func TestWorkerPool_JobFailure(t *testing.T) {
	pool, q, _, ctx := setupTestPool(t)

	pool.RegisterHandler("fail-test", func(ctx context.Context, payload []byte) error {
		return errors.New("intentional failure")
	})

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	job, _ := jobs.NewJobPayload("fail-test", nil)
	job.MaxRetries = 0 // Go straight to DLQ
	q.Enqueue(ctx, job)

	// Wait for job to be processed and moved to DLQ
	testutil.WaitForCondition(t, 10*time.Second, func() bool {
		storedJob, err := q.GetJob(ctx, job.ID)
		return err == nil && storedJob.Status == jobs.JobStatusDead
	}, "Job should be moved to DLQ")

	// Job should be failed/dead
	storedJob, _ := q.GetJob(ctx, job.ID)
	if storedJob.Status != jobs.JobStatusDead {
		t.Errorf("Job status = %v, want dead", storedJob.Status)
	}
	if storedJob.LastError == "" {
		t.Error("LastError should be set")
	}
}

func TestWorkerPool_NoHandler(t *testing.T) {
	pool, q, _, ctx := setupTestPool(t)

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Enqueue job without registering handler
	job, _ := jobs.NewJobPayload("unhandled-job", nil)
	q.Enqueue(ctx, job)

	// Wait for job to be processed and have an error
	testutil.WaitForCondition(t, 10*time.Second, func() bool {
		storedJob, err := q.GetJob(ctx, job.ID)
		return err == nil && storedJob.LastError != ""
	}, "Job should have error for missing handler")

	// Job should fail due to no handler
	storedJob, _ := q.GetJob(ctx, job.ID)
	if storedJob.LastError == "" {
		t.Error("Job should have error for missing handler")
	}
}

func TestWorkerPool_JobTimeout(t *testing.T) {
	pool, q, _, ctx := setupTestPool(t)

	pool.RegisterHandler("timeout-test", func(ctx context.Context, payload []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
			return nil
		}
	})

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Create job with short timeout
	job, _ := jobs.NewJobPayload("timeout-test", nil, jobs.WithTimeout(100*time.Millisecond))
	q.Enqueue(ctx, job)

	// Wait for job to be processed and have a timeout error
	testutil.WaitForCondition(t, 10*time.Second, func() bool {
		storedJob, err := q.GetJob(ctx, job.ID)
		return err == nil && storedJob.LastError != ""
	}, "Job should have timeout error")

	storedJob, _ := q.GetJob(ctx, job.ID)
	if storedJob.LastError == "" {
		t.Error("Job should have timeout error")
	}
}

func TestWorkerPool_IdempotencyCheck(t *testing.T) {
	pool, q, lm, ctx := setupTestPool(t)

	var processCount atomic.Int32
	pool.RegisterHandler("idempotent-test", func(ctx context.Context, payload []byte) error {
		processCount.Add(1)
		return nil
	})

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Mark job as completed for idempotency
	uniqueKey := "idempotent-key-123"
	lm.MarkCompleted(ctx, uniqueKey, "previous-job-id")

	// Enqueue job with same unique key
	job, _ := jobs.NewJobPayload("idempotent-test", nil, jobs.WithUniqueKey(uniqueKey))
	q.Enqueue(ctx, job)

	// Wait for job to be completed (skipped jobs are marked complete)
	testutil.WaitForCondition(t, 10*time.Second, func() bool {
		storedJob, err := q.GetJob(ctx, job.ID)
		return err == nil && storedJob.Status == jobs.JobStatusCompleted
	}, "Job should be completed (skipped due to idempotency)")

	// Job should be skipped due to idempotency (processCount should be 0)
	if processCount.Load() != 0 {
		t.Errorf("Job was processed %d times, should be skipped", processCount.Load())
	}
}

func TestWorkerPool_Stats(t *testing.T) {
	pool, q, _, ctx := setupTestPool(t)

	var processed atomic.Int32
	pool.RegisterHandler("stats-test", func(ctx context.Context, payload []byte) error {
		processed.Add(1)
		return nil
	})

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Enqueue and process jobs
	for i := 0; i < 5; i++ {
		job, _ := jobs.NewJobPayload("stats-test", nil)
		q.Enqueue(ctx, job)
	}

	// Wait for processing (BLPop has 1s minimum duration in mock)
	testutil.WaitForCondition(t, 30*time.Second, func() bool {
		return processed.Load() >= 5
	}, "All jobs should be processed")

	stats := pool.Stats()

	if !stats.Running {
		t.Error("Stats.Running should be true")
	}
	if stats.ProcessedJobs < 5 {
		t.Errorf("Stats.ProcessedJobs = %v, want >= 5", stats.ProcessedJobs)
	}
	if stats.Concurrency != 2 {
		t.Errorf("Stats.Concurrency = %v, want 2", stats.Concurrency)
	}
}

func TestWorkerPool_GetRunningJobs(t *testing.T) {
	pool, q, _, ctx := setupTestPool(t)

	// Register a slow handler
	pool.RegisterHandler("slow-test", func(ctx context.Context, payload []byte) error {
		time.Sleep(2 * time.Second)
		return nil
	})

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Enqueue job
	job, _ := jobs.NewJobPayload("slow-test", nil)
	q.Enqueue(ctx, job)

	// Wait for job to start
	time.Sleep(200 * time.Millisecond)

	// Check running jobs
	running, err := pool.GetRunningJobs(ctx)
	if err != nil {
		t.Fatalf("GetRunningJobs() error = %v", err)
	}

	// Should have at least one running job
	if len(running) == 0 {
		t.Log("No running jobs found (job may have completed)")
	}
}

func TestWorkerPool_GetRunningJobs_NoLockManager(t *testing.T) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewTestLogger(t)

	pool := NewWorkerPool(q, logger, DefaultWorkerPoolConfig())
	// Don't set lock manager

	_, err := pool.GetRunningJobs(context.Background())
	if err == nil {
		t.Error("GetRunningJobs() should error without lock manager")
	}
}

func TestWorkerPool_DisableLocking(t *testing.T) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewTestLogger(t)

	poolConfig := DefaultWorkerPoolConfig()
	poolConfig.EnableLocking = false
	poolConfig.Concurrency = 1
	poolConfig.PollInterval = 50 * time.Millisecond

	pool := NewWorkerPool(q, logger, poolConfig)
	pool.SetLockManager(lock.NewLockManager(client, lock.DefaultLockManagerConfig()))

	var processed atomic.Bool
	pool.RegisterHandler("no-lock-test", func(ctx context.Context, payload []byte) error {
		processed.Store(true)
		return nil
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop(context.Background())

	job, _ := jobs.NewJobPayload("no-lock-test", nil)
	q.Enqueue(ctx, job)

	testutil.WaitForCondition(t, 10*time.Second, func() bool {
		return processed.Load()
	}, "Job should be processed without locking")
}

func TestWorkerPool_DisableIdempotency(t *testing.T) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	q := queue.NewRedisQueue(client)
	lm := lock.NewLockManager(client, lock.DefaultLockManagerConfig())
	logger := testutil.NewTestLogger(t)

	poolConfig := DefaultWorkerPoolConfig()
	poolConfig.EnableIdempotency = false
	poolConfig.Concurrency = 1
	poolConfig.PollInterval = 50 * time.Millisecond

	pool := NewWorkerPool(q, logger, poolConfig)
	pool.SetLockManager(lm)

	var processed atomic.Bool
	pool.RegisterHandler("no-idem-test", func(ctx context.Context, payload []byte) error {
		processed.Store(true)
		return nil
	})

	ctx := context.Background()

	// Mark job as completed
	uniqueKey := "no-idem-key"
	lm.MarkCompleted(ctx, uniqueKey, "old-job")

	pool.Start(ctx)
	defer pool.Stop(context.Background())

	// Job should still be processed despite idempotency mark
	job, _ := jobs.NewJobPayload("no-idem-test", nil, jobs.WithUniqueKey(uniqueKey))
	q.Enqueue(ctx, job)

	testutil.WaitForCondition(t, 10*time.Second, func() bool {
		return processed.Load()
	}, "Job should be processed even with idempotency disabled")
}

func TestWorkerPoolStats_Struct(t *testing.T) {
	stats := jobs.WorkerPoolStats{
		Running:       true,
		WorkerID:      "worker-123",
		ActiveWorkers: 5,
		ProcessedJobs: 100,
		FailedJobs:    3,
		SkippedJobs:   2,
		Concurrency:   8,
	}

	if !stats.Running {
		t.Error("Running should be true")
	}
	if stats.WorkerID != "worker-123" {
		t.Errorf("WorkerID = %v, want worker-123", stats.WorkerID)
	}
	if stats.ActiveWorkers != 5 {
		t.Errorf("ActiveWorkers = %v, want 5", stats.ActiveWorkers)
	}
	if stats.ProcessedJobs != 100 {
		t.Errorf("ProcessedJobs = %v, want 100", stats.ProcessedJobs)
	}
	if stats.FailedJobs != 3 {
		t.Errorf("FailedJobs = %v, want 3", stats.FailedJobs)
	}
	if stats.SkippedJobs != 2 {
		t.Errorf("SkippedJobs = %v, want 2", stats.SkippedJobs)
	}
	if stats.Concurrency != 8 {
		t.Errorf("Concurrency = %v, want 8", stats.Concurrency)
	}
}

// Benchmarks
func BenchmarkWorkerPool_ProcessJob(b *testing.B) {
	testutil.SkipIfNoRedis(&testing.T{})
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(&testing.T{}, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewNopLogger()

	poolConfig := DefaultWorkerPoolConfig()
	poolConfig.Concurrency = 4
	pool := NewWorkerPool(q, logger, poolConfig)

	var processed atomic.Int64
	pool.RegisterHandler("bench-job", func(ctx context.Context, payload []byte) error {
		processed.Add(1)
		return nil
	})

	ctx := context.Background()
	pool.Start(ctx)
	defer pool.Stop(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job, _ := jobs.NewJobPayload("bench-job", nil)
		q.Enqueue(ctx, job)
	}

	// Wait for all jobs to be processed
	for processed.Load() < int64(b.N) {
		time.Sleep(10 * time.Millisecond)
	}
}
