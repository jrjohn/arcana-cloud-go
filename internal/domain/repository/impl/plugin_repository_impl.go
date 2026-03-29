package impl

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
)

// pluginRepository implements repository.PluginRepository by delegating to PluginDAO.
type pluginRepository struct {
	dao dao.PluginDAO
}

// NewPluginRepository creates a new PluginRepository instance.
func NewPluginRepository(pluginDAO dao.PluginDAO) repository.PluginRepository {
	return &pluginRepository{dao: pluginDAO}
}

// Create inserts a new plugin.
func (r *pluginRepository) Create(ctx context.Context, plugin *entity.Plugin) error {
	return r.dao.Create(ctx, plugin)
}

// GetByID retrieves a plugin by its ID.
func (r *pluginRepository) GetByID(ctx context.Context, id uint) (*entity.Plugin, error) {
	return r.dao.FindByID(ctx, id)
}

// GetByKey retrieves a plugin by its unique key.
func (r *pluginRepository) GetByKey(ctx context.Context, key string) (*entity.Plugin, error) {
	return r.dao.FindByKey(ctx, key)
}

// Update modifies an existing plugin.
func (r *pluginRepository) Update(ctx context.Context, plugin *entity.Plugin) error {
	return r.dao.Update(ctx, plugin)
}

// Delete removes a plugin by ID.
func (r *pluginRepository) Delete(ctx context.Context, id uint) error {
	return r.dao.Delete(ctx, id)
}

// DeleteByKey soft-deletes a plugin by its key.
func (r *pluginRepository) DeleteByKey(ctx context.Context, key string) error {
	return r.dao.DeleteByKey(ctx, key)
}

// List retrieves plugins with pagination.
func (r *pluginRepository) List(ctx context.Context, page, size int) ([]*entity.Plugin, int64, error) {
	return r.dao.FindAll(ctx, page, size)
}

// ListByState retrieves all plugins with a specific state.
func (r *pluginRepository) ListByState(ctx context.Context, state entity.PluginState) ([]*entity.Plugin, error) {
	return r.dao.FindByState(ctx, state)
}

// ListEnabled retrieves all enabled plugins.
func (r *pluginRepository) ListEnabled(ctx context.Context) ([]*entity.Plugin, error) {
	return r.dao.FindEnabled(ctx)
}

// ExistsByKey checks if a plugin with the given key exists.
func (r *pluginRepository) ExistsByKey(ctx context.Context, key string) (bool, error) {
	return r.dao.ExistsByKey(ctx, key)
}

// UpdateState updates the state of a plugin.
func (r *pluginRepository) UpdateState(ctx context.Context, id uint, state entity.PluginState) error {
	return r.dao.UpdateState(ctx, id, state)
}
