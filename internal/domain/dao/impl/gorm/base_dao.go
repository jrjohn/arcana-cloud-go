// Package gorm provides GORM-based DAO implementations for SQL databases (MySQL, PostgreSQL).
package gorm

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// baseGormDAO provides common GORM operations for all entity DAOs.
// It implements the generic BaseDAO interface for SQL databases.
type baseGormDAO[T any] struct {
	db *gorm.DB
}

// newBaseGormDAO creates a new base GORM DAO instance.
func newBaseGormDAO[T any](db *gorm.DB) *baseGormDAO[T] {
	return &baseGormDAO[T]{db: db}
}

// Create inserts a new entity into the database.
func (d *baseGormDAO[T]) Create(ctx context.Context, entity *T) error {
	return d.db.WithContext(ctx).Create(entity).Error
}

// FindByID retrieves an entity by its primary key.
// Returns nil, nil if the entity is not found.
func (d *baseGormDAO[T]) FindByID(ctx context.Context, id uint) (*T, error) {
	var entity T
	err := d.db.WithContext(ctx).First(&entity, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// Update modifies an existing entity in the database.
func (d *baseGormDAO[T]) Update(ctx context.Context, entity *T) error {
	return d.db.WithContext(ctx).Save(entity).Error
}

// Delete performs a soft delete on an entity by its ID.
func (d *baseGormDAO[T]) Delete(ctx context.Context, id uint) error {
	var entity T
	return d.db.WithContext(ctx).Delete(&entity, id).Error
}

// FindAll retrieves entities with pagination.
// Returns the entities, total count, and any error.
func (d *baseGormDAO[T]) FindAll(ctx context.Context, page, size int) ([]*T, int64, error) {
	var entities []*T
	var total int64
	offset := (page - 1) * size

	var model T
	if err := d.db.WithContext(ctx).Model(&model).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := d.db.WithContext(ctx).
		Offset(offset).
		Limit(size).
		Find(&entities).Error

	return entities, total, err
}

// Count returns the total number of entities.
func (d *baseGormDAO[T]) Count(ctx context.Context) (int64, error) {
	var count int64
	var model T
	err := d.db.WithContext(ctx).Model(&model).Count(&count).Error
	return count, err
}

// ExistsBy checks if an entity exists by a field value.
func (d *baseGormDAO[T]) ExistsBy(ctx context.Context, field string, value any) (bool, error) {
	var count int64
	var model T
	err := d.db.WithContext(ctx).
		Model(&model).
		Where(field+" = ?", value).
		Count(&count).Error
	return count > 0, err
}

// getDB returns the underlying GORM database instance.
// This is used by entity-specific DAOs to access the database for custom queries.
func (d *baseGormDAO[T]) getDB() *gorm.DB {
	return d.db
}

// findByField retrieves an entity by a specific field value.
// This is a helper method for entity-specific DAOs.
func (d *baseGormDAO[T]) findByField(ctx context.Context, field string, value any) (*T, error) {
	var entity T
	err := d.db.WithContext(ctx).Where(field+" = ?", value).First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// findAllByField retrieves all entities matching a specific field value.
// This is a helper method for entity-specific DAOs.
func (d *baseGormDAO[T]) findAllByField(ctx context.Context, field string, value any) ([]*T, error) {
	var entities []*T
	err := d.db.WithContext(ctx).Where(field+" = ?", value).Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return entities, nil
}

// deleteByField deletes entities matching a specific field value.
// This is a helper method for entity-specific DAOs.
func (d *baseGormDAO[T]) deleteByField(ctx context.Context, field string, value any) error {
	var model T
	return d.db.WithContext(ctx).Where(field+" = ?", value).Delete(&model).Error
}
