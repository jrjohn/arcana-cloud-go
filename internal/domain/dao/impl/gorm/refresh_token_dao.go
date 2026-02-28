package gorm

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// refreshTokenDAO implements dao.RefreshTokenDAO using GORM for SQL databases.
type refreshTokenDAO struct {
	*baseGormDAO[entity.RefreshToken]
}

// NewRefreshTokenDAO creates a new GORM-based RefreshTokenDAO.
func NewRefreshTokenDAO(db *gorm.DB) dao.RefreshTokenDAO {
	return &refreshTokenDAO{
		baseGormDAO: newBaseGormDAO[entity.RefreshToken](db),
	}
}

// FindByToken retrieves a refresh token by its value.
// Only returns non-revoked tokens with preloaded User data.
func (d *refreshTokenDAO) FindByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	var refreshToken entity.RefreshToken
	err := d.getDB().WithContext(ctx).
		Preload("User").
		Where("token = ? AND revoked = ?", token, false).
		First(&refreshToken).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &refreshToken, nil
}

// RevokeByToken revokes a specific refresh token.
func (d *refreshTokenDAO) RevokeByToken(ctx context.Context, token string) error {
	return d.getDB().WithContext(ctx).
		Model(&entity.RefreshToken{}).
		Where("token = ?", token).
		Update("revoked", true).Error
}

// RevokeAllByUserID revokes all refresh tokens for a specific user.
func (d *refreshTokenDAO) RevokeAllByUserID(ctx context.Context, userID uint) error {
	return d.getDB().WithContext(ctx).
		Model(&entity.RefreshToken{}).
		Where("user_id = ?", userID).
		Update("revoked", true).Error
}

// DeleteExpired removes all expired tokens from the database.
func (d *refreshTokenDAO) DeleteExpired(ctx context.Context) error {
	return d.getDB().WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&entity.RefreshToken{}).Error
}

// FindAll retrieves refresh tokens with pagination, ordered by created_at descending.
func (d *refreshTokenDAO) FindAll(ctx context.Context, page, size int) ([]*entity.RefreshToken, int64, error) {
	var tokens []*entity.RefreshToken
	var total int64
	offset := (page - 1) * size

	if err := d.getDB().WithContext(ctx).Model(&entity.RefreshToken{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := d.getDB().WithContext(ctx).
		Preload("User").
		Offset(offset).
		Limit(size).
		Order("created_at DESC").
		Find(&tokens).Error

	return tokens, total, err
}
