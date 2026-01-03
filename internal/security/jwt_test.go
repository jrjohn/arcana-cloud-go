package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

func newTestJWTProvider() *JWTProvider {
	cfg := &config.JWTConfig{
		Secret:               "test-secret-key-for-testing-purposes",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test-issuer",
	}
	return NewJWTProvider(cfg)
}

func newTestUser() *entity.User {
	return &entity.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     entity.RoleUser,
	}
}

func TestNewJWTProvider(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:               "secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "issuer",
	}

	provider := NewJWTProvider(cfg)

	if provider == nil {
		t.Fatal("NewJWTProvider() returned nil")
	}
	if string(provider.secret) != "secret" {
		t.Errorf("secret = %v, want secret", string(provider.secret))
	}
	if provider.accessTokenDuration != time.Hour {
		t.Errorf("accessTokenDuration = %v, want %v", provider.accessTokenDuration, time.Hour)
	}
	if provider.refreshTokenDuration != 24*time.Hour {
		t.Errorf("refreshTokenDuration = %v, want %v", provider.refreshTokenDuration, 24*time.Hour)
	}
	if provider.issuer != "issuer" {
		t.Errorf("issuer = %v, want issuer", provider.issuer)
	}
}

func TestJWTProvider_GenerateAccessToken(t *testing.T) {
	provider := newTestJWTProvider()
	user := newTestUser()

	token, err := provider.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateAccessToken() returned empty token")
	}

	// Validate the token
	claims, err := provider.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", claims.UserID, user.ID)
	}
	if claims.Username != user.Username {
		t.Errorf("Username = %v, want %v", claims.Username, user.Username)
	}
	if claims.Email != user.Email {
		t.Errorf("Email = %v, want %v", claims.Email, user.Email)
	}
	if claims.Role != user.Role {
		t.Errorf("Role = %v, want %v", claims.Role, user.Role)
	}
}

func TestJWTProvider_GenerateRefreshToken(t *testing.T) {
	provider := newTestJWTProvider()
	user := newTestUser()

	token, expiresAt, err := provider.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateRefreshToken() returned empty token")
	}

	if expiresAt.Before(time.Now()) {
		t.Error("expiresAt should be in the future")
	}

	expectedExpiry := time.Now().Add(24 * time.Hour)
	if expiresAt.Before(expectedExpiry.Add(-time.Minute)) || expiresAt.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("expiresAt = %v, expected around %v", expiresAt, expectedExpiry)
	}
}

func TestJWTProvider_ValidateAccessToken(t *testing.T) {
	provider := newTestJWTProvider()
	user := newTestUser()

	tests := []struct {
		name    string
		token   func() string
		wantErr error
	}{
		{
			name: "valid token",
			token: func() string {
				token, _ := provider.GenerateAccessToken(user)
				return token
			},
			wantErr: nil,
		},
		{
			name: "invalid token format",
			token: func() string {
				return "invalid-token"
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "empty token",
			token: func() string {
				return ""
			},
			wantErr: ErrInvalidToken,
		},
		{
			name: "wrong signature",
			token: func() string {
				// Create token with different secret
				otherProvider := NewJWTProvider(&config.JWTConfig{
					Secret:              "different-secret",
					AccessTokenDuration: time.Hour,
					Issuer:              "test",
				})
				token, _ := otherProvider.GenerateAccessToken(user)
				return token
			},
			wantErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.ValidateAccessToken(tt.token())
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("ValidateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("ValidateAccessToken() unexpected error = %v", err)
			}
		})
	}
}

func TestJWTProvider_ValidateAccessToken_Expired(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:               "test-secret",
		AccessTokenDuration:  -time.Hour, // Already expired
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	provider := NewJWTProvider(cfg)
	user := newTestUser()

	token, _ := provider.GenerateAccessToken(user)

	_, err := provider.ValidateAccessToken(token)
	if err != ErrExpiredToken {
		t.Errorf("ValidateAccessToken() error = %v, want %v", err, ErrExpiredToken)
	}
}

