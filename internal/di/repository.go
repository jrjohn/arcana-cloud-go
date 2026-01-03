package di

import (
	"go.uber.org/fx"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository/impl"
)

// RepositoryModule provides repository dependencies.
// Repositories now delegate to the DAO layer for database operations.
var RepositoryModule = fx.Module("repository",
	fx.Provide(
		provideUserRepository,
		provideRefreshTokenRepository,
		providePluginRepository,
		providePluginExtensionRepository,
	),
)

// provideUserRepository creates a UserRepository that delegates to UserDAO.
func provideUserRepository(userDAO dao.UserDAO) repository.UserRepository {
	return impl.NewUserRepository(userDAO)
}

// provideRefreshTokenRepository creates a RefreshTokenRepository that delegates to RefreshTokenDAO.
func provideRefreshTokenRepository(refreshTokenDAO dao.RefreshTokenDAO) repository.RefreshTokenRepository {
	return impl.NewRefreshTokenRepository(refreshTokenDAO)
}

// providePluginRepository creates a PluginRepository that delegates to PluginDAO.
func providePluginRepository(pluginDAO dao.PluginDAO) repository.PluginRepository {
	return impl.NewPluginRepository(pluginDAO)
}

// providePluginExtensionRepository creates a PluginExtensionRepository that delegates to PluginExtensionDAO.
func providePluginExtensionRepository(extensionDAO dao.PluginExtensionDAO) repository.PluginExtensionRepository {
	return impl.NewPluginExtensionRepository(extensionDAO)
}
