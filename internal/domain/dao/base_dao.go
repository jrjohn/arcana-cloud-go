// Package dao defines data access object interfaces for database abstraction.
// The DAO layer provides a clean separation between repository business logic
// and database-specific implementations (MySQL, PostgreSQL, MongoDB).
package dao

import (
	"context"
)

// BaseDAO defines common CRUD operations for all DAOs.
// T is the entity type, ID is the identifier type (uint for SQL, string for MongoDB).
type BaseDAO[T any, ID comparable] interface {
	// Create inserts a new entity into the database.
	Create(ctx context.Context, entity *T) error

	// FindByID retrieves an entity by its primary key.
	// Returns nil, nil if the entity is not found.
	FindByID(ctx context.Context, id ID) (*T, error)

	// Update modifies an existing entity in the database.
	Update(ctx context.Context, entity *T) error

	// Delete removes an entity by its ID.
	// For SQL databases, this performs a soft delete.
	// For MongoDB, this can be configured for soft or hard delete.
	Delete(ctx context.Context, id ID) error

	// FindAll retrieves entities with pagination.
	// Returns the entities, total count, and any error.
	FindAll(ctx context.Context, page, size int) ([]*T, int64, error)

	// Count returns the total number of entities.
	Count(ctx context.Context) (int64, error)

	// ExistsBy checks if an entity exists by a field value.
	ExistsBy(ctx context.Context, field string, value any) (bool, error)
}

// QueryOption represents optional query parameters for advanced queries.
type QueryOption struct {
	OrderBy    string
	Descending bool
	Preloads   []string
	Conditions map[string]any
}

// PageResult wraps paginated results with metadata.
type PageResult[T any] struct {
	Items      []*T
	TotalCount int64
	Page       int
	Size       int
}

// NewPageResult creates a new PageResult with the given parameters.
func NewPageResult[T any](items []*T, totalCount int64, page, size int) *PageResult[T] {
	return &PageResult[T]{
		Items:      items,
		TotalCount: totalCount,
		Page:       page,
		Size:       size,
	}
}

// TotalPages calculates the total number of pages.
func (p *PageResult[T]) TotalPages() int {
	if p.Size <= 0 {
		return 0
	}
	pages := int(p.TotalCount) / p.Size
	if int(p.TotalCount)%p.Size > 0 {
		pages++
	}
	return pages
}

// HasNext returns true if there are more pages after the current one.
func (p *PageResult[T]) HasNext() bool {
	return p.Page < p.TotalPages()
}

// HasPrev returns true if there are pages before the current one.
func (p *PageResult[T]) HasPrev() bool {
	return p.Page > 1
}
