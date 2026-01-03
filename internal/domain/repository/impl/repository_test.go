package impl

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

// MockUserDAO is a mock implementation of dao.UserDAO
type MockUserDAO struct {
	mock.Mock
}

func (m *MockUserDAO) Create(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserDAO) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserDAO) Update(ctx context.Context, user *entity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserDAO) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserDAO) FindAll(ctx context.Context, page, size int) ([]*entity.User, int64, error) {
	args := m.Called(ctx, page, size)
	return args.Get(0).([]*entity.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserDAO) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserDAO) ExistsBy(ctx context.Context, field string, value any) (bool, error) {
	args := m.Called(ctx, field, value)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserDAO) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserDAO) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserDAO) FindByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (*entity.User, error) {
	args := m.Called(ctx, usernameOrEmail)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.User), args.Error(1)
}

func (m *MockUserDAO) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserDAO) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

// MockRefreshTokenDAO is a mock implementation of dao.RefreshTokenDAO
type MockRefreshTokenDAO struct {
	mock.Mock
}

func (m *MockRefreshTokenDAO) Create(ctx context.Context, token *entity.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockRefreshTokenDAO) FindByID(ctx context.Context, id uint) (*entity.RefreshToken, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenDAO) Update(ctx context.Context, token *entity.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockRefreshTokenDAO) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRefreshTokenDAO) FindAll(ctx context.Context, page, size int) ([]*entity.RefreshToken, int64, error) {
	args := m.Called(ctx, page, size)
	return args.Get(0).([]*entity.RefreshToken), args.Get(1).(int64), args.Error(2)
}

func (m *MockRefreshTokenDAO) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRefreshTokenDAO) ExistsBy(ctx context.Context, field string, value any) (bool, error) {
	args := m.Called(ctx, field, value)
	return args.Bool(0), args.Error(1)
}

func (m *MockRefreshTokenDAO) FindByToken(ctx context.Context, token string) (*entity.RefreshToken, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenDAO) RevokeByToken(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockRefreshTokenDAO) RevokeAllByUserID(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRefreshTokenDAO) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockPluginDAO is a mock implementation of dao.PluginDAO
type MockPluginDAO struct {
	mock.Mock
}

func (m *MockPluginDAO) Create(ctx context.Context, plugin *entity.Plugin) error {
	args := m.Called(ctx, plugin)
	return args.Error(0)
}

func (m *MockPluginDAO) FindByID(ctx context.Context, id uint) (*entity.Plugin, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Plugin), args.Error(1)
}

func (m *MockPluginDAO) Update(ctx context.Context, plugin *entity.Plugin) error {
	args := m.Called(ctx, plugin)
	return args.Error(0)
}

func (m *MockPluginDAO) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPluginDAO) FindAll(ctx context.Context, page, size int) ([]*entity.Plugin, int64, error) {
	args := m.Called(ctx, page, size)
	return args.Get(0).([]*entity.Plugin), args.Get(1).(int64), args.Error(2)
}

func (m *MockPluginDAO) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPluginDAO) ExistsBy(ctx context.Context, field string, value any) (bool, error) {
	args := m.Called(ctx, field, value)
	return args.Bool(0), args.Error(1)
}

func (m *MockPluginDAO) FindByKey(ctx context.Context, key string) (*entity.Plugin, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Plugin), args.Error(1)
}

func (m *MockPluginDAO) DeleteByKey(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockPluginDAO) FindByState(ctx context.Context, state entity.PluginState) ([]*entity.Plugin, error) {
	args := m.Called(ctx, state)
	return args.Get(0).([]*entity.Plugin), args.Error(1)
}

func (m *MockPluginDAO) FindEnabled(ctx context.Context) ([]*entity.Plugin, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*entity.Plugin), args.Error(1)
}

func (m *MockPluginDAO) ExistsByKey(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockPluginDAO) UpdateState(ctx context.Context, id uint, state entity.PluginState) error {
	args := m.Called(ctx, id, state)
	return args.Error(0)
}

// MockPluginExtensionDAO is a mock implementation of dao.PluginExtensionDAO
type MockPluginExtensionDAO struct {
	mock.Mock
}

func (m *MockPluginExtensionDAO) Create(ctx context.Context, ext *entity.PluginExtension) error {
	args := m.Called(ctx, ext)
	return args.Error(0)
}

