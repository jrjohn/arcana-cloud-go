package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
	"github.com/jrjohn/arcana-cloud-go/internal/testutil"
)

func setupTestQueue(t *testing.T) (*RedisQueue, context.Context) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	return NewRedisQueue(client), context.Background()
}

func TestNewRedisQueue(t *testing.T) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)

	q := NewRedisQueue(client)
	if q == nil {
		t.Fatal("NewRedisQueue() returned nil")
	}
	if q.client == nil {
		t.Fatal("Queue client is nil")
	}
}

func TestRedisQueue_Enqueue(t *testing.T) {
	q, ctx := setupTestQueue(t)

	job, err := jobs.NewJobPayload("test-job", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	err = q.Enqueue(ctx, job)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Verify job is stored
	storedJob, err := q.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if storedJob.ID != job.ID {
		t.Errorf("Stored job ID = %v, want %v", storedJob.ID, job.ID)
	}
	if storedJob.Type != job.Type {
		t.Errorf("Stored job Type = %v, want %v", storedJob.Type, job.Type)
	}
}

func TestRedisQueue_Enqueue_WithPriority(t *testing.T) {
	q, ctx := setupTestQueue(t)

	priorities := []jobs.Priority{
		jobs.PriorityLow,
		jobs.PriorityNormal,
		jobs.PriorityHigh,
		jobs.PriorityCritical,
	}

	for _, p := range priorities {
		t.Run(p.String(), func(t *testing.T) {
			job, _ := jobs.NewJobPayload("test-job", nil, jobs.WithPriority(p))
			if err := q.Enqueue(ctx, job); err != nil {
				t.Errorf("Enqueue() error = %v", err)
			}
		})
	}
}

func TestRedisQueue_Enqueue_Scheduled(t *testing.T) {
	q, ctx := setupTestQueue(t)

	scheduledTime := time.Now().Add(time.Hour)
	job, _ := jobs.NewJobPayload("scheduled-job", nil, jobs.WithScheduledAt(scheduledTime))

	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Job should be in scheduled set, not in queue
	storedJob, err := q.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if storedJob.ScheduledAt == nil {
		t.Error("ScheduledAt should not be nil")
	}
}

func TestRedisQueue_Enqueue_UniqueKey(t *testing.T) {
	q, ctx := setupTestQueue(t)

	// First job should succeed
	job1, _ := jobs.NewJobPayload("unique-job", nil, jobs.WithUniqueKey("unique-123"))
	if err := q.Enqueue(ctx, job1); err != nil {
		t.Fatalf("First Enqueue() error = %v", err)
	}

	// Second job with same unique key should fail
	job2, _ := jobs.NewJobPayload("unique-job", nil, jobs.WithUniqueKey("unique-123"))
	err := q.Enqueue(ctx, job2)
	if err != jobs.ErrDuplicateJob {
		t.Errorf("Second Enqueue() error = %v, want jobs.ErrDuplicateJob", err)
	}

	// Job with different unique key should succeed
	job3, _ := jobs.NewJobPayload("unique-job", nil, jobs.WithUniqueKey("unique-456"))
	if err := q.Enqueue(ctx, job3); err != nil {
		t.Errorf("Third Enqueue() error = %v", err)
	}
}

func TestRedisQueue_Dequeue(t *testing.T) {
	q, ctx := setupTestQueue(t)

	// Enqueue a job
	job, _ := jobs.NewJobPayload("dequeue-test", map[string]string{"data": "test"})
	if err := q.Enqueue(ctx, job); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Dequeue
	dequeued, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue() error = %v", err)
	}

	if dequeued.ID != job.ID {
		t.Errorf("Dequeued job ID = %v, want %v", dequeued.ID, job.ID)
	}
	if dequeued.Status != jobs.JobStatusRunning {
		t.Errorf("Dequeued job Status = %v, want running", dequeued.Status)
	}
	if dequeued.Attempts != 1 {
		t.Errorf("Dequeued job Attempts = %v, want 1", dequeued.Attempts)
	}
	if dequeued.StartedAt == nil {
		t.Error("Dequeued job StartedAt is nil")
	}
}

