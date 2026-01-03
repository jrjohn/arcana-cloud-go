package configserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ConfigServerConfig holds configuration server settings
type ConfigServerConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	Port          int           `mapstructure:"port"`
	ConfigDir     string        `mapstructure:"config_dir"`
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`
	EnableWatch   bool          `mapstructure:"enable_watch"`
	EncryptionKey string        `mapstructure:"encryption_key"`
}

// DefaultConfigServerConfig returns default configuration
func DefaultConfigServerConfig() *ConfigServerConfig {
	return &ConfigServerConfig{
		Enabled:         false,
		Port:            8888,
		ConfigDir:       "./config-repo",
		RefreshInterval: 30 * time.Second,
		EnableWatch:     true,
	}
}

// ConfigServer is a centralized configuration server
type ConfigServer struct {
	config    *ConfigServerConfig
	configs   map[string]map[string]interface{} // app -> profile -> config
	mutex     sync.RWMutex
	logger    *zap.Logger
	watcher   *fsnotify.Watcher
	listeners []chan ConfigChangeEvent
	server    *http.Server
}

// ConfigChangeEvent represents a configuration change
type ConfigChangeEvent struct {
	Application string
	Profile     string
	Key         string
	OldValue    interface{}
	NewValue    interface{}
	Timestamp   time.Time
}

// NewConfigServer creates a new configuration server
func NewConfigServer(config *ConfigServerConfig, logger *zap.Logger) (*ConfigServer, error) {
	cs := &ConfigServer{
		config:    config,
		configs:   make(map[string]map[string]interface{}),
		logger:    logger,
		listeners: make([]chan ConfigChangeEvent, 0),
	}

	// Load initial configurations
	if err := cs.loadConfigs(); err != nil {
		return nil, fmt.Errorf("failed to load configs: %w", err)
	}

	// Set up file watcher if enabled
	if config.EnableWatch {
		if err := cs.setupWatcher(); err != nil {
			logger.Warn("Failed to setup file watcher", zap.Error(err))
		}
	}

	return cs, nil
}

// Start starts the configuration server
func (cs *ConfigServer) Start() error {
	mux := http.NewServeMux()

	// Endpoints
	mux.HandleFunc("/", cs.handleRoot)
	mux.HandleFunc("/health", cs.handleHealth)
	mux.HandleFunc("/config/", cs.handleConfig)
	mux.HandleFunc("/refresh", cs.handleRefresh)
	mux.HandleFunc("/encrypt", cs.handleEncrypt)
	mux.HandleFunc("/decrypt", cs.handleDecrypt)

	cs.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cs.config.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	cs.logger.Info("Starting configuration server",
		zap.Int("port", cs.config.Port),
		zap.String("config_dir", cs.config.ConfigDir),
	)

	return cs.server.ListenAndServe()
}

// Stop stops the configuration server
func (cs *ConfigServer) Stop(ctx context.Context) error {
	if cs.watcher != nil {
		cs.watcher.Close()
	}
	if cs.server != nil {
		return cs.server.Shutdown(ctx)
	}
	return nil
}

// loadConfigs loads all configurations from the config directory
func (cs *ConfigServer) loadConfigs() error {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// Create config directory if it doesn't exist
	if _, err := os.Stat(cs.config.ConfigDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cs.config.ConfigDir, 0755); err != nil {
			return err
		}
	}

	// Walk through config directory
	return filepath.Walk(cs.config.ConfigDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Only process YAML and JSON files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			return nil
		}

		// Parse filename: {application}-{profile}.{ext}
		filename := strings.TrimSuffix(filepath.Base(path), ext)
		parts := strings.Split(filename, "-")

		application := parts[0]
		profile := "default"
		if len(parts) > 1 {
			profile = strings.Join(parts[1:], "-")
		}

		// Read and parse file
		data, err := os.ReadFile(path)
		if err != nil {
			cs.logger.Warn("Failed to read config file", zap.String("path", path), zap.Error(err))
			return nil
		}

		var config map[string]interface{}
		if ext == ".json" {
			if err := json.Unmarshal(data, &config); err != nil {
				cs.logger.Warn("Failed to parse JSON config", zap.String("path", path), zap.Error(err))
				return nil
			}
		} else {
			if err := yaml.Unmarshal(data, &config); err != nil {
				cs.logger.Warn("Failed to parse YAML config", zap.String("path", path), zap.Error(err))
				return nil
			}
		}

		// Store config
		key := fmt.Sprintf("%s/%s", application, profile)
		cs.configs[key] = config

		cs.logger.Debug("Loaded config",
			zap.String("application", application),
			zap.String("profile", profile),
			zap.String("path", path),
		)

		return nil
	})
}

// setupWatcher sets up file system watcher for config changes
func (cs *ConfigServer) setupWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	cs.watcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					cs.logger.Info("Config file changed", zap.String("file", event.Name))
					if err := cs.loadConfigs(); err != nil {
						cs.logger.Error("Failed to reload configs", zap.Error(err))
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				cs.logger.Error("Watcher error", zap.Error(err))
			}
		}
	}()

	return watcher.Add(cs.config.ConfigDir)
}

// GetConfig returns configuration for an application and profile
func (cs *ConfigServer) GetConfig(application, profile string) (map[string]interface{}, error) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	// Try specific profile first
	key := fmt.Sprintf("%s/%s", application, profile)
	if config, ok := cs.configs[key]; ok {
		return cs.mergeWithDefault(application, config), nil
	}

	// Fall back to default profile
	key = fmt.Sprintf("%s/default", application)
	if config, ok := cs.configs[key]; ok {
		return config, nil
	}

	return nil, fmt.Errorf("configuration not found for %s/%s", application, profile)
}

// mergeWithDefault merges profile-specific config with default
func (cs *ConfigServer) mergeWithDefault(application string, profileConfig map[string]interface{}) map[string]interface{} {
	defaultKey := fmt.Sprintf("%s/default", application)
	defaultConfig, exists := cs.configs[defaultKey]
	if !exists {
		return profileConfig
	}

	// Deep merge default with profile (profile takes precedence)
	return deepMerge(defaultConfig, profileConfig)
}

// deepMerge merges two maps recursively
func deepMerge(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy base
	for k, v := range base {
		result[k] = v
	}

	// Override with new values
	for k, v := range override {
		if baseVal, exists := result[k]; exists {
			// If both are maps, merge recursively
			if baseMap, ok := baseVal.(map[string]interface{}); ok {
				if overrideMap, ok := v.(map[string]interface{}); ok {
					result[k] = deepMerge(baseMap, overrideMap)
					continue
				}
			}
		}
		result[k] = v
	}

	return result
}

// Subscribe subscribes to configuration changes
func (cs *ConfigServer) Subscribe() <-chan ConfigChangeEvent {
	ch := make(chan ConfigChangeEvent, 10)
	cs.mutex.Lock()
	cs.listeners = append(cs.listeners, ch)
	cs.mutex.Unlock()
	return ch
}

// HTTP Handlers

func (cs *ConfigServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"name":    "Arcana Config Server",
		"version": "1.0.0",
	})
}

func (cs *ConfigServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "UP"})
}

func (cs *ConfigServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	// Parse path: /config/{application}/{profile}
	path := strings.TrimPrefix(r.URL.Path, "/config/")
	parts := strings.Split(path, "/")

	application := "application"
	profile := "default"

	if len(parts) >= 1 && parts[0] != "" {
		application = parts[0]
	}
	if len(parts) >= 2 && parts[1] != "" {
		profile = parts[1]
	}

	config, err := cs.GetConfig(application, profile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":     application,
		"profiles": []string{profile},
		"propertySources": []map[string]interface{}{
			{
				"name":   fmt.Sprintf("%s/%s", application, profile),
				"source": config,
			},
		},
	})
}

func (cs *ConfigServer) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := cs.loadConfigs(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "refreshed"})
}

func (cs *ConfigServer) handleEncrypt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Placeholder for encryption
	http.Error(w, "Encryption not implemented", http.StatusNotImplemented)
}

func (cs *ConfigServer) handleDecrypt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Placeholder for decryption
	http.Error(w, "Decryption not implemented", http.StatusNotImplemented)
}
