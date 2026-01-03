package di

import (
	"testing"

	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
)

func TestPrintBanner(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{
		App: config.AppConfig{
			Name:        "test-app",
			Version:     "1.0.0",
			Environment: "test",
		},
		Deployment: config.DeploymentConfig{
			Mode:     config.DeploymentMonolithic,
			Layer:    config.LayerAll,
			Protocol: config.ProtocolHTTP,
		},
	}

	// Just ensure PrintBanner doesn't panic
	PrintBanner(cfg, logger)
}

func TestProvideLogger(t *testing.T) {
	cfg := &config.AppConfig{
		Debug: true,
	}

	logger, err := provideLogger(cfg)
	if err != nil {
		t.Fatalf("provideLogger() error = %v", err)
	}
	if logger == nil {
		t.Error("provideLogger() returned nil")
	}
}

func TestProvideLogger_Production(t *testing.T) {
	cfg := &config.AppConfig{
		Debug: false,
	}

	logger, err := provideLogger(cfg)
	if err != nil {
		t.Fatalf("provideLogger() error = %v", err)
	}
	if logger == nil {
		t.Error("provideLogger() returned nil")
	}
}

func TestModulesNotNil(t *testing.T) {
	tests := []struct {
		name   string
		module interface{}
	}{
		{"AppModule", AppModule},
		{"ConfigModule", ConfigModule},
		{"LoggerModule", LoggerModule},
		{"DatabaseModule", DatabaseModule},
		{"RepositoryModule", RepositoryModule},
		{"SecurityModule", SecurityModule},
		{"ServiceModule", ServiceModule},
		{"MiddlewareModule", MiddlewareModule},
		{"ControllerModule", ControllerModule},
		{"PluginModule", PluginModule},
		{"JobsModule", JobsModule},
		{"HTTPServerModule", HTTPServerModule},
		{"GRPCServerModule", GRPCServerModule},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.module == nil {
				t.Errorf("%s is nil", tt.name)
			}
		})
	}
}