func TestRedisQueue_Dequeue_PriorityOrder(t *testing.T) {
	q, ctx := setupTestQueue(t)

	// Enqueue jobs with different priorities
	lowJob, _ := jobs.NewJobPayload("low-job", nil, jobs.WithPriority(jobs.PriorityLow))
	normalJob, _ := jobs.NewJobPayload("normal-job", nil, jobs.WithPriority(jobs.PriorityNormal))
	highJob, _ := jobs.NewJobPayload("high-job", nil, jobs.WithPriority(jobs.PriorityHigh))
	criticalJob, _ := jobs.NewJobPayload("critical-job", nil, jobs.WithPriority(jobs.PriorityCritical))

	q.Enqueue(ctx, lowJob)
	q.Enqueue(ctx, normalJob)
	q.Enqueue(ctx, highJob)
	q.Enqueue(ctx, criticalJob)

	// Should dequeue in priority order
	expectedOrder := []string{criticalJob.ID, highJob.ID, normalJob.ID, lowJob.ID}
	for i, expectedID := range expectedOrder {
		dequeued, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatalf("Dequeue() %d error = %v", i, err)
		}
		if dequeued.ID != expectedID {
			t.Errorf("Dequeue() %d = %v, want %v", i, dequeued.ID, expectedID)
		}
	}
}

func TestRedisQueue_Dequeue_Empty(t *testing.T) {
	q, ctx := setupTestQueue(t)

	_, err := q.Dequeue(ctx)
	if err != jobs.ErrQueueEmpty {
		t.Errorf("Dequeue() error = %v, want jobs.ErrQueueEmpty", err)
	}
}

func TestRedisQueue_GetJob(t *testing.T) {
	q, ctx := setupTestQueue(t)

	// Create and enqueue job
	job, _ := jobs.NewJobPayload("get-test", map[string]int{"count": 42})
	q.Enqueue(ctx, job)

	// Get job
	retrieved, err := q.GetJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}

	if retrieved.ID != job.ID {
		t.Errorf("ID = %v, want %v", retrieved.ID, job.ID)
	}
	if retrieved.Type != job.Type {
		t.Errorf("Type = %v, want %v", retrieved.Type, job.Type)
	}

	// Verify payload
	var payload map[string]int
	json.Unmarshal(retrieved.Payload, &payload)
	if payload["count"] != 42 {
		t.Errorf("Payload count = %v, want 42", payload["count"])
	}
}

func TestRedisQueue_GetJob_NotFound(t *testing.T) {
	q, ctx := setupTestQueue(t)

	_, err := q.GetJob(ctx, "nonexistent-job-id")
	if err != jobs.ErrJobNotFound {
		t.Errorf("GetJob() error = %v, want jobs.ErrJobNotFound", err)
	}
}

func TestRedisQueue_UpdateJob(t *testing.T) {
	q, ctx := setupTestQueue(t)

	job, _ := jobs.NewJobPayload("update-test", nil)
	q.Enqueue(ctx, job)

	// Update job
	job.Status = jobs.JobStatusRunning
	job.Attempts = 2
	job.LastError = "test error"

	if err := q.UpdateJob(ctx, job); err != nil {
		t.Fatalf("UpdateJob() error = %v", err)
	}

	// Verify update
	updated, _ := q.GetJob(ctx, job.ID)
	if updated.Status != jobs.JobStatusRunning {
		t.Errorf("Status = %v, want running", updated.Status)
	}
	if updated.Attempts != 2 {
		t.Errorf("Attempts = %v, want 2", updated.Attempts)
	}
	if updated.LastError != "test error" {
		t.Errorf("LastError = %v, want 'test error'", updated.LastError)
	}
}

func TestRedisQueue_Complete(t *testing.T) {
	q, ctx := setupTestQueue(t)

	job, _ := jobs.NewJobPayload("complete-test", nil, jobs.WithUniqueKey("complete-key"))
	q.Enqueue(ctx, job)

	// Complete the job
	if err := q.Complete(ctx, job.ID); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	// Verify status
	completed, _ := q.GetJob(ctx, job.ID)
	if completed.Status != jobs.JobStatusCompleted {
		t.Errorf("Status = %v, want completed", completed.Status)
	}
	if completed.CompletedAt == nil {
		t.Error("CompletedAt is nil")
	}

	// Unique key should be cleaned up - new job with same key should work
	job2, _ := jobs.NewJobPayload("complete-test", nil, jobs.WithUniqueKey("complete-key"))
	if err := q.Enqueue(ctx, job2); err != nil {
		t.Errorf("Enqueue after complete should succeed: %v", err)
	}
}

func TestRedisQueue_Fail_WithRetry(t *testing.T) {
	q, ctx := setupTestQueue(t)

	job, _ := jobs.NewJobPayload("fail-test", nil)
	job.MaxRetries = 3
	q.Enqueue(ctx, job)

	// Fail the job
	if err := q.Fail(ctx, job.ID, jobs.ErrJobNotFound); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}

	// Verify status
	failed, _ := q.GetJob(ctx, job.ID)
	if failed.Status != jobs.JobStatusRetrying {
		t.Errorf("Status = %v, want retrying", failed.Status)
	}
	if failed.LastError == "" {
		t.Error("LastError should be set")
	}
	if failed.ScheduledAt == nil {
		t.Error("ScheduledAt should be set for retry")
	}
}

