package grpc

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
)

// Server wraps the gRPC server
type Server struct {
	server   *grpc.Server
	listener net.Listener
	logger   *zap.Logger
	config   *config.GRPCConfig
}

// NewServer creates a new gRPC server
func NewServer(cfg *config.GRPCConfig, logger *zap.Logger) (*Server, error) {
	var opts []grpc.ServerOption

	// Add TLS if enabled
	if cfg.TLSEnabled {
		creds, err := credentials.NewServerTLSFromFile(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// Add interceptors
	opts = append(opts,
		grpc.ChainUnaryInterceptor(
			LoggingInterceptor(logger),
			RecoveryInterceptor(logger),
		),
	)

	server := grpc.NewServer(opts...)

	// Enable reflection for development
	reflection.Register(server)

	return &Server{
		server: server,
		logger: logger,
		config: cfg,
	}, nil
}

// GetServer returns the underlying gRPC server for service registration
func (s *Server) GetServer() *grpc.Server {
	return s.server
}

// Start starts the gRPC server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	s.logger.Info("gRPC server starting", zap.String("address", addr))

	return s.server.Serve(listener)
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	s.logger.Info("stopping gRPC server")
	s.server.GracefulStop()
}

// LoggingInterceptor logs gRPC requests
func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		logger.Debug("gRPC request",
			zap.String("method", info.FullMethod),
		)

		resp, err := handler(ctx, req)

		if err != nil {
			logger.Error("gRPC error",
				zap.String("method", info.FullMethod),
				zap.Error(err),
			)
		}

		return resp, err
	}
}

// RecoveryInterceptor recovers from panics in gRPC handlers
func RecoveryInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC panic recovered",
					zap.String("method", info.FullMethod),
					zap.Any("panic", r),
				)
				err = fmt.Errorf("internal server error")
			}
		}()

		return handler(ctx, req)
	}
}
