package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidSignature = errors.New("invalid token signature")
)

// UserClaims represents the JWT claims for a user
type UserClaims struct {
	UserID   uint            `json:"user_id"`
	Username string          `json:"username"`
	Email    string          `json:"email"`
	Role     entity.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// JWTProvider handles JWT token generation and validation
type JWTProvider struct {
	secret               []byte
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
	issuer               string
}

// NewJWTProvider creates a new JWTProvider instance
func NewJWTProvider(cfg *config.JWTConfig) *JWTProvider {
	return &JWTProvider{
		secret:               []byte(cfg.Secret),
		accessTokenDuration:  cfg.AccessTokenDuration,
		refreshTokenDuration: cfg.RefreshTokenDuration,
		issuer:               cfg.Issuer,
	}
}

// GenerateAccessToken generates a new access token for a user
func (p *JWTProvider) GenerateAccessToken(user *entity.User) (string, error) {
	now := time.Now()
	claims := UserClaims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    p.issuer,
			Subject:   user.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(p.accessTokenDuration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(p.secret)
}

// GenerateRefreshToken generates a new refresh token
func (p *JWTProvider) GenerateRefreshToken(user *entity.User) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(p.refreshTokenDuration)

	claims := jwt.RegisteredClaims{
		ID:        uuid.New().String(), // Unique token ID to prevent prefix collisions
		Issuer:    p.issuer,
		Subject:   user.Username,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(p.secret)
	return tokenString, expiresAt, err
}

// ValidateAccessToken validates an access token and returns the claims
func (p *JWTProvider) ValidateAccessToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignature
		}
		return p.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token
func (p *JWTProvider) ValidateRefreshToken(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignature
		}
		return p.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetAccessTokenDuration returns the access token duration in seconds
func (p *JWTProvider) GetAccessTokenDuration() int64 {
	return int64(p.accessTokenDuration.Seconds())
}
