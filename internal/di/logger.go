package di

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/pkg/logger"
)

// LoggerModule provides logging dependencies
var LoggerModule = fx.Module("logger",
	fx.Provide(provideLogger),
)

func provideLogger(cfg *config.AppConfig) (*zap.Logger, error) {
	return logger.New(logger.Config{
		Level:       "debug",
		Development: cfg.Debug,
		Encoding:    "console",
	})
}
