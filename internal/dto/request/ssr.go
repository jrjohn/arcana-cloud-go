package request

// RenderRequest represents a server-side render request
type RenderRequest struct {
	Component string         `json:"component" binding:"required"`
	Props     map[string]any `json:"props,omitempty"`
	Context   RenderContext  `json:"context,omitempty"`
}

// RenderContext provides additional context for rendering
type RenderContext struct {
	URL       string            `json:"url,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	Cookies   map[string]string `json:"cookies,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}
