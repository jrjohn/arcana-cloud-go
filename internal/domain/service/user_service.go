package service

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
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

// userService implements UserService
type userService struct {
	userRepo       repository.UserRepository
	passwordHasher *security.PasswordHasher
}

// NewUserService creates a new UserService instance
func NewUserService(userRepo repository.UserRepository, passwordHasher *security.PasswordHasher) UserService {
	return &userService{
		userRepo:       userRepo,
		passwordHasher: passwordHasher,
	}
}

func (s *userService) GetByID(ctx context.Context, id uint) (*response.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return s.toUserResponse(user), nil
}

func (s *userService) GetByUsername(ctx context.Context, username string) (*response.UserResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return s.toUserResponse(user), nil
}

func (s *userService) GetByEmail(ctx context.Context, email string) (*response.UserResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return s.toUserResponse(user), nil
}

func (s *userService) List(ctx context.Context, page, size int) (*response.PagedResponse[response.UserResponse], error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	users, total, err := s.userRepo.List(ctx, page, size)
	if err != nil {
		return nil, err
	}

	items := make([]response.UserResponse, len(users))
	for i, user := range users {
		items[i] = *s.toUserResponse(user)
	}

	result := response.NewPagedResponse(items, page, size, total)
	return &result, nil
}

func (s *userService) Update(ctx context.Context, id uint, req *request.UpdateProfileRequest) (*response.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Update fields
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Email != "" && req.Email != user.Email {
		// Check if email is already in use
		exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrUserAlreadyExists
		}
		user.Email = req.Email
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return s.toUserResponse(user), nil
}

func (s *userService) ChangePassword(ctx context.Context, id uint, req *request.ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Verify old password
	if !s.passwordHasher.Verify(req.OldPassword, user.Password) {
		return ErrInvalidCredentials
	}

	// Hash new password
	hashedPassword, err := s.passwordHasher.Hash(req.NewPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	return s.userRepo.Update(ctx, user)
}

func (s *userService) Delete(ctx context.Context, id uint) error {
	return s.userRepo.Delete(ctx, id)
}

func (s *userService) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return s.userRepo.ExistsByUsername(ctx, username)
}

func (s *userService) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return s.userRepo.ExistsByEmail(ctx, email)
}

func (s *userService) toUserResponse(user *entity.User) *response.UserResponse {
	return &response.UserResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Role:       string(user.Role),
		IsActive:   user.IsActive,
		IsVerified: user.IsVerified,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
	}
}
