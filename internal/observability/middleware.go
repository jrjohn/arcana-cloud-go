package observability

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TracingMiddleware returns a Gin middleware for HTTP tracing
func TracingMiddleware(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		// Extract trace context from incoming request
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Start span
		spanName := c.FullPath()
		if spanName == "" {
			spanName = c.Request.URL.Path
		}

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				AttrHTTPMethod.String(c.Request.Method),
				AttrHTTPURL.String(c.Request.URL.String()),
				AttrHTTPRoute.String(spanName),
			),
		)
		defer span.End()

		// Set context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Add response attributes
		statusCode := c.Writer.Status()
		span.SetAttributes(
			AttrHTTPStatusCode.Int(statusCode),
			attribute.Int64("http.response_time_ms", duration.Milliseconds()),
		)

		// Set span status based on HTTP status code
		if statusCode >= 400 {
			span.SetStatus(codes.Error, "HTTP error")
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Record errors
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
		}
	}
}

// UnaryServerInterceptor returns a gRPC server interceptor for tracing
func UnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract trace context from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			ctx = propagator.Extract(ctx, MetadataCarrier(md))
		}

		// Start span
		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				AttrGRPCService.String(serviceName),
				AttrGRPCMethod.String(info.FullMethod),
			),
		)
		defer span.End()

		// Handle request
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		span.SetAttributes(
			attribute.Int64("rpc.response_time_ms", duration.Milliseconds()),
		)

		if err != nil {
			s, _ := status.FromError(err)
			span.SetStatus(codes.Error, s.Message())
			span.RecordError(err)
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", s.Code().String()),
			)
		} else {
			span.SetStatus(codes.Ok, "")
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", "OK"),
			)
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for tracing
func StreamServerInterceptor(serviceName string) grpc.StreamServerInterceptor {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		// Extract trace context from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			ctx = propagator.Extract(ctx, MetadataCarrier(md))
		}

		// Start span
		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				AttrGRPCService.String(serviceName),
				AttrGRPCMethod.String(info.FullMethod),
				attribute.Bool("rpc.grpc.stream", true),
			),
		)
		defer span.End()

		// Wrap stream with traced context
		wrappedStream := &tracedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		err := handler(srv, wrappedStream)

		if err != nil {
			s, _ := status.FromError(err)
			span.SetStatus(codes.Error, s.Message())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

// UnaryClientInterceptor returns a gRPC client interceptor for tracing
func UnaryClientInterceptor(serviceName string) grpc.UnaryClientInterceptor {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Start span
		ctx, span := tracer.Start(ctx, method,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				AttrGRPCService.String(serviceName),
				AttrGRPCMethod.String(method),
				attribute.String("rpc.system", "grpc"),
			),
		)
		defer span.End()

		// Inject trace context into outgoing metadata
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		propagator.Inject(ctx, MetadataCarrier(md))
		ctx = metadata.NewOutgoingContext(ctx, md)

		// Invoke
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(start)

		span.SetAttributes(
			attribute.Int64("rpc.response_time_ms", duration.Milliseconds()),
		)

		if err != nil {
			s, _ := status.FromError(err)
			span.SetStatus(codes.Error, s.Message())
			span.RecordError(err)
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", s.Code().String()),
			)
		} else {
			span.SetStatus(codes.Ok, "")
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", "OK"),
			)
		}

		return err
	}
}

// MetadataCarrier adapts gRPC metadata for propagation
type MetadataCarrier metadata.MD

// Get returns the value for a key
func (mc MetadataCarrier) Get(key string) string {
	vals := metadata.MD(mc).Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// Set sets a key-value pair
func (mc MetadataCarrier) Set(key, value string) {
	metadata.MD(mc).Set(key, value)
}

// Keys returns all keys
func (mc MetadataCarrier) Keys() []string {
	keys := make([]string, 0, len(mc))
	for k := range mc {
		keys = append(keys, k)
	}
	return keys
}

// tracedServerStream wraps a server stream with a traced context
type tracedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *tracedServerStream) Context() context.Context {
	return s.ctx
}
