package security

import (
	"github.com/gin-gonic/gin"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

const (
	// ContextKeyUser is the key for storing user in context
	ContextKeyUser = "current_user"
	// ContextKeyClaims is the key for storing claims in context
	ContextKeyClaims = "current_claims"
)

// SecurityService provides security-related utilities
type SecurityService struct {
	jwtProvider *JWTProvider
}

// NewSecurityService creates a new SecurityService instance
func NewSecurityService(jwtProvider *JWTProvider) *SecurityService {
	return &SecurityService{jwtProvider: jwtProvider}
}

// GetCurrentUser retrieves the current user from the context
func (s *SecurityService) GetCurrentUser(c *gin.Context) *entity.User {
	user, exists := c.Get(ContextKeyUser)
	if !exists {
		return nil
	}
	if u, ok := user.(*entity.User); ok {
		return u
	}
	return nil
}

// GetCurrentUserID retrieves the current user's ID from the context
func (s *SecurityService) GetCurrentUserID(c *gin.Context) uint {
	claims := s.GetCurrentClaims(c)
	if claims != nil {
		return claims.UserID
	}
	return 0
}

// GetCurrentClaims retrieves the current JWT claims from the context
func (s *SecurityService) GetCurrentClaims(c *gin.Context) *UserClaims {
	claims, exists := c.Get(ContextKeyClaims)
	if !exists {
		return nil
	}
	if cl, ok := claims.(*UserClaims); ok {
		return cl
	}
	return nil
}

// SetCurrentUser sets the current user in the context
func (s *SecurityService) SetCurrentUser(c *gin.Context, user *entity.User) {
	c.Set(ContextKeyUser, user)
}

// SetCurrentClaims sets the current claims in the context
func (s *SecurityService) SetCurrentClaims(c *gin.Context, claims *UserClaims) {
	c.Set(ContextKeyClaims, claims)
}

// IsAuthenticated checks if the current request is authenticated
func (s *SecurityService) IsAuthenticated(c *gin.Context) bool {
	return s.GetCurrentClaims(c) != nil
}

// HasRole checks if the current user has the specified role
func (s *SecurityService) HasRole(c *gin.Context, role entity.UserRole) bool {
	claims := s.GetCurrentClaims(c)
	if claims == nil {
		return false
	}
	return claims.Role == role
}

// IsAdmin checks if the current user is an admin
func (s *SecurityService) IsAdmin(c *gin.Context) bool {
	return s.HasRole(c, entity.RoleAdmin)
}
