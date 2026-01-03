package scheduler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
)

const (
	// Common cron expressions
	EveryMinute      = "* * * * *"
	EveryFiveMinutes = "*/5 * * * *"
	EveryHour        = "0 * * * *"
	DailyMidnight    = "0 0 * * *"
	WeeklyMonday     = "0 0 * * 1"
	MonthlyFirst     = "0 0 1 * *"

	// Redis key prefixes for scheduler
	leaderKey             = "arcana:jobs:scheduler:leader"
	cronExecutionPrefix   = "arcana:jobs:cron:execution:"
	cronLockPrefix        = "arcana:jobs:cron:lock:"
)

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	LeaderLockTTL        time.Duration
	CronExecutionLockTTL time.Duration
	CronDeduplicationTTL time.Duration
}

// DefaultSchedulerConfig returns default scheduler configuration
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		LeaderLockTTL:        30 * time.Second,
		CronExecutionLockTTL: 60 * time.Second,
		CronDeduplicationTTL: 24 * time.Hour,
	}
}

// ScheduledJob represents a recurring job
type ScheduledJob struct {
	Name        string
	Schedule    string // Cron expression
	JobType     string
	Payload     any
	Priority    jobs.Priority
	UniqueKey   string // Optional: base key for deduplication
	Tags        []string
	Singleton   bool // If true, only one instance can run at a time
}

