package service

import (
	"context"
	"errors"
	"io"

	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
)

var (
	ErrPluginNotFound      = errors.New("plugin not found")
	ErrPluginAlreadyExists = errors.New("plugin already exists")
	ErrPluginInvalidState  = errors.New("invalid plugin state")
	ErrPluginLoadFailed    = errors.New("failed to load plugin")
)

// PluginService defines the interface for plugin operations
type PluginService interface {
	// Install installs a new plugin
	Install(ctx context.Context, req *request.InstallPluginRequest, file io.Reader) (*response.PluginResponse, error)

	// InstallFromPath installs a plugin from a file path
	InstallFromPath(ctx context.Context, req *request.InstallPluginRequest, filePath string) (*response.PluginResponse, error)

	// GetByKey retrieves a plugin by its key
	GetByKey(ctx context.Context, key string) (*response.PluginDetailResponse, error)

	// List retrieves all plugins with pagination
	List(ctx context.Context, page, size int) (*response.PagedResponse[response.PluginResponse], error)

	// Enable enables a plugin
	Enable(ctx context.Context, key string) (*response.PluginResponse, error)

	// Disable disables a plugin
	Disable(ctx context.Context, key string) (*response.PluginResponse, error)

	// Uninstall removes a plugin
	Uninstall(ctx context.Context, key string) error

	// GetHealth returns the plugin system health status
	GetHealth(ctx context.Context) (*response.PluginHealthResponse, error)
}
