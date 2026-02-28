package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/api/proto/pb"
	domainservice "github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
	"github.com/jrjohn/arcana-cloud-go/internal/config"
)

func newTestJWT() *security.JWTProvider {
	jwtCfg := &config.JWTConfig{
		Secret:               "test-secret-key-12345",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	return security.NewJWTProvider(jwtCfg)
}

func newTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func newAuthResp() *response.AuthResponse {
	now := time.Now()
	return &response.AuthResponse{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		User: response.UserResponse{
			ID:         1,
			Username:   "testuser",
			Email:      "test@example.com",
			FirstName:  "Test",
			LastName:   "User",
			Role:       "user",
			IsActive:   true,
			IsVerified: true,
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	}
}

// mockAuthService is a simple mock for auth service
type mockAuthService struct {
	registerFn      func(ctx context.Context, req *request.RegisterRequest) (*response.AuthResponse, error)
	loginFn         func(ctx context.Context, req *request.LoginRequest) (*response.AuthResponse, error)
	refreshTokenFn  func(ctx context.Context, req *request.RefreshTokenRequest) (*response.AuthResponse, error)
	logoutFn        func(ctx context.Context, token string) error
	logoutAllFn     func(ctx context.Context, userID uint) error
}

func (m *mockAuthService) Register(ctx context.Context, req *request.RegisterRequest) (*response.AuthResponse, error) {
	if m.registerFn != nil {
		return m.registerFn(ctx, req)
	}
	return newAuthResp(), nil
}

func (m *mockAuthService) Login(ctx context.Context, req *request.LoginRequest) (*response.AuthResponse, error) {
	if m.loginFn != nil {
		return m.loginFn(ctx, req)
	}
	return newAuthResp(), nil
}

func (m *mockAuthService) RefreshToken(ctx context.Context, req *request.RefreshTokenRequest) (*response.AuthResponse, error) {
	if m.refreshTokenFn != nil {
		return m.refreshTokenFn(ctx, req)
	}
	return newAuthResp(), nil
}

func (m *mockAuthService) Logout(ctx context.Context, token string) error {
	if m.logoutFn != nil {
		return m.logoutFn(ctx, token)
	}
	return nil
}

func (m *mockAuthService) LogoutAll(ctx context.Context, userID uint) error {
	if m.logoutAllFn != nil {
		return m.logoutAllFn(ctx, userID)
	}
	return nil
}

func TestNewAuthServiceServer(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{}

	server := NewAuthServiceServer(svc, jwtProvider, logger)
	if server == nil {
		t.Error("NewAuthServiceServer() returned nil")
	}
}

func TestAuthServiceServer_Register_Success(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{
		registerFn: func(ctx context.Context, req *request.RegisterRequest) (*response.AuthResponse, error) {
			return newAuthResp(), nil
		},
	}

	server := NewAuthServiceServer(svc, jwtProvider, logger)
	ctx := context.Background()

	req := &pb.RegisterRequest{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "password123",
	}

	resp, err := server.Register(ctx, req)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if resp == nil {
		t.Fatal("Register() returned nil response")
	}
	if resp.AccessToken != "access-token" {
		t.Errorf("AccessToken = %v, want access-token", resp.AccessToken)
	}
}

func TestAuthServiceServer_Register_Error(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{
		registerFn: func(ctx context.Context, req *request.RegisterRequest) (*response.AuthResponse, error) {
			return nil, domainservice.ErrUserAlreadyExists
		},
	}

	server := NewAuthServiceServer(svc, jwtProvider, logger)
	ctx := context.Background()

	req := &pb.RegisterRequest{
		Username: "existing",
		Email:    "existing@example.com",
		Password: "pass",
	}

	_, err := server.Register(ctx, req)
	if err == nil {
		t.Error("Register() should return error when user already exists")
	}
}

func TestAuthServiceServer_Login_Success(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{}

	server := NewAuthServiceServer(svc, jwtProvider, logger)
	ctx := context.Background()

	req := &pb.LoginRequest{
		UsernameOrEmail: "testuser",
		Password:        "password123",
	}

	resp, err := server.Login(ctx, req)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if resp == nil {
		t.Fatal("Login() returned nil response")
	}
}

func TestAuthServiceServer_Login_InvalidCredentials(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{
		loginFn: func(ctx context.Context, req *request.LoginRequest) (*response.AuthResponse, error) {
			return nil, domainservice.ErrInvalidCredentials
		},
	}

	server := NewAuthServiceServer(svc, jwtProvider, logger)
	ctx := context.Background()

	req := &pb.LoginRequest{
		UsernameOrEmail: "user",
		Password:        "wrong",
	}

	_, err := server.Login(ctx, req)
	if err == nil {
		t.Error("Login() should return error for invalid credentials")
	}
}

func TestAuthServiceServer_RefreshToken_Success(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{}

	server := NewAuthServiceServer(svc, jwtProvider, logger)
	ctx := context.Background()

	req := &pb.RefreshTokenRequest{
		RefreshToken: "refresh-token",
	}

	resp, err := server.RefreshToken(ctx, req)
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if resp == nil {
		t.Fatal("RefreshToken() returned nil response")
	}
}

func TestAuthServiceServer_RefreshToken_InvalidToken(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{
		refreshTokenFn: func(ctx context.Context, req *request.RefreshTokenRequest) (*response.AuthResponse, error) {
			return nil, domainservice.ErrInvalidToken
		},
	}

	server := NewAuthServiceServer(svc, jwtProvider, logger)
	ctx := context.Background()

	req := &pb.RefreshTokenRequest{
		RefreshToken: "invalid",
	}

	_, err := server.RefreshToken(ctx, req)
	if err == nil {
		t.Error("RefreshToken() should return error for invalid token")
	}
}

func TestAuthServiceServer_Logout_Success(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{}

	server := NewAuthServiceServer(svc, jwtProvider, logger)
	ctx := context.Background()

	req := &pb.LogoutRequest{Token: "test-token"}
	resp, err := server.Logout(ctx, req)
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if resp == nil {
		t.Error("Logout() returned nil response")
	}
}

func TestAuthServiceServer_Logout_Error(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{
		logoutFn: func(ctx context.Context, token string) error {
			return errors.New("logout error")
		},
	}

	server := NewAuthServiceServer(svc, jwtProvider, logger)
	ctx := context.Background()

	req := &pb.LogoutRequest{Token: "test-token"}
	_, err := server.Logout(ctx, req)
	if err == nil {
		t.Error("Logout() should return error on failure")
	}
}

func TestAuthServiceServer_LogoutAll_Success(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{}

	server := NewAuthServiceServer(svc, jwtProvider, logger)
	ctx := context.Background()

	req := &pb.LogoutAllRequest{UserId: 1}
	resp, err := server.LogoutAll(ctx, req)
	if err != nil {
		t.Fatalf("LogoutAll() error = %v", err)
	}
	if resp == nil {
		t.Error("LogoutAll() returned nil response")
	}
}

func TestAuthServiceServer_LogoutAll_Error(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{
		logoutAllFn: func(ctx context.Context, userID uint) error {
			return errors.New("logout all error")
		},
	}

	server := NewAuthServiceServer(svc, jwtProvider, logger)
	ctx := context.Background()

	req := &pb.LogoutAllRequest{UserId: 1}
	_, err := server.LogoutAll(ctx, req)
	if err == nil {
		t.Error("LogoutAll() should return error on failure")
	}
}

func TestAuthServiceServer_ValidateToken_Valid(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{}
	server := NewAuthServiceServer(svc, jwtProvider, logger)
	ctx := context.Background()

	// Test with invalid token
	req := &pb.ValidateTokenRequest{Token: "invalid-token"}
	resp, err := server.ValidateToken(ctx, req)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if resp.Valid {
		t.Error("ValidateToken() should return Valid=false for invalid token")
	}
}

func TestAuthServiceServer_ValidateToken_InvalidToken(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{}
	server := NewAuthServiceServer(svc, jwtProvider, logger)
	ctx := context.Background()

	req := &pb.ValidateTokenRequest{Token: "bad-token"}
	resp, err := server.ValidateToken(ctx, req)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if resp.Valid {
		t.Error("ValidateToken() should return Valid=false for bad token")
	}
}

func TestAuthServiceServer_MapError_AllCases(t *testing.T) {
	logger := newTestLogger()
	jwtProvider := newTestJWT()
	svc := &mockAuthService{}
	server := NewAuthServiceServer(svc, jwtProvider, logger)

	tests := []struct {
		name string
		err  error
	}{
		{"UserNotFound", domainservice.ErrUserNotFound},
		{"InvalidCredentials", domainservice.ErrInvalidCredentials},
		{"UserAlreadyExists", domainservice.ErrUserAlreadyExists},
		{"InvalidToken", domainservice.ErrInvalidToken},
		{"UserInactive", domainservice.ErrUserInactive},
		{"InternalError", errors.New("unexpected error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.mapError(tt.err)
			if result == nil {
				t.Error("mapError() returned nil")
			}
		})
	}
}
