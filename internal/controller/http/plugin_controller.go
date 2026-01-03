package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
	"github.com/jrjohn/arcana-cloud-go/internal/middleware"
)

// PluginController handles plugin management endpoints
type PluginController struct {
	pluginService  service.PluginService
	authMiddleware *middleware.AuthMiddleware
}

// NewPluginController creates a new PluginController instance
func NewPluginController(pluginService service.PluginService, authMiddleware *middleware.AuthMiddleware) *PluginController {
	return &PluginController{
		pluginService:  pluginService,
		authMiddleware: authMiddleware,
	}
}

// RegisterRoutes registers the plugin routes
func (c *PluginController) RegisterRoutes(router *gin.RouterGroup) {
	plugins := router.Group("/plugins")
	{
		// Public health endpoints
		plugins.GET("/health", c.GetHealth)
		plugins.GET("/health/ready", c.GetReadiness)
		plugins.GET("/health/live", c.GetLiveness)

		// Protected endpoints
		protected := plugins.Group("")
		protected.Use(c.authMiddleware.Authenticate())
		{
			protected.GET("", c.List)
			protected.GET("/:key", c.GetByKey)
			protected.POST("/install", c.authMiddleware.RequireAdmin(), c.Install)
			protected.POST("/:key/enable", c.authMiddleware.RequireAdmin(), c.Enable)
			protected.POST("/:key/disable", c.authMiddleware.RequireAdmin(), c.Disable)
			protected.DELETE("/:key", c.authMiddleware.RequireAdmin(), c.Uninstall)
		}
	}
}

// List retrieves all plugins
// @Summary List all plugins
// @Tags Plugins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param size query int false "Page size" default(10)
// @Success 200 {object} response.ApiResponse[response.PagedResponse[response.PluginResponse]]
// @Router /api/v1/plugins [get]
func (c *PluginController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))

	plugins, err := c.pluginService.List(ctx.Request.Context(), page, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to fetch plugins"))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(plugins))
}

// GetByKey retrieves a plugin by its key
// @Summary Get plugin by key
// @Tags Plugins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param key path string true "Plugin key"
// @Success 200 {object} response.ApiResponse[response.PluginDetailResponse]
// @Router /api/v1/plugins/{key} [get]
func (c *PluginController) GetByKey(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("plugin key is required"))
		return
	}

	plugin, err := c.pluginService.GetByKey(ctx.Request.Context(), key)
	if err != nil {
		switch err {
		case service.ErrPluginNotFound:
			ctx.JSON(http.StatusNotFound, response.NewError[any]("plugin not found"))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to fetch plugin"))
		}
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(plugin))
}

// Install uploads and installs a new plugin
// @Summary Install a new plugin
// @Tags Plugins
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formance file true "Plugin file"
// @Param name formData string true "Plugin name"
// @Param version formData string true "Plugin version"
// @Param type formData string true "Plugin type"
// @Param description formData string false "Plugin description"
// @Param author formData string false "Plugin author"
// @Success 201 {object} response.ApiResponse[response.PluginResponse]
// @Router /api/v1/plugins/install [post]
func (c *PluginController) Install(ctx *gin.Context) {
	// Parse multipart form
	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("plugin file is required"))
		return
	}
	defer file.Close()

	req := &request.InstallPluginRequest{
		Name:        ctx.PostForm("name"),
		Description: ctx.PostForm("description"),
		Version:     ctx.PostForm("version"),
		Author:      ctx.PostForm("author"),
		Type:        ctx.PostForm("type"),
	}

	if req.Name == "" || req.Version == "" || req.Type == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("name, version, and type are required"))
		return
	}

	plugin, err := c.pluginService.Install(ctx.Request.Context(), req, file)
	if err != nil {
		switch err {
		case service.ErrPluginAlreadyExists:
			ctx.JSON(http.StatusConflict, response.NewError[any]("plugin already exists"))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to install plugin"))
		}
		return
	}

	ctx.JSON(http.StatusCreated, response.NewSuccess(plugin, "Plugin installed successfully"))
}

// Enable enables a plugin
// @Summary Enable a plugin
// @Tags Plugins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param key path string true "Plugin key"
// @Success 200 {object} response.ApiResponse[response.PluginResponse]
// @Router /api/v1/plugins/{key}/enable [post]
func (c *PluginController) Enable(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("plugin key is required"))
		return
	}

	plugin, err := c.pluginService.Enable(ctx.Request.Context(), key)
	if err != nil {
		switch err {
		case service.ErrPluginNotFound:
			ctx.JSON(http.StatusNotFound, response.NewError[any]("plugin not found"))
		case service.ErrPluginInvalidState:
			ctx.JSON(http.StatusBadRequest, response.NewError[any]("plugin cannot be enabled in current state"))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to enable plugin"))
		}
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess(plugin, "Plugin enabled successfully"))
}

// Disable disables a plugin
// @Summary Disable a plugin
// @Tags Plugins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param key path string true "Plugin key"
// @Success 200 {object} response.ApiResponse[response.PluginResponse]
// @Router /api/v1/plugins/{key}/disable [post]
func (c *PluginController) Disable(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("plugin key is required"))
		return
	}

	plugin, err := c.pluginService.Disable(ctx.Request.Context(), key)
	if err != nil {
		switch err {
		case service.ErrPluginNotFound:
			ctx.JSON(http.StatusNotFound, response.NewError[any]("plugin not found"))
		case service.ErrPluginInvalidState:
			ctx.JSON(http.StatusBadRequest, response.NewError[any]("plugin cannot be disabled in current state"))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to disable plugin"))
		}
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess(plugin, "Plugin disabled successfully"))
}

// Uninstall removes a plugin
// @Summary Uninstall a plugin
// @Tags Plugins
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param key path string true "Plugin key"
// @Success 200 {object} response.ApiResponse[any]
// @Router /api/v1/plugins/{key} [delete]
func (c *PluginController) Uninstall(ctx *gin.Context) {
	key := ctx.Param("key")
	if key == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("plugin key is required"))
		return
	}

	if err := c.pluginService.Uninstall(ctx.Request.Context(), key); err != nil {
		switch err {
		case service.ErrPluginNotFound:
			ctx.JSON(http.StatusNotFound, response.NewError[any]("plugin not found"))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to uninstall plugin"))
		}
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess[any](nil, "Plugin uninstalled successfully"))
}

// GetHealth returns the plugin system health status
// @Summary Get plugin system health
// @Tags Plugins
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponse[response.PluginHealthResponse]
// @Router /api/v1/plugins/health [get]
func (c *PluginController) GetHealth(ctx *gin.Context) {
	health, err := c.pluginService.GetHealth(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to get health status"))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(health))
}

// GetReadiness returns the Kubernetes readiness probe response
// @Summary Kubernetes readiness probe
// @Tags Plugins
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/plugins/health/ready [get]
func (c *PluginController) GetReadiness(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// GetLiveness returns the Kubernetes liveness probe response
// @Summary Kubernetes liveness probe
// @Tags Plugins
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/plugins/health/live [get]
func (c *PluginController) GetLiveness(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "alive"})
}
