package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/jobs/worker"
)

// HandlerFunc is a typed handler function
type HandlerFunc[T any] func(ctx context.Context, payload T) error

// Registry manages job handler registration
type Registry struct {
	pool   *worker.WorkerPool
	logger *zap.Logger
	mu     sync.RWMutex
	types  map[string]string // jobType -> Go type name for documentation
}

// NewRegistry creates a new handler registry
func NewRegistry(pool *worker.WorkerPool, logger *zap.Logger) *Registry {
	return &Registry{
		pool:   pool,
		logger: logger,
		types:  make(map[string]string),
	}
}

// Register registers a typed handler for a job type
func Register[T any](r *Registry, jobType string, handler HandlerFunc[T]) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Store type info for documentation
	var zero T
	r.types[jobType] = fmt.Sprintf("%T", zero)

	// Create wrapper that deserializes payload
	wrappedHandler := func(ctx context.Context, data []byte) error {
		var payload T
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload: %w", err)
		}
		return handler(ctx, payload)
	}

	r.pool.RegisterHandler(jobType, wrappedHandler)
	r.logger.Info("Registered typed job handler",
		zap.String("job_type", jobType),
		zap.String("payload_type", r.types[jobType]),
	)
}

// ListHandlers returns all registered handler types
func (r *Registry) ListHandlers() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range r.types {
		result[k] = v
	}
	return result
}

// Example job payload types

// EmailJobPayload is the payload for email jobs
type EmailJobPayload struct {
	To          []string          `json:"to"`
	Subject     string            `json:"subject"`
	Body        string            `json:"body"`
	TemplateID  string            `json:"template_id,omitempty"`
	TemplateData map[string]any   `json:"template_data,omitempty"`
}

// WebhookJobPayload is the payload for webhook jobs
type WebhookJobPayload struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"`
	Timeout int               `json:"timeout_seconds,omitempty"`
}

// CleanupJobPayload is the payload for cleanup jobs
type CleanupJobPayload struct {
	Type       string `json:"type"` // "expired_tokens", "old_logs", etc.
	OlderThan  int    `json:"older_than_days"`
	DryRun     bool   `json:"dry_run"`
}

// NotificationJobPayload is the payload for notification jobs
type NotificationJobPayload struct {
	UserID    uint              `json:"user_id"`
	Type      string            `json:"type"` // "push", "sms", "in_app"
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Data      map[string]any    `json:"data,omitempty"`
}

// ReportJobPayload is the payload for report generation jobs
type ReportJobPayload struct {
	ReportType string            `json:"report_type"`
	Format     string            `json:"format"` // "pdf", "csv", "xlsx"
	Parameters map[string]any    `json:"parameters,omitempty"`
	Recipients []string          `json:"recipients,omitempty"`
}

// SyncJobPayload is the payload for data sync jobs
type SyncJobPayload struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	EntityType  string `json:"entity_type"`
	LastSyncAt  string `json:"last_sync_at,omitempty"`
	FullSync    bool   `json:"full_sync"`
}
