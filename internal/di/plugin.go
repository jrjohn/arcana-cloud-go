package di

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/plugin/manager"
)

// PluginModule provides plugin system dependencies
var PluginModule = fx.Module("plugin",
	fx.Provide(providePluginManager),
	fx.Invoke(initializePluginManager),
)

func providePluginManager(cfg *config.PluginConfig, logger *zap.Logger) *manager.Manager {
	return manager.NewManager(cfg.PluginsDirectory, logger)
}

func initializePluginManager(lc fx.Lifecycle, pm *manager.Manager, cfg *config.PluginConfig, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if cfg.AutoLoad {
				logger.Info("Auto-loading plugins")
				if err := pm.LoadAll(ctx); err != nil {
					logger.Warn("Failed to auto-load plugins", zap.Error(err))
				}
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down plugin manager")
			return pm.Shutdown(ctx)
		},
	})
}
