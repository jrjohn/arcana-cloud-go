package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/queue"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/scheduler"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/worker"
	"github.com/jrjohn/arcana-cloud-go/internal/testutil"
)

func setupTestJobService(t *testing.T) (jobs.Service, *queue.RedisQueue, *worker.WorkerPool, *scheduler.Scheduler, context.Context) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)

	q := queue.NewRedisQueue(client)
	logger := testutil.NewTestLogger(t)

	poolConfig := worker.DefaultWorkerPoolConfig()
	pool := worker.NewWorkerPool(q, logger, poolConfig)

	sched := scheduler.NewScheduler(client, q, logger)

	service := jobs.NewJobService(q, pool, sched)
	return service, q, pool, sched, context.Background()
}

func TestNewJobService(t *testing.T) {
	service, _, _, _, _ := setupTestJobService(t)

	if service == nil {
		t.Fatal("NewJobService() returned nil")
	}
}

func TestJobService_Enqueue(t *testing.T) {
	service, q, _, _, ctx := setupTestJobService(t)

	type TestPayload struct {
		Message string `json:"message"`
	}

	jobID, err := service.Enqueue(ctx, "test-job", TestPayload{Message: "hello"})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if jobID == "" {
		t.Error("Enqueue() returned empty job ID")
	}

	// Verify job is in queue
	job, err := q.GetJob(ctx, jobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if job.Type != "test-job" {
		t.Errorf("Job type = %v, want test-job", job.Type)
	}
}

func TestJobService_Enqueue_WithOptions(t *testing.T) {
	service, q, _, _, ctx := setupTestJobService(t)

	jobID, err := service.Enqueue(ctx, "options-job", nil,
		jobs.WithPriority(jobs.PriorityHigh),
		jobs.WithUniqueKey("unique-123"),
		jobs.WithTimeout(10*time.Minute),
	)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	job, _ := q.GetJob(ctx, jobID)
	if job.Priority != jobs.PriorityHigh {
		t.Errorf("Priority = %v, want high", job.Priority)
	}
	if job.UniqueKey != "unique-123" {
		t.Errorf("UniqueKey = %v, want unique-123", job.UniqueKey)
	}
	if job.Timeout != 10*time.Minute {
		t.Errorf("Timeout = %v, want 10m", job.Timeout)
	}
}

func TestJobService_EnqueueAt(t *testing.T) {
	service, q, _, _, ctx := setupTestJobService(t)

	scheduledTime := time.Now().Add(time.Hour)
	jobID, err := service.EnqueueAt(ctx, "scheduled-job", nil, scheduledTime)
	if err != nil {
		t.Fatalf("EnqueueAt() error = %v", err)
	}

	job, _ := q.GetJob(ctx, jobID)
	if job.ScheduledAt == nil {
		t.Error("ScheduledAt should be set")
	}
	if !job.ScheduledAt.Equal(scheduledTime) {
		t.Errorf("ScheduledAt = %v, want %v", job.ScheduledAt, scheduledTime)
	}
}

func TestJobService_EnqueueIn(t *testing.T) {
	service, q, _, _, ctx := setupTestJobService(t)

	before := time.Now()
	jobID, err := service.EnqueueIn(ctx, "delayed-job", nil, time.Hour)
	if err != nil {
		t.Fatalf("EnqueueIn() error = %v", err)
	}
	after := time.Now()

	job, _ := q.GetJob(ctx, jobID)
	if job.ScheduledAt == nil {
		t.Error("ScheduledAt should be set")
	}

	expectedMin := before.Add(time.Hour)
	expectedMax := after.Add(time.Hour)
	if job.ScheduledAt.Before(expectedMin) || job.ScheduledAt.After(expectedMax) {
		t.Errorf("ScheduledAt = %v, expected between %v and %v", job.ScheduledAt, expectedMin, expectedMax)
	}
}

func TestJobService_GetJob(t *testing.T) {
	service, _, _, _, ctx := setupTestJobService(t)

	// Enqueue a job
	jobID, _ := service.Enqueue(ctx, "get-job", map[string]string{"key": "value"})

	// Get the job
	job, err := service.GetJob(ctx, jobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if job.ID != jobID {
		t.Errorf("Job ID = %v, want %v", job.ID, jobID)
	}
	if job.Type != "get-job" {
		t.Errorf("Job Type = %v, want get-job", job.Type)
	}
}

func TestJobService_GetJob_NotFound(t *testing.T) {
	service, _, _, _, ctx := setupTestJobService(t)

	_, err := service.GetJob(ctx, "nonexistent-id")
	if err == nil {
		t.Error("GetJob() should return error for non-existent job")
	}
}

func TestJobService_CancelJob(t *testing.T) {
	service, _, _, _, ctx := setupTestJobService(t)

	jobID, _ := service.Enqueue(ctx, "cancel-job", nil)

	if err := service.CancelJob(ctx, jobID); err != nil {
		t.Fatalf("CancelJob() error = %v", err)
	}

	// Job should be deleted
	_, err := service.GetJob(ctx, jobID)
	if err == nil {
		t.Error("Job should be deleted after cancel")
	}
}

func TestJobService_RetryJob(t *testing.T) {
	service, q, _, _, ctx := setupTestJobService(t)

	// Create and fail a job
	jobID, _ := service.Enqueue(ctx, "retry-job", nil)
	job, _ := q.GetJob(ctx, jobID)
	job.Status = jobs.JobStatusFailed
	job.Attempts = 3
	job.LastError = "previous error"
	q.UpdateJob(ctx, job)

	// Retry
	if err := service.RetryJob(ctx, jobID); err != nil {
		t.Fatalf("RetryJob() error = %v", err)
	}

	// Verify job is reset
	retried, _ := q.GetJob(ctx, jobID)
	if retried.Status != jobs.JobStatusPending {
		t.Errorf("Status = %v, want pending", retried.Status)
	}
	if retried.Attempts != 0 {
		t.Errorf("Attempts = %v, want 0", retried.Attempts)
	}
	if retried.LastError != "" {
		t.Errorf("LastError = %v, want empty", retried.LastError)
	}
}

func TestJobService_GetQueueStats(t *testing.T) {
	service, _, pool, sched, ctx := setupTestJobService(t)

	// Register a scheduled job
	sched.RegisterJob(scheduler.ScheduledJob{
		Name:     "stats-cron",
		Schedule: scheduler.EveryHour,
		JobType:  "cleanup",
	})

	// Start pool and scheduler briefly
	pool.Start(ctx)
	sched.Start(ctx)
	time.Sleep(100 * time.Millisecond)
	defer pool.Stop(context.Background())
	defer sched.Stop(context.Background())

	// Enqueue some jobs
	for i := 0; i < 3; i++ {
		service.Enqueue(ctx, "stats-job", nil)
	}

	stats, err := service.GetQueueStats(ctx)
	if err != nil {
		t.Fatalf("GetQueueStats() error = %v", err)
	}

	if stats == nil {
		t.Fatal("GetQueueStats() returned nil")
	}
	if stats.QueueSizes == nil {
		t.Error("QueueSizes is nil")
	}
	if stats.SchedulerStats.RegisteredJobs != 1 {
		t.Errorf("RegisteredJobs = %v, want 1", stats.SchedulerStats.RegisteredJobs)
	}
}

func TestJobService_GetDLQJobs(t *testing.T) {
	service, q, _, _, ctx := setupTestJobService(t)

	// Create and move jobs to DLQ
	for i := 0; i < 5; i++ {
		jobID, err := service.Enqueue(ctx, "dlq-job", nil)
		if err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}
		job, err := q.GetJob(ctx, jobID)
		if err != nil {
			t.Fatalf("GetJob() error = %v", err)
		}
		job.MaxRetries = 0
		q.UpdateJob(ctx, job)
		q.Fail(ctx, jobID, jobs.ErrJobNotFound)
	}

	dlqJobs, err := service.GetDLQJobs(ctx, 10)
	if err != nil {
		t.Fatalf("GetDLQJobs() error = %v", err)
	}
	if len(dlqJobs) != 5 {
		t.Errorf("len(dlqJobs) = %v, want 5", len(dlqJobs))
	}

	// Test limit
	dlqJobs, _ = service.GetDLQJobs(ctx, 2)
	if len(dlqJobs) != 2 {
		t.Errorf("len(dlqJobs) = %v, want 2 (with limit)", len(dlqJobs))
	}
}

func TestJobService_RetryDLQJob(t *testing.T) {
	service, q, _, _, ctx := setupTestJobService(t)

	// Create and move job to DLQ
	jobID, _ := service.Enqueue(ctx, "retry-dlq-job", nil)
	job, _ := q.GetJob(ctx, jobID)
	job.MaxRetries = 0
	q.UpdateJob(ctx, job)
	q.Fail(ctx, jobID, jobs.ErrJobNotFound)

	// Retry from DLQ
	if err := service.RetryDLQJob(ctx, jobID); err != nil {
		t.Fatalf("RetryDLQJob() error = %v", err)
	}

	// Job should be back in queue (can be dequeued)
	// Note: Dequeue increments attempts from 0 to 1
	dequeued, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue after RetryDLQJob error = %v", err)
	}
	if dequeued.Attempts != 1 {
		t.Errorf("Attempts = %v, want 1 (reset to 0, then +1 on dequeue)", dequeued.Attempts)
	}
}

