package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/handler"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/lock"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/queue"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/scheduler"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/worker"
	"github.com/jrjohn/arcana-cloud-go/pkg/logger"
)

func main() {
	cfg, log := mustLoadConfig()
	defer log.Sync()

	log.Info("Starting Arcana Cloud Worker",
		zap.String("version", cfg.App.Version),
		zap.String("environment", cfg.App.Environment),
	)

	ctx := context.Background()
	redisClient := mustConnectRedis(cfg, ctx, log)
	defer redisClient.Close()

	jobQueue := queue.NewRedisQueue(redisClient)
	lockManager := setupLockManager(redisClient, log)
	pool := setupWorkerPool(jobQueue, lockManager, log)

	registry := handler.NewRegistry(pool, log)
	registerHandlers(registry, log)

	sched := setupScheduler(redisClient, jobQueue, log)
	registerScheduledJobs(sched, log)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := pool.Start(ctx); err != nil {
		log.Fatal("Failed to start worker pool", zap.Error(err))
	}
	if err := sched.Start(ctx); err != nil {
		log.Fatal("Failed to start scheduler", zap.Error(err))
	}

	go startMetricsServer(sched, lockManager, log)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutdown signal received, stopping workers...")
	cancel()
	gracefulShutdown(pool, sched, lockManager, log)
}

func mustLoadConfig() (*config.Config, *zap.Logger) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	log, err := logger.New(logger.Config{
		Level:       "info",
		Development: cfg.App.Debug,
		Encoding:    "json",
	})
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	return cfg, log
}

func mustConnectRedis(cfg *config.Config, ctx context.Context, log *zap.Logger) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	log.Info("Connected to Redis", zap.String("host", cfg.Redis.Host))
	return client
}

func setupLockManager(redisClient *redis.Client, log *zap.Logger) *lock.LockManager {
	lockConfig := lock.DefaultLockManagerConfig()
	lm := lock.NewLockManager(redisClient, lockConfig)
	log.Info("Lock manager initialized",
		zap.String("worker_id", lm.GetWorkerID()),
		zap.Duration("lock_ttl", lockConfig.LockTTL),
		zap.Duration("idempotency_ttl", lockConfig.IdempotencyTTL),
	)
	return lm
}

func setupWorkerPool(jobQueue *queue.RedisQueue, lockManager *lock.LockManager, log *zap.Logger) *worker.WorkerPool {
	workerConfig := worker.DefaultWorkerPoolConfig()
	if concurrency := os.Getenv("ARCANA_WORKER_CONCURRENCY"); concurrency != "" {
		fmt.Sscanf(concurrency, "%d", &workerConfig.Concurrency)
	}
	if os.Getenv("ARCANA_WORKER_DISABLE_LOCKING") == "true" {
		workerConfig.EnableLocking = false
	}
	if os.Getenv("ARCANA_WORKER_DISABLE_IDEMPOTENCY") == "true" {
		workerConfig.EnableIdempotency = false
	}
	pool := worker.NewWorkerPool(jobQueue, log, workerConfig)
	pool.SetLockManager(lockManager)
	return pool
}

func setupScheduler(redisClient *redis.Client, jobQueue *queue.RedisQueue, log *zap.Logger) *scheduler.Scheduler {
	schedConfig := scheduler.DefaultSchedulerConfig()
	return scheduler.NewSchedulerWithConfig(redisClient, jobQueue, log, schedConfig)
}

func startMetricsServer(sched *scheduler.Scheduler, lockManager *lock.LockManager, log *zap.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", jobs.GlobalMetrics.PrometheusHandler())
	mux.HandleFunc("/health", handleHealth(sched, lockManager))
	mux.HandleFunc("/ready", handleReady())
	mux.HandleFunc("/running", handleRunning(lockManager))

	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9100"
	}
	log.Info("Starting metrics server", zap.String("port", metricsPort))
	if err := http.ListenAndServe(":"+metricsPort, mux); err != nil {
		log.Error("Metrics server error", zap.Error(err))
	}
}

func handleHealth(sched *scheduler.Scheduler, lockManager *lock.LockManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := jobs.GlobalMetrics.GetHealthCheck(sched.IsLeader())
		w.Header().Set("Content-Type", "application/json")
		if health.Status == "healthy" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		fmt.Fprintf(w, `{"status":"%s","workers_active":%d,"jobs_pending":%d,"is_leader":%t,"worker_id":"%s"}`,
			health.Status, health.WorkersActive, health.JobsPending, health.IsLeader, lockManager.GetWorkerID())
	}
}

