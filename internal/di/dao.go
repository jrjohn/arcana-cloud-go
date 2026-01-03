package di

import (
	"go.uber.org/fx"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao"
	gormdao "github.com/jrjohn/arcana-cloud-go/internal/domain/dao/gorm"
	mongodao "github.com/jrjohn/arcana-cloud-go/internal/domain/dao/mongo"
)

// DAOModule provides DAO dependencies based on database driver configuration.
// It automatically selects the appropriate DAO implementation (GORM or MongoDB)
// based on the configured database driver.
var DAOModule = fx.Module("dao",
	fx.Provide(
		provideMongoIDCounter,
		provideUserDAO,
		provideRefreshTokenDAO,
		providePluginDAO,
		providePluginExtensionDAO,
	),
)

// provideMongoIDCounter creates an ID counter for MongoDB.
// Returns nil if SQL database is configured.
func provideMongoIDCounter(mongoDB *MongoDatabase) *mongodao.IDCounter {
	if mongoDB.DB == nil {
		return nil
	}
	return mongodao.NewIDCounter(mongoDB.DB)
}

// provideUserDAO creates a UserDAO based on the configured database driver.
func provideUserDAO(
	cfg *config.DatabaseConfig,
	sqlDB *SQLDatabase,
	mongoDB *MongoDatabase,
	idCounter *mongodao.IDCounter,
) dao.UserDAO {
	if cfg.IsMongoDB() {
		return mongodao.NewUserDAO(mongoDB.DB, idCounter)
	}
	return gormdao.NewUserDAO(sqlDB.DB)
}

// provideRefreshTokenDAO creates a RefreshTokenDAO based on the configured database driver.
func provideRefreshTokenDAO(
	cfg *config.DatabaseConfig,
	sqlDB *SQLDatabase,
	mongoDB *MongoDatabase,
	idCounter *mongodao.IDCounter,
	userDAO dao.UserDAO,
) dao.RefreshTokenDAO {
	if cfg.IsMongoDB() {
		return mongodao.NewRefreshTokenDAO(mongoDB.DB, idCounter, userDAO)
	}
	return gormdao.NewRefreshTokenDAO(sqlDB.DB)
}

// providePluginDAO creates a PluginDAO based on the configured database driver.
func providePluginDAO(
	cfg *config.DatabaseConfig,
	sqlDB *SQLDatabase,
	mongoDB *MongoDatabase,
	idCounter *mongodao.IDCounter,
) dao.PluginDAO {
	if cfg.IsMongoDB() {
		return mongodao.NewPluginDAO(mongoDB.DB, idCounter)
	}
	return gormdao.NewPluginDAO(sqlDB.DB)
}

// providePluginExtensionDAO creates a PluginExtensionDAO based on the configured database driver.
func providePluginExtensionDAO(
	cfg *config.DatabaseConfig,
	sqlDB *SQLDatabase,
	mongoDB *MongoDatabase,
	idCounter *mongodao.IDCounter,
) dao.PluginExtensionDAO {
	if cfg.IsMongoDB() {
		return mongodao.NewPluginExtensionDAO(mongoDB.DB, idCounter)
	}
	return gormdao.NewPluginExtensionDAO(sqlDB.DB)
}
