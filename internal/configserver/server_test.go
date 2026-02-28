package configserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestDefaultConfigServerConfig(t *testing.T) {
	cfg := DefaultConfigServerConfig()
	if cfg == nil {
		t.Fatal("DefaultConfigServerConfig() returned nil")
	}
	if cfg.Enabled != false {
		t.Error("Enabled should default to false")
	}
	if cfg.Port != 8888 {
		t.Errorf("Port = %v, want 8888", cfg.Port)
	}
	if cfg.ConfigDir != "./config-repo" {
		t.Errorf("ConfigDir = %v, want ./config-repo", cfg.ConfigDir)
	}
	if !cfg.EnableWatch {
		t.Error("EnableWatch should default to true")
	}
}

func TestNewConfigServer_EmptyDir(t *testing.T) {
	dir, err := os.MkdirTemp("", "configserver-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{
		ConfigDir:   dir,
		EnableWatch: false,
	}

	server, err := NewConfigServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewConfigServer() error = %v", err)
	}
	if server == nil {
		t.Error("NewConfigServer() returned nil")
	}
}

func TestNewConfigServer_WithYAMLConfig(t *testing.T) {
	dir, err := os.MkdirTemp("", "configserver-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a YAML config file
	yamlContent := `server:
  port: 8080
database:
  host: localhost
`
	err = os.WriteFile(filepath.Join(dir, "application-default.yaml"), []byte(yamlContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{
		ConfigDir:   dir,
		EnableWatch: false,
	}

	server, err := NewConfigServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewConfigServer() error = %v", err)
	}

	config, err := server.GetConfig("application", "default")
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}
	if config == nil {
		t.Error("GetConfig() returned nil")
	}
}

func TestNewConfigServer_WithJSONConfig(t *testing.T) {
	dir, err := os.MkdirTemp("", "configserver-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create a JSON config file
	jsonContent := `{"key": "value", "num": 42}`
	err = os.WriteFile(filepath.Join(dir, "myapp-production.json"), []byte(jsonContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{
		ConfigDir:   dir,
		EnableWatch: false,
	}

	server, err := NewConfigServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewConfigServer() error = %v", err)
	}

	config, err := server.GetConfig("myapp", "production")
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}
	if v, ok := config["key"]; !ok || v != "value" {
		t.Errorf("config[key] = %v, want value", v)
	}
}

func TestConfigServer_GetConfig_NotFound(t *testing.T) {
	dir, err := os.MkdirTemp("", "configserver-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{
		ConfigDir:   dir,
		EnableWatch: false,
	}

	server, _ := NewConfigServer(cfg, logger)
	_, err = server.GetConfig("nonexistent", "default")
	if err == nil {
		t.Error("GetConfig() should return error for nonexistent config")
	}
}

func TestConfigServer_GetConfig_FallsBackToDefault(t *testing.T) {
	dir, err := os.MkdirTemp("", "configserver-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create only default profile
	yamlContent := "key: defaultValue\n"
	os.WriteFile(filepath.Join(dir, "app-default.yaml"), []byte(yamlContent), 0644)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{
		ConfigDir:   dir,
		EnableWatch: false,
	}

	server, _ := NewConfigServer(cfg, logger)

	// Request production profile, should fall back to default
	config, err := server.GetConfig("app", "production")
	if err != nil {
		t.Fatalf("GetConfig() should fallback to default, got error: %v", err)
	}
	if config["key"] != "defaultValue" {
		t.Errorf("config[key] = %v, want defaultValue", config["key"])
	}
}

func TestConfigServer_MergeWithDefault(t *testing.T) {
	dir, err := os.MkdirTemp("", "configserver-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Create default and production profiles
	defaultContent := "host: default-host\nport: 8080\n"
	prodContent := "host: prod-host\n"

	os.WriteFile(filepath.Join(dir, "app-default.yaml"), []byte(defaultContent), 0644)
	os.WriteFile(filepath.Join(dir, "app-production.yaml"), []byte(prodContent), 0644)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{
		ConfigDir:   dir,
		EnableWatch: false,
	}

	server, _ := NewConfigServer(cfg, logger)

	config, err := server.GetConfig("app", "production")
	if err != nil {
		t.Fatalf("GetConfig() error = %v", err)
	}

	// Production should override host
	if config["host"] != "prod-host" {
		t.Errorf("config[host] = %v, want prod-host", config["host"])
	}
	// Port should come from default
	if config["port"] == nil {
		t.Error("config[port] should be inherited from default")
	}
}

func TestConfigServer_Subscribe(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{
		ConfigDir:   dir,
		EnableWatch: false,
	}

	server, _ := NewConfigServer(cfg, logger)
	ch := server.Subscribe()
	if ch == nil {
		t.Error("Subscribe() returned nil channel")
	}
}

func TestConfigServer_Stop(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{
		ConfigDir:   dir,
		EnableWatch: false,
	}

	server, _ := NewConfigServer(cfg, logger)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Stop without starting should not panic
	err := server.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

// Test HTTP handlers using recorder
func TestConfigServer_HandleHealth(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{
		ConfigDir:   dir,
		EnableWatch: false,
	}

	server, _ := NewConfigServer(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleHealth() status = %v, want %v", w.Code, http.StatusOK)
	}

	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)
	if result["status"] != "UP" {
		t.Errorf("status = %v, want UP", result["status"])
	}
}

func TestConfigServer_HandleRoot(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{ConfigDir: dir, EnableWatch: false}
	server, _ := NewConfigServer(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	server.handleRoot(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleRoot() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestConfigServer_HandleConfig_Found(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	// Create test config
	yamlContent := "key: value\n"
	os.WriteFile(filepath.Join(dir, "myapp-default.yaml"), []byte(yamlContent), 0644)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{ConfigDir: dir, EnableWatch: false}
	server, _ := NewConfigServer(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/config/myapp/default", nil)
	req.URL.Path = "/config/myapp/default"
	w := httptest.NewRecorder()

	server.handleConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleConfig() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestConfigServer_HandleConfig_NotFound(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{ConfigDir: dir, EnableWatch: false}
	server, _ := NewConfigServer(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/config/", nil)
	req.URL.Path = "/config/"
	w := httptest.NewRecorder()

	server.handleConfig(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("handleConfig() status = %v, want 404", w.Code)
	}
}

func TestConfigServer_HandleRefresh_POST(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{ConfigDir: dir, EnableWatch: false}
	server, _ := NewConfigServer(cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	w := httptest.NewRecorder()

	server.handleRefresh(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleRefresh() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestConfigServer_HandleRefresh_GetMethodNotAllowed(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{ConfigDir: dir, EnableWatch: false}
	server, _ := NewConfigServer(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/refresh", nil)
	w := httptest.NewRecorder()

	server.handleRefresh(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleRefresh() with GET status = %v, want 405", w.Code)
	}
}

func TestConfigServer_HandleEncrypt_NotImplemented(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{ConfigDir: dir, EnableWatch: false}
	server, _ := NewConfigServer(cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/encrypt", nil)
	w := httptest.NewRecorder()

	server.handleEncrypt(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("handleEncrypt() status = %v, want 501", w.Code)
	}
}

func TestConfigServer_HandleEncrypt_MethodNotAllowed(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{ConfigDir: dir, EnableWatch: false}
	server, _ := NewConfigServer(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/encrypt", nil)
	w := httptest.NewRecorder()

	server.handleEncrypt(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleEncrypt() with GET status = %v, want 405", w.Code)
	}
}

func TestConfigServer_HandleDecrypt_NotImplemented(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{ConfigDir: dir, EnableWatch: false}
	server, _ := NewConfigServer(cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/decrypt", nil)
	w := httptest.NewRecorder()

	server.handleDecrypt(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("handleDecrypt() status = %v, want 501", w.Code)
	}
}

func TestConfigServer_HandleDecrypt_MethodNotAllowed(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{ConfigDir: dir, EnableWatch: false}
	server, _ := NewConfigServer(cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/decrypt", nil)
	w := httptest.NewRecorder()

	server.handleDecrypt(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleDecrypt() with GET status = %v, want 405", w.Code)
	}
}

func TestDeepMerge(t *testing.T) {
	base := map[string]interface{}{
		"key1": "base1",
		"key2": "base2",
		"nested": map[string]interface{}{
			"a": "base-a",
			"b": "base-b",
		},
	}
	override := map[string]interface{}{
		"key1": "override1",
		"key3": "override3",
		"nested": map[string]interface{}{
			"a": "override-a",
		},
	}

	result := deepMerge(base, override)

	if result["key1"] != "override1" {
		t.Errorf("key1 = %v, want override1", result["key1"])
	}
	if result["key2"] != "base2" {
		t.Errorf("key2 = %v, want base2", result["key2"])
	}
	if result["key3"] != "override3" {
		t.Errorf("key3 = %v, want override3", result["key3"])
	}
	if nested, ok := result["nested"].(map[string]interface{}); ok {
		if nested["a"] != "override-a" {
			t.Errorf("nested.a = %v, want override-a", nested["a"])
		}
		if nested["b"] != "base-b" {
			t.Errorf("nested.b = %v, want base-b", nested["b"])
		}
	} else {
		t.Error("nested should be a map")
	}
}

func TestConfigServer_WithWatch(t *testing.T) {
	dir, _ := os.MkdirTemp("", "configserver-test-*")
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{
		ConfigDir:   dir,
		EnableWatch: true,
	}

	server, err := NewConfigServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewConfigServer() with watch error = %v", err)
	}
	defer server.Stop(context.Background())

	if server == nil {
		t.Error("NewConfigServer() returned nil")
	}
}

func TestConfigServer_NonExistentDir_Created(t *testing.T) {
	dir := fmt.Sprintf("/tmp/configserver-test-%d", time.Now().UnixNano())
	defer os.RemoveAll(dir)

	logger, _ := zap.NewDevelopment()
	cfg := &ConfigServerConfig{
		ConfigDir:   dir,
		EnableWatch: false,
	}

	server, err := NewConfigServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewConfigServer() should create missing dir, got error: %v", err)
	}
	if server == nil {
		t.Error("NewConfigServer() returned nil")
	}

	// Directory should be created
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("ConfigDir should have been created")
	}
}
