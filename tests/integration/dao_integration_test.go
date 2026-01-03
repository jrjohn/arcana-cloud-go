//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao"
	gormdao "github.com/jrjohn/arcana-cloud-go/internal/domain/dao/gorm"
	mongodao "github.com/jrjohn/arcana-cloud-go/internal/domain/dao/mongo"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
)

// ========================================
// MySQL Integration Tests
// ========================================

func TestIntegration_MySQL_UserDAO(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoMySQL(t)

	config := testutil.DefaultTestConfig()
	db := testutil.NewTestMySQLDB(t, config)

	// Auto-migrate
	require.NoError(t, db.AutoMigrate(&entity.User{}))

	userDAO := gormdao.NewUserDAO(db)
	runUserDAOTests(t, userDAO)
}

func TestIntegration_MySQL_RefreshTokenDAO(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoMySQL(t)

	config := testutil.DefaultTestConfig()
	db := testutil.NewTestMySQLDB(t, config)

	require.NoError(t, db.AutoMigrate(&entity.User{}))
	require.NoError(t, db.AutoMigrate(&entity.RefreshToken{}))

	userDAO := gormdao.NewUserDAO(db)
	tokenDAO := gormdao.NewRefreshTokenDAO(db)
	runRefreshTokenDAOTests(t, userDAO, tokenDAO)
}

func TestIntegration_MySQL_PluginDAO(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoMySQL(t)

	config := testutil.DefaultTestConfig()
	db := testutil.NewTestMySQLDB(t, config)

	require.NoError(t, db.AutoMigrate(&entity.Plugin{}))

	pluginDAO := gormdao.NewPluginDAO(db)
	runPluginDAOTests(t, pluginDAO)
}

func TestIntegration_MySQL_PluginExtensionDAO(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoMySQL(t)

	config := testutil.DefaultTestConfig()
	db := testutil.NewTestMySQLDB(t, config)

	require.NoError(t, db.AutoMigrate(&entity.Plugin{}))
	require.NoError(t, db.AutoMigrate(&entity.PluginExtension{}))

	pluginDAO := gormdao.NewPluginDAO(db)
	extDAO := gormdao.NewPluginExtensionDAO(db)
	runPluginExtensionDAOTests(t, pluginDAO, extDAO)
}

// ========================================
// PostgreSQL Integration Tests
// ========================================

func TestIntegration_Postgres_UserDAO(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoPostgres(t)

	config := testutil.DefaultTestConfig()
	db := testutil.NewTestPostgresDB(t, config)

	// Create tables using raw SQL to avoid GORM introspection issues with PostgreSQL
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		email VARCHAR(100) UNIQUE NOT NULL,
		password TEXT NOT NULL,
		first_name VARCHAR(50),
		last_name VARCHAR(50),
		role VARCHAR(20) NOT NULL DEFAULT 'USER',
		is_active BOOLEAN DEFAULT true,
		is_verified BOOLEAN DEFAULT false,
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ,
		deleted_at TIMESTAMPTZ
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at)`)

	userDAO := gormdao.NewUserDAO(db)
	runUserDAOTests(t, userDAO)
}

func TestIntegration_Postgres_RefreshTokenDAO(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoPostgres(t)

	config := testutil.DefaultTestConfig()
	db := testutil.NewTestPostgresDB(t, config)

	// Create tables using raw SQL to avoid GORM introspection issues with PostgreSQL
	db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		email VARCHAR(100) UNIQUE NOT NULL,
		password TEXT NOT NULL,
		first_name VARCHAR(50),
		last_name VARCHAR(50),
		role VARCHAR(20) NOT NULL DEFAULT 'USER',
		is_active BOOLEAN DEFAULT true,
		is_verified BOOLEAN DEFAULT false,
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ,
		deleted_at TIMESTAMPTZ
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS refresh_tokens (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		token VARCHAR(500) UNIQUE NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		revoked BOOLEAN DEFAULT false,
		created_at TIMESTAMPTZ,
		deleted_at TIMESTAMPTZ
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_deleted_at ON refresh_tokens(deleted_at)`)

	userDAO := gormdao.NewUserDAO(db)
	tokenDAO := gormdao.NewRefreshTokenDAO(db)
	runRefreshTokenDAOTests(t, userDAO, tokenDAO)
}

