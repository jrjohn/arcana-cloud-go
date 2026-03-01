package configserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newDisabledClient(t *testing.T) *ConfigClient {
	t.Helper()
	config := DefaultConfigClientConfig()
	config.Enabled = false
	client, err := NewConfigClient(config, zap.NewNop())
	if err != nil {
		t.Fatalf("NewConfigClient() error = %v", err)
	}
	return client
}

func TestDefaultConfigClientConfig(t *testing.T) {
	config := DefaultConfigClientConfig()

	if config.Enabled {
		t.Error("Enabled should be false by default")
	}
	if config.ServerURL != "http://localhost:8888" {
		t.Errorf("ServerURL = %v, want http://localhost:8888", config.ServerURL)
	}
	if config.Application != "application" {
		t.Errorf("Application = %v, want application", config.Application)
	}
	if config.Profile != "default" {
		t.Errorf("Profile = %v, want default", config.Profile)
	}
	if config.RetryCount != 3 {
		t.Errorf("RetryCount = %v, want 3", config.RetryCount)
	}
	if config.RetryInterval != time.Second {
		t.Errorf("RetryInterval = %v, want 1s", config.RetryInterval)
	}
	if config.RefreshInterval != 30*time.Second {
		t.Errorf("RefreshInterval = %v, want 30s", config.RefreshInterval)
	}
	if config.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", config.Timeout)
	}
}

func TestNewConfigClient_Disabled(t *testing.T) {
	client := newDisabledClient(t)

	if client == nil {
		t.Fatal("NewConfigClient() returned nil")
	}
	if client.cache == nil {
		t.Error("cache should be initialized")
	}
}

