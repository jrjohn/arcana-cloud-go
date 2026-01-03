package logger

import (
	"os"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "development config",
			config: Config{
				Level:       "debug",
				Development: true,
				Encoding:    "console",
			},
			wantErr: false,
		},
		{
			name: "production config",
			config: Config{
				Level:       "info",
				Development: false,
				Encoding:    "json",
			},
			wantErr: false,
		},
		{
			name: "invalid level falls back to info",
			config: Config{
				Level:       "invalid",
				Development: false,
				Encoding:    "json",
			},
			wantErr: false,
		},
		{
			name: "error level",
			config: Config{
				Level:       "error",
				Development: false,
			},
			wantErr: false,
		},
		{
			name: "warn level",
			config: Config{
				Level:       "warn",
				Development: true,
			},
			wantErr: false,
		},
		{
			name: "empty encoding uses default",
			config: Config{
				Level:       "info",
				Development: false,
				Encoding:    "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("New() returned nil logger")
			}
			if logger != nil {
				logger.Sync()
			}
		})
	}
}

func TestNew_LogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "dpanic", "panic", "fatal"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			cfg := Config{
				Level:       level,
				Development: true,
				Encoding:    "console",
			}
			logger, err := New(cfg)
			if err != nil {
				t.Errorf("New() with level %s error = %v", level, err)
				return
			}
			if logger == nil {
				t.Errorf("New() with level %s returned nil", level)
			}
			logger.Sync()
		})
	}
}

func TestDefault(t *testing.T) {
	// Save and restore env vars
	originalLogLevel := os.Getenv("LOG_LEVEL")
	originalAppEnv := os.Getenv("APP_ENV")
	defer func() {
		os.Setenv("LOG_LEVEL", originalLogLevel)
		os.Setenv("APP_ENV", originalAppEnv)
	}()

	tests := []struct {
		name     string
		logLevel string
		appEnv   string
	}{
		{
			name:     "development mode",
			logLevel: "debug",
			appEnv:   "development",
		},
		{
			name:     "production mode",
			logLevel: "info",
			appEnv:   "production",
		},
		{
			name:     "empty env vars",
			logLevel: "",
			appEnv:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("LOG_LEVEL", tt.logLevel)
			os.Setenv("APP_ENV", tt.appEnv)

			logger := Default()
			if logger == nil {
				t.Error("Default() returned nil")
			}
			logger.Sync()
		})
	}
}

func TestWithContext(t *testing.T) {
	logger, err := New(Config{
		Level:       "info",
		Development: true,
		Encoding:    "console",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Add context fields
	contextLogger := WithContext(logger,
		zap.String("service", "test-service"),
		zap.Int("version", 1),
	)

	if contextLogger == nil {
		t.Error("WithContext() returned nil")
	}

	// Verify it's a different logger instance
	if contextLogger == logger {
		t.Error("WithContext() should return a new logger instance")
	}
}

func TestConfig_Struct(t *testing.T) {
	cfg := Config{
		Level:       "debug",
		Development: true,
		Encoding:    "json",
	}

	if cfg.Level != "debug" {
		t.Errorf("Config.Level = %v, want %v", cfg.Level, "debug")
	}
	if !cfg.Development {
		t.Error("Config.Development should be true")
	}
	if cfg.Encoding != "json" {
		t.Errorf("Config.Encoding = %v, want %v", cfg.Encoding, "json")
	}
}

func TestNew_DevelopmentVsProduction(t *testing.T) {
	devLogger, err := New(Config{
		Level:       "debug",
		Development: true,
	})
	if err != nil {
		t.Fatalf("Failed to create dev logger: %v", err)
	}
	defer devLogger.Sync()

	prodLogger, err := New(Config{
		Level:       "info",
		Development: false,
	})
	if err != nil {
		t.Fatalf("Failed to create prod logger: %v", err)
	}
	defer prodLogger.Sync()

	// Both should be non-nil and functional
	if devLogger == nil {
		t.Error("Dev logger is nil")
	}
	if prodLogger == nil {
		t.Error("Prod logger is nil")
	}
}

func TestLoggerIntegration(t *testing.T) {
	// Test that the logger can actually log without panicking
	logger, err := New(Config{
		Level:       "debug",
		Development: true,
		Encoding:    "console",
	})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// These should not panic
	logger.Debug("debug message", zap.String("key", "value"))
	logger.Info("info message", zap.Int("count", 42))
	logger.Warn("warn message", zap.Bool("flag", true))
	logger.Error("error message", zap.Error(nil))
}

func TestWithContext_MultipleFields(t *testing.T) {
	logger, _ := New(Config{Level: "info", Development: true})
	defer logger.Sync()

	// Test with multiple fields
	fields := []zap.Field{
		zap.String("request_id", "abc-123"),
		zap.String("user_id", "user-456"),
		zap.String("trace_id", "trace-789"),
		zap.Int64("timestamp", 1234567890),
		zap.Float64("duration", 0.123),
		zap.Bool("success", true),
	}

	contextLogger := WithContext(logger, fields...)
	if contextLogger == nil {
		t.Error("WithContext() with multiple fields returned nil")
	}
}

func TestWithContext_EmptyFields(t *testing.T) {
	logger, _ := New(Config{Level: "info", Development: true})
	defer logger.Sync()

	contextLogger := WithContext(logger)
	if contextLogger == nil {
		t.Error("WithContext() with no fields returned nil")
	}
}

// Test zapcore level parsing
func TestZapCoreLevelParsing(t *testing.T) {
	levels := map[string]zapcore.Level{
		"debug":  zapcore.DebugLevel,
		"info":   zapcore.InfoLevel,
		"warn":   zapcore.WarnLevel,
		"error":  zapcore.ErrorLevel,
		"dpanic": zapcore.DPanicLevel,
		"panic":  zapcore.PanicLevel,
		"fatal":  zapcore.FatalLevel,
	}

	for str, expected := range levels {
		level, err := zapcore.ParseLevel(str)
		if err != nil {
			t.Errorf("ParseLevel(%s) error = %v", str, err)
			continue
		}
		if level != expected {
			t.Errorf("ParseLevel(%s) = %v, want %v", str, level, expected)
		}
	}
}

// Benchmarks
func BenchmarkNew(b *testing.B) {
	cfg := Config{
		Level:       "info",
		Development: false,
		Encoding:    "json",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger, _ := New(cfg)
		logger.Sync()
	}
}

func BenchmarkDefault(b *testing.B) {
	for i := 0; i < b.N; i++ {
		logger := Default()
		logger.Sync()
	}
}

func BenchmarkWithContext(b *testing.B) {
	logger, _ := New(Config{Level: "info", Development: false})
	defer logger.Sync()
	fields := []zap.Field{
		zap.String("key", "value"),
		zap.Int("count", 42),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		WithContext(logger, fields...)
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	logger, _ := New(Config{Level: "info", Development: false, Encoding: "json"})
	defer logger.Sync()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", zap.String("key", "value"))
	}
}
