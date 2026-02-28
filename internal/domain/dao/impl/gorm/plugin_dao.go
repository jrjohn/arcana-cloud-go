package gorm

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// pluginDAO implements dao.PluginDAO using GORM for SQL databases.
type pluginDAO struct {
	*baseGormDAO[entity.Plugin]
}

// NewPluginDAO creates a new GORM-based PluginDAO.
func NewPluginDAO(db *gorm.DB) dao.PluginDAO {
	return &pluginDAO{
		baseGormDAO: newBaseGormDAO[entity.Plugin](db),
	}
}

// FindByKey retrieves a plugin by its unique key identifier.
func (d *pluginDAO) FindByKey(ctx context.Context, key string) (*entity.Plugin, error) {
	var plugin entity.Plugin
	err := d.getDB().WithContext(ctx).Where(map[string]any{"key": key}).First(&plugin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &plugin, nil
}

// DeleteByKey soft-deletes a plugin by its key.
func (d *pluginDAO) DeleteByKey(ctx context.Context, key string) error {
	return d.getDB().WithContext(ctx).
		Where(map[string]any{"key": key}).
		Delete(&entity.Plugin{}).Error
}

// FindByState retrieves all plugins with a specific state.
func (d *pluginDAO) FindByState(ctx context.Context, state entity.PluginState) ([]*entity.Plugin, error) {
	var plugins []*entity.Plugin
	err := d.getDB().WithContext(ctx).
		Where("state = ?", state).
		Order("name ASC").
		Find(&plugins).Error
	if err != nil {
		return nil, err
	}
	return plugins, nil
}

// FindEnabled retrieves all plugins that are currently enabled.
func (d *pluginDAO) FindEnabled(ctx context.Context) ([]*entity.Plugin, error) {
	return d.FindByState(ctx, entity.PluginStateEnabled)
}

// ExistsByKey checks if a plugin with the given key exists.
func (d *pluginDAO) ExistsByKey(ctx context.Context, key string) (bool, error) {
	var count int64
	err := d.getDB().WithContext(ctx).Model(&entity.Plugin{}).Where(map[string]any{"key": key}).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateState updates the state of a plugin by its ID.
// When enabling a plugin, it also sets the enabled_at timestamp.
func (d *pluginDAO) UpdateState(ctx context.Context, id uint, state entity.PluginState) error {
	updates := map[string]any{
		"state":      state,
		"updated_at": time.Now(),
	}

	// Set enabled_at when enabling the plugin
	if state == entity.PluginStateEnabled {
		now := time.Now()
		updates["enabled_at"] = &now
	}

	return d.getDB().WithContext(ctx).
		Model(&entity.Plugin{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// FindAll retrieves plugins with pagination, ordered by installed_at descending.
func (d *pluginDAO) FindAll(ctx context.Context, page, size int) ([]*entity.Plugin, int64, error) {
	var plugins []*entity.Plugin
	var total int64
	offset := (page - 1) * size

	if err := d.getDB().WithContext(ctx).Model(&entity.Plugin{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := d.getDB().WithContext(ctx).
		Offset(offset).
		Limit(size).
		Order("installed_at DESC").
		Find(&plugins).Error

	return plugins, total, err
}
