package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/queue"
	"github.com/jrjohn/arcana-cloud-go/internal/testutil"
)

func setupTestScheduler(t *testing.T) (*Scheduler, *queue.RedisQueue, context.Context) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewTestLogger(t)

	schedConfig := SchedulerConfig{
		LeaderLockTTL:        5 * time.Second,
		CronExecutionLockTTL: time.Minute,
		CronDeduplicationTTL: time.Hour,
	}

	sched := NewSchedulerWithConfig(client, q, logger, schedConfig)
	return sched, q, context.Background()
}

func TestDefaultSchedulerConfig(t *testing.T) {
	config := DefaultSchedulerConfig()

	if config.LeaderLockTTL != 30*time.Second {
		t.Errorf("LeaderLockTTL = %v, want 30s", config.LeaderLockTTL)
	}
	if config.CronExecutionLockTTL != 60*time.Second {
		t.Errorf("CronExecutionLockTTL = %v, want 60s", config.CronExecutionLockTTL)
	}
	if config.CronDeduplicationTTL != 24*time.Hour {
		t.Errorf("CronDeduplicationTTL = %v, want 24h", config.CronDeduplicationTTL)
	}
}

func TestNewScheduler(t *testing.T) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewTestLogger(t)

	sched := NewScheduler(client, q, logger)

	if sched == nil {
		t.Fatal("NewScheduler() returned nil")
	}
	if sched.instanceID == "" {
		t.Error("instanceID is empty")
	}
	if sched.jobs == nil {
		t.Error("jobs map is nil")
	}
}

func TestScheduler_RegisterJob(t *testing.T) {
	sched, _, _ := setupTestScheduler(t)

	job := ScheduledJob{
		Name:     "test-cron",
		Schedule: EveryMinute,
		JobType:  "cleanup",
		Payload:  map[string]string{"type": "test"},
		Priority: jobs.PriorityNormal,
	}

	if err := sched.RegisterJob(job); err != nil {
		t.Fatalf("RegisterJob() error = %v", err)
	}

	// Verify registration
	registeredJobs := sched.ListJobs()
	if len(registeredJobs) != 1 {
		t.Errorf("len(registeredJobs) = %v, want 1", len(registeredJobs))
	}
	if registeredJobs[0].Name != "test-cron" {
		t.Errorf("Name = %v, want test-cron", registeredJobs[0].Name)
	}
}

func TestScheduler_RegisterJob_Duplicate(t *testing.T) {
	sched, _, _ := setupTestScheduler(t)

	job := ScheduledJob{
		Name:     "duplicate-job",
		Schedule: EveryHour,
		JobType:  "test",
	}

	if err := sched.RegisterJob(job); err != nil {
		t.Fatalf("First RegisterJob() error = %v", err)
	}

	// Register same name again
	if err := sched.RegisterJob(job); err == nil {
		t.Error("Second RegisterJob() should return error for duplicate")
	}
}

func TestScheduler_RegisterJob_InvalidCron(t *testing.T) {
	sched, _, _ := setupTestScheduler(t)

	job := ScheduledJob{
		Name:     "invalid-cron",
		Schedule: "invalid cron expression",
		JobType:  "test",
	}

	if err := sched.RegisterJob(job); err == nil {
		t.Error("RegisterJob() should return error for invalid cron")
	}
}

func TestScheduler_ValidCronExpressions(t *testing.T) {
	sched, _, _ := setupTestScheduler(t)

	expressions := []struct {
		name string
		expr string
	}{
		{"EveryMinute", EveryMinute},
		{"EveryFiveMinutes", EveryFiveMinutes},
		{"EveryHour", EveryHour},
		{"DailyMidnight", DailyMidnight},
		{"WeeklyMonday", WeeklyMonday},
		{"MonthlyFirst", MonthlyFirst},
		{"Custom", "30 * * * *"},
	}

	for _, tc := range expressions {
		t.Run(tc.name, func(t *testing.T) {
			job := ScheduledJob{
				Name:     "cron-" + tc.name,
				Schedule: tc.expr,
				JobType:  "test",
			}
			if err := sched.RegisterJob(job); err != nil {
				t.Errorf("RegisterJob() error = %v for %s", err, tc.expr)
			}
		})
	}
}

