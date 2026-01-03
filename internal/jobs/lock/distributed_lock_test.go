package lock

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jrjohn/arcana-cloud-go/internal/testutil"
)

func setupTestLockManager(t *testing.T) (*LockManager, context.Context) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	lmConfig := DefaultLockManagerConfig()
	lmConfig.LockTTL = 5 * time.Second
	lmConfig.HeartbeatRate = time.Second
	return NewLockManager(client, lmConfig), context.Background()
}

func TestDefaultLockManagerConfig(t *testing.T) {
	config := DefaultLockManagerConfig()

	if config.LockTTL != 5*time.Minute {
		t.Errorf("LockTTL = %v, want 5m", config.LockTTL)
	}
	if config.HeartbeatRate != 30*time.Second {
		t.Errorf("HeartbeatRate = %v, want 30s", config.HeartbeatRate)
	}
	if config.IdempotencyTTL != 24*time.Hour {
		t.Errorf("IdempotencyTTL = %v, want 24h", config.IdempotencyTTL)
	}
}

func TestNewLockManager(t *testing.T) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)

	lm := NewLockManager(client, DefaultLockManagerConfig())

	if lm == nil {
		t.Fatal("NewLockManager() returned nil")
	}
	if lm.workerID == "" {
		t.Error("workerID is empty")
	}
	if lm.activeLocks == nil {
		t.Error("activeLocks is nil")
	}
}

func TestLockManager_GetWorkerID(t *testing.T) {
	lm, _ := setupTestLockManager(t)

	workerID := lm.GetWorkerID()
	if workerID == "" {
		t.Error("GetWorkerID() returned empty string")
	}

	// Should be consistent
	if lm.GetWorkerID() != workerID {
		t.Error("GetWorkerID() is not consistent")
	}
}

func TestLockManager_AcquireLock_Success(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	lock, err := lm.AcquireLock(ctx, "test-job-1")
	if err != nil {
		t.Fatalf("AcquireLock() error = %v", err)
	}
	if lock == nil {
		t.Fatal("AcquireLock() returned nil lock")
	}

	defer lm.ReleaseLock(ctx, lock)

	if !lock.held {
		t.Error("Lock should be held")
	}
	if lock.jobID != "test-job-1" {
		t.Errorf("jobID = %v, want test-job-1", lock.jobID)
	}
}

func TestLockManager_AcquireLock_AlreadyLocked(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	// Acquire first lock
	lock1, err := lm.AcquireLock(ctx, "test-job-2")
	if err != nil {
		t.Fatalf("First AcquireLock() error = %v", err)
	}
	defer lm.ReleaseLock(ctx, lock1)

	// Try to acquire same lock again
	_, err = lm.AcquireLock(ctx, "test-job-2")
	if err != ErrLockNotAcquired {
		t.Errorf("Second AcquireLock() error = %v, want ErrLockNotAcquired", err)
	}
}

func TestLockManager_AcquireLock_DifferentJobs(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	lock1, err := lm.AcquireLock(ctx, "job-a")
	if err != nil {
		t.Fatalf("First AcquireLock() error = %v", err)
	}
	defer lm.ReleaseLock(ctx, lock1)

	lock2, err := lm.AcquireLock(ctx, "job-b")
	if err != nil {
		t.Fatalf("Second AcquireLock() error = %v", err)
	}
	defer lm.ReleaseLock(ctx, lock2)

	if lock1.jobID == lock2.jobID {
		t.Error("Locks should have different job IDs")
	}
}

func TestLockManager_ReleaseLock(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	lock, _ := lm.AcquireLock(ctx, "test-job-3")

	// Release
	if err := lm.ReleaseLock(ctx, lock); err != nil {
		t.Fatalf("ReleaseLock() error = %v", err)
	}

	if lock.IsHeld() {
		t.Error("Lock should not be held after release")
	}

	// Should be able to acquire again
	lock2, err := lm.AcquireLock(ctx, "test-job-3")
	if err != nil {
		t.Fatalf("AcquireLock after release error = %v", err)
	}
	lm.ReleaseLock(ctx, lock2)
}

