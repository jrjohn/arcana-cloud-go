package dao

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// UserDAO extends BaseDAO with user-specific data access operations.
type UserDAO interface {
	BaseDAO[entity.User, uint]

	// FindByUsername retrieves a user by their unique username.
	// Returns nil, nil if the user is not found.
	FindByUsername(ctx context.Context, username string) (*entity.User, error)

	// FindByEmail retrieves a user by their unique email address.
	// Returns nil, nil if the user is not found.
	FindByEmail(ctx context.Context, email string) (*entity.User, error)

	// FindByUsernameOrEmail retrieves a user by username or email.
	// This is useful for login where users can use either identifier.
	// Returns nil, nil if the user is not found.
	FindByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*entity.User, error)

	// ExistsByUsername checks if a user with the given username exists.
	ExistsByUsername(ctx context.Context, username string) (bool, error)

	// ExistsByEmail checks if a user with the given email exists.
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}
