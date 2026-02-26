package manager

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	pluginapi "github.com/jrjohn/arcana-cloud-go/internal/plugin/api"
)

const (
	testPluginName   = testPluginName
	pluginName1      = pluginName1
	pluginName2      = pluginName2
	nonExistingPlugin = nonExistingPlugin
	msgStateFormat   = msgStateFormat
	msgLoadAllError  = msgLoadAllError
	msgListLength    = msgListLength
	msgGetPlugin     = msgGetPlugin
)

func newTestManager(t *testing.T) (*Manager, string) {
	t.Helper()
	dir := t.TempDir()
	logger := zap.NewNop()
	return NewManager(dir, logger), dir
}

func TestPluginState_Constants(t *testing.T) {
	tests := []struct {
		name     string
		state    PluginState
		expected string
	}{
		{"LOADED", StateLoaded, "LOADED"},
		{"STARTED", StateStarted, "STARTED"},
		{"STOPPED", StateStopped, "STOPPED"},
		{"ERROR", StateError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("PluginState = %v, want %v", tt.state, tt.expected)
			}
		})
	}
}

func TestNewManager(t *testing.T) {
	dir := t.TempDir()
	logger := zap.NewNop()

	manager := NewManager(dir, logger)

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}
	if manager.pluginsDir != dir {
		t.Errorf("pluginsDir = %v, want %v", manager.pluginsDir, dir)
	}
	if manager.plugins == nil {
		t.Error("plugins map should be initialized")
	}
	if manager.logger == nil {
		t.Error("logger should be set")
	}
	if manager.events == nil {
		t.Error("events channel should be initialized")
	}
}

func TestManagedPlugin_Struct(t *testing.T) {
	info := pluginapi.PluginInfo{
		Key:     testPluginName,
		Name:    "Test Plugin",
		Version: "1.0.0",
	}

	managed := &ManagedPlugin{
		Info:   info,
		Plugin: nil,
		State:  StateLoaded,
		Path:   "/path/to/plugin.so",
		Error:  nil,
		Config: map[string]any{"key": "value"},
	}

	if managed.Info.Key != testPluginName {
		t.Errorf("Info.Key = %v, want test-plugin", managed.Info.Key)
	}
	if managed.State != StateLoaded {
		t.Errorf(msgStateFormat, managed.State, StateLoaded)
	}
	if managed.Path != "/path/to/plugin.so" {
		t.Errorf("Path = %v, want /path/to/plugin.so", managed.Path)
	}
	if managed.Config["key"] != "value" {
		t.Errorf("Config[key] = %v, want value", managed.Config["key"])
	}
}

func TestManager_LoadAll_EmptyDir(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	err := manager.LoadAll(ctx)
	if err != nil {
		t.Errorf(msgLoadAllError, err)
	}

	plugins := manager.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("ListPlugins() length = %v, want 0", len(plugins))
	}
}

func TestManager_LoadAll_CreatesDirIfNotExists(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir", "plugins")
	logger := zap.NewNop()
	manager := NewManager(dir, logger)

	ctx := context.Background()
	err := manager.LoadAll(ctx)
	if err != nil {
		t.Errorf(msgLoadAllError, err)
	}

	// Directory should exist now
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Directory should have been created")
	}
}

func TestManager_LoadAll_IgnoresNonSoFiles(t *testing.T) {
	manager, dir := newTestManager(t)
	ctx := context.Background()

	// Create non-.so files
	files := []string{"README.md", "config.json", "plugin.txt"}
	for _, f := range files {
		path := filepath.Join(dir, f)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	err := manager.LoadAll(ctx)
	if err != nil {
		t.Errorf(msgLoadAllError, err)
	}

	// Should not load any plugins
	plugins := manager.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("ListPlugins() length = %v, want 0", len(plugins))
	}
}

func TestManager_LoadAll_IgnoresDirectories(t *testing.T) {
	manager, dir := newTestManager(t)
	ctx := context.Background()

	// Create a subdirectory
	subdir := filepath.Join(dir, "subplugin")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	err := manager.LoadAll(ctx)
	if err != nil {
		t.Errorf(msgLoadAllError, err)
	}

	plugins := manager.ListPlugins()
	if len(plugins) != 0 {
		t.Errorf("ListPlugins() length = %v, want 0", len(plugins))
	}
}