// Scheduler manages cron-based job scheduling with leader election
type Scheduler struct {
	redis    *redis.Client
	queue    jobs.Queue
	logger   *zap.Logger
	config   SchedulerConfig
	cron     *cron.Cron
	jobs     map[string]ScheduledJob
	mu       sync.RWMutex

	// Leader election
	instanceID string
	isLeader   bool
	leaderMu   sync.RWMutex

	// State
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewScheduler creates a new scheduler
func NewScheduler(redisClient *redis.Client, jobQueue jobs.Queue, logger *zap.Logger) *Scheduler {
	return NewSchedulerWithConfig(redisClient, jobQueue, logger, DefaultSchedulerConfig())
}

// NewSchedulerWithConfig creates a new scheduler with custom configuration
func NewSchedulerWithConfig(redisClient *redis.Client, jobQueue jobs.Queue, logger *zap.Logger, config SchedulerConfig) *Scheduler {
	return &Scheduler{
		redis:      redisClient,
		queue:      jobQueue,
		logger:     logger,
		config:     config,
		cron:       cron.New(cron.WithSeconds()),
		jobs:       make(map[string]ScheduledJob),
		instanceID: uuid.New().String(),
		stopCh:     make(chan struct{}),
	}
}

// RegisterJob registers a scheduled job
func (s *Scheduler) RegisterJob(job ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[job.Name]; exists {
		return fmt.Errorf("job %s already registered", job.Name)
	}

	// Validate cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(job.Schedule)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	s.jobs[job.Name] = job
	s.logger.Info("Registered scheduled job",
		zap.String("name", job.Name),
		zap.String("schedule", job.Schedule),
		zap.String("job_type", job.JobType),
		zap.Bool("singleton", job.Singleton),
	)

	return nil
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	s.running = true
	s.logger.Info("Starting scheduler", zap.String("instance_id", s.instanceID))

	// Start leader election
	s.wg.Add(1)
	go s.leaderElectionLoop(ctx)

	// Start cron scheduler
	s.setupCronJobs()
	s.cron.Start()

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop(ctx context.Context) error {
	if !s.running {
		return nil
	}

	s.logger.Info("Stopping scheduler")
	s.running = false
	close(s.stopCh)

	// Stop cron
	cronCtx := s.cron.Stop()
	select {
	case <-cronCtx.Done():
	case <-ctx.Done():
	}

	// Release leadership
	s.releaseLeadership(ctx)

	s.wg.Wait()
	return nil
}

// setupCronJobs sets up all registered cron jobs
func (s *Scheduler) setupCronJobs() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, job := range s.jobs {
		j := job // capture loop variable
		_, err := s.cron.AddFunc(j.Schedule, func() {
			s.executeScheduledJob(context.Background(), j)
		})
		if err != nil {
			s.logger.Error("Failed to add cron job",
				zap.String("name", j.Name),
				zap.Error(err),
			)
		}
	}
}

// executeScheduledJob executes a scheduled job if this instance is the leader
func (s *Scheduler) executeScheduledJob(ctx context.Context, job ScheduledJob) {
	s.leaderMu.RLock()
	isLeader := s.isLeader
	s.leaderMu.RUnlock()

	if !isLeader {
		s.logger.Debug("Skipping job execution - not leader",
			zap.String("name", job.Name),
		)
		return
	}

	// Generate execution window key for deduplication
	executionWindow := s.getExecutionWindow(job.Schedule)
	executionKey := s.generateExecutionKey(job.Name, executionWindow)

	// Try to acquire execution lock (prevents duplicate execution in same window)
	acquired, err := s.acquireCronExecutionLock(ctx, executionKey)
	if err != nil {
		s.logger.Error("Failed to acquire cron execution lock",
			zap.String("name", job.Name),
			zap.Error(err),
		)
		return
	}
	if !acquired {
		s.logger.Debug("Cron job already executed in this window",
			zap.String("name", job.Name),
			zap.String("window", executionWindow),
		)
		return
	}

	// For singleton jobs, check if one is already running
	if job.Singleton {
		running, err := s.isSingletonJobRunning(ctx, job.Name)
		if err != nil {
			s.logger.Warn("Failed to check singleton job status", zap.Error(err))
		} else if running {
			s.logger.Info("Singleton job already running, skipping",
				zap.String("name", job.Name),
			)
			return
		}
	}

	s.logger.Info("Executing scheduled job",
		zap.String("name", job.Name),
		zap.String("job_type", job.JobType),
		zap.String("execution_window", executionWindow),
	)

	// Create unique key for job deduplication
	uniqueKey := s.generateJobUniqueKey(job, executionWindow)

	// Create job payload
	opts := []jobs.JobOption{
		jobs.WithPriority(job.Priority),
		jobs.WithTags(append(job.Tags, "scheduled", "cron:"+job.Name)...),
		jobs.WithUniqueKey(uniqueKey),
	}

	payload, err := jobs.NewJobPayload(job.JobType, job.Payload, opts...)
	if err != nil {
		s.logger.Error("Failed to create job payload",
			zap.String("name", job.Name),
			zap.Error(err),
		)
		return
	}

	// Enqueue the job
	if err := s.queue.Enqueue(ctx, payload); err != nil {
		if err == jobs.ErrDuplicateJob {
			s.logger.Debug("Skipping duplicate scheduled job",
				zap.String("name", job.Name),
			)
			return
		}
		s.logger.Error("Failed to enqueue scheduled job",
			zap.String("name", job.Name),
			zap.Error(err),
		)
		return
	}

	s.logger.Info("Scheduled job enqueued",
		zap.String("name", job.Name),
		zap.String("job_id", payload.ID),
		zap.String("unique_key", uniqueKey),
	)
}

// getExecutionWindow returns a time window identifier based on the cron schedule
func (s *Scheduler) getExecutionWindow(schedule string) string {
	now := time.Now().UTC()

	// Determine window size based on schedule frequency
	switch {
	case schedule == EveryMinute || schedule == "* * * * *":
		return now.Format("2006-01-02T15:04")
	case schedule == EveryFiveMinutes || schedule == "*/5 * * * *":
		minute := (now.Minute() / 5) * 5
		return now.Format("2006-01-02T15:") + fmt.Sprintf("%02d", minute)
	case schedule == EveryHour || schedule == "0 * * * *":
		return now.Format("2006-01-02T15")
	case schedule == DailyMidnight || schedule == "0 0 * * *":
		return now.Format("2006-01-02")
	case schedule == WeeklyMonday || schedule == "0 0 * * 1":
		year, week := now.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	case schedule == MonthlyFirst || schedule == "0 0 1 * *":
		return now.Format("2006-01")
	default:
		// Default to minute-level granularity for unknown schedules
		return now.Format("2006-01-02T15:04")
	}
}

// generateExecutionKey creates a unique key for a cron execution window
func (s *Scheduler) generateExecutionKey(jobName, window string) string {
	return fmt.Sprintf("%s:%s", jobName, window)
}

// generateJobUniqueKey creates a unique key for job deduplication
func (s *Scheduler) generateJobUniqueKey(job ScheduledJob, window string) string {
	baseKey := job.UniqueKey
	if baseKey == "" {
		baseKey = job.Name
	}

	// Create a hash of the job details for uniqueness
	data := fmt.Sprintf("%s:%s:%s:%v", baseKey, job.JobType, window, job.Payload)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("cron:%s:%s", job.Name, hex.EncodeToString(hash[:8]))
}

// acquireCronExecutionLock tries to acquire a lock for cron execution
func (s *Scheduler) acquireCronExecutionLock(ctx context.Context, executionKey string) (bool, error) {
	lockKey := cronExecutionPrefix + executionKey

	// Use SETNX to atomically acquire lock
	acquired, err := s.redis.SetNX(ctx, lockKey, s.instanceID, s.config.CronDeduplicationTTL).Result()
	if err != nil {
		return false, err
	}

	return acquired, nil
}

// isSingletonJobRunning checks if a singleton job is currently running
func (s *Scheduler) isSingletonJobRunning(ctx context.Context, jobName string) (bool, error) {
	lockKey := cronLockPrefix + "singleton:" + jobName

	exists, err := s.redis.Exists(ctx, lockKey).Result()
	if err != nil {
		return false, err
	}

	return exists > 0, nil
}

// AcquireSingletonLock acquires a lock for singleton job execution
// This should be called by the job handler for singleton jobs
func (s *Scheduler) AcquireSingletonLock(ctx context.Context, jobName string, ttl time.Duration) (bool, error) {
	lockKey := cronLockPrefix + "singleton:" + jobName

	acquired, err := s.redis.SetNX(ctx, lockKey, s.instanceID, ttl).Result()
	if err != nil {
		return false, err
	}

	return acquired, nil
}

// ReleaseSingletonLock releases a singleton job lock
func (s *Scheduler) ReleaseSingletonLock(ctx context.Context, jobName string) error {
	lockKey := cronLockPrefix + "singleton:" + jobName

	// Only release if we own the lock
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	_, err := s.redis.Eval(ctx, script, []string{lockKey}, s.instanceID).Result()
	return err
}

// leaderElectionLoop continuously tries to acquire/maintain leadership
func (s *Scheduler) leaderElectionLoop(ctx context.Context) {
	defer s.wg.Done()

	// Try to acquire leadership immediately at start
	s.tryAcquireLeadership(ctx)

	ticker := time.NewTicker(s.config.LeaderLockTTL / 3)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tryAcquireLeadership(ctx)
		}
	}
}

