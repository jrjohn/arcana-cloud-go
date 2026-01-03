package repository

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *entity.User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id uint) (*entity.User, error)

	// GetByUsername retrieves a user by username
	GetByUsername(ctx context.Context, username string) (*entity.User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*entity.User, error)

	// GetByUsernameOrEmail retrieves a user by username or email
	GetByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*entity.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *entity.User) error

	// Delete soft-deletes a user by ID
	Delete(ctx context.Context, id uint) error

	// List retrieves users with pagination
	List(ctx context.Context, page, size int) ([]*entity.User, int64, error)

	// ExistsByUsername checks if a username exists
	ExistsByUsername(ctx context.Context, username string) (bool, error)

	// ExistsByEmail checks if an email exists
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// RefreshTokenRepository defines the interface for refresh token operations
type RefreshTokenRepository interface {
	// Create creates a new refresh token
	Create(ctx context.Context, token *entity.RefreshToken) error

	// GetByToken retrieves a refresh token by its value
	GetByToken(ctx context.Context, token string) (*entity.RefreshToken, error)

	// RevokeByToken revokes a specific refresh token
	RevokeByToken(ctx context.Context, token string) error

	// RevokeAllByUserID revokes all refresh tokens for a user
	RevokeAllByUserID(ctx context.Context, userID uint) error

	// DeleteExpired removes all expired tokens
	DeleteExpired(ctx context.Context) error
}
