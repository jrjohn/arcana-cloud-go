package impl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
)

// pluginService implements service.PluginService
type pluginService struct {
	pluginRepo    repository.PluginRepository
	extensionRepo repository.PluginExtensionRepository
	pluginsDir    string
}

// NewPluginService creates a new PluginService instance
func NewPluginService(
	pluginRepo repository.PluginRepository,
	extensionRepo repository.PluginExtensionRepository,
	pluginsDir string,
) service.PluginService {
	return &pluginService{
		pluginRepo:    pluginRepo,
		extensionRepo: extensionRepo,
		pluginsDir:    pluginsDir,
	}
}

func (s *pluginService) Install(ctx context.Context, req *request.InstallPluginRequest, file io.Reader) (*response.PluginResponse, error) {
	// Generate plugin key
	key := s.generatePluginKey(req.Name)

	// Check if plugin already exists
	exists, err := s.pluginRepo.ExistsByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, service.ErrPluginAlreadyExists
	}

	// Create plugins directory if it doesn't exist
	if err := os.MkdirAll(s.pluginsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// Save the plugin file
	pluginPath := filepath.Join(s.pluginsDir, key+".so")
	outFile, err := os.Create(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin file: %w", err)
	}
	defer outFile.Close()

	// Calculate checksum while copying
	hash := sha256.New()
	writer := io.MultiWriter(outFile, hash)
	if _, err := io.Copy(writer, file); err != nil {
		os.Remove(pluginPath)
		return nil, fmt.Errorf("failed to save plugin file: %w", err)
	}

	checksum := hex.EncodeToString(hash.Sum(nil))

	// Marshal config to JSON
	configJSON := ""
	if req.Config != nil {
		configBytes, err := json.Marshal(req.Config)
		if err != nil {
			os.Remove(pluginPath)
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		configJSON = string(configBytes)
	}

	// Create plugin entity
	plugin := &entity.Plugin{
		Key:         key,
		Name:        req.Name,
		Description: req.Description,
		Version:     req.Version,
		Author:      req.Author,
		Type:        entity.PluginType(req.Type),
		State:       entity.PluginStateInstalled,
		Config:      configJSON,
		Checksum:    checksum,
		Path:        pluginPath,
		InstalledAt: time.Now(),
	}

	if err := s.pluginRepo.Create(ctx, plugin); err != nil {
		os.Remove(pluginPath)
		return nil, err
	}

	return s.toPluginResponse(plugin), nil
}

func (s *pluginService) InstallFromPath(ctx context.Context, req *request.InstallPluginRequest, filePath string) (*response.PluginResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin file: %w", err)
	}
	defer file.Close()

	return s.Install(ctx, req, file)
}

func (s *pluginService) GetByKey(ctx context.Context, key string) (*response.PluginDetailResponse, error) {
	plugin, err := s.pluginRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if plugin == nil {
		return nil, service.ErrPluginNotFound
	}

	// Get extensions
	extensions, err := s.extensionRepo.GetByPluginID(ctx, plugin.ID)
	if err != nil {
		return nil, err
	}

	// Parse config
	var config map[string]any
	if plugin.Config != "" {
		if err := json.Unmarshal([]byte(plugin.Config), &config); err != nil {
			config = nil
		}
	}

	resp := &response.PluginDetailResponse{
		PluginResponse: *s.toPluginResponse(plugin),
		Config:         config,
	}

	for _, ext := range extensions {
		resp.Extensions = append(resp.Extensions, response.PluginExtensionResponse{
			ID:      ext.ID,
			Name:    ext.Name,
			Type:    string(ext.Type),
			Path:    ext.Path,
			Handler: ext.Handler,
		})
	}

	return resp, nil
}

func (s *pluginService) List(ctx context.Context, page, size int) (*response.PagedResponse[response.PluginResponse], error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	plugins, total, err := s.pluginRepo.List(ctx, page, size)
	if err != nil {
		return nil, err
	}

	items := make([]response.PluginResponse, len(plugins))
	for i, plugin := range plugins {
		items[i] = *s.toPluginResponse(plugin)
	}

	result := response.NewPagedResponse(items, page, size, total)
	return &result, nil
}

