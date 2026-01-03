package jobs

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("NewMetrics() returned nil")
	}
	if m.JobDurations == nil {
		t.Error("JobDurations is nil")
	}
}

func TestMetrics_RecordJobEnqueued(t *testing.T) {
	m := NewMetrics()

	m.RecordJobEnqueued(PriorityNormal)

	if m.JobsEnqueued.Load() != 1 {
		t.Errorf("JobsEnqueued = %v, want 1", m.JobsEnqueued.Load())
	}
	if m.JobsPending.Load() != 1 {
		t.Errorf("JobsPending = %v, want 1", m.JobsPending.Load())
	}

	// Record more
	m.RecordJobEnqueued(PriorityHigh)
	m.RecordJobEnqueued(PriorityLow)

	if m.JobsEnqueued.Load() != 3 {
		t.Errorf("JobsEnqueued = %v, want 3", m.JobsEnqueued.Load())
	}
	if m.JobsPending.Load() != 3 {
		t.Errorf("JobsPending = %v, want 3", m.JobsPending.Load())
	}
}

func TestMetrics_RecordJobStarted(t *testing.T) {
	m := NewMetrics()

	// First enqueue some jobs
	m.RecordJobEnqueued(PriorityNormal)
	m.RecordJobEnqueued(PriorityNormal)

	// Then start one
	m.RecordJobStarted()

	if m.JobsPending.Load() != 1 {
		t.Errorf("JobsPending = %v, want 1", m.JobsPending.Load())
	}
	if m.JobsRunning.Load() != 1 {
		t.Errorf("JobsRunning = %v, want 1", m.JobsRunning.Load())
	}
	if m.WorkersActive.Load() != 1 {
		t.Errorf("WorkersActive = %v, want 1", m.WorkersActive.Load())
	}
}

func TestMetrics_RecordJobCompleted(t *testing.T) {
	m := NewMetrics()

	// Setup: enqueue and start
	m.RecordJobEnqueued(PriorityNormal)
	m.RecordJobStarted()

	// Complete
	duration := 100 * time.Millisecond
	m.RecordJobCompleted(duration)

	if m.JobsCompleted.Load() != 1 {
		t.Errorf("JobsCompleted = %v, want 1", m.JobsCompleted.Load())
	}
	if m.JobsRunning.Load() != 0 {
		t.Errorf("JobsRunning = %v, want 0", m.JobsRunning.Load())
	}
	if m.WorkersActive.Load() != 0 {
		t.Errorf("WorkersActive = %v, want 0", m.WorkersActive.Load())
	}
	if len(m.JobDurations) != 1 {
		t.Errorf("len(JobDurations) = %v, want 1", len(m.JobDurations))
	}
	if m.JobDurations[0] != duration {
		t.Errorf("JobDurations[0] = %v, want %v", m.JobDurations[0], duration)
	}
}

func TestMetrics_RecordJobFailed_WithRetry(t *testing.T) {
	m := NewMetrics()

	// Setup
	m.RecordJobEnqueued(PriorityNormal)
	m.RecordJobStarted()

	// Fail with retry
	m.RecordJobFailed(true)

	if m.JobsFailed.Load() != 1 {
		t.Errorf("JobsFailed = %v, want 1", m.JobsFailed.Load())
	}
	if m.JobsRetried.Load() != 1 {
		t.Errorf("JobsRetried = %v, want 1", m.JobsRetried.Load())
	}
	if m.JobsRunning.Load() != 0 {
		t.Errorf("JobsRunning = %v, want 0", m.JobsRunning.Load())
	}
	if m.WorkersActive.Load() != 0 {
		t.Errorf("WorkersActive = %v, want 0", m.WorkersActive.Load())
	}
}

func TestMetrics_RecordJobFailed_NoRetry(t *testing.T) {
	m := NewMetrics()

	// Setup
	m.RecordJobEnqueued(PriorityNormal)
	m.RecordJobStarted()

	// Fail without retry
	m.RecordJobFailed(false)

	if m.JobsFailed.Load() != 1 {
		t.Errorf("JobsFailed = %v, want 1", m.JobsFailed.Load())
	}
	if m.JobsRetried.Load() != 0 {
		t.Errorf("JobsRetried = %v, want 0", m.JobsRetried.Load())
	}
}

func TestMetrics_RecordJobDead(t *testing.T) {
	m := NewMetrics()

	m.RecordJobDead()
	m.RecordJobDead()

	if m.JobsDead.Load() != 2 {
		t.Errorf("JobsDead = %v, want 2", m.JobsDead.Load())
	}
}

