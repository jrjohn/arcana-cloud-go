package dao

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// RefreshTokenDAO extends BaseDAO with refresh token-specific data access operations.
type RefreshTokenDAO interface {
	BaseDAO[entity.RefreshToken, uint]

	// FindByToken retrieves a refresh token by its value.
	// Only returns non-revoked tokens.
	// Returns nil, nil if the token is not found or is revoked.
	FindByToken(ctx context.Context, token string) (*entity.RefreshToken, error)

	// RevokeByToken revokes a specific refresh token by setting its revoked flag.
	RevokeByToken(ctx context.Context, token string) error

	// RevokeAllByUserID revokes all refresh tokens for a specific user.
	// This is useful for logout-from-all-devices functionality.
	RevokeAllByUserID(ctx context.Context, userID uint) error

	// DeleteExpired removes all expired tokens from the database.
	// This is typically called by a cleanup job.
	DeleteExpired(ctx context.Context) error
}
