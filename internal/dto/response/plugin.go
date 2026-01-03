package response

import (
	"time"
)

// PluginResponse represents plugin data in responses
type PluginResponse struct {
	ID          uint       `json:"id"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Version     string     `json:"version"`
	Author      string     `json:"author,omitempty"`
	Type        string     `json:"type"`
	State       string     `json:"state"`
	InstalledAt time.Time  `json:"installed_at"`
	EnabledAt   *time.Time `json:"enabled_at,omitempty"`
}

// PluginHealthResponse represents plugin system health status
type PluginHealthResponse struct {
	Status          string `json:"status"`
	TotalPlugins    int    `json:"total_plugins"`
	EnabledPlugins  int    `json:"enabled_plugins"`
	DisabledPlugins int    `json:"disabled_plugins"`
	ErrorPlugins    int    `json:"error_plugins"`
}

// PluginExtensionResponse represents a plugin extension in responses
type PluginExtensionResponse struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Path    string `json:"path,omitempty"`
	Handler string `json:"handler,omitempty"`
}

// PluginDetailResponse represents detailed plugin information
type PluginDetailResponse struct {
	PluginResponse
	Extensions []PluginExtensionResponse `json:"extensions,omitempty"`
	Config     map[string]any            `json:"config,omitempty"`
}
