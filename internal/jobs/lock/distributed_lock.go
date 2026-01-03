package lock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// Key prefixes
	keyPrefixJobLock     = "arcana:jobs:lock:"
	keyPrefixRunningJobs = "arcana:jobs:running"
	keyPrefixIdempotency = "arcana:jobs:idempotency:"
	keyPrefixWorkerJobs  = "arcana:jobs:worker:"

	// Default settings
	defaultLockTTL       = 5 * time.Minute
	defaultHeartbeatRate = 30 * time.Second
)

var (
	ErrLockNotAcquired = errors.New("failed to acquire job lock")
	ErrLockNotHeld     = errors.New("lock not held by this worker")
	ErrJobAlreadyDone  = errors.New("job already completed (idempotency check)")
)

// JobLock represents a distributed lock for job execution
type JobLock struct {
	redis      *redis.Client
	jobID      string
	workerID   string
	lockKey    string
	ttl        time.Duration
	held       bool
	cancelFunc context.CancelFunc
	mu         sync.Mutex
}

// IsHeld returns whether the lock is currently held (thread-safe)
func (jl *JobLock) IsHeld() bool {
	jl.mu.Lock()
	defer jl.mu.Unlock()
	return jl.held
}

// LockManager manages distributed locks for job execution
type LockManager struct {
	redis           *redis.Client
	workerID        string
	lockTTL         time.Duration
	heartbeatRate   time.Duration
	idempotencyTTL  time.Duration
	activeLocks     map[string]*JobLock
	mu              sync.RWMutex
}

// LockManagerConfig holds configuration for the lock manager
type LockManagerConfig struct {
	LockTTL        time.Duration
	HeartbeatRate  time.Duration
	IdempotencyTTL time.Duration
}

// DefaultLockManagerConfig returns default configuration
func DefaultLockManagerConfig() LockManagerConfig {
	return LockManagerConfig{
		LockTTL:        defaultLockTTL,
		HeartbeatRate:  defaultHeartbeatRate,
		IdempotencyTTL: 24 * time.Hour,
	}
}

// NewLockManager creates a new lock manager
func NewLockManager(redisClient *redis.Client, config LockManagerConfig) *LockManager {
	return &LockManager{
		redis:          redisClient,
		workerID:       uuid.New().String(),
		lockTTL:        config.LockTTL,
		heartbeatRate:  config.HeartbeatRate,
		idempotencyTTL: config.IdempotencyTTL,
		activeLocks:    make(map[string]*JobLock),
	}
}

// GetWorkerID returns this worker's unique ID
func (lm *LockManager) GetWorkerID() string {
	return lm.workerID
}

// AcquireLock attempts to acquire an exclusive lock for a job
func (lm *LockManager) AcquireLock(ctx context.Context, jobID string) (*JobLock, error) {
	lockKey := keyPrefixJobLock + jobID

	// Try to acquire lock with SETNX
	lockValue := fmt.Sprintf("%s:%d", lm.workerID, time.Now().UnixNano())
	acquired, err := lm.redis.SetNX(ctx, lockKey, lockValue, lm.lockTTL).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		// Check if the existing lock is stale (owner crashed)
		_, err := lm.redis.Get(ctx, lockKey).Result()
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("failed to check existing lock: %w", err)
		}

		// Lock is held by another worker
		return nil, ErrLockNotAcquired
	}

	// Create lock object
	lockCtx, cancel := context.WithCancel(ctx)
	lock := &JobLock{
		redis:      lm.redis,
		jobID:      jobID,
		workerID:   lm.workerID,
		lockKey:    lockKey,
		ttl:        lm.lockTTL,
		held:       true,
		cancelFunc: cancel,
	}

	// Start heartbeat to maintain lock
	go lock.heartbeat(lockCtx, lm.heartbeatRate)

	// Track in running jobs
	lm.trackRunningJob(ctx, jobID)

	// Store in active locks
	lm.mu.Lock()
	lm.activeLocks[jobID] = lock
	lm.mu.Unlock()

	return lock, nil
}

