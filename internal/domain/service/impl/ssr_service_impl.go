package impl

import (
	"context"
	"sync"
	"time"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
)

// SSRCacheEntry represents a cached render result
type SSRCacheEntry struct {
	Result    *service.SSRRenderResult
	ExpiresAt time.Time
}

// ssrService implements service.SSRService
type ssrService struct {
	reactEngine   service.SSREngine
	angularEngine service.SSREngine
	cache         map[string]*SSRCacheEntry
	cacheMutex    sync.RWMutex
	cacheEnabled  bool
	cacheTTL      time.Duration
	stats         map[string]int64
	statsMutex    sync.RWMutex
}

// NewSSRService creates a new SSRService instance
func NewSSRService(cacheEnabled bool, cacheTTL time.Duration) service.SSRService {
	s := &ssrService{
		cache:        make(map[string]*SSRCacheEntry),
		cacheEnabled: cacheEnabled,
		cacheTTL:     cacheTTL,
		stats: map[string]int64{
			"react_renders":   0,
			"angular_renders": 0,
			"cache_hits":      0,
			"cache_misses":    0,
		},
	}

	// Start cache cleanup goroutine
	go s.cleanupCache()

	return s
}

// SetReactEngine sets the React rendering engine
func (s *ssrService) SetReactEngine(engine service.SSREngine) {
	s.reactEngine = engine
}

// SetAngularEngine sets the Angular rendering engine
func (s *ssrService) SetAngularEngine(engine service.SSREngine) {
	s.angularEngine = engine
}

func (s *ssrService) RenderReact(ctx context.Context, component string, props map[string]any) (*service.SSRRenderResult, error) {
	s.incrementStat("react_renders")

	// Check cache
	if s.cacheEnabled {
		if result := s.getFromCache("react:" + component); result != nil {
			s.incrementStat("cache_hits")
			result.Cached = true
			return result, nil
		}
		s.incrementStat("cache_misses")
	}

	start := time.Now()

	// For now, return a placeholder since we don't have an actual JS engine
	// In production, this would use v8go or goja to execute React SSR
	result := &service.SSRRenderResult{
		HTML:       s.renderPlaceholder("react", component, props),
		CSS:        "",
		Scripts:    []string{"/static/react/bundle.js"},
		State:      props,
		RenderTime: time.Since(start),
		Cached:     false,
	}

	// Cache the result
	if s.cacheEnabled {
		s.putInCache("react:"+component, result)
	}

	return result, nil
}

func (s *ssrService) RenderAngular(ctx context.Context, component string, props map[string]any) (*service.SSRRenderResult, error) {
	s.incrementStat("angular_renders")

	// Check cache
	if s.cacheEnabled {
		if result := s.getFromCache("angular:" + component); result != nil {
			s.incrementStat("cache_hits")
			result.Cached = true
			return result, nil
		}
		s.incrementStat("cache_misses")
	}

	start := time.Now()

	// Placeholder implementation
	result := &service.SSRRenderResult{
		HTML:       s.renderPlaceholder("angular", component, props),
		CSS:        "",
		Scripts:    []string{"/static/angular/main.js"},
		State:      props,
		RenderTime: time.Since(start),
		Cached:     false,
	}

	// Cache the result
	if s.cacheEnabled {
		s.putInCache("angular:"+component, result)
	}

	return result, nil
}

func (s *ssrService) GetStatus(ctx context.Context) (*service.SSRStatus, error) {
	s.statsMutex.RLock()
	statsCopy := make(map[string]int64)
	for k, v := range s.stats {
		statsCopy[k] = v
	}
	s.statsMutex.RUnlock()

	s.cacheMutex.RLock()
	cacheSize := len(s.cache)
	s.cacheMutex.RUnlock()

	return &service.SSRStatus{
		Status:       "ready",
		ReactReady:   s.reactEngine != nil && s.reactEngine.IsReady(),
		AngularReady: s.angularEngine != nil && s.angularEngine.IsReady(),
		CacheEnabled: s.cacheEnabled,
		CacheSize:    cacheSize,
		Stats:        statsCopy,
	}, nil
}

func (s *ssrService) ClearCache(ctx context.Context) error {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.cache = make(map[string]*SSRCacheEntry)
	return nil
}

func (s *ssrService) getFromCache(key string) *service.SSRRenderResult {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	entry, ok := s.cache[key]
	if !ok {
		return nil
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil
	}

	// Return a copy
	return &service.SSRRenderResult{
		HTML:       entry.Result.HTML,
		CSS:        entry.Result.CSS,
		Scripts:    entry.Result.Scripts,
		State:      entry.Result.State,
		RenderTime: entry.Result.RenderTime,
	}
}

func (s *ssrService) putInCache(key string, result *service.SSRRenderResult) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	s.cache[key] = &SSRCacheEntry{
		Result:    result,
		ExpiresAt: time.Now().Add(s.cacheTTL),
	}
}

func (s *ssrService) cleanupCache() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cacheMutex.Lock()
		now := time.Now()
		for key, entry := range s.cache {
			if now.After(entry.ExpiresAt) {
				delete(s.cache, key)
			}
		}
		s.cacheMutex.Unlock()
	}
}

func (s *ssrService) incrementStat(name string) {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()
	s.stats[name]++
}

func (s *ssrService) renderPlaceholder(framework, component string, props map[string]any) string {
	// This is a placeholder. In production, you would use:
	// - v8go (V8 JavaScript engine bindings)
	// - goja (Pure Go JavaScript engine)
	// - or external Node.js process
	return `<div id="ssr-root" data-framework="` + framework + `" data-component="` + component + `">
  <div class="ssr-placeholder">Loading ` + component + `...</div>
  <script>window.__SSR_STATE__ = ` + toJSON(props) + `;</script>
</div>`
}

func toJSON(v any) string {
	if v == nil {
		return "{}"
	}
	// Simple JSON serialization for placeholder
	return "{}"
}