// tryAcquireLeadership attempts to acquire or renew leadership
func (s *Scheduler) tryAcquireLeadership(ctx context.Context) {
	s.leaderMu.Lock()
	defer s.leaderMu.Unlock()

	// Try to set the leader key with NX (only if not exists)
	set, err := s.redis.SetNX(ctx, leaderKey, s.instanceID, s.config.LeaderLockTTL).Result()
	if err != nil {
		s.logger.Error("Failed to acquire leadership", zap.Error(err))
		s.isLeader = false
		return
	}

	if set {
		// We acquired leadership
		if !s.isLeader {
			s.logger.Info("Acquired scheduler leadership", zap.String("instance_id", s.instanceID))
		}
		s.isLeader = true
		return
	}

	// Check if we're already the leader and renew
	currentLeader, err := s.redis.Get(ctx, leaderKey).Result()
	if err != nil {
		s.isLeader = false
		return
	}

	if currentLeader == s.instanceID {
		// Renew our lease
		s.redis.Expire(ctx, leaderKey, s.config.LeaderLockTTL)
		s.isLeader = true
	} else {
		if s.isLeader {
			s.logger.Info("Lost scheduler leadership",
				zap.String("instance_id", s.instanceID),
				zap.String("new_leader", currentLeader),
			)
		}
		s.isLeader = false
	}
}

// releaseLeadership releases leadership when shutting down
func (s *Scheduler) releaseLeadership(ctx context.Context) {
	s.leaderMu.Lock()
	defer s.leaderMu.Unlock()

	if !s.isLeader {
		return
	}

	// Only delete if we're still the leader
	currentLeader, err := s.redis.Get(ctx, leaderKey).Result()
	if err == nil && currentLeader == s.instanceID {
		s.redis.Del(ctx, leaderKey)
		s.logger.Info("Released scheduler leadership", zap.String("instance_id", s.instanceID))
	}

	s.isLeader = false
}

// IsLeader returns whether this instance is the leader
func (s *Scheduler) IsLeader() bool {
	s.leaderMu.RLock()
	defer s.leaderMu.RUnlock()
	return s.isLeader
}

// ListJobs returns all registered scheduled jobs
func (s *Scheduler) ListJobs() []jobs.ScheduledJobInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]jobs.ScheduledJobInfo, 0, len(s.jobs))
	for _, job := range s.jobs {
		result = append(result, jobs.ScheduledJobInfo{
			Name:      job.Name,
			Schedule:  job.Schedule,
			JobType:   job.JobType,
			Priority:  job.Priority.String(),
			Singleton: job.Singleton,
		})
	}
	return result
}

// ListScheduledJobs returns all registered ScheduledJob structs (internal use)
func (s *Scheduler) ListScheduledJobs() []ScheduledJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ScheduledJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		result = append(result, job)
	}
	return result
}

// GetNextRun returns the next scheduled run time for a job
func (s *Scheduler) GetNextRun(jobName string) (time.Time, error) {
	s.mu.RLock()
	job, exists := s.jobs[jobName]
	s.mu.RUnlock()

	if !exists {
		return time.Time{}, fmt.Errorf("job %s not found", jobName)
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(job.Schedule)
	if err != nil {
		return time.Time{}, err
	}

	return schedule.Next(time.Now()), nil
}

// GetRecentExecutions returns recent execution records for a job
func (s *Scheduler) GetRecentExecutions(ctx context.Context, jobName string, limit int) ([]string, error) {
	pattern := cronExecutionPrefix + jobName + ":*"
	var cursor uint64
	var executions []string

	for {
		keys, nextCursor, err := s.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			// Extract the window from the key
			if len(key) > len(cronExecutionPrefix+jobName+":") {
				window := key[len(cronExecutionPrefix+jobName+":"):]
				executions = append(executions, window)
			}
		}

		cursor = nextCursor
		if cursor == 0 || len(executions) >= limit {
			break
		}
	}

	// Sort and limit
	if len(executions) > limit {
		executions = executions[:limit]
	}

	return executions, nil
}