func (m *MockPluginExtensionDAO) FindByID(ctx context.Context, id uint) (*entity.PluginExtension, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.PluginExtension), args.Error(1)
}

func (m *MockPluginExtensionDAO) Update(ctx context.Context, ext *entity.PluginExtension) error {
	args := m.Called(ctx, ext)
	return args.Error(0)
}

func (m *MockPluginExtensionDAO) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPluginExtensionDAO) FindAll(ctx context.Context, page, size int) ([]*entity.PluginExtension, int64, error) {
	args := m.Called(ctx, page, size)
	return args.Get(0).([]*entity.PluginExtension), args.Get(1).(int64), args.Error(2)
}

func (m *MockPluginExtensionDAO) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPluginExtensionDAO) ExistsBy(ctx context.Context, field string, value any) (bool, error) {
	args := m.Called(ctx, field, value)
	return args.Bool(0), args.Error(1)
}

func (m *MockPluginExtensionDAO) FindByPluginID(ctx context.Context, pluginID uint) ([]*entity.PluginExtension, error) {
	args := m.Called(ctx, pluginID)
	return args.Get(0).([]*entity.PluginExtension), args.Error(1)
}

func (m *MockPluginExtensionDAO) DeleteByPluginID(ctx context.Context, pluginID uint) error {
	args := m.Called(ctx, pluginID)
	return args.Error(0)
}