func TestIntegration_Postgres_PluginDAO(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoPostgres(t)

	config := testutil.DefaultTestConfig()
	db := testutil.NewTestPostgresDB(t, config)

	// Create tables using raw SQL to avoid GORM introspection issues with PostgreSQL
	db.Exec(`CREATE TABLE IF NOT EXISTS plugins (
		id BIGSERIAL PRIMARY KEY,
		key VARCHAR(100) UNIQUE NOT NULL,
		name VARCHAR(200) NOT NULL,
		description VARCHAR(1000),
		version VARCHAR(50) NOT NULL,
		author VARCHAR(200),
		type VARCHAR(50) NOT NULL,
		state VARCHAR(20) NOT NULL DEFAULT 'INSTALLED',
		config TEXT,
		checksum VARCHAR(128),
		path VARCHAR(500),
		installed_at TIMESTAMPTZ,
		enabled_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ,
		deleted_at TIMESTAMPTZ
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_plugins_deleted_at ON plugins(deleted_at)`)

	pluginDAO := gormdao.NewPluginDAO(db)
	runPluginDAOTests(t, pluginDAO)
}

func TestIntegration_Postgres_PluginExtensionDAO(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoPostgres(t)

	config := testutil.DefaultTestConfig()
	db := testutil.NewTestPostgresDB(t, config)

	// Create tables using raw SQL to avoid GORM introspection issues with PostgreSQL
	db.Exec(`CREATE TABLE IF NOT EXISTS plugins (
		id BIGSERIAL PRIMARY KEY,
		key VARCHAR(100) UNIQUE NOT NULL,
		name VARCHAR(200) NOT NULL,
		description VARCHAR(1000),
		version VARCHAR(50) NOT NULL,
		author VARCHAR(200),
		type VARCHAR(50) NOT NULL,
		state VARCHAR(20) NOT NULL DEFAULT 'INSTALLED',
		config TEXT,
		checksum VARCHAR(128),
		path VARCHAR(500),
		installed_at TIMESTAMPTZ,
		enabled_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ,
		updated_at TIMESTAMPTZ,
		deleted_at TIMESTAMPTZ
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS plugin_extensions (
		id BIGSERIAL PRIMARY KEY,
		plugin_id BIGINT NOT NULL,
		name VARCHAR(200) NOT NULL,
		type VARCHAR(50) NOT NULL,
		path VARCHAR(500),
		handler VARCHAR(500),
		config TEXT,
		created_at TIMESTAMPTZ,
		deleted_at TIMESTAMPTZ
	)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_plugins_deleted_at ON plugins(deleted_at)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_plugin_extensions_deleted_at ON plugin_extensions(deleted_at)`)

	pluginDAO := gormdao.NewPluginDAO(db)
	extDAO := gormdao.NewPluginExtensionDAO(db)
	runPluginExtensionDAOTests(t, pluginDAO, extDAO)
}

// ========================================
// MongoDB Integration Tests
// ========================================

func setupMongoDAOs(t *testing.T, db *mongo.Database) *mongodao.IDCounter {
	idCounter := mongodao.NewIDCounter(db)
	return idCounter
}

func TestIntegration_MongoDB_UserDAO(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoMongo(t)

	config := testutil.DefaultTestConfig()
	_, db := testutil.NewTestMongoDB(t, config)

	idCounter := setupMongoDAOs(t, db)
	userDAO := mongodao.NewUserDAO(db, idCounter)
	runUserDAOTests(t, userDAO)
}

