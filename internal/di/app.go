package di

import (
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
)

// AppModule aggregates all application modules
var AppModule = fx.Options(
	ConfigModule,
	LoggerModule,
	DatabaseModule,
	DAOModule,          // DAO layer (between Database and Repository)
	RepositoryModule,   // Repository layer (delegates to DAO)
	SecurityModule,
	ServiceModule,
	MiddlewareModule,
	ControllerModule,
	PluginModule,
	JobsModule,         // Job worker system
	HTTPServerModule,
	GRPCServerModule,
	GRPCLayeredModule,  // Layered gRPC architecture
)

// PrintBanner prints the application startup banner
func PrintBanner(cfg *config.Config, logger *zap.Logger) {
	logger.Info("===========================================")
	logger.Info("   Arcana Cloud Go - Enterprise Platform   ")
	logger.Info("===========================================")
	logger.Info("Application Info",
		zap.String("name", cfg.App.Name),
		zap.String("version", cfg.App.Version),
		zap.String("environment", cfg.App.Environment),
	)
	logger.Info("Deployment Config",
		zap.String("mode", string(cfg.Deployment.Mode)),
		zap.String("layer", string(cfg.Deployment.Layer)),
		zap.String("protocol", string(cfg.Deployment.Protocol)),
	)
	logger.Info("===========================================")
}
