package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
)

const (
	// Redis key prefixes
	keyPrefixQueue     = "arcana:jobs:queue:"
	keyPrefixJob       = "arcana:jobs:job:"
	keyPrefixScheduled = "arcana:jobs:scheduled"
	keyPrefixUnique    = "arcana:jobs:unique:"
	keyPrefixDLQ       = "arcana:jobs:dlq"
	keyPrefixStats     = "arcana:jobs:stats"
)

// RedisQueue implements a Redis-backed job queue
type RedisQueue struct {
	client *redis.Client
}

// NewRedisQueue creates a new Redis queue
func NewRedisQueue(client *redis.Client) *RedisQueue {
	return &RedisQueue{client: client}
}

// checkDuplicate returns ErrDuplicateJob if the unique key already exists
func (q *RedisQueue) checkDuplicate(ctx context.Context, uniqueKey string) error {
	exists, err := q.client.Exists(ctx, keyPrefixUnique+uniqueKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check unique key: %w", err)
	}
	if exists > 0 {
		return jobs.ErrDuplicateJob
	}
	return nil
}

// scheduleOrEnqueue adds the job to the scheduled set or immediate priority queue
func (q *RedisQueue) scheduleOrEnqueue(ctx context.Context, job *jobs.JobPayload) error {
	if job.ScheduledAt != nil && job.ScheduledAt.After(time.Now()) {
		score := float64(job.ScheduledAt.Unix())
		return q.client.ZAdd(ctx, keyPrefixScheduled, redis.Z{Score: score, Member: job.ID}).Err()
	}
	return q.client.LPush(ctx, job.Priority.QueueName(), job.ID).Err()
}

// setUniqueKey stores the unique key with the appropriate TTL
func (q *RedisQueue) setUniqueKey(ctx context.Context, job *jobs.JobPayload) error {
	ttl := 24 * time.Hour
	if job.ScheduledAt != nil {
		ttl = time.Until(*job.ScheduledAt) + 24*time.Hour
	}
	return q.client.Set(ctx, keyPrefixUnique+job.UniqueKey, job.ID, ttl).Err()
}

// Enqueue adds a job to the queue
func (q *RedisQueue) Enqueue(ctx context.Context, job *jobs.JobPayload) error {
	if job.UniqueKey != "" {
		if err := q.checkDuplicate(ctx, job.UniqueKey); err != nil {
			return err
		}
	}

	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to serialize job: %w", err)
	}
	if err := q.client.Set(ctx, keyPrefixJob+job.ID, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store job: %w", err)
	}

	if err := q.scheduleOrEnqueue(ctx, job); err != nil {
		return fmt.Errorf("failed to queue job: %w", err)
	}

	if job.UniqueKey != "" {
		if err := q.setUniqueKey(ctx, job); err != nil {
			return fmt.Errorf("failed to set unique key: %w", err)
		}
	}

	q.client.HIncrBy(ctx, keyPrefixStats, "enqueued_total", 1)
	q.client.HIncrBy(ctx, keyPrefixStats, "pending", 1)
	return nil
}

// Dequeue retrieves the next job from the queue
func (q *RedisQueue) Dequeue(ctx context.Context, priorities ...jobs.Priority) (*jobs.JobPayload, error) {
	if len(priorities) == 0 {
		priorities = []jobs.Priority{
			jobs.PriorityCritical,
			jobs.PriorityHigh,
			jobs.PriorityNormal,
			jobs.PriorityLow,
		}
	}

	// Try each priority queue in order
	for _, priority := range priorities {
		queueKey := priority.QueueName()

		// Use RPOP (non-blocking) to avoid 1s minimum timeout of BRPOP
		jobID, err := q.client.RPop(ctx, queueKey).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to dequeue: %w", err)
		}

		job, err := q.GetJob(ctx, jobID)
		if err != nil {
			continue
		}

		// Update job status
		job.Status = jobs.JobStatusRunning
		now := time.Now()
		job.StartedAt = &now
		job.Attempts++

		if err := q.UpdateJob(ctx, job); err != nil {
			return nil, fmt.Errorf("failed to update job status: %w", err)
		}

		q.client.HIncrBy(ctx, keyPrefixStats, "pending", -1)

		return job, nil
	}

	return nil, jobs.ErrQueueEmpty
}

// GetJob retrieves a job by ID
func (q *RedisQueue) GetJob(ctx context.Context, jobID string) (*jobs.JobPayload, error) {
	data, err := q.client.Get(ctx, keyPrefixJob+jobID).Bytes()
	if err == redis.Nil {
		return nil, jobs.ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	var job jobs.JobPayload
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("failed to deserialize job: %w", err)
	}

	return &job, nil
}

