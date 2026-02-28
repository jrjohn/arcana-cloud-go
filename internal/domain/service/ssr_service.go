package service

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSSREngineNotReady = errors.New("SSR engine is not ready")
	ErrSSRRenderFailed   = errors.New("SSR rendering failed")
	ErrComponentNotFound = errors.New("component not found")
)

// SSREngine represents a JavaScript rendering engine
type SSREngine interface {
	// Render renders a component with the given props
	Render(ctx context.Context, component string, props map[string]any) (string, error)

	// IsReady checks if the engine is ready
	IsReady() bool

	// Shutdown gracefully shuts down the engine
	Shutdown() error
}

// SSRRenderResult represents the result of an SSR render
type SSRRenderResult struct {
	HTML       string
	CSS        string
	Scripts    []string
	State      map[string]any
	RenderTime time.Duration
	Cached     bool
}

// SSRStatus represents the SSR engine status
type SSRStatus struct {
	Status       string
	ReactReady   bool
	AngularReady bool
	CacheEnabled bool
	CacheSize    int
	Stats        map[string]int64
}

// SSRService defines the interface for server-side rendering operations
type SSRService interface {
	// RenderReact renders a React component
	RenderReact(ctx context.Context, component string, props map[string]any) (*SSRRenderResult, error)

	// RenderAngular renders an Angular component
	RenderAngular(ctx context.Context, component string, props map[string]any) (*SSRRenderResult, error)

	// GetStatus returns the SSR engine status
	GetStatus(ctx context.Context) (*SSRStatus, error)

	// ClearCache clears the render cache
	ClearCache(ctx context.Context) error
}
