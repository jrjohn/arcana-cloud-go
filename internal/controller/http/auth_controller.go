package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

const msgValidationFailed = "validation failed"

// AuthController handles authentication endpoints
type AuthController struct {
	authService     service.AuthService
	securityService *security.SecurityService
}

// NewAuthController creates a new AuthController instance
func NewAuthController(authService service.AuthService, securityService *security.SecurityService) *AuthController {
	return &AuthController{
		authService:     authService,
		securityService: securityService,
	}
}

// RegisterRoutes registers the auth routes
func (c *AuthController) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/register", c.Register)
		auth.POST("/login", c.Login)
		auth.POST("/refresh", c.RefreshToken)
		auth.POST("/logout", c.Logout)
		auth.POST("/logout-all", c.LogoutAll)
	}
}

// Register handles user registration
// @Summary Register a new user
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body request.RegisterRequest true "Registration request"
// @Success 201 {object} response.ApiResponse[response.AuthResponse]
// @Failure 400 {object} response.ApiResponse[any]
// @Failure 409 {object} response.ApiResponse[any]
// @Router /api/v1/auth/register [post]
func (c *AuthController) Register(ctx *gin.Context) {
	var req request.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorWithDetails[any](msgValidationFailed, err.Error()))
		return
	}

	authResp, err := c.authService.Register(ctx.Request.Context(), &req)
	if err != nil {
		switch err {
		case service.ErrUserAlreadyExists:
			ctx.JSON(http.StatusConflict, response.NewError[any]("user already exists"))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any]("registration failed"))
		}
		return
	}

	ctx.JSON(http.StatusCreated, response.NewSuccess(authResp, "User registered successfully"))
}

// Login handles user login
// @Summary Login with username/email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body request.LoginRequest true "Login request"
// @Success 200 {object} response.ApiResponse[response.AuthResponse]
// @Failure 400 {object} response.ApiResponse[any]
// @Failure 401 {object} response.ApiResponse[any]
// @Router /api/v1/auth/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	var req request.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorWithDetails[any](msgValidationFailed, err.Error()))
		return
	}

	authResp, err := c.authService.Login(ctx.Request.Context(), &req)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			ctx.JSON(http.StatusUnauthorized, response.NewError[any]("invalid credentials"))
		case service.ErrUserInactive:
			ctx.JSON(http.StatusUnauthorized, response.NewError[any]("account is inactive"))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any]("login failed"))
		}
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess(authResp, "Login successful"))
}

// RefreshToken handles token refresh
// @Summary Refresh access token using refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body request.RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} response.ApiResponse[response.AuthResponse]
// @Failure 400 {object} response.ApiResponse[any]
// @Failure 401 {object} response.ApiResponse[any]
// @Router /api/v1/auth/refresh [post]
func (c *AuthController) RefreshToken(ctx *gin.Context) {
	var req request.RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorWithDetails[any](msgValidationFailed, err.Error()))
		return
	}

	authResp, err := c.authService.RefreshToken(ctx.Request.Context(), &req)
	if err != nil {
		switch err {
		case service.ErrInvalidToken:
			ctx.JSON(http.StatusUnauthorized, response.NewError[any]("invalid or expired refresh token"))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any]("token refresh failed"))
		}
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess(authResp, "Token refreshed successfully"))
}

// Logout handles user logout
// @Summary Logout current session
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.ApiResponse[any]
// @Router /api/v1/auth/logout [post]
func (c *AuthController) Logout(ctx *gin.Context) {
	bearerToken := ctx.GetHeader("Authorization")
	if bearerToken != "" && strings.HasPrefix(bearerToken, "Bearer ") {
		token := strings.TrimPrefix(bearerToken, "Bearer ")
		_ = c.authService.Logout(ctx.Request.Context(), token)
	}

	ctx.JSON(http.StatusOK, response.NewSuccess[any](nil, "Logged out successfully"))
}

// LogoutAll handles logout from all sessions
// @Summary Logout all sessions
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.ApiResponse[any]
// @Router /api/v1/auth/logout-all [post]
func (c *AuthController) LogoutAll(ctx *gin.Context) {
	userID := c.securityService.GetCurrentUserID(ctx)
	if userID > 0 {
		_ = c.authService.LogoutAll(ctx.Request.Context(), userID)
	}

	ctx.JSON(http.StatusOK, response.NewSuccess[any](nil, "All sessions logged out successfully"))
}