func TestNewConfigClient_WithServer(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ConfigResponse{
			Name:     "test-app",
			Profiles: []string{"test"},
			PropertySources: []PropertySource{
				{
					Name: "test",
					Source: map[string]interface{}{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := DefaultConfigClientConfig()
	config.Enabled = true
	config.ServerURL = server.URL
	config.Application = "test-app"
	config.Profile = "test"
	config.RetryCount = 0

	client, err := NewConfigClient(config, zap.NewNop())
	if err != nil {
		t.Fatalf("NewConfigClient() error = %v", err)
	}

	val, ok := client.Get("key1")
	if !ok {
		t.Error("key1 should exist")
	}
	if val != "value1" {
		t.Errorf("key1 = %v, want value1", val)
	}
}

func TestNewConfigClient_FailFast(t *testing.T) {
	config := DefaultConfigClientConfig()
	config.Enabled = true
	config.ServerURL = "http://localhost:99999" // Invalid URL
	config.FailFast = true
	config.RetryCount = 0

	_, err := NewConfigClient(config, zap.NewNop())
	if err == nil {
		t.Error("NewConfigClient() should fail with FailFast=true and invalid server")
	}
}

func TestNewConfigClient_FailSoft(t *testing.T) {
	config := DefaultConfigClientConfig()
	config.Enabled = true
	config.ServerURL = "http://localhost:99999" // Invalid URL
	config.FailFast = false
	config.RetryCount = 0

	client, err := NewConfigClient(config, zap.NewNop())
	if err != nil {
		t.Errorf("NewConfigClient() with FailFast=false should not fail, got %v", err)
	}
	if client == nil {
		t.Error("client should not be nil")
	}
}

func TestConfigClient_Get(t *testing.T) {
	client := newDisabledClient(t)

	// Populate cache manually
	client.cache["key1"] = "value1"
	client.cache["num"] = 42

	val, ok := client.Get("key1")
	if !ok {
		t.Error("Get() should return true for existing key")
	}
	if val != "value1" {
		t.Errorf("Get(key1) = %v, want value1", val)
	}

	_, ok = client.Get("missing")
	if ok {
		t.Error("Get() should return false for missing key")
	}
}

func TestConfigClient_GetString(t *testing.T) {
	client := newDisabledClient(t)

	client.cache["str"] = "hello"
	client.cache["num"] = 42

	if client.GetString("str", "default") != "hello" {
		t.Errorf("GetString(str) = %v, want hello", client.GetString("str", "default"))
	}
	if client.GetString("missing", "default") != "default" {
		t.Errorf("GetString(missing) = %v, want default", client.GetString("missing", "default"))
	}
	if client.GetString("num", "default") != "default" {
		t.Errorf("GetString(num) should return default for non-string, got %v", client.GetString("num", "default"))
	}
}

func TestConfigClient_GetInt(t *testing.T) {
	client := newDisabledClient(t)

	client.cache["int"] = 42
	client.cache["int64"] = int64(100)
	client.cache["float"] = float64(3.14)
	client.cache["str"] = "hello"

	if client.GetInt("int", 0) != 42 {
		t.Errorf("GetInt(int) = %v, want 42", client.GetInt("int", 0))
	}
	if client.GetInt("int64", 0) != 100 {
		t.Errorf("GetInt(int64) = %v, want 100", client.GetInt("int64", 0))
	}
	if client.GetInt("float", 0) != 3 {
		t.Errorf("GetInt(float) = %v, want 3", client.GetInt("float", 0))
	}
	if client.GetInt("missing", 99) != 99 {
		t.Errorf("GetInt(missing) = %v, want 99", client.GetInt("missing", 99))
	}
	if client.GetInt("str", 0) != 0 {
		t.Errorf("GetInt(str) should return default, got %v", client.GetInt("str", 0))
	}
}

func TestConfigClient_GetBool(t *testing.T) {
	client := newDisabledClient(t)

	client.cache["bool_true"] = true
	client.cache["bool_false"] = false
	client.cache["str"] = "hello"

	if !client.GetBool("bool_true", false) {
		t.Error("GetBool(bool_true) should return true")
	}
	if client.GetBool("bool_false", true) {
		t.Error("GetBool(bool_false) should return false")
	}
	if !client.GetBool("missing", true) {
		t.Error("GetBool(missing) should return default value true")
	}
	if client.GetBool("str", false) {
		t.Error("GetBool(str) should return default for non-bool")
	}
}

func TestConfigClient_GetAll(t *testing.T) {
	client := newDisabledClient(t)

	client.cache["a"] = 1
	client.cache["b"] = "two"

	all := client.GetAll()
	if len(all) != 2 {
		t.Errorf("GetAll() length = %v, want 2", len(all))
	}
	if all["a"] != 1 {
		t.Errorf("all[a] = %v, want 1", all["a"])
	}
	if all["b"] != "two" {
		t.Errorf("all[b] = %v, want two", all["b"])
	}
}

func TestConfigClient_OnChange(t *testing.T) {
	client := newDisabledClient(t)

	called := false
	client.OnChange(func(config map[string]interface{}) {
		called = true
	})

	if len(client.listeners) != 1 {
		t.Errorf("listeners length = %v, want 1", len(client.listeners))
	}

	_ = called
}

func TestConfigClient_Stop(t *testing.T) {
	client := newDisabledClient(t)

	// Should not panic
	client.Stop()
}

func TestConfigClient_Start_Disabled(t *testing.T) {
	client := newDisabledClient(t)
	// Should not start refresh goroutine when disabled
	client.Start()
}

func TestConfigClient_Start_NoRefreshInterval(t *testing.T) {
	config := DefaultConfigClientConfig()
	config.Enabled = true
	config.RefreshInterval = 0 // No refresh

	client := &ConfigClient{
		config:    config,
		cache:     make(map[string]interface{}),
		logger:    zap.NewNop(),
		stopCh:    make(chan struct{}),
		listeners: make([]func(map[string]interface{}), 0),
	}

	// Should return without starting goroutine
	client.Start()
}

func TestMergePropertySources(t *testing.T) {
	sources := []PropertySource{
		{
			Name: "source1",
			Source: map[string]interface{}{
				"key1": "from-source1",
				"key2": "from-source1",
			},
		},
		{
			Name: "source2",
			Source: map[string]interface{}{
				"key1": "from-source2", // Overrides source1
				"key3": "from-source2",
			},
		},
	}

	merged := mergePropertySources(sources)

	if len(merged) != 3 {
		t.Errorf("merged length = %v, want 3", len(merged))
	}
	// source2 should override source1 for key1
	if merged["key1"] != "from-source1" {
		t.Errorf("merged[key1] = %v, want from-source1 (earlier source wins)", merged["key1"])
	}
	if merged["key2"] != "from-source1" {
		t.Errorf("merged[key2] = %v, want from-source1", merged["key2"])
	}
	if merged["key3"] != "from-source2" {
		t.Errorf("merged[key3] = %v, want from-source2", merged["key3"])
	}
}

func TestConfigEqual(t *testing.T) {
	a := map[string]interface{}{"key": "value", "num": 42}
	b := map[string]interface{}{"key": "value", "num": 42}
	c := map[string]interface{}{"key": "different", "num": 42}
	d := map[string]interface{}{"key": "value"}

	if !configEqual(a, b) {
		t.Error("configEqual(a, b) should be true for identical maps")
	}
	if configEqual(a, c) {
		t.Error("configEqual(a, c) should be false for different values")
	}
	if configEqual(a, d) {
		t.Error("configEqual(a, d) should be false for different lengths")
	}
}

func TestConfigClient_FetchConfig_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	config := DefaultConfigClientConfig()
	config.Enabled = false
	config.ServerURL = server.URL
	config.Application = "test"
	config.Profile = "default"
	config.RetryCount = 0

	client := &ConfigClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		cache:     make(map[string]interface{}),
		logger:    zap.NewNop(),
		stopCh:    make(chan struct{}),
		listeners: make([]func(map[string]interface{}), 0),
	}

	err := client.fetchConfig()
	if err == nil {
		t.Error("fetchConfig() should return error on server error")
	}
}

func TestConfigClient_FetchConfig_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json{"))
	}))
	defer server.Close()

	config := DefaultConfigClientConfig()
	config.Enabled = false
	config.ServerURL = server.URL
	config.RetryCount = 0

	client := &ConfigClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		cache:     make(map[string]interface{}),
		logger:    zap.NewNop(),
		stopCh:    make(chan struct{}),
		listeners: make([]func(map[string]interface{}), 0),
	}

	err := client.fetchConfig()
	if err == nil {
		t.Error("fetchConfig() should return error on invalid JSON")
	}
}

