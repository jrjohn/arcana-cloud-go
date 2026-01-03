package config

import (
	"os"
	"testing"
	"time"
)

func TestDeploymentMode_Constants(t *testing.T) {
	tests := []struct {
		name     string
		mode     DeploymentMode
		expected string
	}{
		{"monolithic", DeploymentMonolithic, "monolithic"},
		{"layered", DeploymentLayered, "layered"},
		{"kubernetes", DeploymentKubernetes, "kubernetes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.mode) != tt.expected {
				t.Errorf("DeploymentMode = %v, want %v", tt.mode, tt.expected)
			}
		})
	}
}

func TestDeploymentLayer_Constants(t *testing.T) {
	tests := []struct {
		name     string
		layer    DeploymentLayer
		expected string
	}{
		{"controller", LayerController, "controller"},
		{"service", LayerService, "service"},
		{"repository", LayerRepository, "repository"},
		{"all", LayerAll, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.layer) != tt.expected {
				t.Errorf("DeploymentLayer = %v, want %v", tt.layer, tt.expected)
			}
		})
	}
}

func TestCommunicationProtocol_Constants(t *testing.T) {
	tests := []struct {
		name     string
		protocol CommunicationProtocol
		expected string
	}{
		{"http", ProtocolHTTP, "http"},
		{"grpc", ProtocolGRPC, "grpc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.protocol) != tt.expected {
				t.Errorf("CommunicationProtocol = %v, want %v", tt.protocol, tt.expected)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				JWT:      JWTConfig{Secret: "test-secret"},
				Database: DatabaseConfig{Name: "test-db"},
			},
			wantErr: false,
		},
		{
			name: "missing JWT secret",
			config: Config{
				JWT:      JWTConfig{Secret: ""},
				Database: DatabaseConfig{Name: "test-db"},
			},
			wantErr: true,
			errMsg:  "JWT secret is required",
		},
		{
			name: "missing database name",
			config: Config{
				JWT:      JWTConfig{Secret: "test-secret"},
				Database: DatabaseConfig{Name: ""},
			},
			wantErr: true,
			errMsg:  "database name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Config.Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestDatabaseConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		config   DatabaseConfig
		expected string
	}{
		{
			name: "mysql DSN",
			config: DatabaseConfig{
				Driver:   "mysql",
				Host:     "localhost",
				Port:     3306,
				Name:     "testdb",
				User:     "root",
				Password: "password",
			},
			expected: "root:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "postgres DSN",
			config: DatabaseConfig{
				Driver:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Name:     "testdb",
				User:     "postgres",
				Password: "password",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=testdb sslmode=disable",
		},
		{
			name: "unknown driver returns empty",
			config: DatabaseConfig{
				Driver: "sqlite",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.DSN(); got != tt.expected {
				t.Errorf("DatabaseConfig.DSN() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDeploymentConfig_IsControllerLayer(t *testing.T) {
	tests := []struct {
		name     string
		config   DeploymentConfig
		expected bool
	}{
		{
			name:     "LayerAll returns true",
			config:   DeploymentConfig{Layer: LayerAll},
			expected: true,
		},
		{
			name:     "LayerController returns true",
			config:   DeploymentConfig{Layer: LayerController},
			expected: true,
		},
		{
			name:     "LayerService returns false",
			config:   DeploymentConfig{Layer: LayerService},
			expected: false,
		},
		{
			name:     "LayerRepository returns false",
			config:   DeploymentConfig{Layer: LayerRepository},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsControllerLayer(); got != tt.expected {
				t.Errorf("DeploymentConfig.IsControllerLayer() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDeploymentConfig_IsServiceLayer(t *testing.T) {
	tests := []struct {
		name     string
		config   DeploymentConfig
		expected bool
	}{
		{
			name:     "LayerAll returns true",
			config:   DeploymentConfig{Layer: LayerAll},
			expected: true,
		},
		{
			name:     "LayerService returns true",
			config:   DeploymentConfig{Layer: LayerService},
			expected: true,
		},
		{
			name:     "LayerController returns false",
			config:   DeploymentConfig{Layer: LayerController},
			expected: false,
		},
		{
			name:     "LayerRepository returns false",
			config:   DeploymentConfig{Layer: LayerRepository},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsServiceLayer(); got != tt.expected {
				t.Errorf("DeploymentConfig.IsServiceLayer() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDeploymentConfig_IsRepositoryLayer(t *testing.T) {
	tests := []struct {
		name     string
		config   DeploymentConfig
		expected bool
	}{
		{
			name:     "LayerAll returns true",
			config:   DeploymentConfig{Layer: LayerAll},
			expected: true,
		},
		{
			name:     "LayerRepository returns true",
			config:   DeploymentConfig{Layer: LayerRepository},
			expected: true,
		},
		{
			name:     "LayerController returns false",
			config:   DeploymentConfig{Layer: LayerController},
			expected: false,
		},
		{
			name:     "LayerService returns false",
			config:   DeploymentConfig{Layer: LayerService},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsRepositoryLayer(); got != tt.expected {
				t.Errorf("DeploymentConfig.IsRepositoryLayer() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDeploymentConfig_IsGRPC(t *testing.T) {
	tests := []struct {
		name     string
		config   DeploymentConfig
		expected bool
	}{
		{
			name:     "gRPC protocol",
			config:   DeploymentConfig{Protocol: ProtocolGRPC},
			expected: true,
		},
		{
			name:     "HTTP protocol",
			config:   DeploymentConfig{Protocol: ProtocolHTTP},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsGRPC(); got != tt.expected {
				t.Errorf("DeploymentConfig.IsGRPC() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoad_WithEnvVars(t *testing.T) {
	// Save and restore env vars
	envVars := []string{
		"ARCANA_JWT_SECRET",
		"ARCANA_DATABASE_NAME",
		"ARCANA_APP_NAME",
		"ARCANA_SERVER_PORT",
	}
	savedVars := make(map[string]string)
	for _, v := range envVars {
		savedVars[v] = os.Getenv(v)
	}
	defer func() {
		for k, v := range savedVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set required env vars
	os.Setenv("ARCANA_JWT_SECRET", "test-secret-key")
	os.Setenv("ARCANA_DATABASE_NAME", "test-db")
	os.Setenv("ARCANA_APP_NAME", "test-app")
	os.Setenv("ARCANA_SERVER_PORT", "9000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.JWT.Secret != "test-secret-key" {
		t.Errorf("JWT.Secret = %v, want test-secret-key", cfg.JWT.Secret)
	}
	if cfg.Database.Name != "test-db" {
		t.Errorf("Database.Name = %v, want test-db", cfg.Database.Name)
	}
}

func TestConfig_Structs(t *testing.T) {
	// Test AppConfig
	appCfg := AppConfig{
		Name:        "test-app",
		Version:     "1.0.0",
		Environment: "development",
		Debug:       true,
	}
	if appCfg.Name != "test-app" {
		t.Errorf("AppConfig.Name = %v, want test-app", appCfg.Name)
	}

	// Test ServerConfig
	serverCfg := ServerConfig{
		Host:         "0.0.0.0",
		Port:         8080,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if serverCfg.Port != 8080 {
		t.Errorf("ServerConfig.Port = %v, want 8080", serverCfg.Port)
	}

	// Test GRPCConfig
	grpcCfg := GRPCConfig{
		Host:       "0.0.0.0",
		Port:       9090,
		TLSEnabled: true,
		CertFile:   "/path/to/cert",
		KeyFile:    "/path/to/key",
		CAFile:     "/path/to/ca",
	}
	if grpcCfg.Port != 9090 {
		t.Errorf("GRPCConfig.Port = %v, want 9090", grpcCfg.Port)
	}
	if !grpcCfg.TLSEnabled {
		t.Error("GRPCConfig.TLSEnabled should be true")
	}

	// Test RedisConfig
	redisCfg := RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "secret",
		DB:       0,
	}
	if redisCfg.Port != 6379 {
		t.Errorf("RedisConfig.Port = %v, want 6379", redisCfg.Port)
	}

	// Test JWTConfig
	jwtCfg := JWTConfig{
		Secret:               "secret-key",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 30 * 24 * time.Hour,
		Issuer:               "test-issuer",
	}
	if jwtCfg.AccessTokenDuration != time.Hour {
		t.Errorf("JWTConfig.AccessTokenDuration = %v, want %v", jwtCfg.AccessTokenDuration, time.Hour)
	}

	// Test PluginConfig
	pluginCfg := PluginConfig{
		Enabled:          true,
		PluginsDirectory: "./plugins",
		AutoLoad:         true,
		HotReload:        true,
	}
	if !pluginCfg.Enabled {
		t.Error("PluginConfig.Enabled should be true")
	}

	// Test SSRConfig
	ssrCfg := SSRConfig{
		Enabled:      true,
		ReactPath:    "./react",
		AngularPath:  "./angular",
		CacheEnabled: true,
		CacheTTL:     3600,
	}
	if ssrCfg.CacheTTL != 3600 {
		t.Errorf("SSRConfig.CacheTTL = %v, want 3600", ssrCfg.CacheTTL)
	}

	// Test DatabaseConfig
	dbCfg := DatabaseConfig{
		Driver:          "mysql",
		Host:            "localhost",
		Port:            3306,
		Name:            "arcana",
		User:            "root",
		Password:        "password",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
	}
	if dbCfg.MaxOpenConns != 25 {
		t.Errorf("DatabaseConfig.MaxOpenConns = %v, want 25", dbCfg.MaxOpenConns)
	}
}

func TestConfig_FullStruct(t *testing.T) {
	cfg := Config{
		App: AppConfig{
			Name:        "arcana-cloud-go",
			Version:     "1.0.0",
			Environment: "test",
			Debug:       true,
		},
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Driver: "mysql",
			Name:   "testdb",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		JWT: JWTConfig{
			Secret: "test-secret",
		},
		Deployment: DeploymentConfig{
			Mode:     DeploymentMonolithic,
			Layer:    LayerAll,
			Protocol: ProtocolGRPC,
		},
		Plugin: PluginConfig{
			Enabled: true,
		},
		SSR: SSRConfig{
			Enabled: true,
		},
		GRPC: GRPCConfig{
			Port: 9090,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Config.Validate() error = %v", err)
	}

	if cfg.App.Name != "arcana-cloud-go" {
		t.Errorf("Config.App.Name = %v, want arcana-cloud-go", cfg.App.Name)
	}
}

// Benchmarks
func BenchmarkDatabaseConfig_DSN_MySQL(b *testing.B) {
	cfg := DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Name:     "testdb",
		User:     "root",
		Password: "password",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.DSN()
	}
}

func BenchmarkDatabaseConfig_DSN_Postgres(b *testing.B) {
	cfg := DatabaseConfig{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		Name:     "testdb",
		User:     "postgres",
		Password: "password",
		SSLMode:  "disable",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.DSN()
	}
}

func BenchmarkConfig_Validate(b *testing.B) {
	cfg := Config{
		JWT:      JWTConfig{Secret: "test-secret"},
		Database: DatabaseConfig{Name: "test-db"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.Validate()
	}
}
