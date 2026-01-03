package testutil

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testIDCounter is used to generate unique test IDs
var testIDCounter uint64

// TestConfig holds test configuration
type TestConfig struct {
	RedisAddr    string
	MySQLDSN     string
	PostgresDSN  string
	MongoURI     string
	MongoDB      string
	UseRealRedis bool
	UseRealMySQL bool
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() TestConfig {
	redisAddr := os.Getenv("TEST_REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6380"
	}

	mysqlDSN := os.Getenv("TEST_MYSQL_DSN")
	if mysqlDSN == "" {
		mysqlDSN = "arcana_test:arcana_test@tcp(localhost:3307)/arcana_test?charset=utf8mb4&parseTime=True&loc=Local"
	}

	postgresDSN := os.Getenv("TEST_POSTGRES_DSN")
	if postgresDSN == "" {
		postgresDSN = "host=localhost port=5433 user=arcana_test password=arcana_test dbname=arcana_test sslmode=disable"
	}

	mongoURI := os.Getenv("TEST_MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://arcana_test:arcana_test@localhost:27018/?authSource=admin"
	}

	mongoDB := os.Getenv("TEST_MONGO_DB")
	if mongoDB == "" {
		mongoDB = "arcana_test"
	}

	return TestConfig{
		RedisAddr:    redisAddr,
		MySQLDSN:     mysqlDSN,
		PostgresDSN:  postgresDSN,
		MongoURI:     mongoURI,
		MongoDB:      mongoDB,
		UseRealRedis: os.Getenv("TEST_USE_REAL_REDIS") == "true",
		UseRealMySQL: os.Getenv("TEST_USE_REAL_MYSQL") == "true",
	}
}

// NewTestLogger creates a test logger
func NewTestLogger(t *testing.T) *zap.Logger {
	return zaptest.NewLogger(t)
}

// NewNopLogger creates a no-op logger for benchmarks
func NewNopLogger() *zap.Logger {
	return zap.NewNop()
}

// NewTestRedisClient creates a Redis client for testing
func NewTestRedisClient(t *testing.T, config TestConfig) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
		DB:   15, // Use DB 15 for tests to avoid conflicts
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// Clean up test database
	client.FlushDB(ctx)

	t.Cleanup(func() {
		client.FlushDB(context.Background())
		client.Close()
	})

	return client
}

// NewTestMySQLDB creates a MySQL connection for testing
func NewTestMySQLDB(t *testing.T, config TestConfig) *gorm.DB {
	db, err := gorm.Open(mysql.Open(config.MySQLDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	t.Cleanup(func() {
		sqlDB.Close()
	})

	return db
}

// CleanupRedisKeys removes keys matching pattern
func CleanupRedisKeys(ctx context.Context, client *redis.Client, pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// WaitForCondition waits for a condition to be true
func WaitForCondition(t *testing.T, timeout time.Duration, condition func() bool, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Timeout waiting for condition: %s", message)
}

// AssertEventually asserts that a condition becomes true within timeout
func AssertEventually(t *testing.T, timeout time.Duration, condition func() bool, msgAndArgs ...interface{}) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("Condition never became true: %v", msgAndArgs)
	return false
}

// GenerateTestID generates a unique test ID using an atomic counter
func GenerateTestID() string {
	id := atomic.AddUint64(&testIDCounter, 1)
	return fmt.Sprintf("test-%d-%d", time.Now().UnixNano(), id)
}

// SkipIfShort skips the test if running in short mode
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}
}

// SkipIfNoRedis skips the test if Redis is not available
func SkipIfNoRedis(t *testing.T) {
	config := DefaultTestConfig()
	client := redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
	}
}

// SkipIfNoMySQL skips the test if MySQL is not available
func SkipIfNoMySQL(t *testing.T) {
	config := DefaultTestConfig()
	db, err := gorm.Open(mysql.Open(config.MySQLDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skip("MySQL not available")
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

// NewTestPostgresDB creates a PostgreSQL connection for testing
func NewTestPostgresDB(t *testing.T, config TestConfig) *gorm.DB {
	db, err := gorm.Open(postgres.Open(config.PostgresDSN), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	t.Cleanup(func() {
		sqlDB.Close()
	})

	return db
}

// SkipIfNoPostgres skips the test if PostgreSQL is not available
func SkipIfNoPostgres(t *testing.T) {
	config := DefaultTestConfig()
	db, err := gorm.Open(postgres.Open(config.PostgresDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skip("PostgreSQL not available")
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

// NewTestMongoDB creates a MongoDB connection for testing
func NewTestMongoDB(t *testing.T, config TestConfig) (*mongo.Client, *mongo.Database) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(config.MongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("MongoDB ping failed: %v", err)
	}

	db := client.Database(config.MongoDB)

	t.Cleanup(func() {
		// Drop test database
		db.Drop(context.Background())
		client.Disconnect(context.Background())
	})

	return client, db
}

// SkipIfNoMongo skips the test if MongoDB is not available
func SkipIfNoMongo(t *testing.T) {
	config := DefaultTestConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(config.MongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		t.Skip("MongoDB not available")
	}
	defer client.Disconnect(context.Background())

	if err := client.Ping(ctx, nil); err != nil {
		t.Skip("MongoDB not available")
	}
}
