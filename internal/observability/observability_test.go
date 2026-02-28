package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ── MetricsConfig ────────────────────────────────────────────────────────────

func TestDefaultMetricsConfig(t *testing.T) {
	cfg := DefaultMetricsConfig()
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "arcana-cloud-go", cfg.ServiceName)
	assert.Equal(t, "/metrics", cfg.PrometheusPath)
}

// ── MetricsProvider (disabled) ───────────────────────────────────────────────

func TestNewMetricsProvider_Disabled(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "test-svc"}
	logger := zap.NewNop()
	mp, err := NewMetricsProvider(cfg, logger)
	require.NoError(t, err)
	assert.NotNil(t, mp)
}

func TestMetricsProvider_Disabled_NoOp(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "test-svc"}
	mp, _ := NewMetricsProvider(cfg, zap.NewNop())
	ctx := context.Background()

	// All record calls should be no-ops (no panic)
	assert.NotPanics(t, func() {
		mp.RecordHTTPRequest(ctx, "GET", "/api/test", 200, 10*time.Millisecond)
		mp.RecordGRPCRequest(ctx, "UserService", "GetUser", true, 5*time.Millisecond)
		mp.RecordDBOperation(ctx, "SELECT", true, 2*time.Millisecond)
		mp.RecordCacheHit(ctx, "redis")
		mp.RecordCacheMiss(ctx, "redis")
		mp.IncrementConnections(ctx, "http")
		mp.DecrementConnections(ctx, "http")
	})
}

func TestMetricsProvider_Handler_Disabled(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "test-svc"}
	mp, _ := NewMetricsProvider(cfg, zap.NewNop())
	handler := mp.Handler()
	assert.NotNil(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	// not-found handler returns 404
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMetricsProvider_Meter_Disabled(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "test-svc"}
	mp, _ := NewMetricsProvider(cfg, zap.NewNop())
	assert.NotNil(t, mp.Meter())
}

func TestMetricsProvider_Shutdown_Disabled(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "test-svc"}
	mp, _ := NewMetricsProvider(cfg, zap.NewNop())
	err := mp.Shutdown(context.Background())
	assert.NoError(t, err)
}

// ── MetricsProvider (enabled) ────────────────────────────────────────────────

func TestNewMetricsProvider_Enabled(t *testing.T) {
	cfg := &MetricsConfig{
		Enabled:        true,
		ServiceName:    "test-svc",
		PrometheusPath: "/metrics",
	}
	mp, err := NewMetricsProvider(cfg, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, mp)

	ctx := context.Background()
	assert.NotPanics(t, func() {
		mp.RecordHTTPRequest(ctx, "POST", "/login", 201, 20*time.Millisecond)
		mp.RecordGRPCRequest(ctx, "AuthService", "Login", false, 15*time.Millisecond)
		mp.RecordDBOperation(ctx, "INSERT", false, 3*time.Millisecond)
		mp.RecordCacheHit(ctx, "user-cache")
		mp.RecordCacheMiss(ctx, "user-cache")
		mp.IncrementConnections(ctx, "grpc")
		mp.DecrementConnections(ctx, "grpc")
	})

	handler := mp.Handler()
	assert.NotNil(t, handler)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	assert.NoError(t, mp.Shutdown(ctx))
}

// ── TracingConfig ─────────────────────────────────────────────────────────────

func TestDefaultTracingConfig(t *testing.T) {
	cfg := DefaultTracingConfig()
	assert.False(t, cfg.Enabled)
	assert.Equal(t, "arcana-cloud-go", cfg.ServiceName)
	assert.Equal(t, "1.0.0", cfg.ServiceVersion)
	assert.Equal(t, float64(1.0), cfg.SamplingRate)
}

func TestNewTracingProvider_Disabled(t *testing.T) {
	cfg := DefaultTracingConfig()
	cfg.Enabled = false
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)
	assert.NotNil(t, tp)
}

func TestNewTracingProvider_Stdout(t *testing.T) {
	cfg := DefaultTracingConfig()
	cfg.Enabled = true
	cfg.ExporterType = "stdout"
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)
	assert.NoError(t, tp.Shutdown(context.Background()))
}

func TestTracingProvider_Shutdown_Disabled(t *testing.T) {
	cfg := DefaultTracingConfig()
	tp, _ := NewTracingProvider(cfg, zap.NewNop())
	assert.NoError(t, tp.Shutdown(context.Background()))
}
