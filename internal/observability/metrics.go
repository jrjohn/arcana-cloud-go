package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	otelprometheus "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	ServiceName   string `mapstructure:"service_name"`
	PrometheusPath string `mapstructure:"prometheus_path"`
}

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:        true,
		ServiceName:    "arcana-cloud-go",
		PrometheusPath: "/metrics",
	}
}

// MetricsProvider manages OpenTelemetry metrics
type MetricsProvider struct {
	config         *MetricsConfig
	meterProvider  *sdkmetric.MeterProvider
	meter          metric.Meter
	logger         *zap.Logger
	registry       *prometheus.Registry
	handler        http.Handler

	// Common metrics
	httpRequestsTotal   metric.Int64Counter
	httpRequestDuration metric.Float64Histogram
	grpcRequestsTotal   metric.Int64Counter
	grpcRequestDuration metric.Float64Histogram
	dbOperationsTotal   metric.Int64Counter
	dbOperationDuration metric.Float64Histogram
	activeConnections   metric.Int64UpDownCounter
	cacheHits           metric.Int64Counter
	cacheMisses         metric.Int64Counter
}

// NewMetricsProvider creates a new metrics provider
func NewMetricsProvider(config *MetricsConfig, logger *zap.Logger) (*MetricsProvider, error) {
	if !config.Enabled {
		return &MetricsProvider{
			config: config,
			meter:  otel.Meter(config.ServiceName),
			logger: logger,
		}, nil
	}

	// Create Prometheus registry
	registry := prometheus.NewRegistry()

	// Create Prometheus exporter with the registry
	exporter, err := otelprometheus.New(
		otelprometheus.WithRegisterer(registry),
	)
	if err != nil {
		return nil, err
	}

	// Create meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)
	otel.SetMeterProvider(meterProvider)

	meter := meterProvider.Meter(config.ServiceName)

	mp := &MetricsProvider{
		config:        config,
		meterProvider: meterProvider,
		meter:         meter,
		logger:        logger,
		registry:      registry,
		handler:       promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	}

	// Initialize common metrics
	if err := mp.initMetrics(); err != nil {
		return nil, err
	}

	logger.Info("OpenTelemetry metrics initialized",
		zap.String("service", config.ServiceName),
		zap.String("prometheus_path", config.PrometheusPath),
	)

	return mp, nil
}

// initMetrics initializes common metrics
func (mp *MetricsProvider) initMetrics() error {
	var err error

	// HTTP metrics
	mp.httpRequestsTotal, err = mp.meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return err
	}

	mp.httpRequestDuration, err = mp.meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	// gRPC metrics
	mp.grpcRequestsTotal, err = mp.meter.Int64Counter(
		"grpc_requests_total",
		metric.WithDescription("Total number of gRPC requests"),
	)
	if err != nil {
		return err
	}

	mp.grpcRequestDuration, err = mp.meter.Float64Histogram(
		"grpc_request_duration_seconds",
		metric.WithDescription("gRPC request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	// Database metrics
	mp.dbOperationsTotal, err = mp.meter.Int64Counter(
		"db_operations_total",
		metric.WithDescription("Total number of database operations"),
	)
	if err != nil {
		return err
	}

	mp.dbOperationDuration, err = mp.meter.Float64Histogram(
		"db_operation_duration_seconds",
		metric.WithDescription("Database operation duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	// Connection metrics
	mp.activeConnections, err = mp.meter.Int64UpDownCounter(
		"active_connections",
		metric.WithDescription("Number of active connections"),
	)
	if err != nil {
		return err
	}

	// Cache metrics
	mp.cacheHits, err = mp.meter.Int64Counter(
		"cache_hits_total",
		metric.WithDescription("Total number of cache hits"),
	)
	if err != nil {
		return err
	}

	mp.cacheMisses, err = mp.meter.Int64Counter(
		"cache_misses_total",
		metric.WithDescription("Total number of cache misses"),
	)
	if err != nil {
		return err
	}

	return nil
}

// RecordHTTPRequest records an HTTP request metric
func (mp *MetricsProvider) RecordHTTPRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	if mp.httpRequestsTotal == nil {
		return
	}

	attrs := metric.WithAttributes(
		AttrHTTPMethod.String(method),
		AttrHTTPRoute.String(path),
		AttrHTTPStatusCode.Int(statusCode),
	)

	mp.httpRequestsTotal.Add(ctx, 1, attrs)
	mp.httpRequestDuration.Record(ctx, duration.Seconds(), attrs)
}

// RecordGRPCRequest records a gRPC request metric
func (mp *MetricsProvider) RecordGRPCRequest(ctx context.Context, service, method string, success bool, duration time.Duration) {
	if mp.grpcRequestsTotal == nil {
		return
	}

	status := "ok"
	if !success {
		status = "error"
	}

	attrs := metric.WithAttributes(
		AttrGRPCService.String(service),
		AttrGRPCMethod.String(method),
		AttrDBOperation.String(status),
	)

	mp.grpcRequestsTotal.Add(ctx, 1, attrs)
	mp.grpcRequestDuration.Record(ctx, duration.Seconds(), attrs)
}

// RecordDBOperation records a database operation metric
func (mp *MetricsProvider) RecordDBOperation(ctx context.Context, operation string, success bool, duration time.Duration) {
	if mp.dbOperationsTotal == nil {
		return
	}

	status := "ok"
	if !success {
		status = "error"
	}

	attrs := metric.WithAttributes(
		AttrDBOperation.String(operation),
		AttrDBSystem.String(status),
	)

	mp.dbOperationsTotal.Add(ctx, 1, attrs)
	mp.dbOperationDuration.Record(ctx, duration.Seconds(), attrs)
}

// RecordCacheHit records a cache hit
func (mp *MetricsProvider) RecordCacheHit(ctx context.Context, cacheName string) {
	if mp.cacheHits == nil {
		return
	}
	mp.cacheHits.Add(ctx, 1, metric.WithAttributes(
		AttrDBSystem.String(cacheName),
	))
}

// RecordCacheMiss records a cache miss
func (mp *MetricsProvider) RecordCacheMiss(ctx context.Context, cacheName string) {
	if mp.cacheMisses == nil {
		return
	}
	mp.cacheMisses.Add(ctx, 1, metric.WithAttributes(
		AttrDBSystem.String(cacheName),
	))
}

// IncrementConnections increments active connections
func (mp *MetricsProvider) IncrementConnections(ctx context.Context, connType string) {
	if mp.activeConnections == nil {
		return
	}
	mp.activeConnections.Add(ctx, 1, metric.WithAttributes(
		AttrDBSystem.String(connType),
	))
}

// DecrementConnections decrements active connections
func (mp *MetricsProvider) DecrementConnections(ctx context.Context, connType string) {
	if mp.activeConnections == nil {
		return
	}
	mp.activeConnections.Add(ctx, -1, metric.WithAttributes(
		AttrDBSystem.String(connType),
	))
}

// Handler returns an HTTP handler for Prometheus metrics
func (mp *MetricsProvider) Handler() http.Handler {
	if mp.handler != nil {
		return mp.handler
	}
	return http.NotFoundHandler()
}

// Meter returns the meter for creating custom metrics
func (mp *MetricsProvider) Meter() metric.Meter {
	return mp.meter
}

// Shutdown gracefully shuts down the metrics provider
func (mp *MetricsProvider) Shutdown(ctx context.Context) error {
	if mp.meterProvider != nil {
		return mp.meterProvider.Shutdown(ctx)
	}
	return nil
}
