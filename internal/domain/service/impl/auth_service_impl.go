package impl

import (
	"context"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

// authService implements service.AuthService
type authService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtProvider      *security.JWTProvider
	passwordHasher   *security.PasswordHasher
}

// NewAuthService creates a new AuthService instance
func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtProvider *security.JWTProvider,
	passwordHasher *security.PasswordHasher,
) service.AuthService {
	return &authService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtProvider:      jwtProvider,
		passwordHasher:   passwordHasher,
	}
}

func (s *authService) Register(ctx context.Context, req *request.RegisterRequest) (*response.AuthResponse, error) {
	// Check if username exists
	exists, err := s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, service.ErrUserAlreadyExists
	}

	// Check if email exists
	exists, err = s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, service.ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := s.passwordHasher.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &entity.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  hashedPassword,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      entity.RoleUser,
		IsActive:  true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate tokens
	return s.generateAuthResponse(ctx, user)
}

func (s *authService) Login(ctx context.Context, req *request.LoginRequest) (*response.AuthResponse, error) {
	// Find user by username or email
	user, err := s.userRepo.GetByUsernameOrEmail(ctx, req.UsernameOrEmail)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, service.ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		return nil, service.ErrUserInactive
	}

	// Verify password
	if !s.passwordHasher.Verify(req.Password, user.Password) {
		return nil, service.ErrInvalidCredentials
	}

	// Generate tokens
	return s.generateAuthResponse(ctx, user)
}

func (s *authService) RefreshToken(ctx context.Context, req *request.RefreshTokenRequest) (*response.AuthResponse, error) {
	// Validate the refresh token JWT
	_, err := s.jwtProvider.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, service.ErrInvalidToken
	}

	// Get the refresh token from database
	refreshToken, err := s.refreshTokenRepo.GetByToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}
	if refreshToken == nil || !refreshToken.IsValid() {
		return nil, service.ErrInvalidToken
	}

	// Revoke the old refresh token
	if err := s.refreshTokenRepo.RevokeByToken(ctx, req.RefreshToken); err != nil {
		return nil, err
	}

	// Get the user
	user, err := s.userRepo.GetByID(ctx, refreshToken.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, service.ErrUserNotFound
	}

	// Check if user is active
	if !user.IsActive {
		return nil, service.ErrUserInactive
	}

	// Generate new tokens
	return s.generateAuthResponse(ctx, user)
}

func (s *authService) Logout(ctx context.Context, token string) error {
	return s.refreshTokenRepo.RevokeByToken(ctx, token)
}

func (s *authService) LogoutAll(ctx context.Context, userID uint) error {
	return s.refreshTokenRepo.RevokeAllByUserID(ctx, userID)
}

func (s *authService) generateAuthResponse(ctx context.Context, user *entity.User) (*response.AuthResponse, error) {
	// Generate access token
	accessToken, err := s.jwtProvider.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshTokenString, expiresAt, err := s.jwtProvider.GenerateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	// Save refresh token to database
	refreshToken := &entity.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenString,
		ExpiresAt: expiresAt,
	}
	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, err
	}

	return &response.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
		TokenType:    "Bearer",
		ExpiresIn:    s.jwtProvider.GetAccessTokenDuration(),
		User: response.UserResponse{
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
		},
	}, nil
}
