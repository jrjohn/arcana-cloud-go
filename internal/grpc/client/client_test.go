package client

import (
	"testing"

	"go.uber.org/zap"
)

func TestDefaultClientConfig(t *testing.T) {
	cfg := DefaultClientConfig("localhost", 9090)
	if cfg.Host != "localhost" {
		t.Errorf("Host = %v, want localhost", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Errorf("Port = %v, want 9090", cfg.Port)
	}
	if cfg.Timeout == 0 {
		t.Error("Timeout should not be 0")
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %v, want 3", cfg.MaxRetries)
	}
	if !cfg.InsecureEnabled {
		t.Error("InsecureEnabled should be true by default")
	}
}

func TestNewGRPCClient_Success(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := DefaultClientConfig("localhost", 9090)

	client, err := NewGRPCClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewGRPCClient() error = %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Error("NewGRPCClient() returned nil")
	}
}

func TestGRPCClient_UserServiceClient(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := DefaultClientConfig("localhost", 9090)

	client, err := NewGRPCClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewGRPCClient() error = %v", err)
	}
	defer client.Close()

	uc := client.UserServiceClient()
	if uc == nil {
		t.Error("UserServiceClient() returned nil")
	}
}

func TestGRPCClient_AuthServiceClient(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := DefaultClientConfig("localhost", 9090)

	client, err := NewGRPCClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewGRPCClient() error = %v", err)
	}
	defer client.Close()

	ac := client.AuthServiceClient()
	if ac == nil {
		t.Error("AuthServiceClient() returned nil")
	}
}

func TestGRPCClient_Close(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := DefaultClientConfig("localhost", 9090)

	client, err := NewGRPCClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewGRPCClient() error = %v", err)
	}

	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestGRPCClient_Close_NilConn(t *testing.T) {
	client := &GRPCClient{}
	if err := client.Close(); err != nil {
		t.Errorf("Close() with nil conn error = %v", err)
	}
}