func TestConfigClient_NotifyListeners(t *testing.T) {
	client := newDisabledClient(t)

	received := make(chan map[string]interface{}, 1)
	client.OnChange(func(config map[string]interface{}) {
		received <- config
	})

	newConfig := map[string]interface{}{"key": "new_value"}
	client.notifyListeners(newConfig)

	select {
	case cfg := <-received:
		if cfg["key"] != "new_value" {
			t.Errorf("received config key = %v, want new_value", cfg["key"])
		}
	case <-time.After(time.Second):
		t.Error("listener was not called")
	}
}

func TestConfigClient_UpdateCache_WithChange(t *testing.T) {
	client := newDisabledClient(t)

	received := make(chan map[string]interface{}, 1)
	client.OnChange(func(config map[string]interface{}) {
		received <- config
	})

	client.cache["old"] = "value"

	newConfig := map[string]interface{}{"new": "data"}
	client.updateCache(newConfig)

	// Old cache should be replaced
	val, ok := client.Get("new")
	if !ok {
		t.Error("new key should exist after updateCache")
	}
	if val != "data" {
		t.Errorf("new key = %v, want data", val)
	}
}

func TestConfigClient_UpdateCache_NoChange(t *testing.T) {
	client := newDisabledClient(t)

	listenerCalled := false
	client.OnChange(func(config map[string]interface{}) {
		listenerCalled = true
	})

	// Set same config twice
	config := map[string]interface{}{"key": "value"}
	client.cache = config
	client.updateCache(config)

	// Small wait to see if listener is called
	time.Sleep(10 * time.Millisecond)

	if listenerCalled {
		t.Error("listener should not be called when config hasn't changed")
	}
}

