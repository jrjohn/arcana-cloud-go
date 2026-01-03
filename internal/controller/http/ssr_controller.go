package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
	"github.com/jrjohn/arcana-cloud-go/internal/middleware"
)

// SSRController handles server-side rendering endpoints
type SSRController struct {
	ssrService     service.SSRService
	authMiddleware *middleware.AuthMiddleware
}

// NewSSRController creates a new SSRController instance
func NewSSRController(ssrService service.SSRService, authMiddleware *middleware.AuthMiddleware) *SSRController {
	return &SSRController{
		ssrService:     ssrService,
		authMiddleware: authMiddleware,
	}
}

// RegisterRoutes registers the SSR routes
func (c *SSRController) RegisterRoutes(router *gin.RouterGroup) {
	ssr := router.Group("/ssr")
	{
		// Public status endpoint
		ssr.GET("/status", c.GetStatus)

		// Protected endpoints
		protected := ssr.Group("")
		protected.Use(c.authMiddleware.Authenticate())
		{
			protected.POST("/react/:component", c.RenderReact)
			protected.POST("/angular/:component", c.RenderAngular)
			protected.POST("/cache/clear", c.authMiddleware.RequireAdmin(), c.ClearCache)
		}
	}
}

// RenderReact renders a React component server-side
// @Summary Render React component
// @Tags SSR
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param component path string true "Component name"
// @Param request body request.RenderRequest false "Render request"
// @Success 200 {object} response.ApiResponse[response.RenderResponse]
// @Router /api/v1/ssr/react/{component} [post]
func (c *SSRController) RenderReact(ctx *gin.Context) {
	component := ctx.Param("component")
	if component == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("component name is required"))
		return
	}

	var req request.RenderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req = request.RenderRequest{}
	}
	req.Component = component

	result, err := c.ssrService.RenderReact(ctx.Request.Context(), component, req.Props)
	if err != nil {
		switch err {
		case service.ErrSSREngineNotReady:
			ctx.JSON(http.StatusServiceUnavailable, response.NewError[any]("SSR engine is not ready"))
		case service.ErrComponentNotFound:
			ctx.JSON(http.StatusNotFound, response.NewError[any]("component not found"))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any]("SSR rendering failed"))
		}
		return
	}

	resp := response.RenderResponse{
		HTML:       result.HTML,
		CSS:        result.CSS,
		Scripts:    result.Scripts,
		State:      result.State,
		RenderTime: result.RenderTime.Milliseconds(),
		Cached:     result.Cached,
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(resp))
}

// RenderAngular renders an Angular component server-side
// @Summary Render Angular component
// @Tags SSR
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param component path string true "Component name"
// @Param request body request.RenderRequest false "Render request"
// @Success 200 {object} response.ApiResponse[response.RenderResponse]
// @Router /api/v1/ssr/angular/{component} [post]
func (c *SSRController) RenderAngular(ctx *gin.Context) {
	component := ctx.Param("component")
	if component == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError[any]("component name is required"))
		return
	}

	var req request.RenderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		req = request.RenderRequest{}
	}
	req.Component = component

	result, err := c.ssrService.RenderAngular(ctx.Request.Context(), component, req.Props)
	if err != nil {
		switch err {
		case service.ErrSSREngineNotReady:
			ctx.JSON(http.StatusServiceUnavailable, response.NewError[any]("SSR engine is not ready"))
		case service.ErrComponentNotFound:
			ctx.JSON(http.StatusNotFound, response.NewError[any]("component not found"))
		default:
			ctx.JSON(http.StatusInternalServerError, response.NewError[any]("SSR rendering failed"))
		}
		return
	}

	resp := response.RenderResponse{
		HTML:       result.HTML,
		CSS:        result.CSS,
		Scripts:    result.Scripts,
		State:      result.State,
		RenderTime: result.RenderTime.Milliseconds(),
		Cached:     result.Cached,
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(resp))
}

// GetStatus returns the SSR engine status
// @Summary Get SSR engine status
// @Tags SSR
// @Accept json
// @Produce json
// @Success 200 {object} response.ApiResponse[response.SSRStatusResponse]
// @Router /api/v1/ssr/status [get]
func (c *SSRController) GetStatus(ctx *gin.Context) {
	status, err := c.ssrService.GetStatus(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to get SSR status"))
		return
	}

	resp := response.SSRStatusResponse{
		Status:       status.Status,
		ReactReady:   status.ReactReady,
		AngularReady: status.AngularReady,
		CacheEnabled: status.CacheEnabled,
		CacheSize:    status.CacheSize,
		Stats:        status.Stats,
	}

	ctx.JSON(http.StatusOK, response.NewSuccessWithData(resp))
}

// ClearCache clears the SSR render cache
// @Summary Clear SSR cache
// @Tags SSR
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.ApiResponse[any]
// @Router /api/v1/ssr/cache/clear [post]
func (c *SSRController) ClearCache(ctx *gin.Context) {
	if err := c.ssrService.ClearCache(ctx.Request.Context()); err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError[any]("failed to clear cache"))
		return
	}

	ctx.JSON(http.StatusOK, response.NewSuccess[any](nil, "Cache cleared successfully"))
}
