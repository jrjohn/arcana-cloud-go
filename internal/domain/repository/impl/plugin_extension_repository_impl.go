package impl

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
)

// pluginExtensionRepository implements repository.PluginExtensionRepository by delegating to PluginExtensionDAO.
type pluginExtensionRepository struct {
	dao dao.PluginExtensionDAO
}

// NewPluginExtensionRepository creates a new PluginExtensionRepository instance.
func NewPluginExtensionRepository(extensionDAO dao.PluginExtensionDAO) repository.PluginExtensionRepository {
	return &pluginExtensionRepository{dao: extensionDAO}
}

// Create inserts a new plugin extension.
func (r *pluginExtensionRepository) Create(ctx context.Context, extension *entity.PluginExtension) error {
	return r.dao.Create(ctx, extension)
}

// GetByPluginID retrieves all extensions belonging to a specific plugin.
func (r *pluginExtensionRepository) GetByPluginID(ctx context.Context, pluginID uint) ([]*entity.PluginExtension, error) {
	return r.dao.FindByPluginID(ctx, pluginID)
}

// DeleteByPluginID deletes all extensions belonging to a specific plugin.
func (r *pluginExtensionRepository) DeleteByPluginID(ctx context.Context, pluginID uint) error {
	return r.dao.DeleteByPluginID(ctx, pluginID)
}
