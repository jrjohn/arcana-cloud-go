package gorm

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// userDAO implements dao.UserDAO using GORM for SQL databases.
type userDAO struct {
	*baseGormDAO[entity.User]
}

// NewUserDAO creates a new GORM-based UserDAO.
func NewUserDAO(db *gorm.DB) dao.UserDAO {
	return &userDAO{
		baseGormDAO: newBaseGormDAO[entity.User](db),
	}
}

// FindByUsername retrieves a user by their unique username.
func (d *userDAO) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	var user entity.User
	err := d.getDB().WithContext(ctx).Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail retrieves a user by their unique email address.
func (d *userDAO) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := d.getDB().WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsernameOrEmail retrieves a user by username or email.
func (d *userDAO) FindByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*entity.User, error) {
	var user entity.User
	err := d.getDB().WithContext(ctx).
		Where("username = ? OR email = ?", usernameOrEmail, usernameOrEmail).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ExistsByUsername checks if a user with the given username exists.
func (d *userDAO) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return d.ExistsBy(ctx, "username", username)
}

// ExistsByEmail checks if a user with the given email exists.
func (d *userDAO) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return d.ExistsBy(ctx, "email", email)
}

// FindAll retrieves users with pagination, ordered by ID descending.
func (d *userDAO) FindAll(ctx context.Context, page, size int) ([]*entity.User, int64, error) {
	var users []*entity.User
	var total int64
	offset := (page - 1) * size

	if err := d.getDB().WithContext(ctx).Model(&entity.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := d.getDB().WithContext(ctx).
		Offset(offset).
		Limit(size).
		Order("id DESC").
		Find(&users).Error

	return users, total, err
}
