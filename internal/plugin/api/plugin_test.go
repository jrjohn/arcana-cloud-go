package api

import (
	"context"
	"net/http"
	"testing"
)

func TestPluginType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		ptype    PluginType
		expected string
	}{
		{"REST_ENDPOINT", TypeRESTEndpoint, "REST_ENDPOINT"},
		{"SERVICE", TypeService, "SERVICE"},
		{"EVENT_LISTENER", TypeEventListener, "EVENT_LISTENER"},
		{"SCHEDULED_JOB", TypeScheduledJob, "SCHEDULED_JOB"},
		{"SSR_VIEW", TypeSSRView, "SSR_VIEW"},
		{"MIDDLEWARE", TypeMiddleware, "MIDDLEWARE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.ptype) != tt.expected {
				t.Errorf("PluginType = %v, want %v", tt.ptype, tt.expected)
			}
		})
	}
}

func TestPluginInfo_Struct(t *testing.T) {
	info := PluginInfo{
		Key:         "test-plugin",
		Name:        "Test Plugin",
		Description: "A test plugin",
		Version:     "1.0.0",
		Author:      "Test Author",
		Type:        TypeRESTEndpoint,
	}

	if info.Key != "test-plugin" {
		t.Errorf("Key = %v, want test-plugin", info.Key)
	}
	if info.Name != "Test Plugin" {
		t.Errorf("Name = %v, want Test Plugin", info.Name)
	}
	if info.Description != "A test plugin" {
		t.Errorf("Description = %v, want A test plugin", info.Description)
	}
	if info.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0", info.Version)
	}
	if info.Author != "Test Author" {
		t.Errorf("Author = %v, want Test Author", info.Author)
	}
	if info.Type != TypeRESTEndpoint {
		t.Errorf("Type = %v, want %v", info.Type, TypeRESTEndpoint)
	}
}

func TestHealthStatus_Struct(t *testing.T) {
	tests := []struct {
		name    string
		status  HealthStatus
		healthy bool
	}{
		{
			name: "healthy status",
			status: HealthStatus{
				Status:  "healthy",
				Message: "All systems operational",
				Details: map[string]string{"db": "connected"},
			},
			healthy: true,
		},
		{
			name: "unhealthy status",
			status: HealthStatus{
				Status:  "unhealthy",
				Message: "Database connection failed",
			},
			healthy: false,
		},
		{
			name: "degraded status",
			status: HealthStatus{
				Status:  "degraded",
				Message: "Some services unavailable",
				Details: map[string]string{"cache": "disconnected"},
			},
			healthy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status.Status == "" {
				t.Error("Status should not be empty")
			}
		})
	}
}

func TestRoute_Struct(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	route := Route{
		Method:      http.MethodGet,
		Path:        "/api/test",
		Handler:     handler,
		Middlewares: []func(http.Handler) http.Handler{middleware},
	}

	if route.Method != http.MethodGet {
		t.Errorf("Method = %v, want GET", route.Method)
	}
	if route.Path != "/api/test" {
		t.Errorf("Path = %v, want /api/test", route.Path)
	}
	if route.Handler == nil {
		t.Error("Handler should not be nil")
	}
	if len(route.Middlewares) != 1 {
		t.Errorf("Middlewares length = %v, want 1", len(route.Middlewares))
	}
}

func TestEvent_Struct(t *testing.T) {
	event := Event{
		Type: "user.created",
		Payload: map[string]any{
			"user_id":  123,
			"username": "testuser",
		},
	}

	if event.Type != "user.created" {
		t.Errorf("Type = %v, want user.created", event.Type)
	}
	if event.Payload == nil {
		t.Error("Payload should not be nil")
	}
	if event.Payload["user_id"] != 123 {
		t.Errorf("Payload user_id = %v, want 123", event.Payload["user_id"])
	}
}

func TestJob_Struct(t *testing.T) {
	job := Job{
		Name:     "cleanup",
		Schedule: "0 0 * * *",
		Handler: func(ctx context.Context) error {
			return nil
		},
	}

	if job.Name != "cleanup" {
		t.Errorf("Name = %v, want cleanup", job.Name)
	}
	if job.Schedule != "0 0 * * *" {
		t.Errorf("Schedule = %v, want 0 0 * * *", job.Schedule)
	}
	if job.Handler == nil {
		t.Error("Handler should not be nil")
	}

	// Test handler execution
	if err := job.Handler(context.Background()); err != nil {
		t.Errorf("Handler() error = %v", err)
	}
}

func TestSSRComponent_Struct(t *testing.T) {
	component := SSRComponent{
		Name:     "TestComponent",
		Template: "<div>{{.content}}</div>",
		Handler: func(ctx context.Context, props map[string]any) (string, error) {
			return "<div>rendered</div>", nil
		},
	}

	if component.Name != "TestComponent" {
		t.Errorf("Name = %v, want TestComponent", component.Name)
	}
	if component.Template == "" {
		t.Error("Template should not be empty")
	}
	if component.Handler == nil {
		t.Error("Handler should not be nil")
	}

	// Test handler execution
	result, err := component.Handler(context.Background(), nil)
	if err != nil {
		t.Errorf("Handler() error = %v", err)
	}
	if result != "<div>rendered</div>" {
		t.Errorf("Handler() result = %v, want <div>rendered</div>", result)
	}
}

// Mock Plugin Implementation for testing
type MockPlugin struct {
	info        PluginInfo
	initCalled  bool
	startCalled bool
	stopCalled  bool
	healthErr   error
}

