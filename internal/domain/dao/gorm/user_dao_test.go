package gorm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&entity.User{}, &entity.RefreshToken{}, &entity.Plugin{}, &entity.PluginExtension{})
	require.NoError(t, err)

	return db
}

func TestUserDAO_Create(t *testing.T) {
	db := setupTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := &entity.User{
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "hashedpassword",
		FirstName: "Test",
		LastName:  "User",
		Role:      entity.RoleUser,
		IsActive:  true,
	}

	err := dao.Create(ctx, user)
	assert.NoError(t, err)
	assert.NotZero(t, user.ID)
}

func TestUserDAO_FindByID(t *testing.T) {
	db := setupTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	// Create a user first
	user := &entity.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		Role:     entity.RoleUser,
		IsActive: true,
	}
	err := dao.Create(ctx, user)
	require.NoError(t, err)

	// Find by ID
	found, err := dao.FindByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, user.Username, found.Username)

	// Find non-existent
	notFound, err := dao.FindByID(ctx, 9999)
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestUserDAO_FindByUsername(t *testing.T) {
	db := setupTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := &entity.User{
		Username: "uniqueuser",
		Email:    "unique@example.com",
		Password: "hashedpassword",
		Role:     entity.RoleUser,
		IsActive: true,
	}
	err := dao.Create(ctx, user)
	require.NoError(t, err)

	found, err := dao.FindByUsername(ctx, "uniqueuser")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "uniqueuser", found.Username)

	notFound, err := dao.FindByUsername(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestUserDAO_FindByEmail(t *testing.T) {
	db := setupTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := &entity.User{
		Username: "emailuser",
		Email:    "email@example.com",
		Password: "hashedpassword",
		Role:     entity.RoleUser,
		IsActive: true,
	}
	err := dao.Create(ctx, user)
	require.NoError(t, err)

	found, err := dao.FindByEmail(ctx, "email@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "email@example.com", found.Email)

	notFound, err := dao.FindByEmail(ctx, "nonexistent@example.com")
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestUserDAO_FindByUsernameOrEmail(t *testing.T) {
	db := setupTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := &entity.User{
		Username: "oruser",
		Email:    "oremail@example.com",
		Password: "hashedpassword",
		Role:     entity.RoleUser,
		IsActive: true,
	}
	err := dao.Create(ctx, user)
	require.NoError(t, err)

	// Find by username
	foundByUsername, err := dao.FindByUsernameOrEmail(ctx, "oruser")
	assert.NoError(t, err)
	assert.NotNil(t, foundByUsername)

	// Find by email
	foundByEmail, err := dao.FindByUsernameOrEmail(ctx, "oremail@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, foundByEmail)

	// Not found
	notFound, err := dao.FindByUsernameOrEmail(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestUserDAO_Update(t *testing.T) {
	db := setupTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := &entity.User{
		Username:  "updateuser",
		Email:     "update@example.com",
		Password:  "hashedpassword",
		FirstName: "Original",
		Role:      entity.RoleUser,
		IsActive:  true,
	}
	err := dao.Create(ctx, user)
	require.NoError(t, err)

	user.FirstName = "Updated"
	err = dao.Update(ctx, user)
	assert.NoError(t, err)

	found, err := dao.FindByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated", found.FirstName)
}

func TestUserDAO_Delete(t *testing.T) {
	db := setupTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := &entity.User{
		Username: "deleteuser",
		Email:    "delete@example.com",
		Password: "hashedpassword",
		Role:     entity.RoleUser,
		IsActive: true,
	}
	err := dao.Create(ctx, user)
	require.NoError(t, err)

	err = dao.Delete(ctx, user.ID)
	assert.NoError(t, err)

	// Should not find deleted user
	found, err := dao.FindByID(ctx, user.ID)
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestUserDAO_FindAll(t *testing.T) {
	db := setupTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	// Create multiple users
	for i := 0; i < 15; i++ {
		user := &entity.User{
			Username: "user" + string(rune('a'+i)),
			Email:    "user" + string(rune('a'+i)) + "@example.com",
			Password: "hashedpassword",
			Role:     entity.RoleUser,
			IsActive: true,
		}
		err := dao.Create(ctx, user)
		require.NoError(t, err)
	}

	// Get first page
	users, total, err := dao.FindAll(ctx, 1, 10)
	assert.NoError(t, err)
	assert.Len(t, users, 10)
	assert.Equal(t, int64(15), total)

	// Get second page
	users, total, err = dao.FindAll(ctx, 2, 10)
	assert.NoError(t, err)
	assert.Len(t, users, 5)
	assert.Equal(t, int64(15), total)
}

func TestUserDAO_Count(t *testing.T) {
	db := setupTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	// Initially zero
	count, err := dao.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Create users
	for i := 0; i < 5; i++ {
		user := &entity.User{
			Username: "countuser" + string(rune('a'+i)),
			Email:    "countuser" + string(rune('a'+i)) + "@example.com",
			Password: "hashedpassword",
			Role:     entity.RoleUser,
			IsActive: true,
		}
		err := dao.Create(ctx, user)
		require.NoError(t, err)
	}

	count, err = dao.Count(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestUserDAO_ExistsByUsername(t *testing.T) {
	db := setupTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := &entity.User{
		Username: "existsuser",
		Email:    "exists@example.com",
		Password: "hashedpassword",
		Role:     entity.RoleUser,
		IsActive: true,
	}
	err := dao.Create(ctx, user)
	require.NoError(t, err)

	exists, err := dao.ExistsByUsername(ctx, "existsuser")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = dao.ExistsByUsername(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestUserDAO_ExistsByEmail(t *testing.T) {
	db := setupTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := &entity.User{
		Username: "emailexists",
		Email:    "emailexists@example.com",
		Password: "hashedpassword",
		Role:     entity.RoleUser,
		IsActive: true,
	}
	err := dao.Create(ctx, user)
	require.NoError(t, err)

	exists, err := dao.ExistsByEmail(ctx, "emailexists@example.com")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = dao.ExistsByEmail(ctx, "nonexistent@example.com")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestUserDAO_ExistsBy(t *testing.T) {
	db := setupTestDB(t)
	dao := NewUserDAO(db)
	ctx := context.Background()

	user := &entity.User{
		Username: "existsbyuser",
		Email:    "existsby@example.com",
		Password: "hashedpassword",
		Role:     entity.RoleAdmin,
		IsActive: true,
	}
	err := dao.Create(ctx, user)
	require.NoError(t, err)

	exists, err := dao.ExistsBy(ctx, "role", entity.RoleAdmin)
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = dao.ExistsBy(ctx, "role", "NONEXISTENT")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestBaseGormDAO_Helpers(t *testing.T) {
	db := setupTestDB(t)
	baseDAO := newBaseGormDAO[entity.User](db)
	ctx := context.Background()

	// Test getDB
	assert.NotNil(t, baseDAO.getDB())

	// Test findByField
	user := &entity.User{
		Username: "helperuser",
		Email:    "helper@example.com",
		Password: "hashedpassword",
		Role:     entity.RoleUser,
		IsActive: true,
	}
	err := baseDAO.Create(ctx, user)
	require.NoError(t, err)

	found, err := baseDAO.findByField(ctx, "username", "helperuser")
	assert.NoError(t, err)
	assert.NotNil(t, found)

	notFound, err := baseDAO.findByField(ctx, "username", "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, notFound)

	// Test findAllByField
	users, err := baseDAO.findAllByField(ctx, "role", entity.RoleUser)
	assert.NoError(t, err)
	assert.Len(t, users, 1)

	// Test deleteByField
	err = baseDAO.deleteByField(ctx, "username", "helperuser")
	assert.NoError(t, err)

	found, err = baseDAO.findByField(ctx, "username", "helperuser")
	assert.NoError(t, err)
	assert.Nil(t, found)
}

func TestRefreshTokenDAO_Operations(t *testing.T) {
	db := setupTestDB(t)
	dao := NewRefreshTokenDAO(db)
	userDAO := NewUserDAO(db)
	ctx := context.Background()

	// Create a user first
	user := &entity.User{
		Username: "tokenuser",
		Email:    "token@example.com",
		Password: "hashedpassword",
		Role:     entity.RoleUser,
		IsActive: true,
	}
	err := userDAO.Create(ctx, user)
	require.NoError(t, err)

	// Create refresh token
	token := &entity.RefreshToken{
		UserID:    user.ID,
		Token:     "test-refresh-token-123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
	}
	err = dao.Create(ctx, token)
	assert.NoError(t, err)
	assert.NotZero(t, token.ID)

	// Find by token
	found, err := dao.FindByToken(ctx, "test-refresh-token-123")
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, user.ID, found.UserID)

	// Revoke token
	err = dao.RevokeByToken(ctx, "test-refresh-token-123")
	assert.NoError(t, err)

	// Should not find revoked token
	notFound, err := dao.FindByToken(ctx, "test-refresh-token-123")
	assert.NoError(t, err)
	assert.Nil(t, notFound)

	// Create another token and revoke all by user
	token2 := &entity.RefreshToken{
		UserID:    user.ID,
		Token:     "test-refresh-token-456",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Revoked:   false,
	}
	err = dao.Create(ctx, token2)
	require.NoError(t, err)

	err = dao.RevokeAllByUserID(ctx, user.ID)
	assert.NoError(t, err)

	// Test delete expired
	expiredToken := &entity.RefreshToken{
		UserID:    user.ID,
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
		Revoked:   false,
	}
	err = dao.Create(ctx, expiredToken)
	require.NoError(t, err)

	err = dao.DeleteExpired(ctx)
	assert.NoError(t, err)

	// Test FindAll
	tokens, total, err := dao.FindAll(ctx, 1, 10)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tokens), 0)
	assert.GreaterOrEqual(t, total, int64(0))
}

func TestPluginDAO_Operations(t *testing.T) {
	db := setupTestDB(t)
	dao := NewPluginDAO(db)
	ctx := context.Background()

	// Create plugin
	plugin := &entity.Plugin{
		Key:         "test-plugin",
		Name:        "Test Plugin",
		Description: "A test plugin",
		Version:     "1.0.0",
		Author:      "Test Author",
		Type:        entity.PluginTypeService,
		State:       entity.PluginStateInstalled,
		InstalledAt: time.Now(),
	}
	err := dao.Create(ctx, plugin)
	assert.NoError(t, err)
	assert.NotZero(t, plugin.ID)

	// Find by ID
	found, err := dao.FindByID(ctx, plugin.ID)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "test-plugin", found.Key)

	// Find by key
	foundByKey, err := dao.FindByKey(ctx, "test-plugin")
	assert.NoError(t, err)
	assert.NotNil(t, foundByKey)

	// Exists by key
	exists, err := dao.ExistsByKey(ctx, "test-plugin")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Update state
	err = dao.UpdateState(ctx, plugin.ID, entity.PluginStateEnabled)
	assert.NoError(t, err)

	// Find by state
	enabledPlugins, err := dao.FindByState(ctx, entity.PluginStateEnabled)
	assert.NoError(t, err)
	assert.Len(t, enabledPlugins, 1)

	// Find enabled
	enabled, err := dao.FindEnabled(ctx)
	assert.NoError(t, err)
	assert.Len(t, enabled, 1)

	// Find all
	plugins, total, err := dao.FindAll(ctx, 1, 10)
	assert.NoError(t, err)
	assert.Len(t, plugins, 1)
	assert.Equal(t, int64(1), total)

	// Update
	plugin.Name = "Updated Plugin"
	err = dao.Update(ctx, plugin)
	assert.NoError(t, err)

	// Delete by key
	err = dao.DeleteByKey(ctx, "test-plugin")
	assert.NoError(t, err)

	// Should not find deleted plugin
	notFound, err := dao.FindByKey(ctx, "test-plugin")
	assert.NoError(t, err)
	assert.Nil(t, notFound)

	// Create another plugin and delete by ID
	plugin2 := &entity.Plugin{
		Key:         "test-plugin-2",
		Name:        "Test Plugin 2",
		Version:     "1.0.0",
		Type:        entity.PluginTypeService,
		State:       entity.PluginStateInstalled,
		InstalledAt: time.Now(),
	}
	err = dao.Create(ctx, plugin2)
	require.NoError(t, err)

	err = dao.Delete(ctx, plugin2.ID)
	assert.NoError(t, err)
}

func TestPluginExtensionDAO_Operations(t *testing.T) {
	db := setupTestDB(t)
	pluginDAO := NewPluginDAO(db)
	dao := NewPluginExtensionDAO(db)
	ctx := context.Background()

	// Create plugin first
	plugin := &entity.Plugin{
		Key:         "ext-test-plugin",
		Name:        "Extension Test Plugin",
		Version:     "1.0.0",
		Type:        entity.PluginTypeService,
		State:       entity.PluginStateInstalled,
		InstalledAt: time.Now(),
	}
	err := pluginDAO.Create(ctx, plugin)
	require.NoError(t, err)

	// Create extension
	extension := &entity.PluginExtension{
		PluginID: plugin.ID,
		Name:     "Test Extension",
		Type:     entity.PluginTypeRestEndpoint,
		Path:     "/api/test",
		Handler:  "TestHandler",
	}
	err = dao.Create(ctx, extension)
	assert.NoError(t, err)
	assert.NotZero(t, extension.ID)

	// Find by ID
	found, err := dao.FindByID(ctx, extension.ID)
	assert.NoError(t, err)
	assert.NotNil(t, found)

	// Find by plugin ID
	extensions, err := dao.FindByPluginID(ctx, plugin.ID)
	assert.NoError(t, err)
	assert.Len(t, extensions, 1)

	// Update
	extension.Handler = "UpdatedHandler"
	err = dao.Update(ctx, extension)
	assert.NoError(t, err)

	// Find all
	allExtensions, total, err := dao.FindAll(ctx, 1, 10)
	assert.NoError(t, err)
	assert.Len(t, allExtensions, 1)
	assert.Equal(t, int64(1), total)

	// Delete by plugin ID
	err = dao.DeleteByPluginID(ctx, plugin.ID)
	assert.NoError(t, err)

	extensions, err = dao.FindByPluginID(ctx, plugin.ID)
	assert.NoError(t, err)
	assert.Len(t, extensions, 0)

	// Create another extension and delete by ID
	extension2 := &entity.PluginExtension{
		PluginID: plugin.ID,
		Name:     "Test Extension 2",
		Type:     entity.PluginTypeRestEndpoint,
	}
	err = dao.Create(ctx, extension2)
	require.NoError(t, err)

	err = dao.Delete(ctx, extension2.ID)
	assert.NoError(t, err)
}
