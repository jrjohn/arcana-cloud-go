package di

import (
	"time"

	"go.uber.org/fx"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

// ServiceModule provides service layer dependencies
var ServiceModule = fx.Module("service",
	fx.Provide(
		provideAuthService,
		provideUserService,
		providePluginService,
		provideSSRService,
	),
)

func provideAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtProvider *security.JWTProvider,
	passwordHasher *security.PasswordHasher,
) service.AuthService {
	return service.NewAuthService(userRepo, refreshTokenRepo, jwtProvider, passwordHasher)
}

func provideUserService(
	userRepo repository.UserRepository,
	passwordHasher *security.PasswordHasher,
) service.UserService {
	return service.NewUserService(userRepo, passwordHasher)
}

func providePluginService(
	pluginRepo repository.PluginRepository,
	extensionRepo repository.PluginExtensionRepository,
	cfg *config.PluginConfig,
) service.PluginService {
	return service.NewPluginService(pluginRepo, extensionRepo, cfg.PluginsDirectory)
}

func provideSSRService(cfg *config.SSRConfig) service.SSRService {
	return service.NewSSRService(cfg.CacheEnabled, time.Duration(cfg.CacheTTL)*time.Second)
}
