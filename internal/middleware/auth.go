package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

// AuthMiddleware provides authentication middleware
type AuthMiddleware struct {
	jwtProvider     *security.JWTProvider
	securityService *security.SecurityService
}

// NewAuthMiddleware creates a new AuthMiddleware instance
func NewAuthMiddleware(jwtProvider *security.JWTProvider, securityService *security.SecurityService) *AuthMiddleware {
	return &AuthMiddleware{
		jwtProvider:     jwtProvider,
		securityService: securityService,
	}
}

// Authenticate validates the JWT token and sets the user in context
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, response.NewError[any]("authorization header required"))
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, response.NewError[any]("invalid authorization header format"))
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := m.jwtProvider.ValidateAccessToken(tokenString)
		if err != nil {
			switch err {
			case security.ErrExpiredToken:
				c.JSON(http.StatusUnauthorized, response.NewError[any]("token has expired"))
			default:
				c.JSON(http.StatusUnauthorized, response.NewError[any]("invalid token"))
			}
			c.Abort()
			return
		}

		// Set claims in context
		m.securityService.SetCurrentClaims(c, claims)

		c.Next()
	}
}

// OptionalAuth validates the JWT token if present but doesn't require it
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]

		claims, err := m.jwtProvider.ValidateAccessToken(tokenString)
		if err == nil {
			m.securityService.SetCurrentClaims(c, claims)
		}

		c.Next()
	}
}

// RequireRole checks if the user has the required role
func (m *AuthMiddleware) RequireRole(roles ...entity.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := m.securityService.GetCurrentClaims(c)
		if claims == nil {
			c.JSON(http.StatusUnauthorized, response.NewError[any]("authentication required"))
			c.Abort()
			return
		}

		for _, role := range roles {
			if claims.Role == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, response.NewError[any]("insufficient permissions"))
		c.Abort()
	}
}

// RequireAdmin checks if the user is an admin
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return m.RequireRole(entity.RoleAdmin)
}
