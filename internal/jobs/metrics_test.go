package jobs

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewMetrics creates a new metrics instance
func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	assert.NotNil(t, m)
	assert.NotNil(t, m.JobDurations)
	assert.Empty(t, m.JobDurations)
}

// TestMetrics_RecordJobEnqueued increments counters
func TestMetrics_RecordJobEnqueued(t *testing.T) {
	m := NewMetrics()

	m.RecordJobEnqueued(PriorityNormal)
	assert.Equal(t, int64(1), m.JobsEnqueued.Load())
	assert.Equal(t, int64(1), m.JobsPending.Load())

	m.RecordJobEnqueued(PriorityHigh)
	assert.Equal(t, int64(2), m.JobsEnqueued.Load())
	assert.Equal(t, int64(2), m.JobsPending.Load())
}

// TestMetrics_RecordJobStarted moves from pending to running
func TestMetrics_RecordJobStarted(t *testing.T) {
	m := NewMetrics()
	m.RecordJobEnqueued(PriorityNormal)
	m.RecordJobEnqueued(PriorityNormal)

	m.RecordJobStarted()
	assert.Equal(t, int64(1), m.JobsPending.Load())
	assert.Equal(t, int64(1), m.JobsRunning.Load())
	assert.Equal(t, int64(1), m.WorkersActive.Load())
}

// TestMetrics_RecordJobCompleted increments completed and records duration
func TestMetrics_RecordJobCompleted(t *testing.T) {
	m := NewMetrics()
	m.RecordJobEnqueued(PriorityNormal)
	m.RecordJobStarted()

	m.RecordJobCompleted(150 * time.Millisecond)
	assert.Equal(t, int64(1), m.JobsCompleted.Load())
	assert.Equal(t, int64(0), m.JobsRunning.Load())
	assert.Equal(t, int64(0), m.WorkersActive.Load())

	m.durationMu.RLock()
	assert.Len(t, m.JobDurations, 1)
	assert.Equal(t, 150*time.Millisecond, m.JobDurations[0])
	m.durationMu.RUnlock()
}

// TestMetrics_RecordJobCompleted_Multiple accumulates durations
func TestMetrics_RecordJobCompleted_Multiple(t *testing.T) {
	m := NewMetrics()
	durations := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond}

	for _, d := range durations {
		m.RecordJobEnqueued(PriorityNormal)
		m.RecordJobStarted()
		m.RecordJobCompleted(d)
	}

	assert.Equal(t, int64(3), m.JobsCompleted.Load())

	m.durationMu.RLock()
	assert.Len(t, m.JobDurations, 3)
	m.durationMu.RUnlock()
}

// TestMetrics_RecordJobFailed_WithRetry increments failed and retried
func TestMetrics_RecordJobFailed_WithRetry(t *testing.T) {
	m := NewMetrics()
	m.RecordJobEnqueued(PriorityNormal)
	m.RecordJobStarted()

	m.RecordJobFailed(true)
	assert.Equal(t, int64(1), m.JobsFailed.Load())
	assert.Equal(t, int64(1), m.JobsRetried.Load())
	assert.Equal(t, int64(0), m.JobsRunning.Load())
	assert.Equal(t, int64(0), m.WorkersActive.Load())
}

// TestMetrics_RecordJobFailed_NoRetry increments failed but not retried
func TestMetrics_RecordJobFailed_NoRetry(t *testing.T) {
	m := NewMetrics()
	m.RecordJobEnqueued(PriorityNormal)
	m.RecordJobStarted()

	m.RecordJobFailed(false)
	assert.Equal(t, int64(1), m.JobsFailed.Load())
	assert.Equal(t, int64(0), m.JobsRetried.Load())
}

// TestMetrics_RecordJobDead increments DLQ counter
func TestMetrics_RecordJobDead(t *testing.T) {
	m := NewMetrics()

	m.RecordJobDead()
	assert.Equal(t, int64(1), m.JobsDead.Load())

	m.RecordJobDead()
	assert.Equal(t, int64(2), m.JobsDead.Load())
}

// TestMetrics_PrometheusHandler returns valid Prometheus metrics
func TestMetrics_PrometheusHandler(t *testing.T) {
	m := NewMetrics()
	m.RecordJobEnqueued(PriorityNormal)
	m.RecordJobEnqueued(PriorityHigh)
	m.RecordJobStarted()
	m.RecordJobCompleted(200 * time.Millisecond)
	m.RecordJobDead()

	handler := m.PrometheusHandler()
	assert.NotNil(t, handler)

	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	assert.Equal(t, 200, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "arcana_jobs_enqueued_total")
	assert.Contains(t, body, "arcana_jobs_completed_total")
	assert.Contains(t, body, "arcana_jobs_failed_total")
	assert.Contains(t, body, "arcana_jobs_dead_total")
	assert.Contains(t, body, "arcana_jobs_pending")
	assert.Contains(t, body, "arcana_jobs_running")
	assert.Contains(t, body, "arcana_workers_active")
	assert.Contains(t, body, "arcana_job_duration_avg_ms")
}

// TestMetrics_PrometheusHandler_NoDurations omits avg when no durations
func TestMetrics_PrometheusHandler_NoDurations(t *testing.T) {
	m := NewMetrics()

	handler := m.PrometheusHandler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	assert.Equal(t, 200, rr.Code)
	body := rr.Body.String()
	// No avg duration when no jobs completed
	assert.NotContains(t, body, "arcana_job_duration_avg_ms")
}

// TestMetrics_GetHealthCheck_Healthy returns healthy status
func TestMetrics_GetHealthCheck_Healthy(t *testing.T) {
	m := NewMetrics()

	hc := m.GetHealthCheck(true)
	assert.Equal(t, "healthy", hc.Status)
	assert.Equal(t, int64(0), hc.WorkersActive)
	assert.Equal(t, int64(0), hc.JobsPending)
	assert.True(t, hc.IsLeader)
}

// TestMetrics_GetHealthCheck_Degraded returns degraded when pending > 1000
func TestMetrics_GetHealthCheck_Degraded(t *testing.T) {
	m := NewMetrics()

	// Set pending to > 1000
	for i := 0; i < 1001; i++ {
		m.JobsPending.Add(1)
	}

	hc := m.GetHealthCheck(false)
	assert.Equal(t, "degraded", hc.Status)
	assert.False(t, hc.IsLeader)
}

// TestMetrics_GetHealthCheck_NotLeader returns non-leader status
func TestMetrics_GetHealthCheck_NotLeader(t *testing.T) {
	m := NewMetrics()

	hc := m.GetHealthCheck(false)
	assert.Equal(t, "healthy", hc.Status)
	assert.False(t, hc.IsLeader)
}

// TestGlobalMetrics is not nil
func TestGlobalMetrics(t *testing.T) {
	assert.NotNil(t, GlobalMetrics)
}
