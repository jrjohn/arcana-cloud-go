package client

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/jrjohn/arcana-cloud-go/api/proto/pb"
)

// GRPCClientConfig holds configuration for gRPC client connections
type GRPCClientConfig struct {
	Host            string
	Port            int
	Timeout         time.Duration
	MaxRetries      int
	KeepAlive       time.Duration
	InsecureEnabled bool
}

// DefaultClientConfig returns default gRPC client configuration
func DefaultClientConfig(host string, port int) *GRPCClientConfig {
	return &GRPCClientConfig{
		Host:            host,
		Port:            port,
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		KeepAlive:       30 * time.Second,
		InsecureEnabled: true,
	}
}

// GRPCClient manages gRPC client connections
type GRPCClient struct {
	conn   *grpc.ClientConn
	config *GRPCClientConfig
	logger *zap.Logger

	userServiceClient pb.UserServiceClient
	authServiceClient pb.AuthServiceClient
}

// NewGRPCClient creates a new gRPC client
func NewGRPCClient(config *GRPCClientConfig, logger *zap.Logger) (*GRPCClient, error) {
	target := fmt.Sprintf("%s:%d", config.Host, config.Port)

	var opts []grpc.DialOption
	if config.InsecureEnabled {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	opts = append(opts,
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)

	logger.Info("connecting to gRPC server", zap.String("target", target))

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	client := &GRPCClient{
		conn:   conn,
		config: config,
		logger: logger,
	}

	// Initialize service clients
	client.userServiceClient = pb.NewUserServiceClient(conn)
	client.authServiceClient = pb.NewAuthServiceClient(conn)

	return client, nil
}

// Close closes the gRPC connection
func (c *GRPCClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// UserServiceClient returns the UserService client
func (c *GRPCClient) UserServiceClient() pb.UserServiceClient {
	return c.userServiceClient
}

// AuthServiceClient returns the AuthService client
func (c *GRPCClient) AuthServiceClient() pb.AuthServiceClient {
	return c.authServiceClient
}

// WaitForReady waits for the gRPC connection to be ready
func (c *GRPCClient) WaitForReady(ctx context.Context) error {
	state := c.conn.GetState()
	for state != 2 { // READY = 2
		if !c.conn.WaitForStateChange(ctx, state) {
			return fmt.Errorf("connection state change timed out")
		}
		state = c.conn.GetState()
		if state == 4 { // SHUTDOWN = 4
			return fmt.Errorf("connection is shutdown")
		}
	}
	return nil
}
