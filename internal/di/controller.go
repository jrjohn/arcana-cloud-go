package di

import (
	"go.uber.org/fx"

	httpctrl "github.com/jrjohn/arcana-cloud-go/internal/controller/http"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/middleware"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

// ControllerModule provides HTTP controller dependencies
var ControllerModule = fx.Module("controller",
	fx.Provide(
		provideAuthController,
		provideUserController,
		providePluginController,
		provideSSRController,
	),
)

func provideAuthController(
	authService service.AuthService,
	securityService *security.SecurityService,
) *httpctrl.AuthController {
	return httpctrl.NewAuthController(authService, securityService)
}

func provideUserController(
	userService service.UserService,
	securityService *security.SecurityService,
	authMiddleware *middleware.AuthMiddleware,
) *httpctrl.UserController {
	return httpctrl.NewUserController(userService, securityService, authMiddleware)
}

func providePluginController(
	pluginService service.PluginService,
	authMiddleware *middleware.AuthMiddleware,
) *httpctrl.PluginController {
	return httpctrl.NewPluginController(pluginService, authMiddleware)
}

func provideSSRController(
	ssrService service.SSRService,
	authMiddleware *middleware.AuthMiddleware,
) *httpctrl.SSRController {
	return httpctrl.NewSSRController(ssrService, authMiddleware)
}
