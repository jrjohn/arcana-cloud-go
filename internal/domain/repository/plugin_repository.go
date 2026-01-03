package repository

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// PluginRepository defines the interface for plugin data operations
type PluginRepository interface {
	// Create creates a new plugin
	Create(ctx context.Context, plugin *entity.Plugin) error

	// GetByID retrieves a plugin by ID
	GetByID(ctx context.Context, id uint) (*entity.Plugin, error)

	// GetByKey retrieves a plugin by its unique key
	GetByKey(ctx context.Context, key string) (*entity.Plugin, error)

	// Update updates an existing plugin
	Update(ctx context.Context, plugin *entity.Plugin) error

	// Delete soft-deletes a plugin by ID
	Delete(ctx context.Context, id uint) error

	// DeleteByKey soft-deletes a plugin by key
	DeleteByKey(ctx context.Context, key string) error

	// List retrieves all plugins with pagination
	List(ctx context.Context, page, size int) ([]*entity.Plugin, int64, error)

	// ListByState retrieves plugins by state
	ListByState(ctx context.Context, state entity.PluginState) ([]*entity.Plugin, error)

	// ListEnabled retrieves all enabled plugins
	ListEnabled(ctx context.Context) ([]*entity.Plugin, error)

	// ExistsByKey checks if a plugin with the given key exists
	ExistsByKey(ctx context.Context, key string) (bool, error)

	// UpdateState updates a plugin's state
	UpdateState(ctx context.Context, id uint, state entity.PluginState) error
}

// PluginExtensionRepository defines the interface for plugin extension operations
type PluginExtensionRepository interface {
	// Create creates a new plugin extension
	Create(ctx context.Context, extension *entity.PluginExtension) error

	// GetByPluginID retrieves all extensions for a plugin
	GetByPluginID(ctx context.Context, pluginID uint) ([]*entity.PluginExtension, error)

	// DeleteByPluginID deletes all extensions for a plugin
	DeleteByPluginID(ctx context.Context, pluginID uint) error
}
