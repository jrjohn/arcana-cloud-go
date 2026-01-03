package request

// InstallPluginRequest represents a plugin installation request
type InstallPluginRequest struct {
	Name        string            `json:"name" binding:"required,max=200"`
	Description string            `json:"description,omitempty" binding:"max=1000"`
	Version     string            `json:"version" binding:"required,max=50"`
	Author      string            `json:"author,omitempty" binding:"max=200"`
	Type        string            `json:"type" binding:"required"`
	Config      map[string]any    `json:"config,omitempty"`
}

// UpdatePluginRequest represents a plugin update request
type UpdatePluginRequest struct {
	Name        string         `json:"name,omitempty" binding:"max=200"`
	Description string         `json:"description,omitempty" binding:"max=1000"`
	Config      map[string]any `json:"config,omitempty"`
}

// PluginActionRequest represents a plugin action request (enable/disable)
type PluginActionRequest struct {
	Action string `json:"action" binding:"required,oneof=enable disable"`
}
