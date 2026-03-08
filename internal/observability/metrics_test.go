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

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

// TestDefaultMetricsConfig verifies the default config values
func TestDefaultMetricsConfig(t *testing.T) {
	cfg := DefaultMetricsConfig()
	assert.NotNil(t, cfg)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "arcana-cloud-go", cfg.ServiceName)
	assert.Equal(t, "/metrics", cfg.PrometheusPath)
}

// TestNewMetricsProvider_Disabled creates a disabled provider
func TestNewMetricsProvider_Disabled(t *testing.T) {
	cfg := &MetricsConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)
	require.NotNil(t, mp)
}

// TestNewMetricsProvider_Enabled creates an enabled provider
func TestNewMetricsProvider_Enabled(t *testing.T) {
	cfg := DefaultMetricsConfig()
	cfg.ServiceName = "test-metrics-enabled"
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)
	require.NotNil(t, mp)
	// Shutdown cleanly
	err = mp.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestMetricsProvider_Handler_Enabled checks handler is set for enabled provider
func TestMetricsProvider_Handler_Enabled(t *testing.T) {
	cfg := DefaultMetricsConfig()
	cfg.ServiceName = "test-handler-enabled"
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)

	handler := mp.Handler()
	assert.NotNil(t, handler)
	defer mp.Shutdown(context.Background())
}

// TestMetricsProvider_Handler_Disabled returns NotFoundHandler when disabled
func TestMetricsProvider_Handler_Disabled(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "disabled"}
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)

	handler := mp.Handler()
	assert.NotNil(t, handler)

	// Should return 404
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// TestMetricsProvider_Meter returns the meter
func TestMetricsProvider_Meter(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "test-meter"}
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)
	meter := mp.Meter()
	assert.NotNil(t, meter)
}

// TestMetricsProvider_RecordHTTPRequest_Nil does not panic when counters are nil
func TestMetricsProvider_RecordHTTPRequest_Nil(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "test-nil"}
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)

	// httpRequestsTotal is nil when disabled, should not panic
	assert.NotPanics(t, func() {
		mp.RecordHTTPRequest(context.Background(), "GET", "/test", 200, 100*time.Millisecond)
	})
}

// TestMetricsProvider_RecordHTTPRequest_Enabled records metrics on enabled provider
func TestMetricsProvider_RecordHTTPRequest_Enabled(t *testing.T) {
	cfg := DefaultMetricsConfig()
	cfg.ServiceName = "test-record-http"
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)
	defer mp.Shutdown(context.Background())

	assert.NotPanics(t, func() {
		mp.RecordHTTPRequest(context.Background(), "POST", "/api/v1/users", 201, 50*time.Millisecond)
		mp.RecordHTTPRequest(context.Background(), "GET", "/api/v1/users", 200, 10*time.Millisecond)
		mp.RecordHTTPRequest(context.Background(), "DELETE", "/api/v1/users/1", 404, 5*time.Millisecond)
	})
}

// TestMetricsProvider_RecordGRPCRequest_Nil does not panic when nil
func TestMetricsProvider_RecordGRPCRequest_Nil(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "test-grpc-nil"}
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		mp.RecordGRPCRequest(context.Background(), "UserService", "GetUser", true, 20*time.Millisecond)
		mp.RecordGRPCRequest(context.Background(), "UserService", "GetUser", false, 5*time.Millisecond)
	})
}

// TestMetricsProvider_RecordGRPCRequest_Enabled records gRPC metrics
func TestMetricsProvider_RecordGRPCRequest_Enabled(t *testing.T) {
	cfg := DefaultMetricsConfig()
	cfg.ServiceName = "test-record-grpc"
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)
	defer mp.Shutdown(context.Background())

	assert.NotPanics(t, func() {
		mp.RecordGRPCRequest(context.Background(), "UserService", "GetUser", true, 20*time.Millisecond)
		mp.RecordGRPCRequest(context.Background(), "AuthService", "Login", false, 5*time.Millisecond)
	})
}

