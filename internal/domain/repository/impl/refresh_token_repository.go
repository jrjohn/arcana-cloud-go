package impl

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
)

// refreshTokenRepository implements repository.RefreshTokenRepository by delegating to RefreshTokenDAO.
type refreshTokenRepository struct {
	dao dao.RefreshTokenDAO
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository instance.
func NewRefreshTokenRepository(refreshTokenDAO dao.RefreshTokenDAO) repository.RefreshTokenRepository {
	return &refreshTokenRepository{dao: refreshTokenDAO}
}

// Create inserts a new refresh token.
func (r *refreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	return r.dao.Create(ctx, token)
}

// GetByToken retrieves a refresh token by its value.
func (r *refreshTokenRepository) GetByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	return r.dao.FindByToken(ctx, token)
}

// RevokeByToken revokes a specific refresh token.
func (r *refreshTokenRepository) RevokeByToken(ctx context.Context, token string) error {
	return r.dao.RevokeByToken(ctx, token)
}

// RevokeAllByUserID revokes all refresh tokens for a specific user.
func (r *refreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uint) error {
	return r.dao.RevokeAllByUserID(ctx, userID)
}

// DeleteExpired removes all expired tokens from the database.
func (r *refreshTokenRepository) DeleteExpired(ctx context.Context) error {
	return r.dao.DeleteExpired(ctx)
}
