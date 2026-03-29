// Package impl provides repository implementations that delegate to the DAO layer.
// This separation allows repositories to focus on business logic while DAOs handle
// database-specific operations.
package impl

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
)

// userRepository implements repository.UserRepository by delegating to UserDAO.
type userRepository struct {
	dao dao.UserDAO
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(userDAO dao.UserDAO) repository.UserRepository {
	return &userRepository{dao: userDAO}
}

// Create inserts a new user.
func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	return r.dao.Create(ctx, user)
}

// GetByID retrieves a user by their ID.
func (r *userRepository) GetByID(ctx context.Context, id uint) (*entity.User, error) {
	return r.dao.FindByID(ctx, id)
}

// GetByUsername retrieves a user by their username.
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*entity.User, error) {
	return r.dao.FindByUsername(ctx, username)
}

// GetByEmail retrieves a user by their email.
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return r.dao.FindByEmail(ctx, email)
}

// GetByUsernameOrEmail retrieves a user by username or email.
func (r *userRepository) GetByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*entity.User, error) {
	return r.dao.FindByUsernameOrEmail(ctx, usernameOrEmail)
}

// Update modifies an existing user.
func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	return r.dao.Update(ctx, user)
}

// Delete removes a user by ID.
func (r *userRepository) Delete(ctx context.Context, id uint) error {
	return r.dao.Delete(ctx, id)
}

// List retrieves users with pagination.
func (r *userRepository) List(ctx context.Context, page, size int) ([]*entity.User, int64, error) {
	return r.dao.FindAll(ctx, page, size)
}

// ExistsByUsername checks if a user with the given username exists.
func (r *userRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return r.dao.ExistsByUsername(ctx, username)
}

// ExistsByEmail checks if a user with the given email exists.
func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.dao.ExistsByEmail(ctx, email)
}
