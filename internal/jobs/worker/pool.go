package worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/lock"
)

// JobHandler is a function that handles a specific job type
type JobHandler func(ctx context.Context, payload []byte) error

// WorkerPoolConfig configures the worker pool
type WorkerPoolConfig struct {
	Concurrency        int           // Number of concurrent workers
	PollInterval       time.Duration // How often to poll for jobs
	ShutdownTimeout    time.Duration // Timeout for graceful shutdown
	EnableLocking      bool          // Enable distributed locking
	EnableIdempotency  bool          // Enable idempotency checks
	StaleJobCleanup    time.Duration // Interval for cleaning stale jobs
	StaleJobThreshold  time.Duration // Time after which a job is considered stale
}

// DefaultWorkerPoolConfig returns sensible defaults
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		Concurrency:        8,
		PollInterval:       100 * time.Millisecond,
		ShutdownTimeout:    30 * time.Second,
		EnableLocking:      true,
		EnableIdempotency:  true,
		StaleJobCleanup:    time.Minute,
		StaleJobThreshold:  10 * time.Minute,
	}
}

// WorkerPool manages a pool of job workers
type WorkerPool struct {
	config      WorkerPoolConfig
	queue       jobs.Queue
	lockManager *lock.LockManager
	logger      *zap.Logger
	handlers    map[string]JobHandler
	mu          sync.RWMutex

	// State
	running atomic.Bool
	wg      sync.WaitGroup
	stopCh  chan struct{}

	// Metrics
	activeWorkers atomic.Int64
	processedJobs atomic.Int64
	failedJobs    atomic.Int64
	skippedJobs   atomic.Int64 // Jobs skipped due to locking/idempotency
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(q jobs.Queue, logger *zap.Logger, config WorkerPoolConfig) *WorkerPool {
	return &WorkerPool{
		config:   config,
		queue:    q,
		logger:   logger,
		handlers: make(map[string]JobHandler),
		stopCh:   make(chan struct{}),
	}
}

// SetLockManager sets the lock manager for distributed locking
func (p *WorkerPool) SetLockManager(lm *lock.LockManager) {
	p.lockManager = lm
	p.logger.Info("Lock manager configured",
		zap.String("worker_id", lm.GetWorkerID()),
		zap.Bool("locking_enabled", p.config.EnableLocking),
		zap.Bool("idempotency_enabled", p.config.EnableIdempotency),
	)
}

// RegisterHandler registers a handler for a job type
func (p *WorkerPool) RegisterHandler(jobType string, handler JobHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[jobType] = handler
	p.logger.Info("Registered job handler", zap.String("type", jobType))
}

// Start starts the worker pool
func (p *WorkerPool) Start(ctx context.Context) error {
	if p.running.Load() {
		return fmt.Errorf("worker pool already running")
	}

	p.running.Store(true)
	p.logger.Info("Starting worker pool",
		zap.Int("concurrency", p.config.Concurrency),
		zap.Duration("poll_interval", p.config.PollInterval),
		zap.Bool("locking_enabled", p.config.EnableLocking && p.lockManager != nil),
		zap.Bool("idempotency_enabled", p.config.EnableIdempotency && p.lockManager != nil),
	)

	// Start workers
	for i := 0; i < p.config.Concurrency; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	// Start scheduled job processor
	p.wg.Add(1)
	go p.scheduledProcessor(ctx)

	// Start stale job cleanup if locking is enabled
	if p.config.EnableLocking && p.lockManager != nil {
		p.wg.Add(1)
		go p.staleJobCleaner(ctx)
	}

	return nil
}

// Stop gracefully stops the worker pool
func (p *WorkerPool) Stop(ctx context.Context) error {
	if !p.running.Load() {
		return nil
	}

	p.logger.Info("Stopping worker pool")
	p.running.Store(false)
	close(p.stopCh)

	// Release all locks held by this worker
	if p.lockManager != nil {
		if err := p.lockManager.ReleaseAllLocks(ctx); err != nil {
			p.logger.Error("Failed to release all locks", zap.Error(err))
		}
	}

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("Worker pool stopped gracefully")
	case <-time.After(p.config.ShutdownTimeout):
		p.logger.Warn("Worker pool shutdown timed out")
	case <-ctx.Done():
		p.logger.Warn("Worker pool shutdown cancelled")
	}

	return nil
}

// worker is a single worker goroutine
func (p *WorkerPool) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	logger := p.logger.With(zap.Int("worker_id", id))
	logger.Debug("Worker started")

	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			// Only log if still running (avoid logging after test cleanup)
			if p.running.Load() {
				logger.Debug("Worker stopping")
			}
			return
		case <-ctx.Done():
			if p.running.Load() {
				logger.Debug("Worker context cancelled")
			}
			return
		case <-ticker.C:
			p.processNextJob(ctx, logger)
		}
	}
}