func TestJobService_PurgeDLQ(t *testing.T) {
	service, q, _, _, ctx := setupTestJobService(t)

	// Create and move jobs to DLQ
	for i := 0; i < 3; i++ {
		jobID, _ := service.Enqueue(ctx, "purge-job", nil)
		job, _ := q.GetJob(ctx, jobID)
		job.MaxRetries = 0
		q.UpdateJob(ctx, job)
		q.Fail(ctx, jobID, jobs.ErrJobNotFound)
	}

	// Purge
	if err := service.PurgeDLQ(ctx); err != nil {
		t.Fatalf("PurgeDLQ() error = %v", err)
	}

	// DLQ should be empty
	dlqJobs, _ := service.GetDLQJobs(ctx, 10)
	if len(dlqJobs) != 0 {
		t.Errorf("len(dlqJobs) after purge = %v, want 0", len(dlqJobs))
	}
}

// Test structs
func TestQueueStats_Struct(t *testing.T) {
	stats := jobs.QueueStats{
		Pending:   10,
		Scheduled: 5,
		Running:   2,
		Completed: 100,
		Failed:    3,
		Dead:      1,
		QueueSizes: map[string]int64{
			"critical": 2,
			"high":     3,
			"normal":   4,
			"low":      1,
		},
		WorkerStats: jobs.WorkerStats{
			Running:       true,
			ActiveWorkers: 4,
			Concurrency:   8,
			ProcessedJobs: 100,
			FailedJobs:    3,
		},
		SchedulerStats: jobs.SchedulerStats{
			IsLeader:          true,
			RegisteredJobs:    5,
			ScheduledJobNames: []string{"job1", "job2"},
		},
	}

	if stats.Pending != 10 {
		t.Errorf("Pending = %v, want 10", stats.Pending)
	}
	if !stats.WorkerStats.Running {
		t.Error("WorkerStats.Running should be true")
	}
	if !stats.SchedulerStats.IsLeader {
		t.Error("SchedulerStats.IsLeader should be true")
	}
}

