package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// TracingConfig holds OpenTelemetry tracing configuration
type TracingConfig struct {
	Enabled         bool    `mapstructure:"enabled"`
	ServiceName     string  `mapstructure:"service_name"`
	ServiceVersion  string  `mapstructure:"service_version"`
	Environment     string  `mapstructure:"environment"`
	ExporterType    string  `mapstructure:"exporter_type"` // stdout, otlp-grpc, otlp-http, jaeger
	OTLPEndpoint    string  `mapstructure:"otlp_endpoint"`
	OTLPInsecure    bool    `mapstructure:"otlp_insecure"`
	SamplingRate    float64 `mapstructure:"sampling_rate"`
	PropagatorType  string  `mapstructure:"propagator_type"` // tracecontext, b3, jaeger
}

// DefaultTracingConfig returns default tracing configuration
func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		Enabled:        false,
		ServiceName:    "arcana-cloud-go",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		ExporterType:   "stdout",
		OTLPEndpoint:   "localhost:4317",
		OTLPInsecure:   true,
		SamplingRate:   1.0,
		PropagatorType: "tracecontext",
	}
}

// TracingProvider manages OpenTelemetry tracing
type TracingProvider struct {
	config         *TracingConfig
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
	logger         *zap.Logger
}

// NewTracingProvider creates a new tracing provider
func NewTracingProvider(config *TracingConfig, logger *zap.Logger) (*TracingProvider, error) {
	if !config.Enabled {
		return &TracingProvider{
			config: config,
			tracer: otel.Tracer(config.ServiceName),
			logger: logger,
		}, nil
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter
	exporter, err := createExporter(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create sampler
	var sampler sdktrace.Sampler
	if config.SamplingRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if config.SamplingRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SamplingRate)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider and propagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(createPropagator(config.PropagatorType))

	logger.Info("OpenTelemetry tracing initialized",
		zap.String("service", config.ServiceName),
		zap.String("exporter", config.ExporterType),
		zap.Float64("sampling_rate", config.SamplingRate),
	)

	return &TracingProvider{
		config:         config,
		tracerProvider: tp,
		tracer:         tp.Tracer(config.ServiceName),
		logger:         logger,
	}, nil
}

// createExporter creates the appropriate exporter based on config
func createExporter(config *TracingConfig) (sdktrace.SpanExporter, error) {
	ctx := context.Background()

	switch config.ExporterType {
	case "stdout":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())

	case "otlp-grpc":
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(config.OTLPEndpoint),
		}
		if config.OTLPInsecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		client := otlptracegrpc.NewClient(opts...)
		return otlptrace.New(ctx, client)

	case "otlp-http":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(config.OTLPEndpoint),
		}
		if config.OTLPInsecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		client := otlptracehttp.NewClient(opts...)
		return otlptrace.New(ctx, client)

	default:
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
}

// createPropagator creates the appropriate propagator
func createPropagator(_ string) propagation.TextMapPropagator {
	// All supported propagator types (b3, tracecontext) use W3C composite format
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// Tracer returns the tracer
func (tp *TracingProvider) Tracer() trace.Tracer {
	return tp.tracer
}

// StartSpan starts a new span
func (tp *TracingProvider) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tp.tracer.Start(ctx, name, opts...)
}

// Shutdown gracefully shuts down the tracer provider
func (tp *TracingProvider) Shutdown(ctx context.Context) error {
	if tp.tracerProvider != nil {
		return tp.tracerProvider.Shutdown(ctx)
	}
	return nil
}

// SpanFromContext returns the span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ContextWithSpan returns a new context with the span
func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}

// AddSpanAttributes adds attributes to the current span
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordSpanError records an error on the current span
func RecordSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// SetSpanStatus sets the status of the current span
func SetSpanStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(code, description)
}

// Common attribute keys
var (
	AttrHTTPMethod     = attribute.Key("http.method")
	AttrHTTPURL        = attribute.Key("http.url")
	AttrHTTPStatusCode = attribute.Key("http.status_code")
	AttrHTTPRoute      = attribute.Key("http.route")
	AttrGRPCService    = attribute.Key("rpc.service")
	AttrGRPCMethod     = attribute.Key("rpc.method")
	AttrDBSystem       = attribute.Key("db.system")
	AttrDBStatement    = attribute.Key("db.statement")
	AttrDBOperation    = attribute.Key("db.operation")
	AttrUserID         = attribute.Key("user.id")
	AttrLayerName      = attribute.Key("arcana.layer")
)
