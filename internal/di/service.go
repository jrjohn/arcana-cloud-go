package di

import (
	"time"

	"go.uber.org/fx"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	serviceimpl "github.com/jrjohn/arcana-cloud-go/internal/domain/service/impl"
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
	return serviceimpl.NewAuthService(userRepo, refreshTokenRepo, jwtProvider, passwordHasher)
}

func provideUserService(
	userRepo repository.UserRepository,
	passwordHasher *security.PasswordHasher,
) service.UserService {
	return serviceimpl.NewUserService(userRepo, passwordHasher)
}

func providePluginService(
	pluginRepo repository.PluginRepository,
	extensionRepo repository.PluginExtensionRepository,
	cfg *config.PluginConfig,
) service.PluginService {
	return serviceimpl.NewPluginService(pluginRepo, extensionRepo, cfg.PluginsDirectory)
}

func provideSSRService(cfg *config.SSRConfig) service.SSRService {
	return serviceimpl.NewSSRService(cfg.CacheEnabled, time.Duration(cfg.CacheTTL)*time.Second)
}
