package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"go.uber.org/zap"

	pluginapi "github.com/jrjohn/arcana-cloud-go/internal/plugin/api"
)

// PluginState represents the state of a managed plugin
type PluginState string

const (
	StateLoaded   PluginState = "LOADED"
	StateStarted  PluginState = "STARTED"
	StateStopped  PluginState = "STOPPED"
	StateError    PluginState = "ERROR"
)

const errFmtPluginNotFound = "plugin %s not found"

// ManagedPlugin represents a loaded plugin
type ManagedPlugin struct {
	Info    pluginapi.PluginInfo
	Plugin  pluginapi.Plugin
	State   PluginState
	Path    string
	Error   error
	Config  map[string]any
}

// Manager manages plugin lifecycle
type Manager struct {
	plugins    map[string]*ManagedPlugin
	pluginsDir string
	logger     *zap.Logger
	mutex      sync.RWMutex
	events     chan pluginapi.Event
}

// NewManager creates a new plugin manager
func NewManager(pluginsDir string, logger *zap.Logger) *Manager {
	return &Manager{
		plugins:    make(map[string]*ManagedPlugin),
		pluginsDir: pluginsDir,
		logger:     logger,
		events:     make(chan pluginapi.Event, 100),
	}
}

// LoadAll loads all plugins from the plugins directory
func (m *Manager) LoadAll(ctx context.Context) error {
	if err := os.MkdirAll(m.pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	entries, err := os.ReadDir(m.pluginsDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".so" {
			continue
		}

		path := filepath.Join(m.pluginsDir, entry.Name())
		if err := m.LoadPlugin(ctx, path); err != nil {
			m.logger.Error("failed to load plugin",
				zap.String("path", path),
				zap.Error(err),
			)
		}
	}

	return nil
}

// LoadPlugin loads a plugin from the given path
func (m *Manager) LoadPlugin(ctx context.Context, path string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Load the Go plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look up the plugin symbol
	sym, err := p.Lookup("ArcanaPlugin")
	if err != nil {
		return fmt.Errorf("plugin does not export ArcanaPlugin: %w", err)
	}

	// Cast to Plugin interface
	arcanaPlugin, ok := sym.(pluginapi.Plugin)
	if !ok {
		return fmt.Errorf("ArcanaPlugin does not implement Plugin interface")
	}

	info := arcanaPlugin.Info()

	// Check if already loaded
	if _, exists := m.plugins[info.Key]; exists {
		return fmt.Errorf("plugin %s is already loaded", info.Key)
	}

	managed := &ManagedPlugin{
		Info:   info,
		Plugin: arcanaPlugin,
		State:  StateLoaded,
		Path:   path,
	}

	m.plugins[info.Key] = managed

	m.logger.Info("plugin loaded",
		zap.String("key", info.Key),
		zap.String("name", info.Name),
		zap.String("version", info.Version),
	)

	return nil
}

// StartPlugin starts a loaded plugin
func (m *Manager) StartPlugin(ctx context.Context, key string, config map[string]any) error {
	m.mutex.Lock()
	managed, exists := m.plugins[key]
	m.mutex.Unlock()

	if !exists {
		return fmt.Errorf(errFmtPluginNotFound, key)
	}

	if managed.State == StateStarted {
		return nil
	}

	// Initialize the plugin
	if err := managed.Plugin.Init(ctx, config); err != nil {
		managed.State = StateError
		managed.Error = err
		return fmt.Errorf("failed to initialize plugin: %w", err)
	}

	// Start the plugin
	if err := managed.Plugin.Start(ctx); err != nil {
		managed.State = StateError
		managed.Error = err
		return fmt.Errorf("failed to start plugin: %w", err)
	}

	managed.State = StateStarted
	managed.Config = config

	m.logger.Info("plugin started", zap.String("key", key))

	return nil
}

// StopPlugin stops a running plugin
func (m *Manager) StopPlugin(ctx context.Context, key string) error {
	m.mutex.Lock()
	managed, exists := m.plugins[key]
	m.mutex.Unlock()

	if !exists {
		return fmt.Errorf(errFmtPluginNotFound, key)
	}

	if managed.State != StateStarted {
		return nil
	}

	if err := managed.Plugin.Stop(ctx); err != nil {
		managed.State = StateError
		managed.Error = err
		return fmt.Errorf("failed to stop plugin: %w", err)
	}

	managed.State = StateStopped

	m.logger.Info("plugin stopped", zap.String("key", key))

	return nil
}

// UnloadPlugin unloads a plugin
func (m *Manager) UnloadPlugin(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	managed, exists := m.plugins[key]
	if !exists {
		return fmt.Errorf(errFmtPluginNotFound, key)
	}

	if managed.State == StateStarted {
		if err := managed.Plugin.Stop(ctx); err != nil {
			m.logger.Warn("error stopping plugin before unload",
				zap.String("key", key),
				zap.Error(err),
			)
		}
	}

	delete(m.plugins, key)

	m.logger.Info("plugin unloaded", zap.String("key", key))

	return nil
}

// GetPlugin returns a managed plugin by key
func (m *Manager) GetPlugin(key string) (*ManagedPlugin, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	managed, exists := m.plugins[key]
	return managed, exists
}

// ListPlugins returns all managed plugins
func (m *Manager) ListPlugins() []*ManagedPlugin {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make([]*ManagedPlugin, 0, len(m.plugins))
	for _, managed := range m.plugins {
		result = append(result, managed)
	}
	return result
}

// GetRESTRoutes returns all REST routes from loaded plugins
func (m *Manager) GetRESTRoutes() []pluginapi.Route {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var routes []pluginapi.Route
	for _, managed := range m.plugins {
		if managed.State != StateStarted {
			continue
		}

		if restPlugin, ok := managed.Plugin.(pluginapi.RESTEndpointPlugin); ok {
			routes = append(routes, restPlugin.Routes()...)
		}
	}
	return routes
}

// GetMiddlewares returns all middleware from loaded plugins
func (m *Manager) GetMiddlewares() []pluginapi.MiddlewarePlugin {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var middlewares []pluginapi.MiddlewarePlugin
	for _, managed := range m.plugins {
		if managed.State != StateStarted {
			continue
		}

		if mwPlugin, ok := managed.Plugin.(pluginapi.MiddlewarePlugin); ok {
			middlewares = append(middlewares, mwPlugin)
		}
	}
	return middlewares
}

// pluginListensToEvent checks if a plugin is interested in the given event type
func pluginListensToEvent(plugin pluginapi.EventListenerPlugin, eventType string) bool {
	for _, e := range plugin.Events() {
		if e == eventType || e == "*" {
			return true
		}
	}
	return false
}

// dispatchEventToPlugin dispatches an event to a plugin in a goroutine
func (m *Manager) dispatchEventToPlugin(ctx context.Context, p pluginapi.EventListenerPlugin, event pluginapi.Event) {
	go func() {
		if err := p.HandleEvent(ctx, event); err != nil {
			m.logger.Error("plugin event handler error",
				zap.String("event", event.Type),
				zap.Error(err),
			)
		}
	}()
}

// EmitEvent emits an event to all event listener plugins
func (m *Manager) EmitEvent(ctx context.Context, event pluginapi.Event) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, managed := range m.plugins {
		if managed.State != StateStarted {
			continue
		}
		elPlugin, ok := managed.Plugin.(pluginapi.EventListenerPlugin)
		if !ok {
			continue
		}
		if pluginListensToEvent(elPlugin, event.Type) {
			m.dispatchEventToPlugin(ctx, elPlugin, event)
		}
	}

	return nil
}

// Shutdown stops all plugins and cleans up
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, managed := range m.plugins {
		if managed.State == StateStarted {
			if err := managed.Plugin.Stop(ctx); err != nil {
				m.logger.Warn("error stopping plugin during shutdown",
					zap.String("key", key),
					zap.Error(err),
				)
			}
		}
	}

	m.plugins = make(map[string]*ManagedPlugin)
	close(m.events)

	m.logger.Info("plugin manager shutdown complete")

	return nil
}
