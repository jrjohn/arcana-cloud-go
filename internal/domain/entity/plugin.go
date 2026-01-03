package entity

import (
	"time"

	"gorm.io/gorm"
)

// PluginState represents the state of a plugin
type PluginState string

const (
	PluginStateInstalled  PluginState = "INSTALLED"
	PluginStateEnabled    PluginState = "ENABLED"
	PluginStateDisabled   PluginState = "DISABLED"
	PluginStateUninstalled PluginState = "UNINSTALLED"
	PluginStateError      PluginState = "ERROR"
)

// PluginType represents the type of plugin
type PluginType string

const (
	PluginTypeRestEndpoint   PluginType = "REST_ENDPOINT"
	PluginTypeService        PluginType = "SERVICE"
	PluginTypeEventListener  PluginType = "EVENT_LISTENER"
	PluginTypeScheduledJob   PluginType = "SCHEDULED_JOB"
	PluginTypeSSRView        PluginType = "SSR_VIEW"
	PluginTypeMiddleware     PluginType = "MIDDLEWARE"
)

// Plugin represents a plugin entity in the system
type Plugin struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Key         string         `gorm:"uniqueIndex;size:100;not null" json:"key"`
	Name        string         `gorm:"size:200;not null" json:"name"`
	Description string         `gorm:"size:1000" json:"description,omitempty"`
	Version     string         `gorm:"size:50;not null" json:"version"`
	Author      string         `gorm:"size:200" json:"author,omitempty"`
	Type        PluginType     `gorm:"size:50;not null" json:"type"`
	State       PluginState    `gorm:"size:20;not null;default:INSTALLED" json:"state"`
	Config      string         `gorm:"type:text" json:"config,omitempty"`
	Checksum    string         `gorm:"size:128" json:"checksum,omitempty"`
	Path        string         `gorm:"size:500" json:"path,omitempty"`
	InstalledAt time.Time      `gorm:"column:installed_at" json:"installed_at"`
	EnabledAt   *time.Time     `gorm:"column:enabled_at" json:"enabled_at,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name for Plugin
func (Plugin) TableName() string {
	return "plugins"
}

// IsEnabled checks if the plugin is enabled
func (p *Plugin) IsEnabled() bool {
	return p.State == PluginStateEnabled
}

// PluginExtension represents an extension point provided by a plugin
type PluginExtension struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	PluginID  uint           `gorm:"index;not null" json:"plugin_id"`
	Name      string         `gorm:"size:200;not null" json:"name"`
	Type      PluginType     `gorm:"size:50;not null" json:"type"`
	Path      string         `gorm:"size:500" json:"path,omitempty"`
	Handler   string         `gorm:"size:500" json:"handler,omitempty"`
	Config    string         `gorm:"type:text" json:"config,omitempty"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Plugin Plugin `gorm:"foreignKey:PluginID" json:"-"`
}

// TableName specifies the table name for PluginExtension
func (PluginExtension) TableName() string {
	return "plugin_extensions"
}
