package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds logger configuration
type Config struct {
	Level       string
	Development bool
	Encoding    string // "json" or "console"
}

// New creates a new zap logger
func New(cfg Config) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	var zapConfig zap.Config

	if cfg.Development {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	if cfg.Encoding != "" {
		zapConfig.Encoding = cfg.Encoding
	}

	zapConfig.Level = zap.NewAtomicLevelAt(level)
	zapConfig.OutputPaths = []string{"stdout"}
	zapConfig.ErrorOutputPaths = []string{"stderr"}

	return zapConfig.Build()
}

// Default creates a default logger
func Default() *zap.Logger {
	logger, err := New(Config{
		Level:       os.Getenv("LOG_LEVEL"),
		Development: os.Getenv("APP_ENV") != "production",
		Encoding:    "console",
	})
	if err != nil {
		// Fallback to a basic logger
		return zap.NewExample()
	}
	return logger
}

// WithContext returns a logger with additional context fields
func WithContext(logger *zap.Logger, fields ...zap.Field) *zap.Logger {
	return logger.With(fields...)
}