func (s *pluginService) Enable(ctx context.Context, key string) (*response.PluginResponse, error) {
	plugin, err := s.pluginRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if plugin == nil {
		return nil, service.ErrPluginNotFound
	}

	if plugin.State == entity.PluginStateEnabled {
		return s.toPluginResponse(plugin), nil
	}

	if plugin.State != entity.PluginStateInstalled && plugin.State != entity.PluginStateDisabled {
		return nil, service.ErrPluginInvalidState
	}

	if err := s.pluginRepo.UpdateState(ctx, plugin.ID, entity.PluginStateEnabled); err != nil {
		return nil, err
	}

	plugin.State = entity.PluginStateEnabled
	now := time.Now()
	plugin.EnabledAt = &now

	return s.toPluginResponse(plugin), nil
}

func (s *pluginService) Disable(ctx context.Context, key string) (*response.PluginResponse, error) {
	plugin, err := s.pluginRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	if plugin == nil {
		return nil, service.ErrPluginNotFound
	}

	if plugin.State == entity.PluginStateDisabled {
		return s.toPluginResponse(plugin), nil
	}

	if plugin.State != entity.PluginStateEnabled {
		return nil, service.ErrPluginInvalidState
	}

	if err := s.pluginRepo.UpdateState(ctx, plugin.ID, entity.PluginStateDisabled); err != nil {
		return nil, err
	}

	plugin.State = entity.PluginStateDisabled
	return s.toPluginResponse(plugin), nil
}

func (s *pluginService) Uninstall(ctx context.Context, key string) error {
	plugin, err := s.pluginRepo.GetByKey(ctx, key)
	if err != nil {
		return err
	}
	if plugin == nil {
		return service.ErrPluginNotFound
	}

	// Delete extensions
	if err := s.extensionRepo.DeleteByPluginID(ctx, plugin.ID); err != nil {
		return err
	}

	// Delete plugin file
	if plugin.Path != "" {
		os.Remove(plugin.Path)
	}

	// Delete plugin record
	return s.pluginRepo.DeleteByKey(ctx, key)
}

func (s *pluginService) GetHealth(ctx context.Context) (*response.PluginHealthResponse, error) {
	enabled, err := s.pluginRepo.ListByState(ctx, entity.PluginStateEnabled)
	if err != nil {
		return nil, err
	}

	disabled, err := s.pluginRepo.ListByState(ctx, entity.PluginStateDisabled)
	if err != nil {
		return nil, err
	}

	errPlugins, err := s.pluginRepo.ListByState(ctx, entity.PluginStateError)
	if err != nil {
		return nil, err
	}

	installed, err := s.pluginRepo.ListByState(ctx, entity.PluginStateInstalled)
	if err != nil {
		return nil, err
	}

	total := len(enabled) + len(disabled) + len(errPlugins) + len(installed)

	status := "healthy"
	if len(errPlugins) > 0 {
		status = "degraded"
	}

	return &response.PluginHealthResponse{
		Status:          status,
		TotalPlugins:    total,
		EnabledPlugins:  len(enabled),
		DisabledPlugins: len(disabled) + len(installed),
		ErrorPlugins:    len(errPlugins),
	}, nil
}

func (s *pluginService) generatePluginKey(name string) string {
	// Generate a key from name + UUID suffix
	sanitized := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	suffix := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s", sanitized, suffix)
}

func (s *pluginService) toPluginResponse(plugin *entity.Plugin) *response.PluginResponse {
	return &response.PluginResponse{
		ID:          plugin.ID,
		Key:         plugin.Key,
		Name:        plugin.Name,
		Description: plugin.Description,
		Version:     plugin.Version,
		Author:      plugin.Author,
		Type:        string(plugin.Type),
		State:       string(plugin.State),
		InstalledAt: plugin.InstalledAt,
		EnabledAt:   plugin.EnabledAt,
	}
}
