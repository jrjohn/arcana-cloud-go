package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
	"github.com/jrjohn/arcana-cloud-go/internal/middleware"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

const (
	msgNotAuthenticated = "not authenticated"
	msgUserNotFound     = "user not found"
	msgFailedFetchUser  = "failed to fetch user"
)

// UserController handles user management endpoints
type UserController struct {
	userService     service.UserService
	securityService *security.SecurityService
	authMiddleware  *middleware.AuthMiddleware
}

// NewUserController creates a new UserController instance
func NewUserController(
	userService service.UserService,
	securityService *security.SecurityService,
	authMiddleware *middleware.AuthMiddleware,
) *UserController {
	return &UserController{
		userService:     userService,
		securityService: securityService,
		authMiddleware:  authMiddleware,
	}
}

// RegisterRoutes registers the user routes
func (c *UserController) RegisterRoutes(router *gin.RouterGroup) {
	users := router.Group("/users")
	users.Use(c.authMiddleware.Authenticate())
	{
		users.GET("", c.authMiddleware.RequireAdmin(), c.List)
		users.GET("/me", c.GetCurrentUser)
		users.PUT("/me", c.UpdateCurrentUser)
		users.PUT("/me/password", c.ChangePassword)
		users.GET("/:id", c.GetByID)
		users.GET("/username/:username", c.GetByUsername)
		users.DELETE("/:id", c.authMiddleware.RequireAdmin(), c.Delete)
	}
}

// List retrieves all users with pagination
// @Summary List all users
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param size query int false "Page size" default(10)
// @Success 200 {object} response.ApiResponse[response.PagedResponse[response.UserResponse]]
// @Router /api/v1/users [get]
func (c *UserController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))

	users, err := c.userService.List(ctx.Request.Context(), page, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to fetch users"))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(users))
}

// GetCurrentUser retrieves the current authenticated user
// @Summary Get current user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.ApiResponse[response.UserResponse]
// @Router /api/v1/users/me [get]
func (c *UserController) GetCurrentUser(ctx *gin.Context) {
	userID := c.securityService.GetCurrentUserID(ctx)
	if userID == 0 {
		ctx.JSON(http.StatusUnauthorized, response.NewError[any](msgNotAuthenticated))
		return
	}

	user, err := c.userService.GetByID(ctx.Request.Context(), userID)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			ctx.JSON(http.StatusNotFound, response.NewError[any](msgUserNotFound))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any](msgFailedFetchUser))
		}
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(user))
}

// UpdateCurrentUser updates the current user's profile
// @Summary Update current user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.UpdateProfileRequest true "Update request"
// @Success 200 {object} response.ApiResponse[response.UserResponse]
// @Router /api/v1/users/me [put]
func (c *UserController) UpdateCurrentUser(ctx *gin.Context) {
	userID := c.securityService.GetCurrentUserID(ctx)
	if userID == 0 {
		ctx.JSON(http.StatusUnauthorized, response.NewError[any](msgNotAuthenticated))
		return
	}

	var req request.UpdateProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorWithDetails[any]("validation failed", err.Error()))
		return
	}

	user, err := c.userService.Update(ctx.Request.Context(), userID, &req)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			ctx.JSON(http.StatusNotFound, response.NewError[any](msgUserNotFound))
		case service.ErrUserAlreadyExists:
			ctx.JSON(http.StatusConflict, response.NewError[any]("email already in use"))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to update user"))
		}
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess(user, "Profile updated successfully"))
}

// ChangePassword changes the current user's password
// @Summary Change password
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.ChangePasswordRequest true "Password change request"
// @Success 200 {object} response.ApiResponse[any]
// @Router /api/v1/users/me/password [put]
func (c *UserController) ChangePassword(ctx *gin.Context) {
	userID := c.securityService.GetCurrentUserID(ctx)
	if userID == 0 {
		ctx.JSON(http.StatusUnauthorized, response.NewError[any](msgNotAuthenticated))
		return
	}

	var req request.ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorWithDetails[any]("validation failed", err.Error()))
		return
	}

	err := c.userService.ChangePassword(ctx.Request.Context(), userID, &req)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			ctx.JSON(http.StatusNotFound, response.NewError[any](msgUserNotFound))
		case service.ErrInvalidCredentials:
			ctx.JSON(http.StatusBadRequest, response.NewError[any]("current password is incorrect"))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to change password"))
		}
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess[any](nil, "Password changed successfully"))
}

// GetByID retrieves a user by ID
// @Summary Get user by ID
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} response.ApiResponse[response.UserResponse]
// @Router /api/v1/users/{id} [get]
func (c *UserController) GetByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("invalid user ID"))
		return
	}

	user, err := c.userService.GetByID(ctx.Request.Context(), uint(id))
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			ctx.JSON(http.StatusNotFound, response.NewError[any](msgUserNotFound))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any](msgFailedFetchUser))
		}
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(user))
}

// GetByUsername retrieves a user by username
// @Summary Get user by username
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param username path string true "Username"
// @Success 200 {object} response.ApiResponse[response.UserResponse]
// @Router /api/v1/users/username/{username} [get]
func (c *UserController) GetByUsername(ctx *gin.Context) {
	username := ctx.Param("username")
	if username == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("username is required"))
		return
	}

	user, err := c.userService.GetByUsername(ctx.Request.Context(), username)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			ctx.JSON(http.StatusNotFound, response.NewError[any](msgUserNotFound))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any](msgFailedFetchUser))
		}
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(user))
}

// Delete removes a user
// @Summary Delete user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} response.ApiResponse[any]
// @Router /api/v1/users/{id} [delete]
func (c *UserController) Delete(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("invalid user ID"))
		return
	}

	if err := c.userService.Delete(ctx.Request.Context(), uint(id)); err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to delete user"))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess[any](nil, "User deleted successfully"))
}
