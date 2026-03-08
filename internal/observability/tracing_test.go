package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// TestDefaultTracingConfig verifies the default tracing config
func TestDefaultTracingConfig(t *testing.T) {
	cfg := DefaultTracingConfig()
	assert.NotNil(t, cfg)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, "arcana-cloud-go", cfg.ServiceName)
	assert.Equal(t, "1.0.0", cfg.ServiceVersion)
	assert.Equal(t, "development", cfg.Environment)
	assert.Equal(t, "stdout", cfg.ExporterType)
	assert.Equal(t, "localhost:4317", cfg.OTLPEndpoint)
	assert.True(t, cfg.OTLPInsecure)
	assert.Equal(t, 1.0, cfg.SamplingRate)
	assert.Equal(t, "tracecontext", cfg.PropagatorType)
}

// TestNewTracingProvider_Disabled creates a disabled provider (no exporter)
func TestNewTracingProvider_Disabled(t *testing.T) {
	cfg := &TracingConfig{
		Enabled:     false,
		ServiceName: "test-tracing",
	}
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)
}

// TestNewTracingProvider_StdoutEnabled creates an enabled stdout provider
func TestNewTracingProvider_StdoutEnabled(t *testing.T) {
	cfg := &TracingConfig{
		Enabled:        true,
		ServiceName:    "test-stdout",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExporterType:   "stdout",
		SamplingRate:   1.0,
		PropagatorType: "tracecontext",
	}
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)
	err = tp.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestNewTracingProvider_AlwaysSample uses sampling rate >= 1
func TestNewTracingProvider_AlwaysSample(t *testing.T) {
	cfg := &TracingConfig{
		Enabled:        true,
		ServiceName:    "test-always-sample",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExporterType:   "stdout",
		SamplingRate:   1.0,
		PropagatorType: "tracecontext",
	}
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)
	defer tp.Shutdown(context.Background())
}

// TestNewTracingProvider_NeverSample uses sampling rate <= 0
func TestNewTracingProvider_NeverSample(t *testing.T) {
	cfg := &TracingConfig{
		Enabled:        true,
		ServiceName:    "test-never-sample",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExporterType:   "stdout",
		SamplingRate:   0.0,
		PropagatorType: "tracecontext",
	}
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)
	defer tp.Shutdown(context.Background())
}

// TestNewTracingProvider_RatioSample uses sampling rate between 0 and 1
func TestNewTracingProvider_RatioSample(t *testing.T) {
	cfg := &TracingConfig{
		Enabled:        true,
		ServiceName:    "test-ratio-sample",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExporterType:   "stdout",
		SamplingRate:   0.5,
		PropagatorType: "b3",
	}
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)
	defer tp.Shutdown(context.Background())
}

// TestNewTracingProvider_DefaultExporter uses default (unknown) exporter type
func TestNewTracingProvider_DefaultExporter(t *testing.T) {
	cfg := &TracingConfig{
		Enabled:        true,
		ServiceName:    "test-default-exporter",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExporterType:   "unknown-type",
		SamplingRate:   1.0,
		PropagatorType: "tracecontext",
	}
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, tp)
	defer tp.Shutdown(context.Background())
}

// TestTracingProvider_Tracer returns a valid tracer
func TestTracingProvider_Tracer(t *testing.T) {
	cfg := &TracingConfig{Enabled: false, ServiceName: "test-tracer"}
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)

	tracer := tp.Tracer()
	assert.NotNil(t, tracer)
}

// TestTracingProvider_StartSpan starts a span
func TestTracingProvider_StartSpan(t *testing.T) {
	cfg := &TracingConfig{Enabled: false, ServiceName: "test-start-span"}
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)

	ctx, span := tp.StartSpan(context.Background(), "test-span")
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	span.End()
}

// TestTracingProvider_Shutdown_Disabled does not error
func TestTracingProvider_Shutdown_Disabled(t *testing.T) {
	cfg := &TracingConfig{Enabled: false, ServiceName: "test-shutdown"}
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)

	err = tp.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestTracingProvider_Shutdown_Enabled shuts down cleanly
func TestTracingProvider_Shutdown_Enabled(t *testing.T) {
	cfg := &TracingConfig{
		Enabled:        true,
		ServiceName:    "test-shutdown-enabled",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		ExporterType:   "stdout",
		SamplingRate:   1.0,
	}
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)

	err = tp.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestSpanFromContext returns the current span
func TestSpanFromContext(t *testing.T) {
	span := SpanFromContext(context.Background())
	assert.NotNil(t, span)
}

// TestContextWithSpan returns a context with the span
func TestContextWithSpan(t *testing.T) {
	cfg := &TracingConfig{Enabled: false, ServiceName: "test-ctx-span"}
	tp, err := NewTracingProvider(cfg, zap.NewNop())
	require.NoError(t, err)

	ctx, span := tp.StartSpan(context.Background(), "parent-span")
	newCtx := ContextWithSpan(ctx, span)
	assert.NotNil(t, newCtx)
	span.End()
}

// TestAddSpanAttributes adds attributes without panic
func TestAddSpanAttributes(t *testing.T) {
	assert.NotPanics(t, func() {
		AddSpanAttributes(context.Background(),
			AttrHTTPMethod.String("GET"),
			AttrHTTPRoute.String("/api/v1/users"),
			AttrHTTPStatusCode.Int(200),
		)
	})
}

// TestRecordSpanError records an error without panic
func TestRecordSpanError(t *testing.T) {
	assert.NotPanics(t, func() {
		RecordSpanError(context.Background(), errors.New("test error"))
	})
}

// TestSetSpanStatus sets span status without panic
func TestSetSpanStatus(t *testing.T) {
	assert.NotPanics(t, func() {
		SetSpanStatus(context.Background(), codes.Ok, "success")
		SetSpanStatus(context.Background(), codes.Error, "something went wrong")
	})
}

// TestAttrKeys verifies common attribute keys are defined
func TestAttrKeys(t *testing.T) {
	assert.Equal(t, "http.method", string(AttrHTTPMethod))
	assert.Equal(t, "http.url", string(AttrHTTPURL))
	assert.Equal(t, "http.status_code", string(AttrHTTPStatusCode))
	assert.Equal(t, "http.route", string(AttrHTTPRoute))
	assert.Equal(t, "rpc.service", string(AttrGRPCService))
	assert.Equal(t, "rpc.method", string(AttrGRPCMethod))
	assert.Equal(t, "db.system", string(AttrDBSystem))
	assert.Equal(t, "db.statement", string(AttrDBStatement))
	assert.Equal(t, "db.operation", string(AttrDBOperation))
	assert.Equal(t, "user.id", string(AttrUserID))
	assert.Equal(t, "arcana.layer", string(AttrLayerName))
}

// TestCreatePropagator returns a non-nil propagator
func TestCreatePropagator(t *testing.T) {
	propagators := []string{"tracecontext", "b3", "jaeger", ""}
	for _, pt := range propagators {
		p := createPropagator(pt)
		assert.NotNil(t, p, "propagator should not be nil for type: %s", pt)
	}
}