func TestLockManager_ReleaseLock_Nil(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	// Should not panic
	if err := lm.ReleaseLock(ctx, nil); err != nil {
		t.Errorf("ReleaseLock(nil) error = %v", err)
	}
}

func TestLockManager_ReleaseLock_NotHeld(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	lock, _ := lm.AcquireLock(ctx, "test-job-4")
	lm.ReleaseLock(ctx, lock) // Release once

	// Release again - should be safe
	if err := lm.ReleaseLock(ctx, lock); err != nil {
		t.Errorf("Double ReleaseLock() error = %v", err)
	}
}

func TestLockManager_CheckIdempotency_NotExists(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	completed, err := lm.CheckIdempotency(ctx, "unique-key-123")
	if err != nil {
		t.Fatalf("CheckIdempotency() error = %v", err)
	}
	if completed {
		t.Error("Job should not be marked as completed")
	}
}

func TestLockManager_CheckIdempotency_EmptyKey(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	completed, err := lm.CheckIdempotency(ctx, "")
	if err != nil {
		t.Fatalf("CheckIdempotency() error = %v", err)
	}
	if completed {
		t.Error("Empty key should return false")
	}
}

func TestLockManager_MarkCompleted(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	// Mark as completed
	if err := lm.MarkCompleted(ctx, "complete-key-1", "job-123"); err != nil {
		t.Fatalf("MarkCompleted() error = %v", err)
	}

	// Check idempotency
	completed, err := lm.CheckIdempotency(ctx, "complete-key-1")
	if err != nil {
		t.Fatalf("CheckIdempotency() error = %v", err)
	}
	if !completed {
		t.Error("Job should be marked as completed")
	}
}

func TestLockManager_MarkCompleted_EmptyKey(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	// Should not error with empty key
	if err := lm.MarkCompleted(ctx, "", "job-456"); err != nil {
		t.Errorf("MarkCompleted() with empty key error = %v", err)
	}
}

func TestLockManager_GetRunningJobs(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	// Acquire some locks
	lock1, _ := lm.AcquireLock(ctx, "running-job-1")
	lock2, _ := lm.AcquireLock(ctx, "running-job-2")
	defer lm.ReleaseLock(ctx, lock1)
	defer lm.ReleaseLock(ctx, lock2)

	// Get running jobs
	running, err := lm.GetRunningJobs(ctx)
	if err != nil {
		t.Fatalf("GetRunningJobs() error = %v", err)
	}

	if len(running) < 2 {
		t.Errorf("len(running) = %v, want >= 2", len(running))
	}

	// Both jobs should be present
	if _, ok := running["running-job-1"]; !ok {
		t.Error("running-job-1 not found in running jobs")
	}
	if _, ok := running["running-job-2"]; !ok {
		t.Error("running-job-2 not found in running jobs")
	}
}

func TestLockManager_GetWorkerJobs(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	// Acquire locks
	lock1, _ := lm.AcquireLock(ctx, "worker-job-1")
	lock2, _ := lm.AcquireLock(ctx, "worker-job-2")
	defer lm.ReleaseLock(ctx, lock1)
	defer lm.ReleaseLock(ctx, lock2)

	// Get this worker's jobs
	jobs, err := lm.GetWorkerJobs(ctx, lm.GetWorkerID())
	if err != nil {
		t.Fatalf("GetWorkerJobs() error = %v", err)
	}

	if len(jobs) < 2 {
		t.Errorf("len(jobs) = %v, want >= 2", len(jobs))
	}
}

func TestLockManager_CleanupStaleJobs(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	// This test would need to simulate stale jobs
	// For now, just verify it doesn't error with no stale jobs
	cleaned, err := lm.CleanupStaleJobs(ctx, time.Minute)
	if err != nil {
		t.Fatalf("CleanupStaleJobs() error = %v", err)
	}
	if len(cleaned) != 0 {
		t.Logf("Cleaned %d stale jobs", len(cleaned))
	}
}