func TestIntegration_MongoDB_RefreshTokenDAO(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoMongo(t)

	config := testutil.DefaultTestConfig()
	_, db := testutil.NewTestMongoDB(t, config)

	idCounter := setupMongoDAOs(t, db)
	userDAO := mongodao.NewUserDAO(db, idCounter)
	tokenDAO := mongodao.NewRefreshTokenDAO(db, idCounter, userDAO)
	runRefreshTokenDAOTests(t, userDAO, tokenDAO)
}

func TestIntegration_MongoDB_PluginDAO(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoMongo(t)

	config := testutil.DefaultTestConfig()
	_, db := testutil.NewTestMongoDB(t, config)

	idCounter := setupMongoDAOs(t, db)
	pluginDAO := mongodao.NewPluginDAO(db, idCounter)
	runPluginDAOTests(t, pluginDAO)
}

func TestIntegration_MongoDB_PluginExtensionDAO(t *testing.T) {
	testutil.SkipIfShort(t)
	testutil.SkipIfNoMongo(t)

	config := testutil.DefaultTestConfig()
	_, db := testutil.NewTestMongoDB(t, config)

	idCounter := setupMongoDAOs(t, db)
	pluginDAO := mongodao.NewPluginDAO(db, idCounter)
	extDAO := mongodao.NewPluginExtensionDAO(db, idCounter)
	runPluginExtensionDAOTests(t, pluginDAO, extDAO)
}

// ========================================
// Shared Test Functions
// ========================================