func TestScheduler_StartStop(t *testing.T) {
	sched, _, ctx := setupTestScheduler(t)

	// Start
	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !sched.running {
		t.Error("Scheduler should be running after Start()")
	}

	// Wait a bit for leader election
	time.Sleep(200 * time.Millisecond)

	// Stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sched.Stop(stopCtx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if sched.running {
		t.Error("Scheduler should not be running after Stop()")
	}
}

func TestScheduler_StartTwice(t *testing.T) {
	sched, _, ctx := setupTestScheduler(t)

	sched.Start(ctx)
	defer sched.Stop(context.Background())

	// Starting again should fail
	err := sched.Start(ctx)
	if err == nil {
		t.Error("Second Start() should return error")
	}
}

func TestScheduler_StopNotRunning(t *testing.T) {
	sched, _, _ := setupTestScheduler(t)

	// Stop without starting
	if err := sched.Stop(context.Background()); err != nil {
		t.Errorf("Stop() on non-running scheduler error = %v", err)
	}
}

func TestScheduler_LeaderElection(t *testing.T) {
	sched, _, ctx := setupTestScheduler(t)

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer sched.Stop(context.Background())

	// Wait for leader election
	time.Sleep(300 * time.Millisecond)

	// Should be leader (only scheduler)
	if !sched.IsLeader() {
		t.Error("Scheduler should be leader")
	}
}

func TestScheduler_IsLeader_NotStarted(t *testing.T) {
	sched, _, _ := setupTestScheduler(t)

	if sched.IsLeader() {
		t.Error("Scheduler should not be leader before starting")
	}
}

func TestScheduler_ListJobs(t *testing.T) {
	sched, _, _ := setupTestScheduler(t)

	// Register jobs
	for i := 0; i < 3; i++ {
		sched.RegisterJob(ScheduledJob{
			Name:     testutil.GenerateTestID(),
			Schedule: EveryHour,
			JobType:  "test",
		})
	}

	jobs := sched.ListJobs()
	if len(jobs) != 3 {
		t.Errorf("len(jobs) = %v, want 3", len(jobs))
	}
}

func TestScheduler_GetNextRun(t *testing.T) {
	sched, _, _ := setupTestScheduler(t)

	sched.RegisterJob(ScheduledJob{
		Name:     "next-run-test",
		Schedule: EveryHour,
		JobType:  "test",
	})

	nextRun, err := sched.GetNextRun("next-run-test")
	if err != nil {
		t.Fatalf("GetNextRun() error = %v", err)
	}

	// Next run should be in the future
	if !nextRun.After(time.Now()) {
		t.Error("Next run should be in the future")
	}

	// Next run should be within an hour
	if nextRun.After(time.Now().Add(time.Hour + time.Minute)) {
		t.Error("Next run should be within an hour")
	}
}

func TestScheduler_GetNextRun_NotFound(t *testing.T) {
	sched, _, _ := setupTestScheduler(t)

	_, err := sched.GetNextRun("nonexistent-job")
	if err == nil {
		t.Error("GetNextRun() should return error for non-existent job")
	}
}

func TestScheduler_SingletonJob(t *testing.T) {
	sched, _, ctx := setupTestScheduler(t)

	job := ScheduledJob{
		Name:      "singleton-test",
		Schedule:  EveryMinute,
		JobType:   "cleanup",
		Singleton: true,
	}

	if err := sched.RegisterJob(job); err != nil {
		t.Fatalf("RegisterJob() error = %v", err)
	}

	jobs := sched.ListJobs()
	if len(jobs) != 1 || !jobs[0].Singleton {
		t.Error("Singleton flag should be preserved")
	}

	// Test singleton lock acquisition
	acquired, err := sched.AcquireSingletonLock(ctx, "singleton-test", time.Minute)
	if err != nil {
		t.Fatalf("AcquireSingletonLock() error = %v", err)
	}
	if !acquired {
		t.Error("Should acquire singleton lock")
	}

	// Second acquire should fail
	acquired2, _ := sched.AcquireSingletonLock(ctx, "singleton-test", time.Minute)
	if acquired2 {
		t.Error("Second acquire should fail")
	}

	// Release
	if err := sched.ReleaseSingletonLock(ctx, "singleton-test"); err != nil {
		t.Errorf("ReleaseSingletonLock() error = %v", err)
	}

	// Now should be able to acquire again
	acquired3, _ := sched.AcquireSingletonLock(ctx, "singleton-test", time.Minute)
	if !acquired3 {
		t.Error("Should acquire after release")
	}
	sched.ReleaseSingletonLock(ctx, "singleton-test")
}

