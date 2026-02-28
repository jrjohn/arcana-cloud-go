package impl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
	"github.com/jrjohn/arcana-cloud-go/internal/testutil/mocks"
)

func setupAuthService(t *testing.T) (service.AuthService, *mocks.MockUserRepository, *mocks.MockRefreshTokenRepository) {
	userRepo := mocks.NewMockUserRepository()
	refreshTokenRepo := mocks.NewMockRefreshTokenRepository()

	jwtConfig := &config.JWTConfig{
		Secret:               "test-secret-key-for-testing-purposes-only",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	jwtProvider := security.NewJWTProvider(jwtConfig)
	passwordHasher := security.NewPasswordHasher()

	authService := NewAuthService(userRepo, refreshTokenRepo, jwtProvider, passwordHasher)
	return authService, userRepo, refreshTokenRepo
}

func TestNewAuthService(t *testing.T) {
	authService, _, _ := setupAuthService(t)
	if authService == nil {
		t.Fatal("NewAuthService() returned nil")
	}
}

func TestAuthService_Register_Success(t *testing.T) {
	authService, _, _ := setupAuthService(t)
	ctx := context.Background()

	req := &request.RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	resp, err := authService.Register(ctx, req)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if resp == nil {
		t.Fatal("Register() returned nil response")
	}
	if resp.AccessToken == "" {
		t.Error("Register() AccessToken is empty")
	}
	if resp.RefreshToken == "" {
		t.Error("Register() RefreshToken is empty")
	}
	if resp.User.Username != "testuser" {
		t.Errorf("Register() Username = %v, want testuser", resp.User.Username)
	}
	if resp.User.Email != "test@example.com" {
		t.Errorf("Register() Email = %v, want test@example.com", resp.User.Email)
	}
}

func TestAuthService_Register_UsernameExists(t *testing.T) {
	authService, userRepo, _ := setupAuthService(t)
	ctx := context.Background()

	// Add existing user
	userRepo.AddUser(&entity.User{
		Username: "existinguser",
		Email:    "existing@example.com",
		Password: "hash",
	})

	req := &request.RegisterRequest{
		Username:  "existinguser",
		Email:     "new@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	_, err := authService.Register(ctx, req)
	if !errors.Is(err, service.ErrUserAlreadyExists) {
		t.Errorf("Register() error = %v, want ErrUserAlreadyExists", err)
	}
}

func TestAuthService_Register_EmailExists(t *testing.T) {
	authService, userRepo, _ := setupAuthService(t)
	ctx := context.Background()

	// Add existing user
	userRepo.AddUser(&entity.User{
		Username: "existinguser",
		Email:    "existing@example.com",
		Password: "hash",
	})

	req := &request.RegisterRequest{
		Username:  "newuser",
		Email:     "existing@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	_, err := authService.Register(ctx, req)
	if !errors.Is(err, service.ErrUserAlreadyExists) {
		t.Errorf("Register() error = %v, want ErrUserAlreadyExists", err)
	}
}

func TestAuthService_Register_UsernameCheckError(t *testing.T) {
	authService, userRepo, _ := setupAuthService(t)
	ctx := context.Background()

	expectedErr := errors.New("database error")
	userRepo.ExistsByUsernameErr = expectedErr

	req := &request.RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	_, err := authService.Register(ctx, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Register() error = %v, want %v", err, expectedErr)
	}
}

func TestAuthService_Register_EmailCheckError(t *testing.T) {
	authService, userRepo, _ := setupAuthService(t)
	ctx := context.Background()

	expectedErr := errors.New("database error")
	userRepo.ExistsByEmailErr = expectedErr

	req := &request.RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	_, err := authService.Register(ctx, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Register() error = %v, want %v", err, expectedErr)
	}
}

func TestAuthService_Register_CreateError(t *testing.T) {
	authService, userRepo, _ := setupAuthService(t)
	ctx := context.Background()

	expectedErr := errors.New("create error")
	userRepo.CreateErr = expectedErr

	req := &request.RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	_, err := authService.Register(ctx, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Register() error = %v, want %v", err, expectedErr)
	}
}

func TestAuthService_Register_RefreshTokenCreateError(t *testing.T) {
	authService, _, refreshTokenRepo := setupAuthService(t)
	ctx := context.Background()

	expectedErr := errors.New("refresh token create error")
	refreshTokenRepo.CreateErr = expectedErr

	req := &request.RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
	}

	_, err := authService.Register(ctx, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Register() error = %v, want %v", err, expectedErr)
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	authService, userRepo, _ := setupAuthService(t)
	ctx := context.Background()

	// Create a password hash
	passwordHasher := security.NewPasswordHasher()
	hashedPassword, _ := passwordHasher.Hash("password123")

	// Add existing user
	userRepo.AddUser(&entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		IsActive: true,
	})

	req := &request.LoginRequest{
		UsernameOrEmail: "testuser",
		Password:        "password123",
	}

	resp, err := authService.Login(ctx, req)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if resp == nil {
		t.Fatal("Login() returned nil response")
	}
	if resp.AccessToken == "" {
		t.Error("Login() AccessToken is empty")
	}
}

func TestAuthService_Login_ByEmail(t *testing.T) {
	authService, userRepo, _ := setupAuthService(t)
	ctx := context.Background()

	passwordHasher := security.NewPasswordHasher()
	hashedPassword, _ := passwordHasher.Hash("password123")

	userRepo.AddUser(&entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		IsActive: true,
	})

	req := &request.LoginRequest{
		UsernameOrEmail: "test@example.com",
		Password:        "password123",
	}

	resp, err := authService.Login(ctx, req)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if resp == nil {
		t.Fatal("Login() returned nil response")
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	authService, _, _ := setupAuthService(t)
	ctx := context.Background()

	req := &request.LoginRequest{
		UsernameOrEmail: "nonexistent",
		Password:        "password123",
	}

	_, err := authService.Login(ctx, req)
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	authService, userRepo, _ := setupAuthService(t)
	ctx := context.Background()

	passwordHasher := security.NewPasswordHasher()
	hashedPassword, _ := passwordHasher.Hash("correctpassword")

	userRepo.AddUser(&entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		IsActive: true,
	})

	req := &request.LoginRequest{
		UsernameOrEmail: "testuser",
		Password:        "wrongpassword",
	}

	_, err := authService.Login(ctx, req)
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthService_Login_UserInactive(t *testing.T) {
	authService, userRepo, _ := setupAuthService(t)
	ctx := context.Background()

	passwordHasher := security.NewPasswordHasher()
	hashedPassword, _ := passwordHasher.Hash("password123")

	userRepo.AddUser(&entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: hashedPassword,
		IsActive: false,
	})

	req := &request.LoginRequest{
		UsernameOrEmail: "testuser",
		Password:        "password123",
	}

	_, err := authService.Login(ctx, req)
	if !errors.Is(err, service.ErrUserInactive) {
		t.Errorf("Login() error = %v, want ErrUserInactive", err)
	}
}

func TestAuthService_Login_LookupError(t *testing.T) {
	authService, userRepo, _ := setupAuthService(t)
	ctx := context.Background()

	expectedErr := errors.New("database error")
	userRepo.GetByUsernameOrEmailErr = expectedErr

	req := &request.LoginRequest{
		UsernameOrEmail: "testuser",
		Password:        "password123",
	}

	_, err := authService.Login(ctx, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("Login() error = %v, want %v", err, expectedErr)
	}
}

func TestAuthService_RefreshToken_Success(t *testing.T) {
	authService, userRepo, refreshTokenRepo := setupAuthService(t)
	ctx := context.Background()

	// Add user
	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	// Generate a valid refresh token
	jwtConfig := &config.JWTConfig{
		Secret:               "test-secret-key-for-testing-purposes-only",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	jwtProvider := security.NewJWTProvider(jwtConfig)
	tokenString, expiresAt, _ := jwtProvider.GenerateRefreshToken(user)

	// Add refresh token to repo
	refreshTokenRepo.AddToken(&entity.RefreshToken{
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: expiresAt,
		Revoked:   false,
	})

	req := &request.RefreshTokenRequest{
		RefreshToken: tokenString,
	}

	resp, err := authService.RefreshToken(ctx, req)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if resp == nil {
		t.Fatal("RefreshToken() returned nil response")
	}
	if resp.AccessToken == "" {
		t.Error("RefreshToken() AccessToken is empty")
	}
}

func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	authService, _, _ := setupAuthService(t)
	ctx := context.Background()

	req := &request.RefreshTokenRequest{
		RefreshToken: "invalid-token",
	}

	_, err := authService.RefreshToken(ctx, req)
	if !errors.Is(err, service.ErrInvalidToken) {
		t.Errorf("RefreshToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestAuthService_RefreshToken_TokenNotInDB(t *testing.T) {
	authService, userRepo, _ := setupAuthService(t)
	ctx := context.Background()

	// Add user
	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	// Generate valid JWT but don't add to repo
	jwtConfig := &config.JWTConfig{
		Secret:               "test-secret-key-for-testing-purposes-only",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	jwtProvider := security.NewJWTProvider(jwtConfig)
	tokenString, _, _ := jwtProvider.GenerateRefreshToken(user)

	req := &request.RefreshTokenRequest{
		RefreshToken: tokenString,
	}

	_, err := authService.RefreshToken(ctx, req)
	if !errors.Is(err, service.ErrInvalidToken) {
		t.Errorf("RefreshToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestAuthService_RefreshToken_RevokedToken(t *testing.T) {
	authService, userRepo, refreshTokenRepo := setupAuthService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	jwtConfig := &config.JWTConfig{
		Secret:               "test-secret-key-for-testing-purposes-only",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	jwtProvider := security.NewJWTProvider(jwtConfig)
	tokenString, expiresAt, _ := jwtProvider.GenerateRefreshToken(user)

	// Add revoked token
	refreshTokenRepo.AddToken(&entity.RefreshToken{
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: expiresAt,
		Revoked:   true,
	})

	req := &request.RefreshTokenRequest{
		RefreshToken: tokenString,
	}

	_, err := authService.RefreshToken(ctx, req)
	if !errors.Is(err, service.ErrInvalidToken) {
		t.Errorf("RefreshToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestAuthService_RefreshToken_UserNotFound(t *testing.T) {
	authService, userRepo, refreshTokenRepo := setupAuthService(t)
	ctx := context.Background()

	user := &entity.User{
		ID:       999,
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	// Don't add user to repo

	jwtConfig := &config.JWTConfig{
		Secret:               "test-secret-key-for-testing-purposes-only",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	jwtProvider := security.NewJWTProvider(jwtConfig)
	tokenString, expiresAt, _ := jwtProvider.GenerateRefreshToken(user)

	refreshTokenRepo.AddToken(&entity.RefreshToken{
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: expiresAt,
		Revoked:   false,
	})

	// Add a different user so the lookup succeeds but wrong user
	userRepo.AddUser(&entity.User{
		Username: "otheruser",
		Email:    "other@example.com",
		Password: "hash",
		IsActive: true,
	})

	req := &request.RefreshTokenRequest{
		RefreshToken: tokenString,
	}

	_, err := authService.RefreshToken(ctx, req)
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Errorf("RefreshToken() error = %v, want ErrUserNotFound", err)
	}
}

func TestAuthService_RefreshToken_UserInactive(t *testing.T) {
	authService, userRepo, refreshTokenRepo := setupAuthService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: false,
	}
	userRepo.AddUser(user)

	jwtConfig := &config.JWTConfig{
		Secret:               "test-secret-key-for-testing-purposes-only",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	jwtProvider := security.NewJWTProvider(jwtConfig)
	tokenString, expiresAt, _ := jwtProvider.GenerateRefreshToken(user)

	refreshTokenRepo.AddToken(&entity.RefreshToken{
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: expiresAt,
		Revoked:   false,
	})

	req := &request.RefreshTokenRequest{
		RefreshToken: tokenString,
	}

	_, err := authService.RefreshToken(ctx, req)
	if !errors.Is(err, service.ErrUserInactive) {
		t.Errorf("RefreshToken() error = %v, want ErrUserInactive", err)
	}
}

func TestAuthService_RefreshToken_GetByTokenError(t *testing.T) {
	authService, userRepo, refreshTokenRepo := setupAuthService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	jwtConfig := &config.JWTConfig{
		Secret:               "test-secret-key-for-testing-purposes-only",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	jwtProvider := security.NewJWTProvider(jwtConfig)
	tokenString, _, _ := jwtProvider.GenerateRefreshToken(user)

	expectedErr := errors.New("database error")
	refreshTokenRepo.GetByTokenErr = expectedErr

	req := &request.RefreshTokenRequest{
		RefreshToken: tokenString,
	}

	_, err := authService.RefreshToken(ctx, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("RefreshToken() error = %v, want %v", err, expectedErr)
	}
}

func TestAuthService_RefreshToken_RevokeError(t *testing.T) {
	authService, userRepo, refreshTokenRepo := setupAuthService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	jwtConfig := &config.JWTConfig{
		Secret:               "test-secret-key-for-testing-purposes-only",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	jwtProvider := security.NewJWTProvider(jwtConfig)
	tokenString, expiresAt, _ := jwtProvider.GenerateRefreshToken(user)

	refreshTokenRepo.AddToken(&entity.RefreshToken{
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: expiresAt,
		Revoked:   false,
	})

	expectedErr := errors.New("revoke error")
	refreshTokenRepo.RevokeByTokenErr = expectedErr

	req := &request.RefreshTokenRequest{
		RefreshToken: tokenString,
	}

	_, err := authService.RefreshToken(ctx, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("RefreshToken() error = %v, want %v", err, expectedErr)
	}
}

func TestAuthService_RefreshToken_GetUserError(t *testing.T) {
	authService, userRepo, refreshTokenRepo := setupAuthService(t)
	ctx := context.Background()

	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hash",
		IsActive: true,
	}
	userRepo.AddUser(user)

	jwtConfig := &config.JWTConfig{
		Secret:               "test-secret-key-for-testing-purposes-only",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	jwtProvider := security.NewJWTProvider(jwtConfig)
	tokenString, expiresAt, _ := jwtProvider.GenerateRefreshToken(user)

	refreshTokenRepo.AddToken(&entity.RefreshToken{
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: expiresAt,
		Revoked:   false,
	})

	expectedErr := errors.New("get user error")
	userRepo.GetByIDErr = expectedErr

	req := &request.RefreshTokenRequest{
		RefreshToken: tokenString,
	}

	_, err := authService.RefreshToken(ctx, req)
	if !errors.Is(err, expectedErr) {
		t.Errorf("RefreshToken() error = %v, want %v", err, expectedErr)
	}
}

func TestAuthService_Logout(t *testing.T) {
	authService, _, refreshTokenRepo := setupAuthService(t)
	ctx := context.Background()

	// Add a token
	refreshTokenRepo.AddToken(&entity.RefreshToken{
		UserID:    1,
		Token:     "test-token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
	})

	err := authService.Logout(ctx, "test-token")
	if err != nil {
		t.Errorf("Logout() error = %v", err)
	}
}

func TestAuthService_Logout_Error(t *testing.T) {
	authService, _, refreshTokenRepo := setupAuthService(t)
	ctx := context.Background()

	expectedErr := errors.New("revoke error")
	refreshTokenRepo.RevokeByTokenErr = expectedErr

	err := authService.Logout(ctx, "test-token")
	if !errors.Is(err, expectedErr) {
		t.Errorf("Logout() error = %v, want %v", err, expectedErr)
	}
}

func TestAuthService_LogoutAll(t *testing.T) {
	authService, _, refreshTokenRepo := setupAuthService(t)
	ctx := context.Background()

	// Add tokens for user
	refreshTokenRepo.AddToken(&entity.RefreshToken{
		UserID:    1,
		Token:     "token1",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
	})
	refreshTokenRepo.AddToken(&entity.RefreshToken{
		UserID:    1,
		Token:     "token2",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
	})

	err := authService.LogoutAll(ctx, 1)
	if err != nil {
		t.Errorf("LogoutAll() error = %v", err)
	}
}

func TestAuthService_LogoutAll_Error(t *testing.T) {
	authService, _, refreshTokenRepo := setupAuthService(t)
	ctx := context.Background()

	expectedErr := errors.New("revoke all error")
	refreshTokenRepo.RevokeAllByUserIDErr = expectedErr

	err := authService.LogoutAll(ctx, 1)
	if !errors.Is(err, expectedErr) {
		t.Errorf("LogoutAll() error = %v, want %v", err, expectedErr)
	}
}

// Error constants tests
func TestServiceErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrUserNotFound", service.ErrUserNotFound, "user not found"},
		{"ErrInvalidCredentials", service.ErrInvalidCredentials, "invalid credentials"},
		{"ErrUserAlreadyExists", service.ErrUserAlreadyExists, "user already exists"},
		{"ErrInvalidToken", service.ErrInvalidToken, "invalid or expired token"},
		{"ErrUserInactive", service.ErrUserInactive, "user account is inactive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("%s.Error() = %v, want %v", tt.name, tt.err.Error(), tt.expected)
			}
		})
	}
}