func TestJWTProvider_ValidateRefreshToken(t *testing.T) {
	provider := newTestJWTProvider()
	user := newTestUser()

	token, _, err := provider.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	claims, err := provider.ValidateRefreshToken(token)
	if err != nil {
		t.Fatalf("ValidateRefreshToken() error = %v", err)
	}

	if claims.Subject != user.Username {
		t.Errorf("Subject = %v, want %v", claims.Subject, user.Username)
	}
	if claims.Issuer != "test-issuer" {
		t.Errorf("Issuer = %v, want test-issuer", claims.Issuer)
	}
}

func TestJWTProvider_ValidateRefreshToken_Invalid(t *testing.T) {
	provider := newTestJWTProvider()

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "invalid token",
			token:   "invalid-token",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.ValidateRefreshToken(tt.token)
			if err != tt.wantErr {
				t.Errorf("ValidateRefreshToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJWTProvider_ValidateRefreshToken_Expired(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret:               "test-secret",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: -time.Hour, // Already expired
		Issuer:               "test",
	}
	provider := NewJWTProvider(cfg)
	user := newTestUser()

	token, _, _ := provider.GenerateRefreshToken(user)

	_, err := provider.ValidateRefreshToken(token)
	if err != ErrExpiredToken {
		t.Errorf("ValidateRefreshToken() error = %v, want %v", err, ErrExpiredToken)
	}
}

func TestJWTProvider_GetAccessTokenDuration(t *testing.T) {
	provider := newTestJWTProvider()

	duration := provider.GetAccessTokenDuration()
	expected := int64(time.Hour.Seconds())

	if duration != expected {
		t.Errorf("GetAccessTokenDuration() = %v, want %v", duration, expected)
	}
}

func TestUserClaims_Struct(t *testing.T) {
	claims := UserClaims{
		UserID:   1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     entity.RoleAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "test-issuer",
			Subject:   "testuser",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	if claims.UserID != 1 {
		t.Errorf("UserID = %v, want 1", claims.UserID)
	}
	if claims.Username != "testuser" {
		t.Errorf("Username = %v, want testuser", claims.Username)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email = %v, want test@example.com", claims.Email)
	}
	if claims.Role != entity.RoleAdmin {
		t.Errorf("Role = %v, want %v", claims.Role, entity.RoleAdmin)
	}
}

func TestErrorConstants(t *testing.T) {
	if ErrInvalidToken.Error() != "invalid token" {
		t.Errorf("ErrInvalidToken = %v, want invalid token", ErrInvalidToken.Error())
	}
	if ErrExpiredToken.Error() != "token has expired" {
		t.Errorf("ErrExpiredToken = %v, want token has expired", ErrExpiredToken.Error())
	}
	if ErrInvalidSignature.Error() != "invalid token signature" {
		t.Errorf("ErrInvalidSignature = %v, want invalid token signature", ErrInvalidSignature.Error())
	}
}

func TestJWTProvider_DifferentRoles(t *testing.T) {
	provider := newTestJWTProvider()

	roles := []entity.UserRole{entity.RoleUser, entity.RoleAdmin}

	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			user := &entity.User{
				ID:       1,
				Username: "testuser",
				Email:    "test@example.com",
				Role:     role,
			}

			token, err := provider.GenerateAccessToken(user)
			if err != nil {
				t.Fatalf("GenerateAccessToken() error = %v", err)
			}

			claims, err := provider.ValidateAccessToken(token)
			if err != nil {
				t.Fatalf("ValidateAccessToken() error = %v", err)
			}

			if claims.Role != role {
				t.Errorf("Role = %v, want %v", claims.Role, role)
			}
		})
	}
}

// Benchmarks
func BenchmarkJWTProvider_GenerateAccessToken(b *testing.B) {
	provider := newTestJWTProvider()
	user := newTestUser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.GenerateAccessToken(user)
	}
}

func BenchmarkJWTProvider_ValidateAccessToken(b *testing.B) {
	provider := newTestJWTProvider()
	user := newTestUser()
	token, _ := provider.GenerateAccessToken(user)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.ValidateAccessToken(token)
	}
}

func BenchmarkJWTProvider_GenerateRefreshToken(b *testing.B) {
	provider := newTestJWTProvider()
	user := newTestUser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.GenerateRefreshToken(user)
	}
}