func TestMetrics_PrometheusHandler(t *testing.T) {
	m := NewMetrics()

	// Record some metrics
	m.RecordJobEnqueued(PriorityNormal)
	m.RecordJobEnqueued(PriorityHigh)
	m.RecordJobStarted()
	m.RecordJobCompleted(100 * time.Millisecond)
	m.RecordJobEnqueued(PriorityLow)
	m.RecordJobStarted()
	m.RecordJobFailed(true)
	m.RecordJobDead()

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	// Call handler
	handler := m.PrometheusHandler()
	handler(w, req)

	// Check response
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %v, want 200", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/plain; version=0.0.4" {
		t.Errorf("Content-Type = %v, want text/plain; version=0.0.4", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check for expected metrics
	expectedMetrics := []string{
		"arcana_jobs_enqueued_total",
		"arcana_jobs_completed_total",
		"arcana_jobs_failed_total",
		"arcana_jobs_retried_total",
		"arcana_jobs_dead_total",
		"arcana_jobs_pending",
		"arcana_jobs_running",
		"arcana_workers_active",
		"arcana_job_duration_avg_ms",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(bodyStr, metric) {
			t.Errorf("Response missing metric: %s", metric)
		}
	}

	// Check for HELP and TYPE annotations
	if !strings.Contains(bodyStr, "# HELP") {
		t.Error("Response missing # HELP annotations")
	}
	if !strings.Contains(bodyStr, "# TYPE") {
		t.Error("Response missing # TYPE annotations")
	}
}

func TestMetrics_PrometheusHandler_NoDurations(t *testing.T) {
	m := NewMetrics()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler := m.PrometheusHandler()
	handler(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	bodyStr := string(body)

	// Should not have avg duration when no jobs completed
	if strings.Contains(bodyStr, "arcana_job_duration_avg_ms") {
		t.Error("Should not include avg duration when no jobs completed")
	}
}

func TestMetrics_GetHealthCheck_Healthy(t *testing.T) {
	m := NewMetrics()
	m.WorkersActive.Store(5)
	m.JobsPending.Store(100)

	health := m.GetHealthCheck(true)

	if health.Status != "healthy" {
		t.Errorf("Status = %v, want healthy", health.Status)
	}
	if health.WorkersActive != 5 {
		t.Errorf("WorkersActive = %v, want 5", health.WorkersActive)
	}
	if health.JobsPending != 100 {
		t.Errorf("JobsPending = %v, want 100", health.JobsPending)
	}
	if !health.IsLeader {
		t.Error("IsLeader = false, want true")
	}
}

func TestMetrics_GetHealthCheck_Degraded(t *testing.T) {
	m := NewMetrics()
	m.JobsPending.Store(1001) // More than 1000 pending

	health := m.GetHealthCheck(false)

	if health.Status != "degraded" {
		t.Errorf("Status = %v, want degraded", health.Status)
	}
	if health.IsLeader {
		t.Error("IsLeader = true, want false")
	}
}

func TestMetrics_GetHealthCheck_Boundary(t *testing.T) {
	m := NewMetrics()

	// Exactly 1000 should be healthy
	m.JobsPending.Store(1000)
	health := m.GetHealthCheck(true)
	if health.Status != "healthy" {
		t.Errorf("Status at 1000 = %v, want healthy", health.Status)
	}

	// 1001 should be degraded
	m.JobsPending.Store(1001)
	health = m.GetHealthCheck(true)
	if health.Status != "degraded" {
		t.Errorf("Status at 1001 = %v, want degraded", health.Status)
	}
}

func TestHealthCheck_Struct(t *testing.T) {
	hc := HealthCheck{
		Status:        "healthy",
		WorkersActive: 10,
		JobsPending:   50,
		IsLeader:      true,
	}

	if hc.Status != "healthy" {
		t.Errorf("Status = %v, want healthy", hc.Status)
	}
	if hc.WorkersActive != 10 {
		t.Errorf("WorkersActive = %v, want 10", hc.WorkersActive)
	}
	if hc.JobsPending != 50 {
		t.Errorf("JobsPending = %v, want 50", hc.JobsPending)
	}
	if !hc.IsLeader {
		t.Error("IsLeader = false, want true")
	}
}

func TestGlobalMetrics(t *testing.T) {
	// GlobalMetrics should be initialized
	if GlobalMetrics == nil {
		t.Fatal("GlobalMetrics is nil")
	}

	// Test that it works
	initialEnqueued := GlobalMetrics.JobsEnqueued.Load()
	GlobalMetrics.RecordJobEnqueued(PriorityNormal)
	if GlobalMetrics.JobsEnqueued.Load() != initialEnqueued+1 {
		t.Error("GlobalMetrics.RecordJobEnqueued() failed")
	}
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
	m := NewMetrics()
	done := make(chan bool)

	// Concurrent enqueue
	for i := 0; i < 100; i++ {
		go func() {
			m.RecordJobEnqueued(PriorityNormal)
			done <- true
		}()
	}

	// Concurrent start
	for i := 0; i < 50; i++ {
		go func() {
			m.RecordJobStarted()
			done <- true
		}()
	}

	// Concurrent complete
	for i := 0; i < 30; i++ {
		go func() {
			m.RecordJobCompleted(time.Millisecond)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 180; i++ {
		<-done
	}

	// Verify counters
	if m.JobsEnqueued.Load() != 100 {
		t.Errorf("JobsEnqueued = %v, want 100", m.JobsEnqueued.Load())
	}
	if m.JobsCompleted.Load() != 30 {
		t.Errorf("JobsCompleted = %v, want 30", m.JobsCompleted.Load())
	}
}

// Benchmarks
func BenchmarkMetrics_RecordJobEnqueued(b *testing.B) {
	m := NewMetrics()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordJobEnqueued(PriorityNormal)
	}
}

func BenchmarkMetrics_RecordJobCompleted(b *testing.B) {
	m := NewMetrics()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.RecordJobCompleted(time.Millisecond)
	}
}

func BenchmarkMetrics_GetHealthCheck(b *testing.B) {
	m := NewMetrics()
	m.WorkersActive.Store(10)
	m.JobsPending.Store(500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.GetHealthCheck(true)
	}
}

func BenchmarkMetrics_PrometheusHandler(b *testing.B) {
	m := NewMetrics()
	m.RecordJobEnqueued(PriorityNormal)
	m.RecordJobStarted()
	m.RecordJobCompleted(time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	handler := m.PrometheusHandler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler(w, req)
	}
}
