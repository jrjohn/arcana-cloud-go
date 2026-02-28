package service

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
)

// UserService defines the interface for user operations
type UserService interface {
	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id uint) (*response.UserResponse, error)

	// GetByUsername retrieves a user by username
	GetByUsername(ctx context.Context, username string) (*response.UserResponse, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*response.UserResponse, error)

	// List retrieves users with pagination
	List(ctx context.Context, page, size int) (*response.PagedResponse[response.UserResponse], error)

	// Update updates a user's profile
	Update(ctx context.Context, id uint, req *request.UpdateProfileRequest) (*response.UserResponse, error)

	// ChangePassword changes a user's password
	ChangePassword(ctx context.Context, id uint, req *request.ChangePasswordRequest) error

	// Delete soft-deletes a user
	Delete(ctx context.Context, id uint) error

	// ExistsByUsername checks if a username exists
	ExistsByUsername(ctx context.Context, username string) (bool, error)

	// ExistsByEmail checks if an email exists
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}
