package gorm

import (
	"context"

	"gorm.io/gorm"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// pluginExtensionDAO implements dao.PluginExtensionDAO using GORM for SQL databases.
type pluginExtensionDAO struct {
	*baseGormDAO[entity.PluginExtension]
}

// NewPluginExtensionDAO creates a new GORM-based PluginExtensionDAO.
func NewPluginExtensionDAO(db *gorm.DB) dao.PluginExtensionDAO {
	return &pluginExtensionDAO{
		baseGormDAO: newBaseGormDAO[entity.PluginExtension](db),
	}
}

// FindByPluginID retrieves all extensions belonging to a specific plugin.
func (d *pluginExtensionDAO) FindByPluginID(ctx context.Context, pluginID uint) ([]*entity.PluginExtension, error) {
	var extensions []*entity.PluginExtension
	err := d.getDB().WithContext(ctx).
		Where("plugin_id = ?", pluginID).
		Find(&extensions).Error
	if err != nil {
		return nil, err
	}
	return extensions, nil
}

// DeleteByPluginID deletes all extensions belonging to a specific plugin.
func (d *pluginExtensionDAO) DeleteByPluginID(ctx context.Context, pluginID uint) error {
	return d.getDB().WithContext(ctx).
		Where("plugin_id = ?", pluginID).
		Delete(&entity.PluginExtension{}).Error
}

// FindAll retrieves plugin extensions with pagination, ordered by created_at descending.
func (d *pluginExtensionDAO) FindAll(ctx context.Context, page, size int) ([]*entity.PluginExtension, int64, error) {
	var extensions []*entity.PluginExtension
	var total int64
	offset := (page - 1) * size

	if err := d.getDB().WithContext(ctx).Model(&entity.PluginExtension{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := d.getDB().WithContext(ctx).
		Preload("Plugin").
		Offset(offset).
		Limit(size).
		Order("created_at DESC").
		Find(&extensions).Error

	return extensions, total, err
}
