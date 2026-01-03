package di

import (
	"go.uber.org/fx"

	"github.com/jrjohn/arcana-cloud-go/internal/middleware"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

// MiddlewareModule provides middleware dependencies
var MiddlewareModule = fx.Module("middleware",
	fx.Provide(provideAuthMiddleware),
)

func provideAuthMiddleware(
	jwtProvider *security.JWTProvider,
	securityService *security.SecurityService,
) *middleware.AuthMiddleware {
	return middleware.NewAuthMiddleware(jwtProvider, securityService)
}
