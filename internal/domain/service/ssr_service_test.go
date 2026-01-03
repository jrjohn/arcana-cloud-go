package service

import (
	"context"
	"testing"
	"time"
)

func TestNewSSRService(t *testing.T) {
	svc := NewSSRService(true, 5*time.Minute)
	if svc == nil {
		t.Fatal("NewSSRService() returned nil")
	}
}

func TestSSRService_RenderReact_Success(t *testing.T) {
	svc := NewSSRService(false, 5*time.Minute)
	ctx := context.Background()

	props := map[string]any{
		"title": "Test Title",
		"count": 42,
	}

	result, err := svc.RenderReact(ctx, "TestComponent", props)
	if err != nil {
		t.Fatalf("RenderReact() error = %v", err)
	}
	if result == nil {
		t.Fatal("RenderReact() returned nil result")
	}
	if result.HTML == "" {
		t.Error("RenderReact() HTML is empty")
	}
	if result.Cached {
		t.Error("RenderReact() Cached should be false for first render")
	}
}

func TestSSRService_RenderReact_WithCache(t *testing.T) {
	svc := NewSSRService(true, 5*time.Minute)
	ctx := context.Background()

	props := map[string]any{"title": "Test"}

	// First render
	result1, err := svc.RenderReact(ctx, "CachedComponent", props)
	if err != nil {
		t.Fatalf("First RenderReact() error = %v", err)
	}
	if result1.Cached {
		t.Error("First RenderReact() Cached should be false")
	}

	// Second render should be cached
	result2, err := svc.RenderReact(ctx, "CachedComponent", props)
	if err != nil {
		t.Fatalf("Second RenderReact() error = %v", err)
	}
	if !result2.Cached {
		t.Error("Second RenderReact() Cached should be true")
	}
}

func TestSSRService_RenderAngular_Success(t *testing.T) {
	svc := NewSSRService(false, 5*time.Minute)
	ctx := context.Background()

	props := map[string]any{
		"title": "Test Title",
		"count": 42,
	}

	result, err := svc.RenderAngular(ctx, "TestComponent", props)
	if err != nil {
		t.Fatalf("RenderAngular() error = %v", err)
	}
	if result == nil {
		t.Fatal("RenderAngular() returned nil result")
	}
	if result.HTML == "" {
		t.Error("RenderAngular() HTML is empty")
	}
	if result.Cached {
		t.Error("RenderAngular() Cached should be false for first render")
	}
}

func TestSSRService_RenderAngular_WithCache(t *testing.T) {
	svc := NewSSRService(true, 5*time.Minute)
	ctx := context.Background()

	props := map[string]any{"title": "Test"}

	// First render
	result1, err := svc.RenderAngular(ctx, "AngularComponent", props)
	if err != nil {
		t.Fatalf("First RenderAngular() error = %v", err)
	}
	if result1.Cached {
		t.Error("First RenderAngular() Cached should be false")
	}

	// Second render should be cached
	result2, err := svc.RenderAngular(ctx, "AngularComponent", props)
	if err != nil {
		t.Fatalf("Second RenderAngular() error = %v", err)
	}
	if !result2.Cached {
		t.Error("Second RenderAngular() Cached should be true")
	}
}

func TestSSRService_GetStatus(t *testing.T) {
	svc := NewSSRService(true, 5*time.Minute)
	ctx := context.Background()

	status, err := svc.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if status == nil {
		t.Fatal("GetStatus() returned nil status")
	}
	if status.Status != "ready" {
		t.Errorf("GetStatus() Status = %v, want ready", status.Status)
	}
	if !status.CacheEnabled {
		t.Error("GetStatus() CacheEnabled should be true")
	}
}

func TestSSRService_GetStatus_NoCache(t *testing.T) {
	svc := NewSSRService(false, 5*time.Minute)
	ctx := context.Background()

	status, err := svc.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if status.CacheEnabled {
		t.Error("GetStatus() CacheEnabled should be false")
	}
}

func TestSSRService_GetStatus_WithCacheEntries(t *testing.T) {
	svc := NewSSRService(true, 5*time.Minute)
	ctx := context.Background()

	// Render some components to populate cache
	svc.RenderReact(ctx, "Component1", nil)
	svc.RenderReact(ctx, "Component2", nil)
	svc.RenderAngular(ctx, "Component3", nil)

	status, err := svc.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if status.CacheSize != 3 {
		t.Errorf("GetStatus() CacheSize = %v, want 3", status.CacheSize)
	}
}

func TestSSRService_ClearCache(t *testing.T) {
	svc := NewSSRService(true, 5*time.Minute)
	ctx := context.Background()

	// Populate cache
	svc.RenderReact(ctx, "Component1", nil)
	svc.RenderReact(ctx, "Component2", nil)

	// Clear cache
	err := svc.ClearCache(ctx)
	if err != nil {
		t.Fatalf("ClearCache() error = %v", err)
	}

	// Verify cache is empty
	status, _ := svc.GetStatus(ctx)
	if status.CacheSize != 0 {
		t.Errorf("ClearCache() CacheSize = %v, want 0", status.CacheSize)
	}
}