func handleReady() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}
}

func handleRunning(lockManager *lock.LockManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runningJobs, err := lockManager.GetRunningJobs(r.Context())
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"running_jobs":%d,"jobs":{`, len(runningJobs))
		first := true
		for jobID, workerInfo := range runningJobs {
			if !first {
				w.Write([]byte(","))
			}
			fmt.Fprintf(w, `"%s":"%s"`, jobID, workerInfo)
			first = false
		}
		w.Write([]byte("}}"))
	}
}

func gracefulShutdown(pool *worker.WorkerPool, sched *scheduler.Scheduler, lockManager *lock.LockManager, log *zap.Logger) {
	workerConfig := worker.DefaultWorkerPoolConfig()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), workerConfig.ShutdownTimeout)
	defer cancel()

	if err := sched.Stop(shutdownCtx); err != nil {
		log.Error("Error stopping scheduler", zap.Error(err))
	}
	if err := pool.Stop(shutdownCtx); err != nil {
		log.Error("Error stopping worker pool", zap.Error(err))
	}
	if err := lockManager.ReleaseAllLocks(shutdownCtx); err != nil {
		log.Error("Error releasing locks", zap.Error(err))
	}
	log.Info("Worker shutdown complete")
}

func registerHandlers(registry *handler.Registry, log *zap.Logger) {
	// Register all job handlers
	handler.Register(registry, "email", func(ctx context.Context, payload handler.EmailJobPayload) error {
		log.Info("Processing email job",
			zap.Strings("to", payload.To),
			zap.String("subject", payload.Subject),
		)
		// Implement email sending logic
		return nil
	})

	handler.Register(registry, "webhook", func(ctx context.Context, payload handler.WebhookJobPayload) error {
		log.Info("Processing webhook job",
			zap.String("url", payload.URL),
			zap.String("method", payload.Method),
		)
		// Implement webhook logic
		return nil
	})

	handler.Register(registry, "cleanup", func(ctx context.Context, payload handler.CleanupJobPayload) error {
		log.Info("Processing cleanup job",
			zap.String("type", payload.Type),
			zap.Int("older_than_days", payload.OlderThan),
		)
		// Implement cleanup logic
		return nil
	})

	handler.Register(registry, "notification", func(ctx context.Context, payload handler.NotificationJobPayload) error {
		log.Info("Processing notification job",
			zap.Uint("user_id", payload.UserID),
			zap.String("type", payload.Type),
		)
		// Implement notification logic
		return nil
	})

	handler.Register(registry, "report", func(ctx context.Context, payload handler.ReportJobPayload) error {
		log.Info("Processing report job",
			zap.String("report_type", payload.ReportType),
			zap.String("format", payload.Format),
		)
		// Implement report generation logic
		return nil
	})

	handler.Register(registry, "sync", func(ctx context.Context, payload handler.SyncJobPayload) error {
		log.Info("Processing sync job",
			zap.String("source", payload.Source),
			zap.String("destination", payload.Destination),
		)
		// Implement sync logic
		return nil
	})

	log.Info("Registered job handlers")
}

func registerScheduledJobs(sched *scheduler.Scheduler, log *zap.Logger) {
	// Daily cleanup - singleton to prevent overlap
	sched.RegisterJob(scheduler.ScheduledJob{
		Name:     "daily-token-cleanup",
		Schedule: scheduler.DailyMidnight,
		JobType:  "cleanup",
		Payload: handler.CleanupJobPayload{
			Type:      "expired_tokens",
			OlderThan: 30,
		},
		Priority:  jobs.PriorityLow,
		Singleton: true, // Only one instance can run at a time
	})

	// Hourly stats sync
	sched.RegisterJob(scheduler.ScheduledJob{
		Name:     "hourly-stats-sync",
		Schedule: scheduler.EveryHour,
		JobType:  "sync",
		Payload: handler.SyncJobPayload{
			Source:      "database",
			Destination: "cache",
			EntityType:  "stats",
		},
		Priority:  jobs.PriorityNormal,
		Singleton: false,
	})

	log.Info("Registered scheduled jobs")
}