// Tests for UserRepository
func TestUserRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		mockDAO := new(MockUserDAO)
		repo := NewUserRepository(mockDAO)

		user := &entity.User{Username: "test", Email: "test@example.com"}
		mockDAO.On("Create", ctx, user).Return(nil)

		err := repo.Create(ctx, user)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("GetByID", func(t *testing.T) {
		mockDAO := new(MockUserDAO)
		repo := NewUserRepository(mockDAO)

		expectedUser := &entity.User{ID: 1, Username: "test"}
		mockDAO.On("FindByID", ctx, uint(1)).Return(expectedUser, nil)

		user, err := repo.GetByID(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		mockDAO.AssertExpectations(t)
	})

	t.Run("GetByUsername", func(t *testing.T) {
		mockDAO := new(MockUserDAO)
		repo := NewUserRepository(mockDAO)

		expectedUser := &entity.User{ID: 1, Username: "test"}
		mockDAO.On("FindByUsername", ctx, "test").Return(expectedUser, nil)

		user, err := repo.GetByUsername(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		mockDAO.AssertExpectations(t)
	})

	t.Run("GetByEmail", func(t *testing.T) {
		mockDAO := new(MockUserDAO)
		repo := NewUserRepository(mockDAO)

		expectedUser := &entity.User{ID: 1, Email: "test@example.com"}
		mockDAO.On("FindByEmail", ctx, "test@example.com").Return(expectedUser, nil)

		user, err := repo.GetByEmail(ctx, "test@example.com")
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		mockDAO.AssertExpectations(t)
	})

	t.Run("GetByUsernameOrEmail", func(t *testing.T) {
		mockDAO := new(MockUserDAO)
		repo := NewUserRepository(mockDAO)

		expectedUser := &entity.User{ID: 1, Username: "test"}
		mockDAO.On("FindByUsernameOrEmail", ctx, "test").Return(expectedUser, nil)

		user, err := repo.GetByUsernameOrEmail(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		mockDAO.AssertExpectations(t)
	})

	t.Run("Update", func(t *testing.T) {
		mockDAO := new(MockUserDAO)
		repo := NewUserRepository(mockDAO)

		user := &entity.User{ID: 1, Username: "updated"}
		mockDAO.On("Update", ctx, user).Return(nil)

		err := repo.Update(ctx, user)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("Delete", func(t *testing.T) {
		mockDAO := new(MockUserDAO)
		repo := NewUserRepository(mockDAO)

		mockDAO.On("Delete", ctx, uint(1)).Return(nil)

		err := repo.Delete(ctx, 1)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("List", func(t *testing.T) {
		mockDAO := new(MockUserDAO)
		repo := NewUserRepository(mockDAO)

		expectedUsers := []*entity.User{{ID: 1}, {ID: 2}}
		mockDAO.On("FindAll", ctx, 1, 10).Return(expectedUsers, int64(2), nil)

		users, total, err := repo.List(ctx, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, expectedUsers, users)
		assert.Equal(t, int64(2), total)
		mockDAO.AssertExpectations(t)
	})

	t.Run("ExistsByUsername", func(t *testing.T) {
		mockDAO := new(MockUserDAO)
		repo := NewUserRepository(mockDAO)

		mockDAO.On("ExistsByUsername", ctx, "test").Return(true, nil)

		exists, err := repo.ExistsByUsername(ctx, "test")
		assert.NoError(t, err)
		assert.True(t, exists)
		mockDAO.AssertExpectations(t)
	})

	t.Run("ExistsByEmail", func(t *testing.T) {
		mockDAO := new(MockUserDAO)
		repo := NewUserRepository(mockDAO)

		mockDAO.On("ExistsByEmail", ctx, "test@example.com").Return(true, nil)

		exists, err := repo.ExistsByEmail(ctx, "test@example.com")
		assert.NoError(t, err)
		assert.True(t, exists)
		mockDAO.AssertExpectations(t)
	})
}

// Tests for RefreshTokenRepository
func TestRefreshTokenRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		mockDAO := new(MockRefreshTokenDAO)
		repo := NewRefreshTokenRepository(mockDAO)

		token := &entity.RefreshToken{Token: "test-token"}
		mockDAO.On("Create", ctx, token).Return(nil)

		err := repo.Create(ctx, token)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("GetByToken", func(t *testing.T) {
		mockDAO := new(MockRefreshTokenDAO)
		repo := NewRefreshTokenRepository(mockDAO)

		expectedToken := &entity.RefreshToken{ID: 1, Token: "test-token"}
		mockDAO.On("FindByToken", ctx, "test-token").Return(expectedToken, nil)

		token, err := repo.GetByToken(ctx, "test-token")
		assert.NoError(t, err)
		assert.Equal(t, expectedToken, token)
		mockDAO.AssertExpectations(t)
	})

	t.Run("RevokeByToken", func(t *testing.T) {
		mockDAO := new(MockRefreshTokenDAO)
		repo := NewRefreshTokenRepository(mockDAO)

		mockDAO.On("RevokeByToken", ctx, "test-token").Return(nil)

		err := repo.RevokeByToken(ctx, "test-token")
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("RevokeAllByUserID", func(t *testing.T) {
		mockDAO := new(MockRefreshTokenDAO)
		repo := NewRefreshTokenRepository(mockDAO)

		mockDAO.On("RevokeAllByUserID", ctx, uint(1)).Return(nil)

		err := repo.RevokeAllByUserID(ctx, 1)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("DeleteExpired", func(t *testing.T) {
		mockDAO := new(MockRefreshTokenDAO)
		repo := NewRefreshTokenRepository(mockDAO)

		mockDAO.On("DeleteExpired", ctx).Return(nil)

		err := repo.DeleteExpired(ctx)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})
}

// Tests for PluginRepository
func TestPluginRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		mockDAO := new(MockPluginDAO)
		repo := NewPluginRepository(mockDAO)

		plugin := &entity.Plugin{Key: "test-plugin", InstalledAt: time.Now()}
		mockDAO.On("Create", ctx, plugin).Return(nil)

		err := repo.Create(ctx, plugin)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("GetByID", func(t *testing.T) {
		mockDAO := new(MockPluginDAO)
		repo := NewPluginRepository(mockDAO)

		expectedPlugin := &entity.Plugin{ID: 1, Key: "test"}
		mockDAO.On("FindByID", ctx, uint(1)).Return(expectedPlugin, nil)

		plugin, err := repo.GetByID(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, expectedPlugin, plugin)
		mockDAO.AssertExpectations(t)
	})

	t.Run("GetByKey", func(t *testing.T) {
		mockDAO := new(MockPluginDAO)
		repo := NewPluginRepository(mockDAO)

		expectedPlugin := &entity.Plugin{ID: 1, Key: "test-plugin"}
		mockDAO.On("FindByKey", ctx, "test-plugin").Return(expectedPlugin, nil)

		plugin, err := repo.GetByKey(ctx, "test-plugin")
		assert.NoError(t, err)
		assert.Equal(t, expectedPlugin, plugin)
		mockDAO.AssertExpectations(t)
	})

	t.Run("Update", func(t *testing.T) {
		mockDAO := new(MockPluginDAO)
		repo := NewPluginRepository(mockDAO)

		plugin := &entity.Plugin{ID: 1, Key: "updated"}
		mockDAO.On("Update", ctx, plugin).Return(nil)

		err := repo.Update(ctx, plugin)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("Delete", func(t *testing.T) {
		mockDAO := new(MockPluginDAO)
		repo := NewPluginRepository(mockDAO)

		mockDAO.On("Delete", ctx, uint(1)).Return(nil)

		err := repo.Delete(ctx, 1)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("DeleteByKey", func(t *testing.T) {
		mockDAO := new(MockPluginDAO)
		repo := NewPluginRepository(mockDAO)

		mockDAO.On("DeleteByKey", ctx, "test-plugin").Return(nil)

		err := repo.DeleteByKey(ctx, "test-plugin")
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("List", func(t *testing.T) {
		mockDAO := new(MockPluginDAO)
		repo := NewPluginRepository(mockDAO)

		expectedPlugins := []*entity.Plugin{{ID: 1}, {ID: 2}}
		mockDAO.On("FindAll", ctx, 1, 10).Return(expectedPlugins, int64(2), nil)

		plugins, total, err := repo.List(ctx, 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, expectedPlugins, plugins)
		assert.Equal(t, int64(2), total)
		mockDAO.AssertExpectations(t)
	})

	t.Run("ListByState", func(t *testing.T) {
		mockDAO := new(MockPluginDAO)
		repo := NewPluginRepository(mockDAO)

		expectedPlugins := []*entity.Plugin{{ID: 1, State: entity.PluginStateEnabled}}
		mockDAO.On("FindByState", ctx, entity.PluginStateEnabled).Return(expectedPlugins, nil)

		plugins, err := repo.ListByState(ctx, entity.PluginStateEnabled)
		assert.NoError(t, err)
		assert.Equal(t, expectedPlugins, plugins)
		mockDAO.AssertExpectations(t)
	})

	t.Run("ListEnabled", func(t *testing.T) {
		mockDAO := new(MockPluginDAO)
		repo := NewPluginRepository(mockDAO)

		expectedPlugins := []*entity.Plugin{{ID: 1, State: entity.PluginStateEnabled}}
		mockDAO.On("FindEnabled", ctx).Return(expectedPlugins, nil)

		plugins, err := repo.ListEnabled(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedPlugins, plugins)
		mockDAO.AssertExpectations(t)
	})

	t.Run("ExistsByKey", func(t *testing.T) {
		mockDAO := new(MockPluginDAO)
		repo := NewPluginRepository(mockDAO)

		mockDAO.On("ExistsByKey", ctx, "test-plugin").Return(true, nil)

		exists, err := repo.ExistsByKey(ctx, "test-plugin")
		assert.NoError(t, err)
		assert.True(t, exists)
		mockDAO.AssertExpectations(t)
	})

	t.Run("UpdateState", func(t *testing.T) {
		mockDAO := new(MockPluginDAO)
		repo := NewPluginRepository(mockDAO)

		mockDAO.On("UpdateState", ctx, uint(1), entity.PluginStateEnabled).Return(nil)

		err := repo.UpdateState(ctx, 1, entity.PluginStateEnabled)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})
}

// Tests for PluginExtensionRepository
func TestPluginExtensionRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		mockDAO := new(MockPluginExtensionDAO)
		repo := NewPluginExtensionRepository(mockDAO)

		ext := &entity.PluginExtension{Name: "test-ext"}
		mockDAO.On("Create", ctx, ext).Return(nil)

		err := repo.Create(ctx, ext)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})

	t.Run("GetByPluginID", func(t *testing.T) {
		mockDAO := new(MockPluginExtensionDAO)
		repo := NewPluginExtensionRepository(mockDAO)

		expectedExts := []*entity.PluginExtension{{ID: 1}, {ID: 2}}
		mockDAO.On("FindByPluginID", ctx, uint(1)).Return(expectedExts, nil)

		exts, err := repo.GetByPluginID(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, expectedExts, exts)
		mockDAO.AssertExpectations(t)
	})

	t.Run("DeleteByPluginID", func(t *testing.T) {
		mockDAO := new(MockPluginExtensionDAO)
		repo := NewPluginExtensionRepository(mockDAO)

		mockDAO.On("DeleteByPluginID", ctx, uint(1)).Return(nil)

		err := repo.DeleteByPluginID(ctx, 1)
		assert.NoError(t, err)
		mockDAO.AssertExpectations(t)
	})
}
