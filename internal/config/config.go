package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// DeploymentMode represents the deployment configuration
type DeploymentMode string

const (
	DeploymentMonolithic  DeploymentMode = "monolithic"
	DeploymentLayered     DeploymentMode = "layered"
	DeploymentKubernetes  DeploymentMode = "kubernetes"
)

// DeploymentLayer represents which layer this instance serves
type DeploymentLayer string

const (
	LayerController DeploymentLayer = "controller"
	LayerService    DeploymentLayer = "service"
	LayerRepository DeploymentLayer = "repository"
	LayerAll        DeploymentLayer = ""
)

// CommunicationProtocol represents the inter-service communication protocol
type CommunicationProtocol string

const (
	ProtocolHTTP CommunicationProtocol = "http"
	ProtocolGRPC CommunicationProtocol = "grpc"
)

// DatabaseDriver represents supported database drivers
type DatabaseDriver string

const (
	DriverMySQL    DatabaseDriver = "mysql"
	DriverPostgres DatabaseDriver = "postgres"
	DriverMongoDB  DatabaseDriver = "mongodb"
)

// Config holds all application configuration
type Config struct {
	App        AppConfig        `mapstructure:"app"`
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Deployment DeploymentConfig `mapstructure:"deployment"`
	Plugin     PluginConfig     `mapstructure:"plugin"`
	SSR        SSRConfig        `mapstructure:"ssr"`
	GRPC       GRPCConfig       `mapstructure:"grpc"`
}

