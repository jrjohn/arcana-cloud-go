package di

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// SQLDatabase wraps *gorm.DB for SQL databases (MySQL, PostgreSQL).
// DB may be nil if MongoDB is configured.
type SQLDatabase struct {
	DB *gorm.DB
}

// MongoDatabase wraps *mongo.Database for MongoDB.
// DB may be nil if a SQL database is configured.
type MongoDatabase struct {
	DB     *mongo.Database
	Client *mongo.Client
}

// DatabaseModule provides database dependencies based on config
var DatabaseModule = fx.Module("database",
	fx.Provide(
		provideSQLDatabase,
		provideMongoDatabase,
	),
	fx.Invoke(runMigrations),
)

// provideSQLDatabase creates a GORM database connection for SQL databases.
func provideSQLDatabase(lc fx.Lifecycle, cfg *config.DatabaseConfig, logger *zap.Logger) (*SQLDatabase, error) {
	// Return nil DB if MongoDB is configured
	if cfg.IsMongoDB() {
		logger.Info("MongoDB configured, skipping SQL database")
		return &SQLDatabase{DB: nil}, nil
	}

	var dialector gorm.Dialector
	switch cfg.Driver {
	case string(config.DriverMySQL):
		dialector = mysql.Open(cfg.DSN())
	case string(config.DriverPostgres):
		dialector = postgres.Open(cfg.DSN())
	default:
		return nil, fmt.Errorf("unsupported SQL driver: %s", cfg.Driver)
	}

	logger.Info("Connecting to SQL database",
		zap.String("driver", cfg.Driver),
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
	)

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Register lifecycle hooks
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("Closing SQL database connection")
			return sqlDB.Close()
		},
	})

	return &SQLDatabase{DB: db}, nil
}

// provideMongoDatabase creates a MongoDB database connection.
func provideMongoDatabase(lc fx.Lifecycle, cfg *config.DatabaseConfig, logger *zap.Logger) (*MongoDatabase, error) {
	// Return nil DB if SQL is configured
	if !cfg.IsMongoDB() {
		logger.Info("SQL database configured, skipping MongoDB")
		return &MongoDatabase{DB: nil, Client: nil}, nil
	}

	logger.Info("Connecting to MongoDB",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name),
	)

	clientOpts := options.Client().ApplyURI(cfg.MongoURI())
	client, err := mongo.Connect(context.Background(), clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(context.Background(), nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.Name)

	// Register lifecycle hooks
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			logger.Info("Closing MongoDB connection")
			return client.Disconnect(ctx)
		},
	})

	return &MongoDatabase{DB: db, Client: client}, nil
}

// runMigrations runs database migrations based on the configured driver.
func runMigrations(sqlDB *SQLDatabase, mongoDB *MongoDatabase, cfg *config.DatabaseConfig, logger *zap.Logger) error {
	if sqlDB.DB != nil {
		logger.Info("Running SQL database migrations")
		return sqlDB.DB.AutoMigrate(
			&entity.User{},
			&entity.RefreshToken{},
			&entity.Plugin{},
			&entity.PluginExtension{},
		)
	}

	if mongoDB.DB != nil {
		logger.Info("Creating MongoDB indexes")
		return createMongoIndexes(mongoDB.DB, logger)
	}

	return nil
}

// createMongoIndexes creates necessary indexes for MongoDB collections.
func createMongoIndexes(db *mongo.Database, logger *zap.Logger) error {
	ctx := context.Background()

	// Users collection indexes
	usersCollection := db.Collection("users")
	userIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "numeric_id", Value: 1}},
		},
	}
	if _, err := usersCollection.Indexes().CreateMany(ctx, userIndexes); err != nil {
		logger.Error("Failed to create user indexes", zap.Error(err))
		return err
	}

	// Plugins collection indexes
	pluginsCollection := db.Collection("plugins")
	pluginIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "key", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "numeric_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "state", Value: 1}},
		},
	}
	if _, err := pluginsCollection.Indexes().CreateMany(ctx, pluginIndexes); err != nil {
		logger.Error("Failed to create plugin indexes", zap.Error(err))
		return err
	}

	// Refresh tokens collection indexes
	tokensCollection := db.Collection("refresh_tokens")
	tokenIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "expires_at", Value: 1}},
		},
	}
	if _, err := tokensCollection.Indexes().CreateMany(ctx, tokenIndexes); err != nil {
		logger.Error("Failed to create refresh token indexes", zap.Error(err))
		return err
	}

	// Plugin extensions collection indexes
	extensionsCollection := db.Collection("plugin_extensions")
	extensionIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "plugin_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "numeric_id", Value: 1}},
		},
	}
	if _, err := extensionsCollection.Indexes().CreateMany(ctx, extensionIndexes); err != nil {
		logger.Error("Failed to create plugin extension indexes", zap.Error(err))
		return err
	}

	// Counters collection for auto-increment IDs
	countersCollection := db.Collection("counters")
	counterIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}
	if _, err := countersCollection.Indexes().CreateMany(ctx, counterIndexes); err != nil {
		logger.Error("Failed to create counter indexes", zap.Error(err))
		return err
	}

	logger.Info("MongoDB indexes created successfully")
	return nil
}
