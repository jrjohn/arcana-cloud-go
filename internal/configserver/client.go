package configserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConfigClientConfig holds configuration client settings
type ConfigClientConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	ServerURL       string        `mapstructure:"server_url"`
	Application     string        `mapstructure:"application"`
	Profile         string        `mapstructure:"profile"`
	FailFast        bool          `mapstructure:"fail_fast"`
	RetryCount      int           `mapstructure:"retry_count"`
	RetryInterval   time.Duration `mapstructure:"retry_interval"`
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`
	Timeout         time.Duration `mapstructure:"timeout"`
}

// DefaultConfigClientConfig returns default configuration
func DefaultConfigClientConfig() *ConfigClientConfig {
	return &ConfigClientConfig{
		Enabled:         false,
		ServerURL:       "http://localhost:8888",
		Application:     "application",
		Profile:         "default",
		FailFast:        false,
		RetryCount:      3,
		RetryInterval:   time.Second,
		RefreshInterval: 30 * time.Second,
		Timeout:         5 * time.Second,
	}
}

// ConfigResponse represents the response from config server
type ConfigResponse struct {
	Name            string           `json:"name"`
	Profiles        []string         `json:"profiles"`
	PropertySources []PropertySource `json:"propertySources"`
}

// PropertySource represents a property source in config response
type PropertySource struct {
	Name   string                 `json:"name"`
	Source map[string]interface{} `json:"source"`
}

// ConfigClient is a client for the centralized configuration server
type ConfigClient struct {
	config     *ConfigClientConfig
	httpClient *http.Client
	cache      map[string]interface{}
	mutex      sync.RWMutex
	logger     *zap.Logger
	stopCh     chan struct{}
	listeners  []func(map[string]interface{})
}

// NewConfigClient creates a new configuration client
func NewConfigClient(config *ConfigClientConfig, logger *zap.Logger) (*ConfigClient, error) {
	client := &ConfigClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		cache:     make(map[string]interface{}),
		logger:    logger,
		stopCh:    make(chan struct{}),
		listeners: make([]func(map[string]interface{}), 0),
	}

	// Fetch initial configuration
	if config.Enabled {
		if err := client.fetchConfig(); err != nil {
			if config.FailFast {
				return nil, fmt.Errorf("failed to fetch initial config: %w", err)
			}
			logger.Warn("Failed to fetch initial config, will retry", zap.Error(err))
		}
	}

	return client, nil
}

// Start starts the config client with periodic refresh
func (c *ConfigClient) Start() {
	if !c.config.Enabled || c.config.RefreshInterval <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(c.config.RefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := c.Refresh(); err != nil {
					c.logger.Warn("Failed to refresh config", zap.Error(err))
				}
			case <-c.stopCh:
				return
			}
		}
	}()
}

// Stop stops the config client
func (c *ConfigClient) Stop() {
	close(c.stopCh)
}

// fetchConfig fetches configuration from the server
func (c *ConfigClient) fetchConfig() error {
	url := fmt.Sprintf("%s/config/%s/%s", c.config.ServerURL, c.config.Application, c.config.Profile)

	var lastErr error
	for i := 0; i <= c.config.RetryCount; i++ {
		if i > 0 {
			time.Sleep(c.config.RetryInterval)
		}

		resp, err := c.httpClient.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("config server returned %d: %s", resp.StatusCode, string(body))
			continue
		}

		var configResp ConfigResponse
		if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
			lastErr = err
			continue
		}

		// Merge all property sources
		newConfig := make(map[string]interface{})
		for i := len(configResp.PropertySources) - 1; i >= 0; i-- {
			for k, v := range configResp.PropertySources[i].Source {
				newConfig[k] = v
			}
		}

		// Update cache
		c.mutex.Lock()
		oldConfig := c.cache
		c.cache = newConfig
		c.mutex.Unlock()

		// Notify listeners if config changed
		if !configEqual(oldConfig, newConfig) {
			c.notifyListeners(newConfig)
		}

		c.logger.Info("Configuration loaded",
			zap.String("application", c.config.Application),
			zap.String("profile", c.config.Profile),
			zap.Int("properties", len(newConfig)),
		)

		return nil
	}

	return lastErr
}

// Refresh refreshes configuration from the server
func (c *ConfigClient) Refresh() error {
	return c.fetchConfig()
}

// Get returns a configuration value
func (c *ConfigClient) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	value, ok := c.cache[key]
	return value, ok
}

// GetString returns a string configuration value
func (c *ConfigClient) GetString(key string, defaultValue string) string {
	value, ok := c.Get(key)
	if !ok {
		return defaultValue
	}
	if str, ok := value.(string); ok {
		return str
	}
	return defaultValue
}

// GetInt returns an int configuration value
func (c *ConfigClient) GetInt(key string, defaultValue int) int {
	value, ok := c.Get(key)
	if !ok {
		return defaultValue
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return defaultValue
}

// GetBool returns a bool configuration value
func (c *ConfigClient) GetBool(key string, defaultValue bool) bool {
	value, ok := c.Get(key)
	if !ok {
		return defaultValue
	}
	if b, ok := value.(bool); ok {
		return b
	}
	return defaultValue
}

// GetAll returns all configuration values
func (c *ConfigClient) GetAll() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	result := make(map[string]interface{})
	for k, v := range c.cache {
		result[k] = v
	}
	return result
}

// OnChange registers a callback for configuration changes
func (c *ConfigClient) OnChange(callback func(map[string]interface{})) {
	c.mutex.Lock()
	c.listeners = append(c.listeners, callback)
	c.mutex.Unlock()
}

// notifyListeners notifies all registered listeners
func (c *ConfigClient) notifyListeners(config map[string]interface{}) {
	c.mutex.RLock()
	listeners := c.listeners
	c.mutex.RUnlock()

	for _, listener := range listeners {
		go listener(config)
	}
}

// RefreshConfig triggers a refresh via the config server
func (c *ConfigClient) RefreshConfig(ctx context.Context) error {
	url := fmt.Sprintf("%s/refresh", c.config.ServerURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("refresh failed: %s", string(body))
	}

	// Fetch updated config
	return c.fetchConfig()
}

// configEqual compares two config maps for equality
func configEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || v != bv {
			return false
		}
	}
	return true
}