// UpdateJob updates a job's data
func (q *RedisQueue) UpdateJob(ctx context.Context, job *jobs.JobPayload) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to serialize job: %w", err)
	}

	if err := q.client.Set(ctx, keyPrefixJob+job.ID, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// Complete marks a job as completed
func (q *RedisQueue) Complete(ctx context.Context, jobID string) error {
	job, err := q.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.Status = jobs.JobStatusCompleted
	now := time.Now()
	job.CompletedAt = &now

	if err := q.UpdateJob(ctx, job); err != nil {
		return err
	}

	// Clean up unique key
	if job.UniqueKey != "" {
		q.client.Del(ctx, keyPrefixUnique+job.UniqueKey)
	}

	q.client.HIncrBy(ctx, keyPrefixStats, "completed_total", 1)

	return nil
}

// Fail marks a job as failed and handles retry logic
func (q *RedisQueue) Fail(ctx context.Context, jobID string, jobErr error) error {
	job, err := q.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	job.LastError = jobErr.Error()

	// Check if we should retry
	if job.Attempts < job.MaxRetries {
		job.Status = jobs.JobStatusRetrying
		delay := job.RetryPolicy.CalculateDelay(job.Attempts)
		scheduledAt := time.Now().Add(delay)
		job.ScheduledAt = &scheduledAt

		if err := q.UpdateJob(ctx, job); err != nil {
			return err
		}

		// Add to scheduled queue
		score := float64(scheduledAt.Unix())
		if err := q.client.ZAdd(ctx, keyPrefixScheduled, redis.Z{
			Score:  score,
			Member: job.ID,
		}).Err(); err != nil {
			return fmt.Errorf("failed to schedule retry: %w", err)
		}

		q.client.HIncrBy(ctx, keyPrefixStats, "retries_total", 1)
	} else {
		// Move to DLQ
		job.Status = jobs.JobStatusDead
		if err := q.UpdateJob(ctx, job); err != nil {
			return err
		}

		if err := q.client.LPush(ctx, keyPrefixDLQ, job.ID).Err(); err != nil {
			return fmt.Errorf("failed to move to DLQ: %w", err)
		}

		// Clean up unique key
		if job.UniqueKey != "" {
			q.client.Del(ctx, keyPrefixUnique+job.UniqueKey)
		}

		q.client.HIncrBy(ctx, keyPrefixStats, "dead_total", 1)
	}

	q.client.HIncrBy(ctx, keyPrefixStats, "failed_total", 1)

	return nil
}

// ProcessScheduled moves scheduled jobs that are due to their queues
func (q *RedisQueue) ProcessScheduled(ctx context.Context) (int, error) {
	now := time.Now().Unix()

	// Get all jobs scheduled for now or earlier
	jobIDs, err := q.client.ZRangeByScore(ctx, keyPrefixScheduled, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", now),
	}).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get scheduled jobs: %w", err)
	}

	processed := 0
	for _, jobID := range jobIDs {
		job, err := q.GetJob(ctx, jobID)
		if err != nil {
			continue
		}

		// Remove from scheduled set
		if err := q.client.ZRem(ctx, keyPrefixScheduled, jobID).Err(); err != nil {
			continue
		}

		// Reset scheduled time and add to queue
		job.ScheduledAt = nil
		job.Status = jobs.JobStatusPending
		if err := q.UpdateJob(ctx, job); err != nil {
			continue
		}

		// Add to priority queue
		queueKey := job.Priority.QueueName()
		if err := q.client.LPush(ctx, queueKey, job.ID).Err(); err != nil {
			continue
		}

		processed++
	}

	return processed, nil
}

// GetDLQJobs retrieves jobs from the dead letter queue
func (q *RedisQueue) GetDLQJobs(ctx context.Context, limit int64) ([]*jobs.JobPayload, error) {
	jobIDs, err := q.client.LRange(ctx, keyPrefixDLQ, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get DLQ jobs: %w", err)
	}

	var dlqJobs []*jobs.JobPayload
	for _, jobID := range jobIDs {
		job, err := q.GetJob(ctx, jobID)
		if err != nil {
			continue
		}
		dlqJobs = append(dlqJobs, job)
	}

	return dlqJobs, nil
}

// RetryDLQJob moves a job from DLQ back to the queue
func (q *RedisQueue) RetryDLQJob(ctx context.Context, jobID string) error {
	job, err := q.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Remove from DLQ
	if err := q.client.LRem(ctx, keyPrefixDLQ, 1, jobID).Err(); err != nil {
		return fmt.Errorf("failed to remove from DLQ: %w", err)
	}

	// Reset job state
	job.Status = jobs.JobStatusPending
	job.Attempts = 0
	job.LastError = ""
	job.ID = uuid.New().String() // New ID to avoid conflicts

	// Re-enqueue
	return q.Enqueue(ctx, job)
}

// DeleteJob removes a job completely
func (q *RedisQueue) DeleteJob(ctx context.Context, jobID string) error {
	job, err := q.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	// Remove from all possible locations
	q.client.Del(ctx, keyPrefixJob+jobID)
	q.client.LRem(ctx, job.Priority.QueueName(), 0, jobID)
	q.client.ZRem(ctx, keyPrefixScheduled, jobID)
	q.client.LRem(ctx, keyPrefixDLQ, 0, jobID)

	if job.UniqueKey != "" {
		q.client.Del(ctx, keyPrefixUnique+job.UniqueKey)
	}

	return nil
}

// RequeueJob adds a job back to the queue
func (q *RedisQueue) RequeueJob(ctx context.Context, jobID string, queueKey string) error {
	return q.client.LPush(ctx, queueKey, jobID).Err()
}

// GetStats returns queue statistics
func (q *RedisQueue) GetStats(ctx context.Context) (map[string]int64, error) {
	stats, err := q.client.HGetAll(ctx, keyPrefixStats).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	result := make(map[string]int64)
	for k, v := range stats {
		var val int64
		fmt.Sscanf(v, "%d", &val)
		result[k] = val
	}

	// Add queue sizes
	for _, p := range []jobs.Priority{jobs.PriorityCritical, jobs.PriorityHigh, jobs.PriorityNormal, jobs.PriorityLow} {
		size, _ := q.client.LLen(ctx, p.QueueName()).Result()
		result["queue_"+p.String()] = size
	}

	// Add scheduled and DLQ sizes
	scheduled, _ := q.client.ZCard(ctx, keyPrefixScheduled).Result()
	result["scheduled"] = scheduled

	dlq, _ := q.client.LLen(ctx, keyPrefixDLQ).Result()
	result["dlq"] = dlq

	return result, nil
}