func TestManager_GetPlugin(t *testing.T) {
	manager, _ := newTestManager(t)

	// Add a mock plugin manually
	manager.plugins[testPluginName] = &ManagedPlugin{
		Info: pluginapi.PluginInfo{
			Key:  testPluginName,
			Name: "Test",
		},
		State: StateLoaded,
	}

	t.Run("existing plugin", func(t *testing.T) {
		plugin, exists := manager.GetPlugin(testPluginName)
		if !exists {
			t.Error("Plugin should exist")
		}
		if plugin == nil {
			t.Error("Plugin should not be nil")
		}
		if plugin.Info.Key != testPluginName {
			t.Errorf("Plugin Key = %v, want test-plugin", plugin.Info.Key)
		}
	})

	t.Run("non-existing plugin", func(t *testing.T) {
		plugin, exists := manager.GetPlugin(nonExistingPlugin)
		if exists {
			t.Error("Plugin should not exist")
		}
		if plugin != nil {
			t.Error("Plugin should be nil")
		}
	})
}

func TestManager_ListPlugins(t *testing.T) {
	manager, _ := newTestManager(t)

	// Add mock plugins
	manager.plugins[pluginName1] = &ManagedPlugin{
		Info:  pluginapi.PluginInfo{Key: pluginName1},
		State: StateLoaded,
	}
	manager.plugins[pluginName2] = &ManagedPlugin{
		Info:  pluginapi.PluginInfo{Key: pluginName2},
		State: StateStarted,
	}

	plugins := manager.ListPlugins()
	if len(plugins) != 2 {
		t.Errorf("ListPlugins() length = %v, want 2", len(plugins))
	}
}

func TestManager_StartPlugin_NotFound(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	err := manager.StartPlugin(ctx, nonExistingPlugin, nil)
	if err == nil {
		t.Error("StartPlugin() should return error for non-existing plugin")
	}
}

func TestManager_StopPlugin_NotFound(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	err := manager.StopPlugin(ctx, nonExistingPlugin)
	if err == nil {
		t.Error("StopPlugin() should return error for non-existing plugin")
	}
}

func TestManager_StopPlugin_NotStarted(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	manager.plugins[testPluginName] = &ManagedPlugin{
		Info:  pluginapi.PluginInfo{Key: testPluginName},
		State: StateLoaded, // Not started
	}

	err := manager.StopPlugin(ctx, testPluginName)
	if err != nil {
		t.Errorf("StopPlugin() for non-started plugin should not return error, got %v", err)
	}
}

func TestManager_UnloadPlugin_NotFound(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	err := manager.UnloadPlugin(ctx, nonExistingPlugin)
	if err == nil {
		t.Error("UnloadPlugin() should return error for non-existing plugin")
	}
}

func TestManager_GetRESTRoutes_NoPlugins(t *testing.T) {
	manager, _ := newTestManager(t)

	routes := manager.GetRESTRoutes()
	if len(routes) != 0 {
		t.Errorf("GetRESTRoutes() length = %v, want 0", len(routes))
	}
}

func TestManager_GetMiddlewares_NoPlugins(t *testing.T) {
	manager, _ := newTestManager(t)

	middlewares := manager.GetMiddlewares()
	if len(middlewares) != 0 {
		t.Errorf("GetMiddlewares() length = %v, want 0", len(middlewares))
	}
}

func TestManager_EmitEvent_NoListeners(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	event := pluginapi.Event{
		Type:    "test.event",
		Payload: map[string]any{"key": "value"},
	}

	err := manager.EmitEvent(ctx, event)
	if err != nil {
		t.Errorf("EmitEvent() error = %v", err)
	}
}

func TestManager_Shutdown(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	// Add some mock plugins
	manager.plugins[pluginName1] = &ManagedPlugin{
		Info:  pluginapi.PluginInfo{Key: pluginName1},
		State: StateLoaded,
	}
	manager.plugins[pluginName2] = &ManagedPlugin{
		Info:  pluginapi.PluginInfo{Key: pluginName2},
		State: StateStopped,
	}

	err := manager.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	if len(manager.plugins) != 0 {
		t.Errorf("plugins should be empty after shutdown, got %d", len(manager.plugins))
	}
}

