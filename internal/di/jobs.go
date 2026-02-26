package di

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	httpctrl "github.com/jrjohn/arcana-cloud-go/internal/controller/http"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/handler"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/lock"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/queue"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/scheduler"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/worker"
	"github.com/jrjohn/arcana-cloud-go/internal/middleware"
)

// JobsModule provides job worker system dependencies
var JobsModule = fx.Module("jobs",
	fx.Provide(
		provideRedisClient,
		provideJobQueue,
		provideLockManager,
		provideWorkerPool,
		provideScheduler,
		provideJobService,
		provideHandlerRegistry,
		provideJobController,
	),
	fx.Invoke(
		registerDefaultHandlers,
		registerDefaultScheduledJobs,
		startJobWorkers,
	),
)

// WorkerConfig holds worker pool configuration
type WorkerConfig struct {
	Enabled     bool `mapstructure:"enabled"`
	Concurrency int  `mapstructure:"concurrency"`
}

func provideRedisClient(lc fx.Lifecycle, cfg *config.RedisConfig, logger *zap.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
	)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("Closing Redis connection")
			return client.Close()
		},
	})

	return client, nil
}

func provideJobQueue(client *redis.Client) *queue.RedisQueue {
	return queue.NewRedisQueue(client)
}

func provideLockManager(client *redis.Client, logger *zap.Logger) *lock.LockManager {
	config := lock.DefaultLockManagerConfig()
	lm := lock.NewLockManager(client, config)
	logger.Info("Lock manager initialized",
		zap.String("worker_id", lm.GetWorkerID()),
		zap.Duration("lock_ttl", config.LockTTL),
		zap.Duration("idempotency_ttl", config.IdempotencyTTL),
	)
	return lm
}

func provideWorkerPool(q *queue.RedisQueue, lm *lock.LockManager, logger *zap.Logger) *worker.WorkerPool {
	config := worker.DefaultWorkerPoolConfig()
	pool := worker.NewWorkerPool(q, logger, config)
	pool.SetLockManager(lm)
	return pool
}

func provideScheduler(client *redis.Client, q *queue.RedisQueue, logger *zap.Logger) *scheduler.Scheduler {
	return scheduler.NewScheduler(client, q, logger)
}

func provideJobService(q *queue.RedisQueue, pool *worker.WorkerPool, sched *scheduler.Scheduler) jobs.Service {
	return jobs.NewJobService(q, pool, sched)
}

func provideHandlerRegistry(pool *worker.WorkerPool, logger *zap.Logger) *handler.Registry {
	return handler.NewRegistry(pool, logger)
}

func provideJobController(
	jobService jobs.Service,
	sched *scheduler.Scheduler,
	authMiddleware *middleware.AuthMiddleware,
) *httpctrl.JobController {
	return httpctrl.NewJobController(jobService, sched, authMiddleware)
}