func TestSSRService_CacheExpiration(t *testing.T) {
	// Use very short TTL for testing
	svc := NewSSRService(true, 10*time.Millisecond)
	ctx := context.Background()

	// First render
	result1, _ := svc.RenderReact(ctx, "ExpiringComponent", nil)
	if result1.Cached {
		t.Error("First render should not be cached")
	}

	// Wait for cache to expire
	time.Sleep(20 * time.Millisecond)

	// Second render should not be cached (expired)
	result2, _ := svc.RenderReact(ctx, "ExpiringComponent", nil)
	if result2.Cached {
		t.Error("Second render should not be cached after expiration")
	}
}

func TestSSRService_Stats(t *testing.T) {
	svc := NewSSRService(true, 5*time.Minute)
	ctx := context.Background()

	// Render components
	svc.RenderReact(ctx, "ReactComp", nil)
	svc.RenderReact(ctx, "ReactComp", nil) // Cache hit
	svc.RenderAngular(ctx, "AngularComp", nil)

	status, _ := svc.GetStatus(ctx)

	if status.Stats["react_renders"] != 2 {
		t.Errorf("Stats[react_renders] = %v, want 2", status.Stats["react_renders"])
	}
	if status.Stats["angular_renders"] != 1 {
		t.Errorf("Stats[angular_renders] = %v, want 1", status.Stats["angular_renders"])
	}
	if status.Stats["cache_hits"] != 1 {
		t.Errorf("Stats[cache_hits] = %v, want 1", status.Stats["cache_hits"])
	}
	if status.Stats["cache_misses"] != 2 {
		t.Errorf("Stats[cache_misses] = %v, want 2", status.Stats["cache_misses"])
	}
}

func TestSSRService_RenderPlaceholder_NilProps(t *testing.T) {
	svc := NewSSRService(false, 5*time.Minute)
	ctx := context.Background()

	result, err := svc.RenderReact(ctx, "NilPropsComponent", nil)
	if err != nil {
		t.Fatalf("RenderReact() error = %v", err)
	}
	if result.HTML == "" {
		t.Error("RenderReact() HTML should not be empty")
	}
}

func TestSSRService_ConcurrentRenders(t *testing.T) {
	svc := NewSSRService(true, 5*time.Minute)
	ctx := context.Background()

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			_, err := svc.RenderReact(ctx, "ConcurrentComponent", map[string]any{"id": id})
			if err != nil {
				t.Errorf("RenderReact(%d) error = %v", id, err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestSSRRenderResult_Fields(t *testing.T) {
	result := &SSRRenderResult{
		HTML:       "<div>Test</div>",
		CSS:        ".test { color: red; }",
		Scripts:    []string{"/js/app.js"},
		State:      map[string]any{"key": "value"},
		RenderTime: 100 * time.Millisecond,
		Cached:     true,
	}

	if result.HTML != "<div>Test</div>" {
		t.Errorf("HTML = %v, want <div>Test</div>", result.HTML)
	}
	if result.CSS != ".test { color: red; }" {
		t.Errorf("CSS = %v, want .test { color: red; }", result.CSS)
	}
	if len(result.Scripts) != 1 {
		t.Errorf("Scripts length = %v, want 1", len(result.Scripts))
	}
	if result.State["key"] != "value" {
		t.Errorf("State[key] = %v, want value", result.State["key"])
	}
	if result.RenderTime != 100*time.Millisecond {
		t.Errorf("RenderTime = %v, want 100ms", result.RenderTime)
	}
	if !result.Cached {
		t.Error("Cached = false, want true")
	}
}

func TestSSRCacheEntry_Fields(t *testing.T) {
	result := &SSRRenderResult{HTML: "<div>Test</div>"}
	entry := &SSRCacheEntry{
		Result:    result,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if entry.Result != result {
		t.Error("Result mismatch")
	}
	if entry.ExpiresAt.Before(time.Now()) {
		t.Error("ExpiresAt should be in the future")
	}
}

func TestSSRStatus_Fields(t *testing.T) {
	status := &SSRStatus{
		Status:       "ready",
		ReactReady:   true,
		AngularReady: false,
		CacheEnabled: true,
		CacheSize:    5,
		Stats:        map[string]int64{"renders": 10},
	}

	if status.Status != "ready" {
		t.Errorf("Status = %v, want ready", status.Status)
	}
	if !status.ReactReady {
		t.Error("ReactReady = false, want true")
	}
	if status.AngularReady {
		t.Error("AngularReady = true, want false")
	}
	if !status.CacheEnabled {
		t.Error("CacheEnabled = false, want true")
	}
	if status.CacheSize != 5 {
		t.Errorf("CacheSize = %v, want 5", status.CacheSize)
	}
	if status.Stats["renders"] != 10 {
		t.Errorf("Stats[renders] = %v, want 10", status.Stats["renders"])
	}
}

// Test SSR error constants
func TestSSRServiceErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrSSREngineNotReady", ErrSSREngineNotReady, "SSR engine is not ready"},
		{"ErrSSRRenderFailed", ErrSSRRenderFailed, "SSR rendering failed"},
		{"ErrComponentNotFound", ErrComponentNotFound, "component not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("%s.Error() = %v, want %v", tt.name, tt.err.Error(), tt.expected)
			}
		})
	}
}

// Test toJSON helper function
func TestToJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"nil", nil, "{}"},
		{"empty map", map[string]any{}, "{}"},
		{"map with values", map[string]any{"key": "value"}, "{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toJSON(tt.input)
			if result != tt.expected {
				t.Errorf("toJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}
