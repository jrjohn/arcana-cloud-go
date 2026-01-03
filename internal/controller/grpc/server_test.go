package grpc

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
)

func TestNewServer(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.GRPCConfig{
		Host:       "localhost",
		Port:       50051,
		TLSEnabled: false,
	}

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
	if server.server == nil {
		t.Error("NewServer() server field is nil")
	}
	if server.logger == nil {
		t.Error("NewServer() logger field is nil")
	}
	if server.config == nil {
		t.Error("NewServer() config field is nil")
	}
}

func TestNewServer_TLSEnabled_InvalidFiles(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.GRPCConfig{
		Host:       "localhost",
		Port:       50051,
		TLSEnabled: true,
		CertFile:   "/nonexistent/cert.pem",
		KeyFile:    "/nonexistent/key.pem",
	}

	_, err := NewServer(cfg, logger)
	if err == nil {
		t.Error("NewServer() expected error for invalid TLS files")
	}
}

func TestServer_GetServer(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.GRPCConfig{
		Host:       "localhost",
		Port:       50051,
		TLSEnabled: false,
	}

	server, _ := NewServer(cfg, logger)
	grpcServer := server.GetServer()
	if grpcServer == nil {
		t.Error("GetServer() returned nil")
	}
}

func TestServer_Stop(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.GRPCConfig{
		Host:       "localhost",
		Port:       50052,
		TLSEnabled: false,
	}

	server, _ := NewServer(cfg, logger)
	// Just ensure Stop doesn't panic
	server.Stop()
}

func TestLoggingInterceptor(t *testing.T) {
	logger := zap.NewNop()
	interceptor := LoggingInterceptor(logger)

	if interceptor == nil {
		t.Fatal("LoggingInterceptor() returned nil")
	}

	// Test success case
	t.Run("success", func(t *testing.T) {
		handler := func(ctx context.Context, req any) (any, error) {
			return "response", nil
		}
		info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

		resp, err := interceptor(context.Background(), "request", info, handler)
		if err != nil {
			t.Errorf("LoggingInterceptor() error = %v", err)
		}
		if resp != "response" {
			t.Errorf("LoggingInterceptor() resp = %v, want response", resp)
		}
	})

	// Test error case
	t.Run("error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		handler := func(ctx context.Context, req any) (any, error) {
			return nil, expectedErr
		}
		info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

		_, err := interceptor(context.Background(), "request", info, handler)
		if !errors.Is(err, expectedErr) {
			t.Errorf("LoggingInterceptor() error = %v, want %v", err, expectedErr)
		}
	})
}

func TestRecoveryInterceptor(t *testing.T) {
	logger := zap.NewNop()
	interceptor := RecoveryInterceptor(logger)

	if interceptor == nil {
		t.Fatal("RecoveryInterceptor() returned nil")
	}

	// Test success case
	t.Run("success", func(t *testing.T) {
		handler := func(ctx context.Context, req any) (any, error) {
			return "response", nil
		}
		info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

		resp, err := interceptor(context.Background(), "request", info, handler)
		if err != nil {
			t.Errorf("RecoveryInterceptor() error = %v", err)
		}
		if resp != "response" {
			t.Errorf("RecoveryInterceptor() resp = %v, want response", resp)
		}
	})

	// Test panic recovery
	t.Run("panic recovery", func(t *testing.T) {
		handler := func(ctx context.Context, req any) (any, error) {
			panic("test panic")
		}
		info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

		resp, err := interceptor(context.Background(), "request", info, handler)
		if err == nil {
			t.Error("RecoveryInterceptor() expected error after panic")
		}
		if resp != nil {
			t.Errorf("RecoveryInterceptor() resp = %v, want nil", resp)
		}
		if err.Error() != "internal server error" {
			t.Errorf("RecoveryInterceptor() error = %v, want 'internal server error'", err)
		}
	})
}
