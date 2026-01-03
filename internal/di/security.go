package di

import (
	"go.uber.org/fx"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

// SecurityModule provides security-related dependencies
var SecurityModule = fx.Module("security",
	fx.Provide(
		provideJWTProvider,
		providePasswordHasher,
		provideSecurityService,
	),
)

func provideJWTProvider(cfg *config.JWTConfig) *security.JWTProvider {
	return security.NewJWTProvider(cfg)
}

func providePasswordHasher() *security.PasswordHasher {
	return security.NewPasswordHasher()
}

func provideSecurityService(jwtProvider *security.JWTProvider) *security.SecurityService {
	return security.NewSecurityService(jwtProvider)
}
