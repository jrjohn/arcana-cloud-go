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
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.New(logger.Config{
		Level:       "info",
		Development: cfg.App.Debug,
		Encoding:    "json",
	})
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("Starting Arcana Cloud Worker",
		zap.String("version", cfg.App.Version),
		zap.String("environment", cfg.App.Environment),
	)

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	log.Info("Connected to Redis", zap.String("host", cfg.Redis.Host))

	// Initialize job queue
	jobQueue := queue.NewRedisQueue(redisClient)

	// Initialize lock manager for distributed locking
	lockConfig := lock.DefaultLockManagerConfig()
	lockManager := lock.NewLockManager(redisClient, lockConfig)
	log.Info("Lock manager initialized",
		zap.String("worker_id", lockManager.GetWorkerID()),
		zap.Duration("lock_ttl", lockConfig.LockTTL),
		zap.Duration("idempotency_ttl", lockConfig.IdempotencyTTL),
	)

	// Initialize worker pool
	workerConfig := worker.DefaultWorkerPoolConfig()
	if concurrency := os.Getenv("ARCANA_WORKER_CONCURRENCY"); concurrency != "" {
		fmt.Sscanf(concurrency, "%d", &workerConfig.Concurrency)
	}

	// Configure locking/idempotency from environment
	if os.Getenv("ARCANA_WORKER_DISABLE_LOCKING") == "true" {
		workerConfig.EnableLocking = false
	}
	if os.Getenv("ARCANA_WORKER_DISABLE_IDEMPOTENCY") == "true" {
		workerConfig.EnableIdempotency = false
	}

	pool := worker.NewWorkerPool(jobQueue, log, workerConfig)
	pool.SetLockManager(lockManager)

	// Initialize handler registry and register handlers
	registry := handler.NewRegistry(pool, log)
	registerHandlers(registry, log)

	// Initialize scheduler
	schedConfig := scheduler.DefaultSchedulerConfig()
	sched := scheduler.NewSchedulerWithConfig(redisClient, jobQueue, log, schedConfig)
	registerScheduledJobs(sched, log)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker pool
	if err := pool.Start(ctx); err != nil {
		log.Fatal("Failed to start worker pool", zap.Error(err))
	}

	// Start scheduler
	if err := sched.Start(ctx); err != nil {
		log.Fatal("Failed to start scheduler", zap.Error(err))
	}

	// Start metrics server
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", jobs.GlobalMetrics.PrometheusHandler())
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			health := jobs.GlobalMetrics.GetHealthCheck(sched.IsLeader())
			w.Header().Set("Content-Type", "application/json")
			if health.Status == "healthy" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			fmt.Fprintf(w, `{"status":"%s","workers_active":%d,"jobs_pending":%d,"is_leader":%t,"worker_id":"%s"}`,
				health.Status, health.WorkersActive, health.JobsPending, health.IsLeader, lockManager.GetWorkerID())
		})
		mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ready"}`))
		})

		// Running jobs endpoint
		mux.HandleFunc("/running", func(w http.ResponseWriter, r *http.Request) {
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
		})

		metricsPort := os.Getenv("METRICS_PORT")
		if metricsPort == "" {
			metricsPort = "9100"
		}

		log.Info("Starting metrics server", zap.String("port", metricsPort))
		if err := http.ListenAndServe(":"+metricsPort, mux); err != nil {
			log.Error("Metrics server error", zap.Error(err))
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutdown signal received, stopping workers...")
	cancel()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), workerConfig.ShutdownTimeout)
	defer shutdownCancel()

	if err := sched.Stop(shutdownCtx); err != nil {
		log.Error("Error stopping scheduler", zap.Error(err))
	}

	if err := pool.Stop(shutdownCtx); err != nil {
		log.Error("Error stopping worker pool", zap.Error(err))
	}

	// Release all locks held by this worker
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