func TestWorkerStats_Struct(t *testing.T) {
	stats := jobs.WorkerStats{
		Running:       true,
		ActiveWorkers: 5,
		Concurrency:   10,
		ProcessedJobs: 500,
		FailedJobs:    10,
	}

	if !stats.Running {
		t.Error("Running should be true")
	}
	if stats.ActiveWorkers != 5 {
		t.Errorf("ActiveWorkers = %v, want 5", stats.ActiveWorkers)
	}
}

func TestSchedulerStats_Struct(t *testing.T) {
	stats := jobs.SchedulerStats{
		IsLeader:          true,
		RegisteredJobs:    3,
		ScheduledJobNames: []string{"cleanup", "sync", "report"},
	}

	if !stats.IsLeader {
		t.Error("IsLeader should be true")
	}
	if stats.RegisteredJobs != 3 {
		t.Errorf("RegisteredJobs = %v, want 3", stats.RegisteredJobs)
	}
	if len(stats.ScheduledJobNames) != 3 {
		t.Errorf("len(ScheduledJobNames) = %v, want 3", len(stats.ScheduledJobNames))
	}
}

func TestScheduledJobInfo_Struct(t *testing.T) {
	info := jobs.ScheduledJobInfo{
		Name:     "daily-cleanup",
		Schedule: "0 0 * * *",
		JobType:  "cleanup",
		NextRun:  time.Now().Add(time.Hour),
		Priority: "low",
	}

	if info.Name != "daily-cleanup" {
		t.Errorf("Name = %v, want daily-cleanup", info.Name)
	}
	if info.Schedule != "0 0 * * *" {
		t.Errorf("Schedule = %v", info.Schedule)
	}
	if info.JobType != "cleanup" {
		t.Errorf("JobType = %v, want cleanup", info.JobType)
	}
	if info.Priority != "low" {
		t.Errorf("Priority = %v, want low", info.Priority)
	}
}

// Benchmarks
func BenchmarkJobService_Enqueue(b *testing.B) {
	testutil.SkipIfNoRedis(&testing.T{})
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(&testing.T{}, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewNopLogger()
	pool := worker.NewWorkerPool(q, logger, worker.DefaultWorkerPoolConfig())
	sched := scheduler.NewScheduler(client, q, logger)
	service := jobs.NewJobService(q, pool, sched)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.Enqueue(ctx, "bench-job", nil)
	}
}

func BenchmarkJobService_GetJob(b *testing.B) {
	testutil.SkipIfNoRedis(&testing.T{})
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(&testing.T{}, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewNopLogger()
	pool := worker.NewWorkerPool(q, logger, worker.DefaultWorkerPoolConfig())
	sched := scheduler.NewScheduler(client, q, logger)
	service := jobs.NewJobService(q, pool, sched)
	ctx := context.Background()

	jobID, _ := service.Enqueue(ctx, "bench-job", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetJob(ctx, jobID)
	}
}
