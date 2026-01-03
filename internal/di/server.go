package di

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	httpctrl "github.com/jrjohn/arcana-cloud-go/internal/controller/http"
	grpcctrl "github.com/jrjohn/arcana-cloud-go/internal/controller/grpc"
	"github.com/jrjohn/arcana-cloud-go/internal/middleware"
)

// HTTPServerModule provides HTTP server dependencies
var HTTPServerModule = fx.Module("http_server",
	fx.Provide(provideGinEngine),
	fx.Provide(provideHTTPServer),
	fx.Invoke(registerHTTPRoutes),
	fx.Invoke(startHTTPServer),
)

// GRPCServerModule provides gRPC server dependencies
var GRPCServerModule = fx.Module("grpc_server",
	fx.Provide(provideGRPCServer),
	fx.Invoke(startGRPCServer),
)

func provideGinEngine(cfg *config.AppConfig, logger *zap.Logger) *gin.Engine {
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	return router
}

func provideHTTPServer(cfg *config.ServerConfig, router *gin.Engine) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}

func provideGRPCServer(cfg *config.GRPCConfig, logger *zap.Logger) (*grpcctrl.Server, error) {
	return grpcctrl.NewServer(cfg, logger)
}

// Controllers is a struct that holds all HTTP controllers for fx to inject
type Controllers struct {
	fx.In

	Auth   *httpctrl.AuthController
	User   *httpctrl.UserController
	Plugin *httpctrl.PluginController
	SSR    *httpctrl.SSRController
	Job    *httpctrl.JobController
}

func registerHTTPRoutes(router *gin.Engine, controllers Controllers) {
	// Health endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// API routes
	api := router.Group("/api/v1")

	controllers.Auth.RegisterRoutes(api)
	controllers.User.RegisterRoutes(api)
	controllers.Plugin.RegisterRoutes(api)
	controllers.SSR.RegisterRoutes(api)
	controllers.Job.RegisterRoutes(api)
}

func startHTTPServer(lc fx.Lifecycle, server *http.Server, cfg *config.DeploymentConfig, logger *zap.Logger) {
	// Always start HTTP server for health endpoints
	// In layered mode, non-controller layers only serve /health and /ready
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting HTTP server", zap.String("address", server.Addr))
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("HTTP server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping HTTP server")
			return server.Shutdown(ctx)
		},
	})
}

func startGRPCServer(lc fx.Lifecycle, server *grpcctrl.Server, cfg *config.DeploymentConfig, logger *zap.Logger) {
	if !cfg.IsServiceLayer() || !cfg.IsGRPC() {
		return
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := server.Start(); err != nil {
					logger.Error("gRPC server error", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			server.Stop()
			return nil
		},
	})
}