func TestLockManager_ReleaseAllLocks(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	// Acquire multiple locks
	lock1, _ := lm.AcquireLock(ctx, "release-all-1")
	lock2, _ := lm.AcquireLock(ctx, "release-all-2")
	lock3, _ := lm.AcquireLock(ctx, "release-all-3")

	_ = lock1 // Prevent unused variable warnings
	_ = lock2
	_ = lock3

	// Release all
	if err := lm.ReleaseAllLocks(ctx); err != nil {
		t.Fatalf("ReleaseAllLocks() error = %v", err)
	}

	// All locks should be released - can acquire again
	newLock, err := lm.AcquireLock(ctx, "release-all-1")
	if err != nil {
		t.Errorf("AcquireLock after ReleaseAllLocks error = %v", err)
	}
	if newLock != nil {
		lm.ReleaseLock(ctx, newLock)
	}
}

func TestLockManager_ConcurrentAcquire(t *testing.T) {
	lm, ctx := setupTestLockManager(t)

	jobID := "concurrent-job"
	successCount := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Try to acquire same lock from multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lock, err := lm.AcquireLock(ctx, jobID)
			if err == nil && lock != nil {
				mu.Lock()
				successCount++
				mu.Unlock()
				time.Sleep(10 * time.Millisecond)
				lm.ReleaseLock(ctx, lock)
			}
		}()
	}

	wg.Wait()

	// Only one should succeed initially
	if successCount == 0 {
		t.Error("At least one goroutine should acquire the lock")
	}
}

func TestLockManager_HeartbeatExtendsLock(t *testing.T) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)

	// Short TTL and heartbeat for testing
	lmConfig := LockManagerConfig{
		LockTTL:        2 * time.Second,
		HeartbeatRate:  500 * time.Millisecond,
		IdempotencyTTL: time.Hour,
	}
	lm := NewLockManager(client, lmConfig)
	ctx := context.Background()

	lock, err := lm.AcquireLock(ctx, "heartbeat-job")
	if err != nil {
		t.Fatalf("AcquireLock() error = %v", err)
	}

	// Wait for heartbeat to extend the lock
	time.Sleep(1500 * time.Millisecond)

	// Lock should still be held (heartbeat extended it)
	if !lock.IsHeld() {
		t.Error("Lock should still be held due to heartbeat")
	}

	lm.ReleaseLock(ctx, lock)
}

// Error cases
func TestErrors(t *testing.T) {
	t.Run("ErrLockNotAcquired", func(t *testing.T) {
		if ErrLockNotAcquired.Error() != "failed to acquire job lock" {
			t.Errorf("ErrLockNotAcquired = %v", ErrLockNotAcquired)
		}
	})

	t.Run("ErrLockNotHeld", func(t *testing.T) {
		if ErrLockNotHeld.Error() != "lock not held by this worker" {
			t.Errorf("ErrLockNotHeld = %v", ErrLockNotHeld)
		}
	})

	t.Run("ErrJobAlreadyDone", func(t *testing.T) {
		if ErrJobAlreadyDone.Error() != "job already completed (idempotency check)" {
			t.Errorf("ErrJobAlreadyDone = %v", ErrJobAlreadyDone)
		}
	})
}

// Benchmarks
func BenchmarkLockManager_AcquireRelease(b *testing.B) {
	testutil.SkipIfNoRedis(&testing.T{})
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(&testing.T{}, config)
	lm := NewLockManager(client, DefaultLockManagerConfig())
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jobID := testutil.GenerateTestID()
		lock, _ := lm.AcquireLock(ctx, jobID)
		if lock != nil {
			lm.ReleaseLock(ctx, lock)
		}
	}
}

func BenchmarkLockManager_CheckIdempotency(b *testing.B) {
	testutil.SkipIfNoRedis(&testing.T{})
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(&testing.T{}, config)
	lm := NewLockManager(client, DefaultLockManagerConfig())
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lm.CheckIdempotency(ctx, "bench-key")
	}
}