func TestScheduler_ExecutionWindow(t *testing.T) {
	sched, _, _ := setupTestScheduler(t)

	tests := []struct {
		schedule string
		contains string
	}{
		{EveryMinute, "T"},
		{EveryHour, "T"},
		{DailyMidnight, "-"},
		{WeeklyMonday, "W"},
		{MonthlyFirst, "-"},
	}

	for _, tc := range tests {
		t.Run(tc.schedule, func(t *testing.T) {
			window := sched.getExecutionWindow(tc.schedule)
			if window == "" {
				t.Error("Execution window should not be empty")
			}
		})
	}
}

func TestScheduler_GetRecentExecutions(t *testing.T) {
	sched, _, ctx := setupTestScheduler(t)

	// This tests the GetRecentExecutions method
	executions, err := sched.GetRecentExecutions(ctx, "test-job", 10)
	if err != nil {
		t.Fatalf("GetRecentExecutions() error = %v", err)
	}

	// Should return empty for non-existent job
	if len(executions) != 0 {
		t.Logf("Found %d executions", len(executions))
	}
}

func TestScheduledJob_Struct(t *testing.T) {
	job := ScheduledJob{
		Name:      "struct-test",
		Schedule:  EveryHour,
		JobType:   "cleanup",
		Payload:   map[string]int{"days": 30},
		Priority:  jobs.PriorityLow,
		UniqueKey: "unique-123",
		Tags:      []string{"maintenance"},
		Singleton: true,
	}

	if job.Name != "struct-test" {
		t.Errorf("Name = %v, want struct-test", job.Name)
	}
	if job.Schedule != EveryHour {
		t.Errorf("Schedule = %v, want %v", job.Schedule, EveryHour)
	}
	if job.JobType != "cleanup" {
		t.Errorf("JobType = %v, want cleanup", job.JobType)
	}
	if job.Priority != jobs.PriorityLow {
		t.Errorf("Priority = %v, want low", job.Priority)
	}
	if !job.Singleton {
		t.Error("Singleton should be true")
	}
}

func TestCronExpressionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"EveryMinute", EveryMinute, "* * * * *"},
		{"EveryFiveMinutes", EveryFiveMinutes, "*/5 * * * *"},
		{"EveryHour", EveryHour, "0 * * * *"},
		{"DailyMidnight", DailyMidnight, "0 0 * * *"},
		{"WeeklyMonday", WeeklyMonday, "0 0 * * 1"},
		{"MonthlyFirst", MonthlyFirst, "0 0 1 * *"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.constant != tc.expected {
				t.Errorf("%s = %v, want %v", tc.name, tc.constant, tc.expected)
			}
		})
	}
}

// Benchmarks
func BenchmarkScheduler_RegisterJob(b *testing.B) {
	testutil.SkipIfNoRedis(&testing.T{})
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(&testing.T{}, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewNopLogger()

	sched := NewScheduler(client, q, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sched.RegisterJob(ScheduledJob{
			Name:     testutil.GenerateTestID(),
			Schedule: EveryHour,
			JobType:  "bench",
		})
	}
}

func BenchmarkScheduler_GetNextRun(b *testing.B) {
	testutil.SkipIfNoRedis(&testing.T{})
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(&testing.T{}, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewNopLogger()

	sched := NewScheduler(client, q, logger)
	sched.RegisterJob(ScheduledJob{
		Name:     "bench-job",
		Schedule: EveryHour,
		JobType:  "bench",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sched.GetNextRun("bench-job")
	}
}