func TestConfigClient_Refresh(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ConfigResponse{
			Name:     "app",
			Profiles: []string{"default"},
			PropertySources: []PropertySource{
				{
					Name:   "app",
					Source: map[string]interface{}{"refreshed": "yes"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := DefaultConfigClientConfig()
	config.Enabled = false
	config.ServerURL = server.URL
	config.Application = "app"
	config.Profile = "default"
	config.RetryCount = 0

	client := &ConfigClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		cache:     make(map[string]interface{}),
		logger:    zap.NewNop(),
		stopCh:    make(chan struct{}),
		listeners: make([]func(map[string]interface{}), 0),
	}

	err := client.Refresh()
	if err != nil {
		t.Errorf("Refresh() error = %v", err)
	}

	val, ok := client.Get("refreshed")
	if !ok {
		t.Error("refreshed key should exist after Refresh()")
	}
	if val != "yes" {
		t.Errorf("refreshed = %v, want yes", val)
	}
}

func TestConfigClient_RefreshConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/refresh" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Return updated config
		resp := ConfigResponse{
			Name:     "app",
			Profiles: []string{"default"},
			PropertySources: []PropertySource{
				{
					Name:   "app",
					Source: map[string]interface{}{"after_refresh": "true"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := DefaultConfigClientConfig()
	config.Enabled = false
	config.ServerURL = server.URL
	config.Application = "app"
	config.Profile = "default"
	config.RetryCount = 0

	client := &ConfigClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		cache:     make(map[string]interface{}),
		logger:    zap.NewNop(),
		stopCh:    make(chan struct{}),
		listeners: make([]func(map[string]interface{}), 0),
	}

	err := client.RefreshConfig(context.Background())
	if err != nil {
		t.Errorf("RefreshConfig() error = %v", err)
	}
}

func TestConfigClient_RefreshConfig_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("refresh failed"))
		}
	}))
	defer server.Close()

	config := DefaultConfigClientConfig()
	config.Enabled = false
	config.ServerURL = server.URL

	client := &ConfigClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		cache:     make(map[string]interface{}),
		logger:    zap.NewNop(),
		stopCh:    make(chan struct{}),
		listeners: make([]func(map[string]interface{}), 0),
	}

	err := client.RefreshConfig(context.Background())
	if err == nil {
		t.Error("RefreshConfig() should return error on server error")
	}
}

func TestPropertySource_Struct(t *testing.T) {
	ps := PropertySource{
		Name: "test",
		Source: map[string]interface{}{
			"key": "value",
		},
	}

	if ps.Name != "test" {
		t.Errorf("Name = %v, want test", ps.Name)
	}
	if ps.Source["key"] != "value" {
		t.Errorf("Source[key] = %v, want value", ps.Source["key"])
	}
}

func TestConfigResponse_Struct(t *testing.T) {
	cr := ConfigResponse{
		Name:     "my-app",
		Profiles: []string{"dev", "test"},
		PropertySources: []PropertySource{
			{Name: "ps1", Source: map[string]interface{}{"a": "1"}},
		},
	}

	if cr.Name != "my-app" {
		t.Errorf("Name = %v, want my-app", cr.Name)
	}
	if len(cr.Profiles) != 2 {
		t.Errorf("Profiles length = %v, want 2", len(cr.Profiles))
	}
	if len(cr.PropertySources) != 1 {
		t.Errorf("PropertySources length = %v, want 1", len(cr.PropertySources))
	}
}
