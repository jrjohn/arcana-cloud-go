package dao

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// PluginDAO extends BaseDAO with plugin-specific data access operations.
type PluginDAO interface {
	BaseDAO[entity.Plugin, uint]

	// FindByKey retrieves a plugin by its unique key identifier.
	// Returns nil, nil if the plugin is not found.
	FindByKey(ctx context.Context, key string) (*entity.Plugin, error)

	// DeleteByKey soft-deletes a plugin by its key.
	DeleteByKey(ctx context.Context, key string) error

	// FindByState retrieves all plugins with a specific state.
	FindByState(ctx context.Context, state entity.PluginState) ([]*entity.Plugin, error)

	// FindEnabled retrieves all plugins that are currently enabled.
	// This is a convenience method equivalent to FindByState(ctx, PluginStateEnabled).
	FindEnabled(ctx context.Context) ([]*entity.Plugin, error)

	// ExistsByKey checks if a plugin with the given key exists.
	ExistsByKey(ctx context.Context, key string) (bool, error)

	// UpdateState updates the state of a plugin by its ID.
	// This also updates the enabled_at timestamp when enabling a plugin.
	UpdateState(ctx context.Context, id uint, state entity.PluginState) error
}