// TestMetricsProvider_RecordDBOperation_Nil does not panic when nil
func TestMetricsProvider_RecordDBOperation_Nil(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "test-db-nil"}
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		mp.RecordDBOperation(context.Background(), "select", true, 15*time.Millisecond)
		mp.RecordDBOperation(context.Background(), "insert", false, 30*time.Millisecond)
	})
}

// TestMetricsProvider_RecordDBOperation_Enabled records DB metrics
func TestMetricsProvider_RecordDBOperation_Enabled(t *testing.T) {
	cfg := DefaultMetricsConfig()
	cfg.ServiceName = "test-record-db"
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)
	defer mp.Shutdown(context.Background())

	assert.NotPanics(t, func() {
		mp.RecordDBOperation(context.Background(), "select", true, 15*time.Millisecond)
		mp.RecordDBOperation(context.Background(), "insert", false, 30*time.Millisecond)
		mp.RecordDBOperation(context.Background(), "update", true, 25*time.Millisecond)
	})
}

// TestMetricsProvider_RecordCacheHit_Nil does not panic when nil
func TestMetricsProvider_RecordCacheHit_Nil(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "test-cache-nil"}
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		mp.RecordCacheHit(context.Background(), "redis")
		mp.RecordCacheMiss(context.Background(), "redis")
	})
}

// TestMetricsProvider_RecordCache_Enabled records cache metrics
func TestMetricsProvider_RecordCache_Enabled(t *testing.T) {
	cfg := DefaultMetricsConfig()
	cfg.ServiceName = "test-record-cache"
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)
	defer mp.Shutdown(context.Background())

	assert.NotPanics(t, func() {
		mp.RecordCacheHit(context.Background(), "redis")
		mp.RecordCacheHit(context.Background(), "memory")
		mp.RecordCacheMiss(context.Background(), "redis")
		mp.RecordCacheMiss(context.Background(), "memory")
	})
}

// TestMetricsProvider_Connections_Nil does not panic when nil
func TestMetricsProvider_Connections_Nil(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "test-conn-nil"}
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		mp.IncrementConnections(context.Background(), "websocket")
		mp.DecrementConnections(context.Background(), "websocket")
	})
}

// TestMetricsProvider_Connections_Enabled records connections
func TestMetricsProvider_Connections_Enabled(t *testing.T) {
	cfg := DefaultMetricsConfig()
	cfg.ServiceName = "test-record-conn"
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)
	defer mp.Shutdown(context.Background())

	assert.NotPanics(t, func() {
		mp.IncrementConnections(context.Background(), "websocket")
		mp.IncrementConnections(context.Background(), "grpc")
		mp.DecrementConnections(context.Background(), "websocket")
	})
}

// TestMetricsProvider_Shutdown_Nil does not error when nil meter provider
func TestMetricsProvider_Shutdown_Nil(t *testing.T) {
	cfg := &MetricsConfig{Enabled: false, ServiceName: "test-shutdown-nil"}
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)

	err = mp.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestMetricsProvider_Shutdown_Enabled shuts down cleanly
func TestMetricsProvider_Shutdown_Enabled(t *testing.T) {
	cfg := DefaultMetricsConfig()
	cfg.ServiceName = "test-shutdown-enabled"
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)

	err = mp.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestMetricsProvider_Handler_ServesMetrics verifies handler returns metrics data
func TestMetricsProvider_Handler_ServesMetrics(t *testing.T) {
	cfg := DefaultMetricsConfig()
	cfg.ServiceName = "test-handler-serves"
	mp, err := NewMetricsProvider(cfg, testLogger())
	require.NoError(t, err)
	defer mp.Shutdown(context.Background())

	// Record some metrics first
	mp.RecordHTTPRequest(context.Background(), "GET", "/test", 200, 10*time.Millisecond)

	handler := mp.Handler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// Prometheus returns 200 OK
	assert.Equal(t, http.StatusOK, rr.Code)
}