// registerDefaultHandlers registers the default job handlers
func registerDefaultHandlers(registry *handler.Registry, logger *zap.Logger) {
	// Register email job handler
	handler.Register(registry, "email", func(ctx context.Context, payload handler.EmailJobPayload) error {
		logger.Info("Processing email job",
			zap.Strings("to", payload.To),
			zap.String("subject", payload.Subject),
		)
		// Note: Email sending is handled by external mail service integration
		return nil
	})

	// Register webhook job handler
	handler.Register(registry, "webhook", func(ctx context.Context, payload handler.WebhookJobPayload) error {
		logger.Info("Processing webhook job",
			zap.String("url", payload.URL),
			zap.String("method", payload.Method),
		)
		// Note: Webhook delivery is handled by external HTTP client
		return nil
	})

	// Register cleanup job handler
	handler.Register(registry, "cleanup", func(ctx context.Context, payload handler.CleanupJobPayload) error {
		logger.Info("Processing cleanup job",
			zap.String("type", payload.Type),
			zap.Int("older_than_days", payload.OlderThan),
			zap.Bool("dry_run", payload.DryRun),
		)
		// Note: Cleanup logic delegates to repository layer
		return nil
	})

	// Register notification job handler
	handler.Register(registry, "notification", func(ctx context.Context, payload handler.NotificationJobPayload) error {
		logger.Info("Processing notification job",
			zap.Uint("user_id", payload.UserID),
			zap.String("type", payload.Type),
			zap.String("title", payload.Title),
		)
		// Note: Notification delivery is handled by notification service
		return nil
	})

	// Register report job handler
	handler.Register(registry, "report", func(ctx context.Context, payload handler.ReportJobPayload) error {
		logger.Info("Processing report job",
			zap.String("report_type", payload.ReportType),
			zap.String("format", payload.Format),
		)
		// Note: Report generation is handled by report service
		return nil
	})

	// Register sync job handler
	handler.Register(registry, "sync", func(ctx context.Context, payload handler.SyncJobPayload) error {
		logger.Info("Processing sync job",
			zap.String("source", payload.Source),
			zap.String("destination", payload.Destination),
			zap.String("entity_type", payload.EntityType),
		)
		// Note: Sync logic is handled by synchronization service
		return nil
	})

	logger.Info("Registered default job handlers")
}

// registerDefaultScheduledJobs registers the default scheduled jobs
func registerDefaultScheduledJobs(sched *scheduler.Scheduler, logger *zap.Logger) {
	// Register cleanup job - runs daily at midnight (singleton to prevent overlap)
	if err := sched.RegisterJob(scheduler.ScheduledJob{
		Name:     "daily-token-cleanup",
		Schedule: scheduler.DailyMidnight,
		JobType:  "cleanup",
		Payload: handler.CleanupJobPayload{
			Type:      "expired_tokens",
			OlderThan: 30,
			DryRun:    false,
		},
		Priority:  jobs.PriorityLow,
		Tags:      []string{"maintenance", "cleanup"},
		Singleton: true, // Only one instance can run at a time
	}); err != nil {
		logger.Warn("Failed to register daily-token-cleanup job", zap.Error(err))
	}

	// Register hourly stats job
	if err := sched.RegisterJob(scheduler.ScheduledJob{
		Name:     "hourly-stats-sync",
		Schedule: scheduler.EveryHour,
		JobType:  "sync",
		Payload: handler.SyncJobPayload{
			Source:      "database",
			Destination: "cache",
			EntityType:  "stats",
			FullSync:    false,
		},
		Priority:  jobs.PriorityNormal,
		Tags:      []string{"stats", "sync"},
		Singleton: false, // Multiple instances can run
	}); err != nil {
		logger.Warn("Failed to register hourly-stats-sync job", zap.Error(err))
	}

	logger.Info("Registered default scheduled jobs")
}

// startJobWorkers starts the worker pool and scheduler
func startJobWorkers(lc fx.Lifecycle, pool *worker.WorkerPool, sched *scheduler.Scheduler, lm *lock.LockManager, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting job worker pool",
				zap.String("worker_id", lm.GetWorkerID()),
			)
			if err := pool.Start(ctx); err != nil {
				return fmt.Errorf("failed to start worker pool: %w", err)
			}

			logger.Info("Starting job scheduler")
			if err := sched.Start(ctx); err != nil {
				return fmt.Errorf("failed to start scheduler: %w", err)
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping job scheduler")
			if err := sched.Stop(ctx); err != nil {
				logger.Warn("Error stopping scheduler", zap.Error(err))
			}

			logger.Info("Stopping job worker pool")
			if err := pool.Stop(ctx); err != nil {
				logger.Warn("Error stopping worker pool", zap.Error(err))
			}

			logger.Info("Releasing all job locks")
			if err := lm.ReleaseAllLocks(ctx); err != nil {
				logger.Warn("Error releasing locks", zap.Error(err))
			}

			return nil
		},
	})
}
