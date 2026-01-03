package jobs

import (
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics collects job system metrics for Prometheus
type Metrics struct {
	// Counters
	JobsEnqueued   atomic.Int64
	JobsCompleted  atomic.Int64
	JobsFailed     atomic.Int64
	JobsRetried    atomic.Int64
	JobsDead       atomic.Int64

	// Gauges
	JobsPending    atomic.Int64
	JobsRunning    atomic.Int64
	WorkersActive  atomic.Int64

	// Histograms (simplified - in production use prometheus client)
	JobDurations []time.Duration
	durationMu   sync.RWMutex
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		JobDurations: make([]time.Duration, 0),
	}
}

// Global metrics instance
var GlobalMetrics = NewMetrics()

// RecordJobEnqueued records a job being enqueued
func (m *Metrics) RecordJobEnqueued(priority Priority) {
	m.JobsEnqueued.Add(1)
	m.JobsPending.Add(1)
}

// RecordJobStarted records a job starting execution
func (m *Metrics) RecordJobStarted() {
	m.JobsPending.Add(-1)
	m.JobsRunning.Add(1)
	m.WorkersActive.Add(1)
}

// RecordJobCompleted records a job completing successfully
func (m *Metrics) RecordJobCompleted(duration time.Duration) {
	m.JobsCompleted.Add(1)
	m.JobsRunning.Add(-1)
	m.WorkersActive.Add(-1)
	m.durationMu.Lock()
	m.JobDurations = append(m.JobDurations, duration)
	m.durationMu.Unlock()
}

// RecordJobFailed records a job failure
func (m *Metrics) RecordJobFailed(willRetry bool) {
	m.JobsFailed.Add(1)
	m.JobsRunning.Add(-1)
	m.WorkersActive.Add(-1)
	if willRetry {
		m.JobsRetried.Add(1)
	}
}

// RecordJobDead records a job moved to DLQ
func (m *Metrics) RecordJobDead() {
	m.JobsDead.Add(1)
}

// PrometheusHandler returns an HTTP handler for Prometheus metrics
func (m *Metrics) PrometheusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		// Write metrics in Prometheus format
		writeMetric(w, "arcana_jobs_enqueued_total", "counter", "Total jobs enqueued", m.JobsEnqueued.Load())
		writeMetric(w, "arcana_jobs_completed_total", "counter", "Total jobs completed", m.JobsCompleted.Load())
		writeMetric(w, "arcana_jobs_failed_total", "counter", "Total jobs failed", m.JobsFailed.Load())
		writeMetric(w, "arcana_jobs_retried_total", "counter", "Total jobs retried", m.JobsRetried.Load())
		writeMetric(w, "arcana_jobs_dead_total", "counter", "Total jobs moved to DLQ", m.JobsDead.Load())
		writeMetric(w, "arcana_jobs_pending", "gauge", "Current pending jobs", m.JobsPending.Load())
		writeMetric(w, "arcana_jobs_running", "gauge", "Current running jobs", m.JobsRunning.Load())
		writeMetric(w, "arcana_workers_active", "gauge", "Active worker count", m.WorkersActive.Load())

		// Calculate average duration
		m.durationMu.RLock()
		durations := make([]time.Duration, len(m.JobDurations))
		copy(durations, m.JobDurations)
		m.durationMu.RUnlock()

		if len(durations) > 0 {
			var total time.Duration
			for _, d := range durations {
				total += d
			}
			avg := float64(total.Milliseconds()) / float64(len(durations))
			writeMetricFloat(w, "arcana_job_duration_avg_ms", "gauge", "Average job duration in ms", avg)
		}
	}
}

func writeMetric(w http.ResponseWriter, name, metricType, help string, value int64) {
	w.Write([]byte("# HELP " + name + " " + help + "\n"))
	w.Write([]byte("# TYPE " + name + " " + metricType + "\n"))
	w.Write([]byte(name + " " + strconv.FormatInt(value, 10) + "\n"))
}

func writeMetricFloat(w http.ResponseWriter, name, metricType, help string, value float64) {
	w.Write([]byte("# HELP " + name + " " + help + "\n"))
	w.Write([]byte("# TYPE " + name + " " + metricType + "\n"))
	w.Write([]byte(name + " " + strconv.FormatFloat(value, 'f', 2, 64) + "\n"))
}

// HealthCheck returns the health status of the job system
type HealthCheck struct {
	Status        string `json:"status"`
	WorkersActive int64  `json:"workers_active"`
	JobsPending   int64  `json:"jobs_pending"`
	IsLeader      bool   `json:"is_leader"`
}

// GetHealthCheck returns current health status
func (m *Metrics) GetHealthCheck(isLeader bool) HealthCheck {
	status := "healthy"
	pending := m.JobsPending.Load()
	if pending > 1000 {
		status = "degraded"
	}

	return HealthCheck{
		Status:        status,
		WorkersActive: m.WorkersActive.Load(),
		JobsPending:   pending,
		IsLeader:      isLeader,
	}
}
