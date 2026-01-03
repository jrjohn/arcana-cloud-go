package mocks

import (
	"context"
	"sync"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/repository"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mu     sync.RWMutex
	users  map[uint]*entity.User
	nextID uint

	// Error injection
	CreateErr              error
	GetByIDErr             error
	GetByUsernameErr       error
	GetByEmailErr          error
	GetByUsernameOrEmailErr error
	UpdateErr              error
	DeleteErr              error
	ListErr                error
	ExistsByUsernameErr    error
	ExistsByEmailErr       error
}

var _ repository.UserRepository = (*MockUserRepository)(nil)

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users:  make(map[uint]*entity.User),
		nextID: 1,
	}
}

func (r *MockUserRepository) Create(ctx context.Context, user *entity.User) error {
	if r.CreateErr != nil {
		return r.CreateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	user.ID = r.nextID
	r.nextID++
	r.users[user.ID] = user
	return nil
}

func (r *MockUserRepository) GetByID(ctx context.Context, id uint) (*entity.User, error) {
	if r.GetByIDErr != nil {
		return nil, r.GetByIDErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if user, ok := r.users[id]; ok {
		return user, nil
	}
	return nil, nil
}

func (r *MockUserRepository) GetByUsername(ctx context.Context, username string) (*entity.User, error) {
	if r.GetByUsernameErr != nil {
		return nil, r.GetByUsernameErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, nil
}

func (r *MockUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	if r.GetByEmailErr != nil {
		return nil, r.GetByEmailErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, nil
}

func (r *MockUserRepository) GetByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*entity.User, error) {
	if r.GetByUsernameOrEmailErr != nil {
		return nil, r.GetByUsernameOrEmailErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.Username == usernameOrEmail || user.Email == usernameOrEmail {
			return user, nil
		}
	}
	return nil, nil
}

func (r *MockUserRepository) Update(ctx context.Context, user *entity.User) error {
	if r.UpdateErr != nil {
		return r.UpdateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[user.ID]; ok {
		r.users[user.ID] = user
		return nil
	}
	return nil
}

func (r *MockUserRepository) Delete(ctx context.Context, id uint) error {
	if r.DeleteErr != nil {
		return r.DeleteErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.users, id)
	return nil
}

func (r *MockUserRepository) List(ctx context.Context, page, size int) ([]*entity.User, int64, error) {
	if r.ListErr != nil {
		return nil, 0, r.ListErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*entity.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}

	// Simple pagination
	start := (page - 1) * size
	if start >= len(users) {
		return []*entity.User{}, int64(len(r.users)), nil
	}
	end := start + size
	if end > len(users) {
		end = len(users)
	}

	return users[start:end], int64(len(r.users)), nil
}

func (r *MockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if r.ExistsByUsernameErr != nil {
		return false, r.ExistsByUsernameErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.Username == username {
			return true, nil
		}
	}
	return false, nil
}

func (r *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if r.ExistsByEmailErr != nil {
		return false, r.ExistsByEmailErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.Email == email {
			return true, nil
		}
	}
	return false, nil
}

// AddUser adds a user directly (for test setup)
func (r *MockUserRepository) AddUser(user *entity.User) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if user.ID == 0 {
		user.ID = r.nextID
		r.nextID++
	}
	r.users[user.ID] = user
}

// MockRefreshTokenRepository is a mock implementation of RefreshTokenRepository
type MockRefreshTokenRepository struct {
	mu     sync.RWMutex
	tokens map[uint]*entity.RefreshToken
	nextID uint

	// Error injection
	CreateErr            error
	GetByTokenErr        error
	RevokeByTokenErr     error
	RevokeAllByUserIDErr error
	DeleteExpiredErr     error
}

var _ repository.RefreshTokenRepository = (*MockRefreshTokenRepository)(nil)

func NewMockRefreshTokenRepository() *MockRefreshTokenRepository {
	return &MockRefreshTokenRepository{
		tokens: make(map[uint]*entity.RefreshToken),
		nextID: 1,
	}
}

func (r *MockRefreshTokenRepository) Create(ctx context.Context, token *entity.RefreshToken) error {
	if r.CreateErr != nil {
		return r.CreateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	token.ID = r.nextID
	r.nextID++
	r.tokens[token.ID] = token
	return nil
}

func (r *MockRefreshTokenRepository) GetByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	if r.GetByTokenErr != nil {
		return nil, r.GetByTokenErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, rt := range r.tokens {
		if rt.Token == token && !rt.Revoked {
			return rt, nil
		}
	}
	return nil, nil
}

func (r *MockRefreshTokenRepository) RevokeByToken(ctx context.Context, token string) error {
	if r.RevokeByTokenErr != nil {
		return r.RevokeByTokenErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rt := range r.tokens {
		if rt.Token == token {
			rt.Revoked = true
			return nil
		}
	}
	return nil
}

func (r *MockRefreshTokenRepository) RevokeAllByUserID(ctx context.Context, userID uint) error {
	if r.RevokeAllByUserIDErr != nil {
		return r.RevokeAllByUserIDErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rt := range r.tokens {
		if rt.UserID == userID {
			rt.Revoked = true
		}
	}
	return nil
}

func (r *MockRefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	if r.DeleteExpiredErr != nil {
		return r.DeleteExpiredErr
	}
	return nil
}

// AddToken adds a token directly (for test setup)
func (r *MockRefreshTokenRepository) AddToken(token *entity.RefreshToken) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if token.ID == 0 {
		token.ID = r.nextID
		r.nextID++
	}
	r.tokens[token.ID] = token
}

// MockPluginRepository is a mock implementation of PluginRepository
type MockPluginRepository struct {
	mu      sync.RWMutex
	plugins map[uint]*entity.Plugin
	nextID  uint

	// Error injection
	CreateErr       error
	GetByIDErr      error
	GetByKeyErr     error
	UpdateErr       error
	DeleteErr       error
	DeleteByKeyErr  error
	ListErr         error
	ListByStateErr  error
	ListEnabledErr  error
	ExistsByKeyErr  error
	UpdateStateErr  error
}

var _ repository.PluginRepository = (*MockPluginRepository)(nil)

func NewMockPluginRepository() *MockPluginRepository {
	return &MockPluginRepository{
		plugins: make(map[uint]*entity.Plugin),
		nextID:  1,
	}
}

func (r *MockPluginRepository) Create(ctx context.Context, plugin *entity.Plugin) error {
	if r.CreateErr != nil {
		return r.CreateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	plugin.ID = r.nextID
	r.nextID++
	r.plugins[plugin.ID] = plugin
	return nil
}

func (r *MockPluginRepository) GetByID(ctx context.Context, id uint) (*entity.Plugin, error) {
	if r.GetByIDErr != nil {
		return nil, r.GetByIDErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if plugin, ok := r.plugins[id]; ok {
		return plugin, nil
	}
	return nil, nil
}

func (r *MockPluginRepository) GetByKey(ctx context.Context, key string) (*entity.Plugin, error) {
	if r.GetByKeyErr != nil {
		return nil, r.GetByKeyErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, plugin := range r.plugins {
		if plugin.Key == key {
			return plugin, nil
		}
	}
	return nil, nil
}

func (r *MockPluginRepository) Update(ctx context.Context, plugin *entity.Plugin) error {
	if r.UpdateErr != nil {
		return r.UpdateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.plugins[plugin.ID]; ok {
		r.plugins[plugin.ID] = plugin
	}
	return nil
}

func (r *MockPluginRepository) Delete(ctx context.Context, id uint) error {
	if r.DeleteErr != nil {
		return r.DeleteErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.plugins, id)
	return nil
}

func (r *MockPluginRepository) DeleteByKey(ctx context.Context, key string) error {
	if r.DeleteByKeyErr != nil {
		return r.DeleteByKeyErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, plugin := range r.plugins {
		if plugin.Key == key {
			delete(r.plugins, id)
			return nil
		}
	}
	return nil
}

func (r *MockPluginRepository) List(ctx context.Context, page, size int) ([]*entity.Plugin, int64, error) {
	if r.ListErr != nil {
		return nil, 0, r.ListErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]*entity.Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}

	start := (page - 1) * size
	if start >= len(plugins) {
		return []*entity.Plugin{}, int64(len(r.plugins)), nil
	}
	end := start + size
	if end > len(plugins) {
		end = len(plugins)
	}

	return plugins[start:end], int64(len(r.plugins)), nil
}

func (r *MockPluginRepository) ListByState(ctx context.Context, state entity.PluginState) ([]*entity.Plugin, error) {
	if r.ListByStateErr != nil {
		return nil, r.ListByStateErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*entity.Plugin, 0)
	for _, plugin := range r.plugins {
		if plugin.State == state {
			result = append(result, plugin)
		}
	}
	return result, nil
}

func (r *MockPluginRepository) ListEnabled(ctx context.Context) ([]*entity.Plugin, error) {
	if r.ListEnabledErr != nil {
		return nil, r.ListEnabledErr
	}
	return r.ListByState(ctx, entity.PluginStateEnabled)
}

func (r *MockPluginRepository) ExistsByKey(ctx context.Context, key string) (bool, error) {
	if r.ExistsByKeyErr != nil {
		return false, r.ExistsByKeyErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, plugin := range r.plugins {
		if plugin.Key == key {
			return true, nil
		}
	}
	return false, nil
}

func (r *MockPluginRepository) UpdateState(ctx context.Context, id uint, state entity.PluginState) error {
	if r.UpdateStateErr != nil {
		return r.UpdateStateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if plugin, ok := r.plugins[id]; ok {
		plugin.State = state
	}
	return nil
}

// AddPlugin adds a plugin directly (for test setup)
func (r *MockPluginRepository) AddPlugin(plugin *entity.Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if plugin.ID == 0 {
		plugin.ID = r.nextID
		r.nextID++
	}
	r.plugins[plugin.ID] = plugin
}

// MockPluginExtensionRepository is a mock implementation of PluginExtensionRepository
type MockPluginExtensionRepository struct {
	mu         sync.RWMutex
	extensions map[uint]*entity.PluginExtension
	nextID     uint

	// Error injection
	CreateErr           error
	GetByPluginIDErr    error
	DeleteByPluginIDErr error
}

var _ repository.PluginExtensionRepository = (*MockPluginExtensionRepository)(nil)

func NewMockPluginExtensionRepository() *MockPluginExtensionRepository {
	return &MockPluginExtensionRepository{
		extensions: make(map[uint]*entity.PluginExtension),
		nextID:     1,
	}
}

func (r *MockPluginExtensionRepository) Create(ctx context.Context, extension *entity.PluginExtension) error {
	if r.CreateErr != nil {
		return r.CreateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	extension.ID = r.nextID
	r.nextID++
	r.extensions[extension.ID] = extension
	return nil
}

func (r *MockPluginExtensionRepository) GetByPluginID(ctx context.Context, pluginID uint) ([]*entity.PluginExtension, error) {
	if r.GetByPluginIDErr != nil {
		return nil, r.GetByPluginIDErr
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*entity.PluginExtension, 0)
	for _, ext := range r.extensions {
		if ext.PluginID == pluginID {
			result = append(result, ext)
		}
	}
	return result, nil
}

func (r *MockPluginExtensionRepository) DeleteByPluginID(ctx context.Context, pluginID uint) error {
	if r.DeleteByPluginIDErr != nil {
		return r.DeleteByPluginIDErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, ext := range r.extensions {
		if ext.PluginID == pluginID {
			delete(r.extensions, id)
		}
	}
	return nil
}

// AddExtension adds an extension directly (for test setup)
func (r *MockPluginExtensionRepository) AddExtension(ext *entity.PluginExtension) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ext.ID == 0 {
		ext.ID = r.nextID
		r.nextID++
	}
	r.extensions[ext.ID] = ext
}