func TestRedisQueue_Fail_MoveToDLQ(t *testing.T) {
	q, ctx := setupTestQueue(t)

	job, _ := jobs.NewJobPayload("dlq-test", nil)
	job.MaxRetries = 1
	job.Attempts = 1 // Already at max retries
	q.Enqueue(ctx, job)

	// Fail the job
	if err := q.Fail(ctx, job.ID, jobs.ErrJobNotFound); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}

	// Verify status
	dead, _ := q.GetJob(ctx, job.ID)
	if dead.Status != jobs.JobStatusDead {
		t.Errorf("Status = %v, want dead", dead.Status)
	}

	// Should be in DLQ
	dlqJobs, _ := q.GetDLQJobs(ctx, 10)
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

func TestRedisQueue_ProcessScheduled(t *testing.T) {
	q, ctx := setupTestQueue(t)

	// Schedule a job for 1ms in the future (will be in scheduled set)
	scheduledAt := time.Now().Add(10 * time.Millisecond)
	job, _ := jobs.NewJobPayload("scheduled-test", nil, jobs.WithScheduledAt(scheduledAt))
	q.Enqueue(ctx, job)

	// Wait for scheduled time to pass
	time.Sleep(20 * time.Millisecond)

	// Process scheduled jobs
	processed, err := q.ProcessScheduled(ctx)
	if err != nil {
		t.Fatalf("ProcessScheduled() error = %v", err)
	}
	if processed != 1 {
		t.Errorf("Processed = %v, want 1", processed)
	}

	// Job should now be in queue
	dequeued, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue() error = %v", err)
	}
	if dequeued.ID != job.ID {
		t.Errorf("Dequeued ID = %v, want %v", dequeued.ID, job.ID)
	}
}

func TestRedisQueue_ProcessScheduled_FutureJob(t *testing.T) {
	q, ctx := setupTestQueue(t)

	// Schedule a job for the future
	job, _ := jobs.NewJobPayload("future-test", nil, jobs.WithScheduledAt(time.Now().Add(time.Hour)))
	q.Enqueue(ctx, job)

	// Process scheduled jobs
	processed, err := q.ProcessScheduled(ctx)
	if err != nil {
		t.Fatalf("ProcessScheduled() error = %v", err)
	}
	if processed != 0 {
		t.Errorf("Processed = %v, want 0", processed)
	}
}

func TestRedisQueue_GetDLQJobs(t *testing.T) {
	q, ctx := setupTestQueue(t)

	// Create and fail some jobs
	for i := 0; i < 5; i++ {
		job, _ := jobs.NewJobPayload("dlq-list-test", nil)
		job.MaxRetries = 0 // Go straight to DLQ
		q.Enqueue(ctx, job)
		q.Fail(ctx, job.ID, jobs.ErrJobNotFound)
	}

	// Get DLQ jobs
	dlqJobs, err := q.GetDLQJobs(ctx, 10)
	if err != nil {
		t.Fatalf("GetDLQJobs() error = %v", err)
	}
	if len(dlqJobs) != 5 {
		t.Errorf("len(dlqJobs) = %v, want 5", len(dlqJobs))
	}

	// Test limit
	dlqJobs, err = q.GetDLQJobs(ctx, 2)
	if err != nil {
		t.Fatalf("GetDLQJobs() error = %v", err)
	}
	if len(dlqJobs) != 2 {
		t.Errorf("len(dlqJobs) = %v, want 2", len(dlqJobs))
	}
}

func TestRedisQueue_RetryDLQJob(t *testing.T) {
	q, ctx := setupTestQueue(t)

	// Create and fail a job
	job, _ := jobs.NewJobPayload("retry-dlq-test", nil)
	job.MaxRetries = 0
	q.Enqueue(ctx, job)
	q.Fail(ctx, job.ID, jobs.ErrJobNotFound)

	// Retry from DLQ
	if err := q.RetryDLQJob(ctx, job.ID); err != nil {
		t.Fatalf("RetryDLQJob() error = %v", err)
	}

	// Should be able to dequeue (Dequeue increments attempts from 0 to 1)
	dequeued, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue() error = %v", err)
	}
	if dequeued.Attempts != 1 {
		t.Errorf("Attempts = %v, want 1 (reset to 0, then +1 on dequeue)", dequeued.Attempts)
	}
	if dequeued.Status != jobs.JobStatusRunning {
		t.Errorf("Status = %v, want running", dequeued.Status)
	}
}

