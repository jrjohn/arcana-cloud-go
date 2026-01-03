package di

import (
	"go.uber.org/fx"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
)

// ConfigModule provides configuration dependencies
var ConfigModule = fx.Module("config",
	fx.Provide(
		config.Load,
		provideAppConfig,
		provideServerConfig,
		provideGRPCConfig,
		provideDatabaseConfig,
		provideRedisConfig,
		provideJWTConfig,
		provideDeploymentConfig,
		providePluginConfig,
		provideSSRConfig,
	),
)

func provideAppConfig(cfg *config.Config) *config.AppConfig {
	return &cfg.App
}

func provideServerConfig(cfg *config.Config) *config.ServerConfig {
	return &cfg.Server
}

func provideGRPCConfig(cfg *config.Config) *config.GRPCConfig {
	return &cfg.GRPC
}

func provideDatabaseConfig(cfg *config.Config) *config.DatabaseConfig {
	return &cfg.Database
}

func provideRedisConfig(cfg *config.Config) *config.RedisConfig {
	return &cfg.Redis
}

func provideJWTConfig(cfg *config.Config) *config.JWTConfig {
	return &cfg.JWT
}

func provideDeploymentConfig(cfg *config.Config) *config.DeploymentConfig {
	return &cfg.Deployment
}

func providePluginConfig(cfg *config.Config) *config.PluginConfig {
	return &cfg.Plugin
}

func provideSSRConfig(cfg *config.Config) *config.SSRConfig {
	return &cfg.SSR
}
