package service

import (
	"context"
	"errors"

	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUserInactive       = errors.New("user account is inactive")
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	// Register creates a new user account
	Register(ctx context.Context, req *request.RegisterRequest) (*response.AuthResponse, error)

	// Login authenticates a user and returns tokens
	Login(ctx context.Context, req *request.LoginRequest) (*response.AuthResponse, error)

	// RefreshToken generates new tokens using a refresh token
	RefreshToken(ctx context.Context, req *request.RefreshTokenRequest) (*response.AuthResponse, error)

	// Logout invalidates the current token
	Logout(ctx context.Context, token string) error

	// LogoutAll invalidates all tokens for a user
	LogoutAll(ctx context.Context, userID uint) error
}