func (m *MockPlugin) Info() PluginInfo {
	return m.info
}

func (m *MockPlugin) Init(ctx context.Context, config map[string]any) error {
	m.initCalled = true
	return nil
}

func (m *MockPlugin) Start(ctx context.Context) error {
	m.startCalled = true
	return nil
}

func (m *MockPlugin) Stop(ctx context.Context) error {
	m.stopCalled = true
	return nil
}

func (m *MockPlugin) Health(ctx context.Context) (HealthStatus, error) {
	if m.healthErr != nil {
		return HealthStatus{Status: "unhealthy"}, m.healthErr
	}
	return HealthStatus{Status: "healthy"}, nil
}

func TestMockPlugin_Interface(t *testing.T) {
	mock := &MockPlugin{
		info: PluginInfo{
			Key:     "mock-plugin",
			Name:    "Mock Plugin",
			Version: "1.0.0",
			Type:    TypeService,
		},
	}

	// Verify it implements Plugin interface
	var _ Plugin = mock

	// Test Info
	info := mock.Info()
	if info.Key != "mock-plugin" {
		t.Errorf("Info().Key = %v, want mock-plugin", info.Key)
	}

	// Test lifecycle
	ctx := context.Background()

	if err := mock.Init(ctx, nil); err != nil {
		t.Errorf("Init() error = %v", err)
	}
	if !mock.initCalled {
		t.Error("Init should have been called")
	}

	if err := mock.Start(ctx); err != nil {
		t.Errorf("Start() error = %v", err)
	}
	if !mock.startCalled {
		t.Error("Start should have been called")
	}

	health, err := mock.Health(ctx)
	if err != nil {
		t.Errorf("Health() error = %v", err)
	}
	if health.Status != "healthy" {
		t.Errorf("Health().Status = %v, want healthy", health.Status)
	}

	if err := mock.Stop(ctx); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
	if !mock.stopCalled {
		t.Error("Stop should have been called")
	}
}

// Mock RESTEndpointPlugin
type MockRESTPlugin struct {
	MockPlugin
	routes []Route
}

func (m *MockRESTPlugin) Routes() []Route {
	return m.routes
}

func TestMockRESTPlugin_Interface(t *testing.T) {
	mock := &MockRESTPlugin{
		MockPlugin: MockPlugin{
			info: PluginInfo{
				Key:  "rest-plugin",
				Type: TypeRESTEndpoint,
			},
		},
		routes: []Route{
			{Method: "GET", Path: "/api/test"},
			{Method: "POST", Path: "/api/test"},
		},
	}

	// Verify it implements RESTEndpointPlugin interface
	var _ RESTEndpointPlugin = mock

	routes := mock.Routes()
	if len(routes) != 2 {
		t.Errorf("Routes() length = %v, want 2", len(routes))
	}
}

// Mock EventListenerPlugin
type MockEventPlugin struct {
	MockPlugin
	events []string
	handled []Event
}

func (m *MockEventPlugin) Events() []string {
	return m.events
}

func (m *MockEventPlugin) HandleEvent(ctx context.Context, event Event) error {
	m.handled = append(m.handled, event)
	return nil
}

func TestMockEventPlugin_Interface(t *testing.T) {
	mock := &MockEventPlugin{
		MockPlugin: MockPlugin{
			info: PluginInfo{
				Key:  "event-plugin",
				Type: TypeEventListener,
			},
		},
		events: []string{"user.created", "user.deleted"},
	}

	// Verify it implements EventListenerPlugin interface
	var _ EventListenerPlugin = mock

	events := mock.Events()
	if len(events) != 2 {
		t.Errorf("Events() length = %v, want 2", len(events))
	}

	// Test event handling
	event := Event{Type: "user.created", Payload: map[string]any{"id": 1}}
	if err := mock.HandleEvent(context.Background(), event); err != nil {
		t.Errorf("HandleEvent() error = %v", err)
	}
	if len(mock.handled) != 1 {
		t.Errorf("handled length = %v, want 1", len(mock.handled))
	}
}

// Mock MiddlewarePlugin
type MockMiddlewarePlugin struct {
	MockPlugin
	priority int
}

func (m *MockMiddlewarePlugin) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return next
	}
}

func (m *MockMiddlewarePlugin) Priority() int {
	return m.priority
}

func TestMockMiddlewarePlugin_Interface(t *testing.T) {
	mock := &MockMiddlewarePlugin{
		MockPlugin: MockPlugin{
			info: PluginInfo{
				Key:  "middleware-plugin",
				Type: TypeMiddleware,
			},
		},
		priority: 10,
	}

	// Verify it implements MiddlewarePlugin interface
	var _ MiddlewarePlugin = mock

	if mock.Priority() != 10 {
		t.Errorf("Priority() = %v, want 10", mock.Priority())
	}

	mw := mock.Middleware()
	if mw == nil {
		t.Error("Middleware() should not return nil")
	}
}

// Benchmarks
func BenchmarkPluginInfo_Creation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = PluginInfo{
			Key:         "benchmark-plugin",
			Name:        "Benchmark Plugin",
			Description: "A benchmark plugin",
			Version:     "1.0.0",
			Author:      "Author",
			Type:        TypeRESTEndpoint,
		}
	}
}

func BenchmarkEvent_Creation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Event{
			Type: "test.event",
			Payload: map[string]any{
				"key": "value",
			},
		}
	}
}
