package api

import (
	"context"
	"net/http"
)

// PluginType represents the type of plugin
type PluginType string

const (
	TypeRESTEndpoint  PluginType = "REST_ENDPOINT"
	TypeService       PluginType = "SERVICE"
	TypeEventListener PluginType = "EVENT_LISTENER"
	TypeScheduledJob  PluginType = "SCHEDULED_JOB"
	TypeSSRView       PluginType = "SSR_VIEW"
	TypeMiddleware    PluginType = "MIDDLEWARE"
)

// PluginInfo contains plugin metadata
type PluginInfo struct {
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Version     string     `json:"version"`
	Author      string     `json:"author"`
	Type        PluginType `json:"type"`
}

// Plugin is the interface that all plugins must implement
type Plugin interface {
	// Info returns the plugin metadata
	Info() PluginInfo

	// Init initializes the plugin
	Init(ctx context.Context, config map[string]any) error

	// Start starts the plugin
	Start(ctx context.Context) error

	// Stop stops the plugin
	Stop(ctx context.Context) error

	// Health returns the plugin health status
	Health(ctx context.Context) (HealthStatus, error)
}

// HealthStatus represents plugin health
type HealthStatus struct {
	Status  string            `json:"status"` // "healthy", "unhealthy", "degraded"
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// RESTEndpointPlugin extends Plugin with REST endpoint capabilities
type RESTEndpointPlugin interface {
	Plugin

	// Routes returns the HTTP routes provided by this plugin
	Routes() []Route
}

// Route represents an HTTP route
type Route struct {
	Method      string
	Path        string
	Handler     http.HandlerFunc
	Middlewares []func(http.Handler) http.Handler
}

// ServicePlugin extends Plugin with service capabilities
type ServicePlugin interface {
	Plugin

	// Services returns the services provided by this plugin
	Services() map[string]any
}

// EventListenerPlugin extends Plugin with event handling capabilities
type EventListenerPlugin interface {
	Plugin

	// Events returns the events this plugin listens to
	Events() []string

	// HandleEvent handles an event
	HandleEvent(ctx context.Context, event Event) error
}

// Event represents an application event
type Event struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

// ScheduledJobPlugin extends Plugin with scheduled job capabilities
type ScheduledJobPlugin interface {
	Plugin

	// Jobs returns the scheduled jobs provided by this plugin
	Jobs() []Job
}

// Job represents a scheduled job
type Job struct {
	Name     string
	Schedule string // cron expression
	Handler  func(ctx context.Context) error
}

// MiddlewarePlugin extends Plugin with middleware capabilities
type MiddlewarePlugin interface {
	Plugin

	// Middleware returns the HTTP middleware
	Middleware() func(http.Handler) http.Handler

	// Priority returns the middleware priority (lower = earlier)
	Priority() int
}

// SSRViewPlugin extends Plugin with SSR view capabilities
type SSRViewPlugin interface {
	Plugin

	// Components returns the SSR components provided by this plugin
	Components() []SSRComponent
}

// SSRComponent represents an SSR component
type SSRComponent struct {
	Name     string
	Template string
	Handler  func(ctx context.Context, props map[string]any) (string, error)
}

// PluginContext provides context for plugins
type PluginContext interface {
	// Config returns the plugin configuration
	Config() map[string]any

	// Logger returns a logger for the plugin
	Logger() Logger

	// EmitEvent emits an event
	EmitEvent(ctx context.Context, event Event) error

	// GetService retrieves a service by name
	GetService(name string) (any, error)
}

// Logger is the logging interface for plugins
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