func runUserDAOTests(t *testing.T, userDAO dao.UserDAO) {
	ctx := context.Background()

	t.Run("Create and FindByID", func(t *testing.T) {
		user := &entity.User{
			Username: "testuser_" + testutil.GenerateTestID(),
			Email:    "test_" + testutil.GenerateTestID() + "@example.com",
			Password: "hashedpassword",
			Role:     entity.RoleUser,
		}

		err := userDAO.Create(ctx, user)
		require.NoError(t, err)
		assert.NotZero(t, user.ID)

		found, err := userDAO.FindByID(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, user.Username, found.Username)
		assert.Equal(t, user.Email, found.Email)
	})

	t.Run("FindByUsername", func(t *testing.T) {
		username := "findbyusername_" + testutil.GenerateTestID()
		user := &entity.User{
			Username: username,
			Email:    username + "@example.com",
			Password: "hash",
			Role:     entity.RoleUser,
		}
		require.NoError(t, userDAO.Create(ctx, user))

		found, err := userDAO.FindByUsername(ctx, username)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, username, found.Username)
	})

	t.Run("FindByEmail", func(t *testing.T) {
		email := "findbyemail_" + testutil.GenerateTestID() + "@example.com"
		user := &entity.User{
			Username: "email_" + testutil.GenerateTestID(),
			Email:    email,
			Password: "hash",
			Role:     entity.RoleUser,
		}
		require.NoError(t, userDAO.Create(ctx, user))

		found, err := userDAO.FindByEmail(ctx, email)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, email, found.Email)
	})

	t.Run("FindByUsernameOrEmail", func(t *testing.T) {
		username := "userormail_" + testutil.GenerateTestID()
		email := username + "@example.com"
		user := &entity.User{
			Username: username,
			Email:    email,
			Password: "hash",
			Role:     entity.RoleUser,
		}
		require.NoError(t, userDAO.Create(ctx, user))

		// Find by username
		found, err := userDAO.FindByUsernameOrEmail(ctx, username)
		require.NoError(t, err)
		require.NotNil(t, found)

		// Find by email
		found, err = userDAO.FindByUsernameOrEmail(ctx, email)
		require.NoError(t, err)
		require.NotNil(t, found)
	})

	t.Run("ExistsByUsername", func(t *testing.T) {
		username := "exists_" + testutil.GenerateTestID()
		user := &entity.User{
			Username: username,
			Email:    username + "@example.com",
			Password: "hash",
			Role:     entity.RoleUser,
		}
		require.NoError(t, userDAO.Create(ctx, user))

		exists, err := userDAO.ExistsByUsername(ctx, username)
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = userDAO.ExistsByUsername(ctx, "nonexistent_"+testutil.GenerateTestID())
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("ExistsByEmail", func(t *testing.T) {
		email := "existsemail_" + testutil.GenerateTestID() + "@example.com"
		user := &entity.User{
			Username: "user_" + testutil.GenerateTestID(),
			Email:    email,
			Password: "hash",
			Role:     entity.RoleUser,
		}
		require.NoError(t, userDAO.Create(ctx, user))

		exists, err := userDAO.ExistsByEmail(ctx, email)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Update", func(t *testing.T) {
		user := &entity.User{
			Username: "update_" + testutil.GenerateTestID(),
			Email:    "update_" + testutil.GenerateTestID() + "@example.com",
			Password: "hash",
			Role:     entity.RoleUser,
		}
		require.NoError(t, userDAO.Create(ctx, user))

		user.Role = entity.RoleAdmin
		err := userDAO.Update(ctx, user)
		require.NoError(t, err)

		found, err := userDAO.FindByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, entity.RoleAdmin, found.Role)
	})

	t.Run("Delete", func(t *testing.T) {
		user := &entity.User{
			Username: "delete_" + testutil.GenerateTestID(),
			Email:    "delete_" + testutil.GenerateTestID() + "@example.com",
			Password: "hash",
			Role:     entity.RoleUser,
		}
		require.NoError(t, userDAO.Create(ctx, user))

		err := userDAO.Delete(ctx, user.ID)
		require.NoError(t, err)

		found, err := userDAO.FindByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("FindAll", func(t *testing.T) {
		// Create some users
		for i := 0; i < 3; i++ {
			user := &entity.User{
				Username: "findall_" + testutil.GenerateTestID(),
				Email:    "findall_" + testutil.GenerateTestID() + "@example.com",
				Password: "hash",
				Role:     entity.RoleUser,
			}
			require.NoError(t, userDAO.Create(ctx, user))
		}

		users, total, err := userDAO.FindAll(ctx, 1, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(3))
		assert.NotEmpty(t, users)
	})

	t.Run("Count", func(t *testing.T) {
		count, err := userDAO.Count(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(0))
	})
}

func runRefreshTokenDAOTests(t *testing.T, userDAO dao.UserDAO, tokenDAO dao.RefreshTokenDAO) {
	ctx := context.Background()

	// Create a test user first
	user := &entity.User{
		Username: "tokenuser_" + testutil.GenerateTestID(),
		Email:    "tokenuser_" + testutil.GenerateTestID() + "@example.com",
		Password: "hash",
		Role:     entity.RoleUser,
	}
	require.NoError(t, userDAO.Create(ctx, user))

	t.Run("Create and FindByToken", func(t *testing.T) {
		tokenValue := "token_" + testutil.GenerateTestID()
		token := &entity.RefreshToken{
			UserID:    user.ID,
			Token:     tokenValue,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Revoked:   false,
		}

		err := tokenDAO.Create(ctx, token)
		require.NoError(t, err)
		assert.NotZero(t, token.ID)

		found, err := tokenDAO.FindByToken(ctx, tokenValue)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, tokenValue, found.Token)
		assert.Equal(t, user.ID, found.UserID)
	})

	t.Run("RevokeByToken", func(t *testing.T) {
		tokenValue := "revoke_" + testutil.GenerateTestID()
		token := &entity.RefreshToken{
			UserID:    user.ID,
			Token:     tokenValue,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Revoked:   false,
		}
		require.NoError(t, tokenDAO.Create(ctx, token))

		err := tokenDAO.RevokeByToken(ctx, tokenValue)
		require.NoError(t, err)

		// Revoked tokens should not be found
		found, err := tokenDAO.FindByToken(ctx, tokenValue)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("RevokeAllByUserID", func(t *testing.T) {
		// Create multiple tokens for a user
		anotherUser := &entity.User{
			Username: "revokeall_" + testutil.GenerateTestID(),
			Email:    "revokeall_" + testutil.GenerateTestID() + "@example.com",
			Password: "hash",
			Role:     entity.RoleUser,
		}
		require.NoError(t, userDAO.Create(ctx, anotherUser))

		tokens := []string{
			"revokeall1_" + testutil.GenerateTestID(),
			"revokeall2_" + testutil.GenerateTestID(),
		}

		for _, tok := range tokens {
			token := &entity.RefreshToken{
				UserID:    anotherUser.ID,
				Token:     tok,
				ExpiresAt: time.Now().Add(24 * time.Hour),
				Revoked:   false,
			}
			require.NoError(t, tokenDAO.Create(ctx, token))
		}

		err := tokenDAO.RevokeAllByUserID(ctx, anotherUser.ID)
		require.NoError(t, err)

		// All tokens should be revoked
		for _, tok := range tokens {
			found, err := tokenDAO.FindByToken(ctx, tok)
			require.NoError(t, err)
			assert.Nil(t, found)
		}
	})

	t.Run("DeleteExpired", func(t *testing.T) {
		expiredToken := &entity.RefreshToken{
			UserID:    user.ID,
			Token:     "expired_" + testutil.GenerateTestID(),
			ExpiresAt: time.Now().Add(-1 * time.Hour),
			Revoked:   false,
		}
		require.NoError(t, tokenDAO.Create(ctx, expiredToken))

		err := tokenDAO.DeleteExpired(ctx)
		require.NoError(t, err)
	})

	t.Run("FindAll", func(t *testing.T) {
		tokens, total, err := tokenDAO.FindAll(ctx, 1, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(0))
		_ = tokens // Just ensure it doesn't panic
	})
}

func runPluginDAOTests(t *testing.T, pluginDAO dao.PluginDAO) {
	ctx := context.Background()

	t.Run("Create and FindByID", func(t *testing.T) {
		plugin := &entity.Plugin{
			Key:         "plugin_" + testutil.GenerateTestID(),
			Name:        "Test Plugin",
			Description: "A test plugin",
			Version:     "1.0.0",
			Type:        entity.PluginTypeService,
			State:       entity.PluginStateInstalled,
			InstalledAt: time.Now(),
		}

		err := pluginDAO.Create(ctx, plugin)
		require.NoError(t, err)
		assert.NotZero(t, plugin.ID)

		found, err := pluginDAO.FindByID(ctx, plugin.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, plugin.Key, found.Key)
	})

	t.Run("FindByKey", func(t *testing.T) {
		key := "findbykey_" + testutil.GenerateTestID()
		plugin := &entity.Plugin{
			Key:         key,
			Name:        "Find By Key Plugin",
			Description: "Test",
			Version:     "1.0.0",
			Type:        entity.PluginTypeService,
			State:       entity.PluginStateInstalled,
			InstalledAt: time.Now(),
		}
		require.NoError(t, pluginDAO.Create(ctx, plugin))

		found, err := pluginDAO.FindByKey(ctx, key)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, key, found.Key)
	})

	t.Run("FindByState", func(t *testing.T) {
		key := "state_" + testutil.GenerateTestID()
		plugin := &entity.Plugin{
			Key:         key,
			Name:        "State Plugin",
			Description: "Test",
			Version:     "1.0.0",
			Type:        entity.PluginTypeService,
			State:       entity.PluginStateEnabled,
			InstalledAt: time.Now(),
		}
		require.NoError(t, pluginDAO.Create(ctx, plugin))

		plugins, err := pluginDAO.FindByState(ctx, entity.PluginStateEnabled)
		require.NoError(t, err)
		assert.NotEmpty(t, plugins)
	})

	t.Run("UpdateState", func(t *testing.T) {
		key := "updatestate_" + testutil.GenerateTestID()
		plugin := &entity.Plugin{
			Key:         key,
			Name:        "Update State Plugin",
			Description: "Test",
			Version:     "1.0.0",
			Type:        entity.PluginTypeService,
			State:       entity.PluginStateInstalled,
			InstalledAt: time.Now(),
		}
		require.NoError(t, pluginDAO.Create(ctx, plugin))

		err := pluginDAO.UpdateState(ctx, plugin.ID, entity.PluginStateEnabled)
		require.NoError(t, err)

		found, err := pluginDAO.FindByID(ctx, plugin.ID)
		require.NoError(t, err)
		assert.Equal(t, entity.PluginStateEnabled, found.State)
	})

	t.Run("ExistsByKey", func(t *testing.T) {
		key := "existsbykey_" + testutil.GenerateTestID()
		plugin := &entity.Plugin{
			Key:         key,
			Name:        "Exists Plugin",
			Description: "Test",
			Version:     "1.0.0",
			Type:        entity.PluginTypeService,
			State:       entity.PluginStateInstalled,
			InstalledAt: time.Now(),
		}
		require.NoError(t, pluginDAO.Create(ctx, plugin))

		exists, err := pluginDAO.ExistsByKey(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = pluginDAO.ExistsByKey(ctx, "nonexistent_"+testutil.GenerateTestID())
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func runPluginExtensionDAOTests(t *testing.T, pluginDAO dao.PluginDAO, extDAO dao.PluginExtensionDAO) {
	ctx := context.Background()

	// Create a test plugin first
	plugin := &entity.Plugin{
		Key:         "extplugin_" + testutil.GenerateTestID(),
		Name:        "Extension Test Plugin",
		Description: "Test",
		Version:     "1.0.0",
		Type:        entity.PluginTypeService,
		State:       entity.PluginStateInstalled,
		InstalledAt: time.Now(),
	}
	require.NoError(t, pluginDAO.Create(ctx, plugin))

	t.Run("Create and FindByID", func(t *testing.T) {
		ext := &entity.PluginExtension{
			PluginID: plugin.ID,
			Name:     "ext_" + testutil.GenerateTestID(),
			Type:     entity.PluginTypeRestEndpoint,
			Handler:  "TestHandler",
			Config:   "{}",
		}

		err := extDAO.Create(ctx, ext)
		require.NoError(t, err)
		assert.NotZero(t, ext.ID)

		found, err := extDAO.FindByID(ctx, ext.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, ext.Name, found.Name)
	})

	t.Run("FindByPluginID", func(t *testing.T) {
		// Create some extensions
		for i := 0; i < 3; i++ {
			ext := &entity.PluginExtension{
				PluginID: plugin.ID,
				Name:     "ext_findby_" + testutil.GenerateTestID(),
				Type:     entity.PluginTypeService,
				Handler:  "Handler",
				Config:   "{}",
			}
			require.NoError(t, extDAO.Create(ctx, ext))
		}

		exts, err := extDAO.FindByPluginID(ctx, plugin.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(exts), 3)
	})

	t.Run("DeleteByPluginID", func(t *testing.T) {
		// Create a new plugin with extensions
		newPlugin := &entity.Plugin{
			Key:         "deleteplugin_" + testutil.GenerateTestID(),
			Name:        "Delete Plugin",
			Description: "Test",
			Version:     "1.0.0",
			Type:        entity.PluginTypeService,
			State:       entity.PluginStateInstalled,
			InstalledAt: time.Now(),
		}
		require.NoError(t, pluginDAO.Create(ctx, newPlugin))

		ext := &entity.PluginExtension{
			PluginID: newPlugin.ID,
			Name:     "ext_delete_" + testutil.GenerateTestID(),
			Type:     entity.PluginTypeService,
			Handler:  "Handler",
			Config:   "{}",
		}
		require.NoError(t, extDAO.Create(ctx, ext))

		err := extDAO.DeleteByPluginID(ctx, newPlugin.ID)
		require.NoError(t, err)

		exts, err := extDAO.FindByPluginID(ctx, newPlugin.ID)
		require.NoError(t, err)
		assert.Empty(t, exts)
	})
}