// AppConfig holds application-level settings
type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
	Debug       bool   `mapstructure:"debug"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// GRPCConfig holds gRPC server settings
type GRPCConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	TLSEnabled bool   `mapstructure:"tls_enabled"`
	CertFile   string `mapstructure:"cert_file"`
	KeyFile    string `mapstructure:"key_file"`
	CAFile     string `mapstructure:"ca_file"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Name            string        `mapstructure:"name"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	// MongoDB-specific settings
	AuthSource string `mapstructure:"auth_source"`
	ReplicaSet string `mapstructure:"replica_set"`
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig holds JWT token settings
type JWTConfig struct {
	Secret               string        `mapstructure:"secret"`
	AccessTokenDuration  time.Duration `mapstructure:"access_token_duration"`
	RefreshTokenDuration time.Duration `mapstructure:"refresh_token_duration"`
	Issuer               string        `mapstructure:"issuer"`
}

// DeploymentConfig holds deployment-specific settings
type DeploymentConfig struct {
	Mode     DeploymentMode        `mapstructure:"mode"`
	Layer    DeploymentLayer       `mapstructure:"layer"`
	Protocol CommunicationProtocol `mapstructure:"protocol"`
}

// PluginConfig holds plugin system settings
type PluginConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	PluginsDirectory string `mapstructure:"plugins_directory"`
	AutoLoad         bool   `mapstructure:"auto_load"`
	HotReload        bool   `mapstructure:"hot_reload"`
}

// SSRConfig holds server-side rendering settings
type SSRConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	ReactPath    string `mapstructure:"react_path"`
	AngularPath  string `mapstructure:"angular_path"`
	CacheEnabled bool   `mapstructure:"cache_enabled"`
	CacheTTL     int    `mapstructure:"cache_ttl"`
}

// Load reads configuration from file and environment variables
func Load() (*Config, error) {
	v := viper.New()

	// Set config file details
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/arcana-cloud/")

	// Set environment variable prefix
	v.SetEnvPrefix("ARCANA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set defaults
	setDefaults(v)

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required settings
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("app.name", "arcana-cloud-go")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.environment", "development")
	v.SetDefault("app.debug", true)

	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)
	v.SetDefault("server.idle_timeout", 60*time.Second)

	// gRPC defaults
	v.SetDefault("grpc.host", "0.0.0.0")
	v.SetDefault("grpc.port", 9090)
	v.SetDefault("grpc.tls_enabled", false)

	// Database defaults
	v.SetDefault("database.driver", "mysql")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.name", "arcana_cloud")
	v.SetDefault("database.user", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// JWT defaults
	v.SetDefault("jwt.secret", os.Getenv("JWT_SECRET"))
	v.SetDefault("jwt.access_token_duration", time.Hour)
	v.SetDefault("jwt.refresh_token_duration", 30*24*time.Hour)
	v.SetDefault("jwt.issuer", "arcana-cloud")

	// Deployment defaults
	v.SetDefault("deployment.mode", DeploymentMonolithic)
	v.SetDefault("deployment.layer", LayerAll)
	v.SetDefault("deployment.protocol", ProtocolGRPC)

	// Plugin defaults
	v.SetDefault("plugin.enabled", true)
	v.SetDefault("plugin.plugins_directory", "./plugins")
	v.SetDefault("plugin.auto_load", true)
	v.SetDefault("plugin.hot_reload", true)

	// SSR defaults
	v.SetDefault("ssr.enabled", true)
	v.SetDefault("ssr.react_path", "./arcana-web/react-app")
	v.SetDefault("ssr.angular_path", "./arcana-web/angular-app")
	v.SetDefault("ssr.cache_enabled", true)
	v.SetDefault("ssr.cache_ttl", 3600)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	return nil
}

// DSN returns the database connection string for SQL databases.
func (c *DatabaseConfig) DSN() string {
	switch c.Driver {
	case string(DriverMySQL):
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			c.User, c.Password, c.Host, c.Port, c.Name)
	case string(DriverPostgres):
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
	default:
		return ""
	}
}

// MongoURI returns the MongoDB connection URI.
func (c *DatabaseConfig) MongoURI() string {
	if c.User != "" && c.Password != "" {
		uri := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
			c.User, c.Password, c.Host, c.Port, c.Name)
		return c.appendMongoOptions(uri)
	}
	uri := fmt.Sprintf("mongodb://%s:%d/%s", c.Host, c.Port, c.Name)
	return c.appendMongoOptions(uri)
}

// appendMongoOptions adds optional query parameters to the MongoDB URI.
func (c *DatabaseConfig) appendMongoOptions(uri string) string {
	params := []string{}
	if c.AuthSource != "" {
		params = append(params, "authSource="+c.AuthSource)
	}
	if c.ReplicaSet != "" {
		params = append(params, "replicaSet="+c.ReplicaSet)
	}
	if len(params) > 0 {
		uri += "?" + strings.Join(params, "&")
	}
	return uri
}

// IsMongoDB returns true if MongoDB driver is configured.
func (c *DatabaseConfig) IsMongoDB() bool {
	return c.Driver == string(DriverMongoDB)
}

// IsSQL returns true if a SQL driver (MySQL or PostgreSQL) is configured.
func (c *DatabaseConfig) IsSQL() bool {
	return c.Driver == string(DriverMySQL) || c.Driver == string(DriverPostgres)
}

// IsMySQL returns true if MySQL driver is configured.
func (c *DatabaseConfig) IsMySQL() bool {
	return c.Driver == string(DriverMySQL)
}

// IsPostgres returns true if PostgreSQL driver is configured.
func (c *DatabaseConfig) IsPostgres() bool {
	return c.Driver == string(DriverPostgres)
}

// IsControllerLayer checks if this instance should run the controller layer
func (c *DeploymentConfig) IsControllerLayer() bool {
	return c.Layer == LayerAll || c.Layer == LayerController
}

// IsServiceLayer checks if this instance should run the service layer
func (c *DeploymentConfig) IsServiceLayer() bool {
	return c.Layer == LayerAll || c.Layer == LayerService
}

// IsRepositoryLayer checks if this instance should run the repository layer
func (c *DeploymentConfig) IsRepositoryLayer() bool {
	return c.Layer == LayerAll || c.Layer == LayerRepository
}

// IsGRPC checks if gRPC protocol is enabled
func (c *DeploymentConfig) IsGRPC() bool {
	return c.Protocol == ProtocolGRPC
}
