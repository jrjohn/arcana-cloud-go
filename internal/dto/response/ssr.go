package response

// RenderResponse represents a server-side render response
type RenderResponse struct {
	HTML       string         `json:"html"`
	CSS        string         `json:"css,omitempty"`
	Scripts    []string       `json:"scripts,omitempty"`
	State      map[string]any `json:"state,omitempty"`
	RenderTime int64          `json:"render_time_ms"`
	Cached     bool           `json:"cached"`
}

// SSRStatusResponse represents the SSR engine status
type SSRStatusResponse struct {
	Status       string            `json:"status"`
	ReactReady   bool              `json:"react_ready"`
	AngularReady bool              `json:"angular_ready"`
	CacheEnabled bool              `json:"cache_enabled"`
	CacheSize    int               `json:"cache_size"`
	Stats        map[string]int64  `json:"stats,omitempty"`
}