func TestRedisQueue_DeleteJob(t *testing.T) {
	q, ctx := setupTestQueue(t)

	job, _ := jobs.NewJobPayload("delete-test", nil, jobs.WithUniqueKey("delete-key"))
	q.Enqueue(ctx, job)

	// Delete
	if err := q.DeleteJob(ctx, job.ID); err != nil {
		t.Fatalf("DeleteJob() error = %v", err)
	}

	// Should not be found
	_, err := q.GetJob(ctx, job.ID)
	if err != jobs.ErrJobNotFound {
		t.Errorf("GetJob() error = %v, want jobs.ErrJobNotFound", err)
	}

	// Unique key should be cleaned up
	job2, _ := jobs.NewJobPayload("delete-test", nil, jobs.WithUniqueKey("delete-key"))
	if err := q.Enqueue(ctx, job2); err != nil {
		t.Errorf("Enqueue after delete should succeed: %v", err)
	}
}

func TestRedisQueue_RequeueJob(t *testing.T) {
	q, ctx := setupTestQueue(t)

	job, _ := jobs.NewJobPayload("requeue-test", nil, jobs.WithPriority(jobs.PriorityHigh))
	q.Enqueue(ctx, job)

	// Dequeue
	dequeued, _ := q.Dequeue(ctx)

	// Requeue
	if err := q.RequeueJob(ctx, dequeued.ID, jobs.PriorityHigh.QueueName()); err != nil {
		t.Fatalf("RequeueJob() error = %v", err)
	}

	// Should be able to dequeue again
	requeued, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue after requeue error = %v", err)
	}
	if requeued.ID != job.ID {
		t.Errorf("Requeued ID = %v, want %v", requeued.ID, job.ID)
	}
}

func TestRedisQueue_GetStats(t *testing.T) {
	q, ctx := setupTestQueue(t)

	// Enqueue some jobs
	for i := 0; i < 3; i++ {
		job, _ := jobs.NewJobPayload("stats-test", nil, jobs.WithPriority(jobs.PriorityNormal))
		q.Enqueue(ctx, job)
	}

	// Complete one
	dequeued, _ := q.Dequeue(ctx)
	q.Complete(ctx, dequeued.ID)

	// Get stats
	stats, err := q.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if stats["enqueued_total"] != 3 {
		t.Errorf("enqueued_total = %v, want 3", stats["enqueued_total"])
	}
	if stats["completed_total"] != 1 {
		t.Errorf("completed_total = %v, want 1", stats["completed_total"])
	}
}

// Error cases
func TestRedisQueue_Errors(t *testing.T) {
	t.Run("ErrJobNotFound", func(t *testing.T) {
		if jobs.ErrJobNotFound.Error() != "job not found" {
			t.Errorf("jobs.ErrJobNotFound = %v", jobs.ErrJobNotFound)
		}
	})

	t.Run("ErrDuplicateJob", func(t *testing.T) {
		if jobs.ErrDuplicateJob.Error() != "duplicate job with same unique key" {
			t.Errorf("jobs.ErrDuplicateJob = %v", jobs.ErrDuplicateJob)
		}
	})

	t.Run("ErrQueueEmpty", func(t *testing.T) {
		if jobs.ErrQueueEmpty.Error() != "queue is empty" {
			t.Errorf("jobs.ErrQueueEmpty = %v", jobs.ErrQueueEmpty)
		}
	})

	t.Run("ErrJobAlreadyTaken", func(t *testing.T) {
		if jobs.ErrJobAlreadyTaken.Error() != "job already taken by another worker" {
			t.Errorf("jobs.ErrJobAlreadyTaken = %v", jobs.ErrJobAlreadyTaken)
		}
	})
}

// Benchmarks
func BenchmarkRedisQueue_Enqueue(b *testing.B) {
	testutil.SkipIfNoRedis(&testing.T{})
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(&testing.T{}, config)
	q := NewRedisQueue(client)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job, _ := jobs.NewJobPayload("bench-job", nil)
		q.Enqueue(ctx, job)
	}
}

func BenchmarkRedisQueue_Dequeue(b *testing.B) {
	testutil.SkipIfNoRedis(&testing.T{})
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(&testing.T{}, config)
	q := NewRedisQueue(client)
	ctx := context.Background()

	// Pre-populate queue
	for i := 0; i < b.N; i++ {
		job, _ := jobs.NewJobPayload("bench-job", nil)
		q.Enqueue(ctx, job)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Dequeue(ctx)
	}
}
