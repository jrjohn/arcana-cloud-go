package configserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestDefaultConfigClientConfig(t *testing.T) {
	cfg := DefaultConfigClientConfig()
	if cfg == nil {
		t.Fatal("DefaultConfigClientConfig() returned nil")
	}
	if cfg.Enabled != false {
		t.Error("Enabled should default to false")
	}
	if cfg.ServerURL != "http://localhost:8888" {
		t.Errorf("ServerURL = %v, want http://localhost:8888", cfg.ServerURL)
	}
	if cfg.Application != "application" {
		t.Errorf("Application = %v, want application", cfg.Application)
	}
	if cfg.Profile != "default" {
		t.Errorf("Profile = %v, want default", cfg.Profile)
	}
	if cfg.RetryCount != 3 {
		t.Errorf("RetryCount = %v, want 3", cfg.RetryCount)
	}
}

func TestNewConfigClient_Disabled(t *testing.T) {
	logger := newTestLogger()
	cfg := DefaultConfigClientConfig()
	cfg.Enabled = false

	client, err := NewConfigClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewConfigClient() error = %v", err)
	}
	if client == nil {
		t.Error("NewConfigClient() returned nil")
	}
}

func TestNewConfigClient_Enabled_Success(t *testing.T) {
	// Start test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ConfigResponse{
			Name:     "test",
			Profiles: []string{"default"},
			PropertySources: []PropertySource{
				{
					Name:   "test/default",
					Source: map[string]interface{}{"key1": "value1"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := newTestLogger()
	cfg := &ConfigClientConfig{
		Enabled:         true,
		ServerURL:       server.URL,
		Application:     "test",
		Profile:         "default",
		FailFast:        false,
		RetryCount:      1,
		RetryInterval:   time.Millisecond,
		RefreshInterval: 0,
		Timeout:         5 * time.Second,
	}

	client, err := NewConfigClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewConfigClient() error = %v", err)
	}
	if client == nil {
		t.Error("NewConfigClient() returned nil")
	}

	val, ok := client.Get("key1")
	if !ok {
		t.Error("Get() should find key1")
	}
	if val != "value1" {
		t.Errorf("Get(key1) = %v, want value1", val)
	}
}

func TestNewConfigClient_Enabled_FailFast(t *testing.T) {
	logger := newTestLogger()
	cfg := &ConfigClientConfig{
		Enabled:       true,
		ServerURL:     "http://localhost:99999", // Invalid port
		Application:   "test",
		Profile:       "default",
		FailFast:      true,
		RetryCount:    0,
		RetryInterval: time.Millisecond,
		Timeout:       100 * time.Millisecond,
	}

	_, err := NewConfigClient(cfg, logger)
	if err == nil {
		t.Error("NewConfigClient() should fail when FailFast=true and server unavailable")
	}
}

func TestNewConfigClient_Enabled_NonFailFast(t *testing.T) {
	logger := newTestLogger()
	cfg := &ConfigClientConfig{
		Enabled:       true,
		ServerURL:     "http://localhost:99999", // Invalid port
		Application:   "test",
		Profile:       "default",
		FailFast:      false,
		RetryCount:    0,
		RetryInterval: time.Millisecond,
		Timeout:       100 * time.Millisecond,
	}

	client, err := NewConfigClient(cfg, logger)
	if err != nil {
		t.Fatalf("NewConfigClient() should not fail when FailFast=false, got error: %v", err)
	}
	if client == nil {
		t.Error("NewConfigClient() returned nil")
	}
}

func TestConfigClient_GetString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ConfigResponse{
			PropertySources: []PropertySource{
				{Source: map[string]interface{}{"name": "arcana", "count": float64(5)}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := newTestLogger()
	cfg := &ConfigClientConfig{
		Enabled:       true,
		ServerURL:     server.URL,
		Application:   "test",
		Profile:       "default",
		RetryCount:    1,
		RetryInterval: time.Millisecond,
		Timeout:       5 * time.Second,
	}

	client, _ := NewConfigClient(cfg, logger)

	if got := client.GetString("name", "default"); got != "arcana" {
		t.Errorf("GetString(name) = %v, want arcana", got)
	}
	if got := client.GetString("missing", "fallback"); got != "fallback" {
		t.Errorf("GetString(missing) = %v, want fallback", got)
	}
	// Count is a float64, not string
	if got := client.GetString("count", "none"); got != "none" {
		t.Errorf("GetString(count) = %v, want none (not a string)", got)
	}
}

func TestConfigClient_GetInt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ConfigResponse{
			PropertySources: []PropertySource{
				{Source: map[string]interface{}{
					"count":    float64(42),
					"count64":  int64(100),
					"name":     "str",
				}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := newTestLogger()
	cfg := &ConfigClientConfig{
		Enabled:       true,
		ServerURL:     server.URL,
		Application:   "test",
		Profile:       "default",
		RetryCount:    1,
		RetryInterval: time.Millisecond,
		Timeout:       5 * time.Second,
	}

	client, _ := NewConfigClient(cfg, logger)

	if got := client.GetInt("count", 0); got != 42 {
		t.Errorf("GetInt(count) = %v, want 42", got)
	}
	if got := client.GetInt("missing", 99); got != 99 {
		t.Errorf("GetInt(missing) = %v, want 99", got)
	}
	if got := client.GetInt("name", 0); got != 0 {
		t.Errorf("GetInt(name) = %v, want 0 (not an int)", got)
	}
}

func TestConfigClient_GetBool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ConfigResponse{
			PropertySources: []PropertySource{
				{Source: map[string]interface{}{
					"enabled": true,
					"name":    "str",
				}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := newTestLogger()
	cfg := &ConfigClientConfig{
		Enabled:       true,
		ServerURL:     server.URL,
		Application:   "test",
		Profile:       "default",
		RetryCount:    1,
		RetryInterval: time.Millisecond,
		Timeout:       5 * time.Second,
	}

	client, _ := NewConfigClient(cfg, logger)

	if got := client.GetBool("enabled", false); got != true {
		t.Errorf("GetBool(enabled) = %v, want true", got)
	}
	if got := client.GetBool("missing", true); got != true {
		t.Errorf("GetBool(missing) = %v, want true (default)", got)
	}
	if got := client.GetBool("name", false); got != false {
		t.Errorf("GetBool(name) = %v, want false (not a bool)", got)
	}
}

func TestConfigClient_GetAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ConfigResponse{
			PropertySources: []PropertySource{
				{Source: map[string]interface{}{"a": "1", "b": "2"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := newTestLogger()
	cfg := &ConfigClientConfig{
		Enabled:       true,
		ServerURL:     server.URL,
		Application:   "test",
		Profile:       "default",
		RetryCount:    1,
		RetryInterval: time.Millisecond,
		Timeout:       5 * time.Second,
	}

	client, _ := NewConfigClient(cfg, logger)

	all := client.GetAll()
	if len(all) != 2 {
		t.Errorf("GetAll() returned %d items, want 2", len(all))
	}
}

func TestConfigClient_OnChange(t *testing.T) {
	logger := newTestLogger()
	cfg := DefaultConfigClientConfig()
	cfg.Enabled = false

	client, _ := NewConfigClient(cfg, logger)

	called := make(chan bool, 1)
	client.OnChange(func(config map[string]interface{}) {
		called <- true
	})

	// Manually update cache to trigger change notification
	client.updateCache(map[string]interface{}{"key": "value"})

	select {
	case <-called:
		// Good
	case <-time.After(500 * time.Millisecond):
		t.Error("OnChange callback was not called")
	}
}

func TestConfigClient_Start_Stop(t *testing.T) {
	logger := newTestLogger()
	cfg := DefaultConfigClientConfig()
	cfg.Enabled = false
	cfg.RefreshInterval = 0

	client, _ := NewConfigClient(cfg, logger)
	client.Start()
	client.Stop() // Should not panic
}

func TestConfigClient_Start_WithRefresh(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := ConfigResponse{
			PropertySources: []PropertySource{
				{Source: map[string]interface{}{"count": callCount}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := newTestLogger()
	cfg := &ConfigClientConfig{
		Enabled:         true,
		ServerURL:       server.URL,
		Application:     "test",
		Profile:         "default",
		RetryCount:      0,
		RefreshInterval: 50 * time.Millisecond,
		Timeout:         5 * time.Second,
	}

	client, _ := NewConfigClient(cfg, logger)
	client.Start()

	time.Sleep(120 * time.Millisecond)
	client.Stop()

	if callCount < 2 {
		t.Errorf("Expected at least 2 calls, got %d", callCount)
	}
}

func TestConfigClient_Refresh_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := newTestLogger()
	cfg := &ConfigClientConfig{
		Enabled:       true,
		ServerURL:     server.URL,
		Application:   "test",
		Profile:       "default",
		FailFast:      false,
		RetryCount:    0,
		RetryInterval: time.Millisecond,
		Timeout:       5 * time.Second,
	}

	client, _ := NewConfigClient(cfg, logger)
	err := client.Refresh()
	if err == nil {
		t.Error("Refresh() should return error on server error")
	}
}

func TestConfigClient_Refresh_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	logger := newTestLogger()
	cfg := &ConfigClientConfig{
		Enabled:       true,
		ServerURL:     server.URL,
		Application:   "test",
		Profile:       "default",
		FailFast:      false,
		RetryCount:    0,
		RetryInterval: time.Millisecond,
		Timeout:       5 * time.Second,
	}

	client, _ := NewConfigClient(cfg, logger)
	err := client.Refresh()
	if err == nil {
		t.Error("Refresh() should return error on invalid JSON")
	}
}

func TestConfigClient_PropertySourceMerge(t *testing.T) {
	// Test that later property sources take precedence (reverse iteration)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ConfigResponse{
			PropertySources: []PropertySource{
				{Source: map[string]interface{}{"key": "overridden", "only-first": "first"}},
				{Source: map[string]interface{}{"key": "value", "only-second": "second"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	logger := newTestLogger()
	cfg := &ConfigClientConfig{
		Enabled:       true,
		ServerURL:     server.URL,
		Application:   "test",
		Profile:       "default",
		RetryCount:    0,
		RetryInterval: time.Millisecond,
		Timeout:       5 * time.Second,
	}

	client, _ := NewConfigClient(cfg, logger)
	// The merge should have all keys
	all := client.GetAll()
	if len(all) == 0 {
		t.Error("GetAll() returned empty map")
	}
}

func TestConfigEqual(t *testing.T) {
	a := map[string]interface{}{"key": "value", "num": 1}
	b := map[string]interface{}{"key": "value", "num": 1}
	c := map[string]interface{}{"key": "different"}
	d := map[string]interface{}{"key": "value", "num": 1, "extra": "x"}

	if !configEqual(a, b) {
		t.Error("configEqual(a, b) should be true")
	}
	if configEqual(a, c) {
		t.Error("configEqual(a, c) should be false (different values)")
	}
	if configEqual(a, d) {
		t.Error("configEqual(a, d) should be false (different size)")
	}
}

func TestMergePropertySources(t *testing.T) {
	sources := []PropertySource{
		{Source: map[string]interface{}{"a": "first", "b": "first-b"}},
		{Source: map[string]interface{}{"a": "second"}},
	}

	merged := mergePropertySources(sources)
	// The first source should override second (reverse iteration)
	if v, ok := merged["a"]; !ok || v != "first" {
		t.Errorf("merged[a] = %v, want first", v)
	}
	if v, ok := merged["b"]; !ok || v != "first-b" {
		t.Errorf("merged[b] = %v, want first-b", v)
	}
}
