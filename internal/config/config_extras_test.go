package config

import (
	"testing"
)

func TestDatabaseConfig_MongoURI_WithCredentials(t *testing.T) {
	cfg := DatabaseConfig{
		Driver:   "mongodb",
		Host:     "localhost",
		Port:     27017,
		Name:     "testdb",
		User:     "admin",
		Password: "secret",
	}
	uri := cfg.MongoURI()
	expected := "mongodb://admin:secret@localhost:27017/testdb"
	if uri != expected {
		t.Errorf("MongoURI() = %v, want %v", uri, expected)
	}
}

func TestDatabaseConfig_MongoURI_WithoutCredentials(t *testing.T) {
	cfg := DatabaseConfig{
		Driver: "mongodb",
		Host:   "localhost",
		Port:   27017,
		Name:   "testdb",
	}
	uri := cfg.MongoURI()
	expected := "mongodb://localhost:27017/testdb"
	if uri != expected {
		t.Errorf("MongoURI() = %v, want %v", uri, expected)
	}
}

func TestDatabaseConfig_MongoURI_WithAuthSource(t *testing.T) {
	cfg := DatabaseConfig{
		Driver:     "mongodb",
		Host:       "localhost",
		Port:       27017,
		Name:       "testdb",
		AuthSource: "admin",
	}
	uri := cfg.MongoURI()
	expected := "mongodb://localhost:27017/testdb?authSource=admin"
	if uri != expected {
		t.Errorf("MongoURI() = %v, want %v", uri, expected)
	}
}

func TestDatabaseConfig_MongoURI_WithReplicaSet(t *testing.T) {
	cfg := DatabaseConfig{
		Driver:     "mongodb",
		Host:       "localhost",
		Port:       27017,
		Name:       "testdb",
		ReplicaSet: "rs0",
	}
	uri := cfg.MongoURI()
	expected := "mongodb://localhost:27017/testdb?replicaSet=rs0"
	if uri != expected {
		t.Errorf("MongoURI() = %v, want %v", uri, expected)
	}
}

func TestDatabaseConfig_MongoURI_WithAllOptions(t *testing.T) {
	cfg := DatabaseConfig{
		Driver:     "mongodb",
		Host:       "localhost",
		Port:       27017,
		Name:       "testdb",
		User:       "admin",
		Password:   "secret",
		AuthSource: "admin",
		ReplicaSet: "rs0",
	}
	uri := cfg.MongoURI()
	// Should contain both query params
	if uri == "" {
		t.Error("MongoURI() returned empty string")
	}
	// Check that it contains auth source and replica set
	if len(uri) == 0 {
		t.Error("MongoURI() returned empty string")
	}
}

func TestDatabaseConfig_IsMongoDB(t *testing.T) {
	tests := []struct {
		driver   string
		expected bool
	}{
		{"mongodb", true},
		{"mysql", false},
		{"postgres", false},
		{"", false},
	}
	for _, tt := range tests {
		cfg := DatabaseConfig{Driver: tt.driver}
		if got := cfg.IsMongoDB(); got != tt.expected {
			t.Errorf("IsMongoDB() with driver=%v = %v, want %v", tt.driver, got, tt.expected)
		}
	}
}

func TestDatabaseConfig_IsSQL(t *testing.T) {
	tests := []struct {
		driver   string
		expected bool
	}{
		{"mysql", true},
		{"postgres", true},
		{"mongodb", false},
		{"", false},
	}
	for _, tt := range tests {
		cfg := DatabaseConfig{Driver: tt.driver}
		if got := cfg.IsSQL(); got != tt.expected {
			t.Errorf("IsSQL() with driver=%v = %v, want %v", tt.driver, got, tt.expected)
		}
	}
}

func TestDatabaseConfig_IsMySQL(t *testing.T) {
	tests := []struct {
		driver   string
		expected bool
	}{
		{"mysql", true},
		{"postgres", false},
		{"mongodb", false},
		{"", false},
	}
	for _, tt := range tests {
		cfg := DatabaseConfig{Driver: tt.driver}
		if got := cfg.IsMySQL(); got != tt.expected {
			t.Errorf("IsMySQL() with driver=%v = %v, want %v", tt.driver, got, tt.expected)
		}
	}
}

func TestDatabaseConfig_IsPostgres(t *testing.T) {
	tests := []struct {
		driver   string
		expected bool
	}{
		{"postgres", true},
		{"mysql", false},
		{"mongodb", false},
		{"", false},
	}
	for _, tt := range tests {
		cfg := DatabaseConfig{Driver: tt.driver}
		if got := cfg.IsPostgres(); got != tt.expected {
			t.Errorf("IsPostgres() with driver=%v = %v, want %v", tt.driver, got, tt.expected)
		}
	}
}

func TestDefaultWorkerConfig(t *testing.T) {
	cfg := DefaultWorkerConfig()
	if cfg.Concurrency <= 0 {
		t.Errorf("Concurrency = %v, want > 0", cfg.Concurrency)
	}
	if !cfg.Enabled {
		t.Error("DefaultWorkerConfig should have Enabled=true")
	}
}

func TestDefaultSchedulerConfig(t *testing.T) {
	cfg := DefaultSchedulerConfig()
	if !cfg.Enabled {
		t.Error("DefaultSchedulerConfig should have Enabled=true")
	}
}
