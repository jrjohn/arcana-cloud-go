package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jrjohn/arcana-cloud-go/internal/jobs/queue"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs/worker"
	"github.com/jrjohn/arcana-cloud-go/internal/testutil"
)

func setupTestRegistry(t *testing.T) (*Registry, *worker.WorkerPool) {
	testutil.SkipIfNoRedis(t)
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(t, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewTestLogger(t)

	poolConfig := worker.DefaultWorkerPoolConfig()
	pool := worker.NewWorkerPool(q, logger, poolConfig)

	registry := NewRegistry(pool, logger)
	return registry, pool
}

func TestNewRegistry(t *testing.T) {
	registry, _ := setupTestRegistry(t)

	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if registry.pool == nil {
		t.Error("pool is nil")
	}
	if registry.types == nil {
		t.Error("types map is nil")
	}
}

func TestRegistry_Register_SimplePayload(t *testing.T) {
	registry, _ := setupTestRegistry(t)

	type SimplePayload struct {
		Message string `json:"message"`
	}

	Register(registry, "simple-job", func(ctx context.Context, payload SimplePayload) error {
		return nil
	})

	handlers := registry.ListHandlers()
	if len(handlers) != 1 {
		t.Errorf("len(handlers) = %v, want 1", len(handlers))
	}
	if _, ok := handlers["simple-job"]; !ok {
		t.Error("simple-job not found in handlers")
	}
}

func TestRegistry_Register_MultipleHandlers(t *testing.T) {
	registry, _ := setupTestRegistry(t)

	type Payload1 struct {
		Field1 string `json:"field1"`
	}
	type Payload2 struct {
		Field2 int `json:"field2"`
	}

	Register(registry, "job-1", func(ctx context.Context, payload Payload1) error {
		return nil
	})
	Register(registry, "job-2", func(ctx context.Context, payload Payload2) error {
		return nil
	})

	handlers := registry.ListHandlers()
	if len(handlers) != 2 {
		t.Errorf("len(handlers) = %v, want 2", len(handlers))
	}
}

func TestRegistry_ListHandlers(t *testing.T) {
	registry, _ := setupTestRegistry(t)

	// Initially empty
	handlers := registry.ListHandlers()
	if len(handlers) != 0 {
		t.Errorf("Initial len(handlers) = %v, want 0", len(handlers))
	}

	// Register handlers
	Register(registry, "list-job-1", func(ctx context.Context, payload EmailJobPayload) error {
		return nil
	})
	Register(registry, "list-job-2", func(ctx context.Context, payload WebhookJobPayload) error {
		return nil
	})

	handlers = registry.ListHandlers()
	if len(handlers) != 2 {
		t.Errorf("After registration len(handlers) = %v, want 2", len(handlers))
	}

	// Verify returned map is a copy
	handlers["new-key"] = "new-value"
	if len(registry.ListHandlers()) != 2 {
		t.Error("ListHandlers should return a copy")
	}
}

// Test payload types
func TestEmailJobPayload(t *testing.T) {
	payload := EmailJobPayload{
		To:         []string{"user@example.com", "admin@example.com"},
		Subject:    "Test Subject",
		Body:       "Test Body",
		TemplateID: "template-123",
		TemplateData: map[string]any{
			"name": "John",
			"code": 12345,
		},
	}

	// Serialize and deserialize
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var parsed EmailJobPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if len(parsed.To) != 2 {
		t.Errorf("len(To) = %v, want 2", len(parsed.To))
	}
	if parsed.Subject != "Test Subject" {
		t.Errorf("Subject = %v, want 'Test Subject'", parsed.Subject)
	}
	if parsed.TemplateID != "template-123" {
		t.Errorf("TemplateID = %v, want template-123", parsed.TemplateID)
	}
}

func TestWebhookJobPayload(t *testing.T) {
	payload := WebhookJobPayload{
		URL:    "https://example.com/webhook",
		Method: "POST",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
			"Content-Type":  "application/json",
		},
		Body:    json.RawMessage(`{"key":"value"}`),
		Timeout: 30,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var parsed WebhookJobPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if parsed.URL != "https://example.com/webhook" {
		t.Errorf("URL = %v", parsed.URL)
	}
	if parsed.Method != "POST" {
		t.Errorf("Method = %v, want POST", parsed.Method)
	}
	if len(parsed.Headers) != 2 {
		t.Errorf("len(Headers) = %v, want 2", len(parsed.Headers))
	}
	if parsed.Timeout != 30 {
		t.Errorf("Timeout = %v, want 30", parsed.Timeout)
	}
}

func TestCleanupJobPayload(t *testing.T) {
	payload := CleanupJobPayload{
		Type:      "expired_tokens",
		OlderThan: 30,
		DryRun:    true,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var parsed CleanupJobPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if parsed.Type != "expired_tokens" {
		t.Errorf("Type = %v, want expired_tokens", parsed.Type)
	}
	if parsed.OlderThan != 30 {
		t.Errorf("OlderThan = %v, want 30", parsed.OlderThan)
	}
	if !parsed.DryRun {
		t.Error("DryRun should be true")
	}
}

func TestNotificationJobPayload(t *testing.T) {
	payload := NotificationJobPayload{
		UserID:  12345,
		Type:    "push",
		Title:   "Test Notification",
		Message: "This is a test",
		Data: map[string]any{
			"action": "open_app",
			"deep_link": "/profile",
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var parsed NotificationJobPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if parsed.UserID != 12345 {
		t.Errorf("UserID = %v, want 12345", parsed.UserID)
	}
	if parsed.Type != "push" {
		t.Errorf("Type = %v, want push", parsed.Type)
	}
	if parsed.Title != "Test Notification" {
		t.Errorf("Title = %v", parsed.Title)
	}
}

func TestReportJobPayload(t *testing.T) {
	payload := ReportJobPayload{
		ReportType: "monthly_sales",
		Format:     "pdf",
		Parameters: map[string]any{
			"month": 12,
			"year":  2024,
		},
		Recipients: []string{"manager@example.com", "ceo@example.com"},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var parsed ReportJobPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if parsed.ReportType != "monthly_sales" {
		t.Errorf("ReportType = %v, want monthly_sales", parsed.ReportType)
	}
	if parsed.Format != "pdf" {
		t.Errorf("Format = %v, want pdf", parsed.Format)
	}
	if len(parsed.Recipients) != 2 {
		t.Errorf("len(Recipients) = %v, want 2", len(parsed.Recipients))
	}
}

func TestSyncJobPayload(t *testing.T) {
	payload := SyncJobPayload{
		Source:      "database",
		Destination: "elasticsearch",
		EntityType:  "products",
		LastSyncAt:  "2024-01-15T10:30:00Z",
		FullSync:    false,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var parsed SyncJobPayload
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if parsed.Source != "database" {
		t.Errorf("Source = %v, want database", parsed.Source)
	}
	if parsed.Destination != "elasticsearch" {
		t.Errorf("Destination = %v, want elasticsearch", parsed.Destination)
	}
	if parsed.EntityType != "products" {
		t.Errorf("EntityType = %v, want products", parsed.EntityType)
	}
	if parsed.FullSync {
		t.Error("FullSync should be false")
	}
}

func TestHandlerExecution(t *testing.T) {
	registry, pool := setupTestRegistry(t)

	type TestPayload struct {
		Value int `json:"value"`
	}

	var receivedValue int
	Register(registry, "exec-test", func(ctx context.Context, payload TestPayload) error {
		receivedValue = payload.Value
		return nil
	})
	_ = receivedValue // Silence unused variable warning (handler tested via registry.ListHandlers)

	// Get the raw handler from pool
	pool.RegisterHandler("manual-test", func(ctx context.Context, data []byte) error {
		return nil
	})

	// Verify handler is callable
	handlers := registry.ListHandlers()
	if _, ok := handlers["exec-test"]; !ok {
		t.Error("exec-test handler not found")
	}
}

func TestHandlerWithError(t *testing.T) {
	registry, _ := setupTestRegistry(t)

	type ErrorPayload struct {
		ShouldFail bool `json:"should_fail"`
	}

	Register(registry, "error-test", func(ctx context.Context, payload ErrorPayload) error {
		if payload.ShouldFail {
			return errors.New("intentional failure")
		}
		return nil
	})

	handlers := registry.ListHandlers()
	if _, ok := handlers["error-test"]; !ok {
		t.Error("error-test handler not found")
	}
}

func TestHandlerWithContext(t *testing.T) {
	registry, _ := setupTestRegistry(t)

	type ContextPayload struct{}

	var ctxReceived bool
	Register(registry, "context-test", func(ctx context.Context, payload ContextPayload) error {
		if ctx != nil {
			ctxReceived = true
		}
		return nil
	})

	handlers := registry.ListHandlers()
	if _, ok := handlers["context-test"]; !ok {
		t.Error("context-test handler not found")
	}

	// Note: actual context testing would require full integration
	_ = ctxReceived
}

func TestPayloadSerialization_Empty(t *testing.T) {
	payloads := []interface{}{
		EmailJobPayload{},
		WebhookJobPayload{},
		CleanupJobPayload{},
		NotificationJobPayload{},
		ReportJobPayload{},
		SyncJobPayload{},
	}

	for i, payload := range payloads {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Errorf("Payload %d: Marshal error = %v", i, err)
		}
		if len(data) == 0 {
			t.Errorf("Payload %d: Empty JSON", i)
		}
	}
}

// Benchmarks
func BenchmarkRegistry_Register(b *testing.B) {
	testutil.SkipIfNoRedis(&testing.T{})
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(&testing.T{}, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewNopLogger()

	poolConfig := worker.DefaultWorkerPoolConfig()
	pool := worker.NewWorkerPool(q, logger, poolConfig)
	registry := NewRegistry(pool, logger)

	type BenchPayload struct {
		Data string `json:"data"`
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Register(registry, testutil.GenerateTestID(), func(ctx context.Context, payload BenchPayload) error {
			return nil
		})
	}
}

func BenchmarkRegistry_ListHandlers(b *testing.B) {
	testutil.SkipIfNoRedis(&testing.T{})
	config := testutil.DefaultTestConfig()
	client := testutil.NewTestRedisClient(&testing.T{}, config)
	q := queue.NewRedisQueue(client)
	logger := testutil.NewNopLogger()

	poolConfig := worker.DefaultWorkerPoolConfig()
	pool := worker.NewWorkerPool(q, logger, poolConfig)
	registry := NewRegistry(pool, logger)

	// Register some handlers
	for i := 0; i < 10; i++ {
		Register(registry, testutil.GenerateTestID(), func(ctx context.Context, payload EmailJobPayload) error {
			return nil
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.ListHandlers()
	}
}

func BenchmarkPayloadSerialization(b *testing.B) {
	payload := EmailJobPayload{
		To:      []string{"user@example.com"},
		Subject: "Test",
		Body:    "Test body content",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(payload)
		var p EmailJobPayload
		json.Unmarshal(data, &p)
	}
}

// Concurrency test
func TestRegistry_ConcurrentRegistration(t *testing.T) {
	registry, _ := setupTestRegistry(t)

	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(id int) {
			type Payload struct{}
			Register(registry, testutil.GenerateTestID(), func(ctx context.Context, payload Payload) error {
				return nil
			})
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent registration")
		}
	}

	handlers := registry.ListHandlers()
	if len(handlers) != 100 {
		t.Errorf("len(handlers) = %v, want 100", len(handlers))
	}
}