// processNextJob attempts to process the next available job
func (p *WorkerPool) processNextJob(ctx context.Context, logger *zap.Logger) {
	// Dequeue job
	job, err := p.queue.Dequeue(ctx)
	if err == jobs.ErrQueueEmpty {
		return
	}
	if err != nil {
		// Don't log errors during shutdown
		if p.running.Load() {
			logger.Error("Failed to dequeue job", zap.Error(err))
		}
		return
	}

	logger = logger.With(
		zap.String("job_id", job.ID),
		zap.String("job_type", job.Type),
		zap.Int("attempt", job.Attempts),
	)

	// Check idempotency before processing
	if p.config.EnableIdempotency && p.lockManager != nil && job.UniqueKey != "" {
		completed, err := p.lockManager.CheckIdempotency(ctx, job.UniqueKey)
		if err != nil {
			logger.Warn("Failed to check idempotency", zap.Error(err))
		} else if completed {
			logger.Info("Job already completed (idempotency check), skipping")
			p.queue.Complete(ctx, job.ID)
			p.skippedJobs.Add(1)
			return
		}
	}

	// Acquire distributed lock if enabled
	var jobLock *lock.JobLock
	if p.config.EnableLocking && p.lockManager != nil {
		var err error
		jobLock, err = p.lockManager.AcquireLock(ctx, job.ID)
		if err != nil {
			if err == lock.ErrLockNotAcquired {
				logger.Debug("Job already locked by another worker, skipping")
				// Re-queue the job since we couldn't acquire the lock
				p.requeueJob(ctx, job, logger)
			} else {
				logger.Error("Failed to acquire job lock", zap.Error(err))
			}
			p.skippedJobs.Add(1)
			return
		}
		defer func() {
			if err := p.lockManager.ReleaseLock(ctx, jobLock); err != nil {
				logger.Warn("Failed to release job lock", zap.Error(err))
			}
		}()
	}

	p.activeWorkers.Add(1)
	jobs.GlobalMetrics.RecordJobStarted()
	defer func() {
		p.activeWorkers.Add(-1)
	}()

	logger.Info("Processing job")

	// Get handler
	p.mu.RLock()
	handler, ok := p.handlers[job.Type]
	p.mu.RUnlock()

	if !ok {
		logger.Error("No handler registered for job type")
		p.queue.Fail(ctx, job.ID, fmt.Errorf("no handler for job type: %s", job.Type))
		p.failedJobs.Add(1)
		jobs.GlobalMetrics.RecordJobFailed(false)
		return
	}

	// Execute with timeout
	execCtx, cancel := context.WithTimeout(ctx, job.Timeout)
	defer cancel()

	start := time.Now()
	err = handler(execCtx, job.Payload)
	duration := time.Since(start)

	if err != nil {
		logger.Error("Job failed",
			zap.Error(err),
			zap.Duration("duration", duration),
		)
		p.queue.Fail(ctx, job.ID, err)
		p.failedJobs.Add(1)
		willRetry := job.Attempts < job.MaxRetries
		jobs.GlobalMetrics.RecordJobFailed(willRetry)
	} else {
		logger.Info("Job completed",
			zap.Duration("duration", duration),
		)
		p.queue.Complete(ctx, job.ID)
		p.processedJobs.Add(1)
		jobs.GlobalMetrics.RecordJobCompleted(duration)

		// Mark as completed for idempotency
		if p.config.EnableIdempotency && p.lockManager != nil && job.UniqueKey != "" {
			if err := p.lockManager.MarkCompleted(ctx, job.UniqueKey, job.ID); err != nil {
				logger.Warn("Failed to mark job as completed for idempotency", zap.Error(err))
			}
		}
	}
}

// requeueJob re-queues a job that couldn't be processed
func (p *WorkerPool) requeueJob(ctx context.Context, job *jobs.JobPayload, logger *zap.Logger) {
	// Reset job status
	job.Status = jobs.JobStatusPending
	job.StartedAt = nil
	job.Attempts-- // Don't count this as an attempt

	if err := p.queue.UpdateJob(ctx, job); err != nil {
		logger.Error("Failed to update job for requeue", zap.Error(err))
		return
	}

	// Add back to queue
	queueKey := job.Priority.QueueName()
	if err := p.queue.RequeueJob(ctx, job.ID, queueKey); err != nil {
		logger.Error("Failed to requeue job", zap.Error(err))
	}
}

// scheduledProcessor moves scheduled jobs to queues
func (p *WorkerPool) scheduledProcessor(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			processed, err := p.queue.ProcessScheduled(ctx)
			if err != nil {
				p.logger.Error("Failed to process scheduled jobs", zap.Error(err))
			} else if processed > 0 {
				p.logger.Debug("Processed scheduled jobs", zap.Int("count", processed))
			}
		}
	}
}

// staleJobCleaner periodically cleans up stale job locks
func (p *WorkerPool) staleJobCleaner(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.StaleJobCleanup)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleaned, err := p.lockManager.CleanupStaleJobs(ctx, p.config.StaleJobThreshold)
			if err != nil {
				p.logger.Error("Failed to cleanup stale jobs", zap.Error(err))
			} else if len(cleaned) > 0 {
				p.logger.Info("Cleaned up stale job locks",
					zap.Int("count", len(cleaned)),
					zap.Strings("job_ids", cleaned),
				)
			}
		}
	}
}

// Stats returns worker pool statistics
func (p *WorkerPool) Stats() jobs.WorkerPoolStats {
	stats := jobs.WorkerPoolStats{
		Running:       p.running.Load(),
		ActiveWorkers: p.activeWorkers.Load(),
		ProcessedJobs: p.processedJobs.Load(),
		FailedJobs:    p.failedJobs.Load(),
		SkippedJobs:   p.skippedJobs.Load(),
		Concurrency:   p.config.Concurrency,
	}

	if p.lockManager != nil {
		stats.WorkerID = p.lockManager.GetWorkerID()
	}

	return stats
}

// GetRunningJobs returns information about currently running jobs
func (p *WorkerPool) GetRunningJobs(ctx context.Context) (map[string]string, error) {
	if p.lockManager == nil {
		return nil, fmt.Errorf("lock manager not configured")
	}
	return p.lockManager.GetRunningJobs(ctx)
}
