package dao

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// PluginExtensionDAO extends BaseDAO with plugin extension-specific data access operations.
type PluginExtensionDAO interface {
	BaseDAO[entity.PluginExtension, uint]

	// FindByPluginID retrieves all extensions belonging to a specific plugin.
	FindByPluginID(ctx context.Context, pluginID uint) ([]*entity.PluginExtension, error)

	// DeleteByPluginID deletes all extensions belonging to a specific plugin.
	// This is typically called when uninstalling a plugin.
	DeleteByPluginID(ctx context.Context, pluginID uint) error
}