// ReleaseLock releases a job lock
func (lm *LockManager) ReleaseLock(ctx context.Context, lock *JobLock) error {
	if lock == nil {
		return nil
	}

	lock.mu.Lock()
	defer lock.mu.Unlock()

	if !lock.held {
		return nil
	}

	// Stop heartbeat
	lock.cancelFunc()
	lock.held = false

	// Delete lock only if we still own it (compare-and-delete with prefix match)
	// Use string.find with plain=true (4th arg) to avoid Lua pattern matching issues with UUID hyphens
	script := `
		local val = redis.call("get", KEYS[1])
		if val and string.find(val, ARGV[1], 1, true) then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	lockValue := fmt.Sprintf("%s:", lm.workerID)
	_, err := lm.redis.Eval(ctx, script, []string{lock.lockKey}, lockValue).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	// Remove from running jobs
	lm.untrackRunningJob(ctx, lock.jobID)

	// Remove from active locks
	lm.mu.Lock()
	delete(lm.activeLocks, lock.jobID)
	lm.mu.Unlock()

	return nil
}

// heartbeat maintains the lock by extending TTL periodically
func (jl *JobLock) heartbeat(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			jl.mu.Lock()
			if !jl.held {
				jl.mu.Unlock()
				return
			}

			// Extend lock TTL only if we still own it
			// Use string.find with plain=true (4th arg) to avoid Lua pattern matching issues with UUID hyphens
			script := `
				local val = redis.call("get", KEYS[1])
				if val and string.find(val, ARGV[1], 1, true) then
					return redis.call("pexpire", KEYS[1], ARGV[2])
				else
					return 0
				end
			`
			result, err := jl.redis.Eval(ctx, script, []string{jl.lockKey},
				jl.workerID+":", jl.ttl.Milliseconds()).Int()
			if err != nil || result == 0 {
				// Lost the lock
				jl.held = false
				jl.mu.Unlock()
				return
			}
			jl.mu.Unlock()
		}
	}
}

// trackRunningJob adds a job to the running jobs set
func (lm *LockManager) trackRunningJob(ctx context.Context, jobID string) {
	// Add to global running jobs set with worker info
	lm.redis.HSet(ctx, keyPrefixRunningJobs, jobID, fmt.Sprintf("%s:%d", lm.workerID, time.Now().Unix()))

	// Add to this worker's job list
	lm.redis.SAdd(ctx, keyPrefixWorkerJobs+lm.workerID, jobID)
}

// untrackRunningJob removes a job from the running jobs set
func (lm *LockManager) untrackRunningJob(ctx context.Context, jobID string) {
	lm.redis.HDel(ctx, keyPrefixRunningJobs, jobID)
	lm.redis.SRem(ctx, keyPrefixWorkerJobs+lm.workerID, jobID)
}

// CheckIdempotency checks if a job with the given unique key has already been completed
func (lm *LockManager) CheckIdempotency(ctx context.Context, uniqueKey string) (bool, error) {
	if uniqueKey == "" {
		return false, nil
	}

	exists, err := lm.redis.Exists(ctx, keyPrefixIdempotency+uniqueKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check idempotency: %w", err)
	}

	return exists > 0, nil
}

// MarkCompleted marks a job as completed for idempotency tracking
func (lm *LockManager) MarkCompleted(ctx context.Context, uniqueKey string, jobID string) error {
	if uniqueKey == "" {
		return nil
	}

	// Store completion record with job ID and timestamp
	value := fmt.Sprintf("%s:%d", jobID, time.Now().Unix())
	if err := lm.redis.Set(ctx, keyPrefixIdempotency+uniqueKey, value, lm.idempotencyTTL).Err(); err != nil {
		return fmt.Errorf("failed to mark job as completed: %w", err)
	}

	return nil
}

// GetRunningJobs returns all currently running jobs
func (lm *LockManager) GetRunningJobs(ctx context.Context) (map[string]string, error) {
	return lm.redis.HGetAll(ctx, keyPrefixRunningJobs).Result()
}

// GetWorkerJobs returns jobs being processed by a specific worker
func (lm *LockManager) GetWorkerJobs(ctx context.Context, workerID string) ([]string, error) {
	return lm.redis.SMembers(ctx, keyPrefixWorkerJobs+workerID).Result()
}

// CleanupStaleJobs finds and releases locks for jobs whose workers have crashed
func (lm *LockManager) CleanupStaleJobs(ctx context.Context, staleDuration time.Duration) ([]string, error) {
	runningJobs, err := lm.redis.HGetAll(ctx, keyPrefixRunningJobs).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get running jobs: %w", err)
	}

	var cleaned []string
	now := time.Now().Unix()

	for jobID, value := range runningJobs {
		// Parse worker:timestamp
		var workerID string
		var timestamp int64
		fmt.Sscanf(value, "%s:%d", &workerID, &timestamp)

		// Check if job is stale
		if now-timestamp > int64(staleDuration.Seconds()) {
			// Remove stale lock
			lockKey := keyPrefixJobLock + jobID
			lm.redis.Del(ctx, lockKey)
			lm.redis.HDel(ctx, keyPrefixRunningJobs, jobID)
			cleaned = append(cleaned, jobID)
		}
	}

	return cleaned, nil
}

// ReleaseAllLocks releases all locks held by this worker (for graceful shutdown)
func (lm *LockManager) ReleaseAllLocks(ctx context.Context) error {
	lm.mu.Lock()
	locks := make([]*JobLock, 0, len(lm.activeLocks))
	for _, lock := range lm.activeLocks {
		locks = append(locks, lock)
	}
	lm.mu.Unlock()

	for _, lock := range locks {
		if err := lm.ReleaseLock(ctx, lock); err != nil {
			// Log but continue releasing other locks
			continue
		}
	}

	// Clean up worker job list
	lm.redis.Del(ctx, keyPrefixWorkerJobs+lm.workerID)

	return nil
}