func TestManager_Concurrency(t *testing.T) {
	manager, _ := newTestManager(t)

	// Add a plugin
	manager.plugins[testPluginName] = &ManagedPlugin{
		Info:  pluginapi.PluginInfo{Key: testPluginName},
		State: StateLoaded,
	}

	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			manager.GetPlugin(testPluginName)
			manager.ListPlugins()
			manager.GetRESTRoutes()
			manager.GetMiddlewares()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Mock plugin for testing
type testMockPlugin struct {
	info        pluginapi.PluginInfo
	initError   error
	startError  error
	stopError   error
}

func (p *testMockPlugin) Info() pluginapi.PluginInfo {
	return p.info
}

func (p *testMockPlugin) Init(ctx context.Context, config map[string]any) error {
	return p.initError
}

func (p *testMockPlugin) Start(ctx context.Context) error {
	return p.startError
}

func (p *testMockPlugin) Stop(ctx context.Context) error {
	return p.stopError
}

func (p *testMockPlugin) Health(ctx context.Context) (pluginapi.HealthStatus, error) {
	return pluginapi.HealthStatus{Status: "healthy"}, nil
}

func TestManager_StartPlugin_WithMock(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	mockPlugin := &testMockPlugin{
		info: pluginapi.PluginInfo{
			Key:     "mock-plugin",
			Name:    "Mock",
			Version: "1.0.0",
		},
	}

	manager.plugins["mock-plugin"] = &ManagedPlugin{
		Info:   mockPlugin.info,
		Plugin: mockPlugin,
		State:  StateLoaded,
	}

	err := manager.StartPlugin(ctx, "mock-plugin", nil)
	if err != nil {
		t.Errorf("StartPlugin() error = %v", err)
	}

	plugin, _ := manager.GetPlugin("mock-plugin")
	if plugin.State != StateStarted {
		t.Errorf(msgStateFormat, plugin.State, StateStarted)
	}
}

func TestManager_StartPlugin_AlreadyStarted(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	mockPlugin := &testMockPlugin{
		info: pluginapi.PluginInfo{Key: "mock-plugin"},
	}

	manager.plugins["mock-plugin"] = &ManagedPlugin{
		Info:   mockPlugin.info,
		Plugin: mockPlugin,
		State:  StateStarted, // Already started
	}

	err := manager.StartPlugin(ctx, "mock-plugin", nil)
	if err != nil {
		t.Errorf("StartPlugin() on already started plugin should not error, got %v", err)
	}
}

func TestManager_StopPlugin_WithMock(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	mockPlugin := &testMockPlugin{
		info: pluginapi.PluginInfo{Key: "mock-plugin"},
	}

	manager.plugins["mock-plugin"] = &ManagedPlugin{
		Info:   mockPlugin.info,
		Plugin: mockPlugin,
		State:  StateStarted,
	}

	err := manager.StopPlugin(ctx, "mock-plugin")
	if err != nil {
		t.Errorf("StopPlugin() error = %v", err)
	}

	plugin, _ := manager.GetPlugin("mock-plugin")
	if plugin.State != StateStopped {
		t.Errorf(msgStateFormat, plugin.State, StateStopped)
	}
}

func TestManager_UnloadPlugin_WithMock(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	mockPlugin := &testMockPlugin{
		info: pluginapi.PluginInfo{Key: "mock-plugin"},
	}

	manager.plugins["mock-plugin"] = &ManagedPlugin{
		Info:   mockPlugin.info,
		Plugin: mockPlugin,
		State:  StateLoaded,
	}

	err := manager.UnloadPlugin(ctx, "mock-plugin")
	if err != nil {
		t.Errorf("UnloadPlugin() error = %v", err)
	}

	_, exists := manager.GetPlugin("mock-plugin")
	if exists {
		t.Error("Plugin should not exist after unload")
	}
}

func TestManager_UnloadPlugin_StopsIfStarted(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	mockPlugin := &testMockPlugin{
		info: pluginapi.PluginInfo{Key: "mock-plugin"},
	}

	manager.plugins["mock-plugin"] = &ManagedPlugin{
		Info:   mockPlugin.info,
		Plugin: mockPlugin,
		State:  StateStarted, // Started plugin
	}

	err := manager.UnloadPlugin(ctx, "mock-plugin")
	if err != nil {
		t.Errorf("UnloadPlugin() error = %v", err)
	}

	_, exists := manager.GetPlugin("mock-plugin")
	if exists {
		t.Error("Plugin should not exist after unload")
	}
}

// Benchmarks
func BenchmarkManager_GetPlugin(b *testing.B) {
	manager := NewManager("/tmp/plugins", zap.NewNop())
	manager.plugins[testPluginName] = &ManagedPlugin{
		Info:  pluginapi.PluginInfo{Key: testPluginName},
		State: StateLoaded,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.GetPlugin(testPluginName)
	}
}

func BenchmarkManager_ListPlugins(b *testing.B) {
	manager := NewManager("/tmp/plugins", zap.NewNop())
	for i := 0; i < 10; i++ {
		key := "plugin-" + string(rune('A'+i))
		manager.plugins[key] = &ManagedPlugin{
			Info:  pluginapi.PluginInfo{Key: key},
			State: StateLoaded,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.ListPlugins()
	}
}
