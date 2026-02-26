package mapper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/jrjohn/arcana-cloud-go/internal/domain/dao/mongo/document"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

const (
	testEmail     = "test@example.com"
	testToken     = "test-token"
	testPluginKey = "test-plugin"
	testExtension = "Test Extension"
	toDocumentNil   = "nil user"
	toDocumentValid = "valid user"
	toEntityNil     = "nil document"
	toEntityValid   = "valid document"
)

func TestUserMapper_ToDocument(t *testing.T) {
	mapper := NewUserMapper()

	t.Run("nil user", func(t *testing.T) {
		doc := mapper.ToDocument(nil)
		assert.Nil(t, doc)
	})

	t.Run("valid user", func(t *testing.T) {
		now := time.Now()
		user := &entity.User{
			ID:         1,
			Username:   "testuser",
			Email:      testEmail,
			Password:   "hashedpass",
			FirstName:  "Test",
			LastName:   "User",
			Role:       entity.RoleAdmin,
			IsActive:   true,
			IsVerified: true,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		doc := mapper.ToDocument(user)
		assert.NotNil(t, doc)
		assert.Equal(t, uint(1), doc.NumericID)
		assert.Equal(t, "testuser", doc.Username)
		assert.Equal(t, testEmail, doc.Email)
		assert.Equal(t, "hashedpass", doc.Password)
		assert.Equal(t, "Test", doc.FirstName)
		assert.Equal(t, "User", doc.LastName)
		assert.Equal(t, "ADMIN", doc.Role)
		assert.True(t, doc.IsActive)
		assert.True(t, doc.IsVerified)
		assert.Nil(t, doc.DeletedAt)
	})

	t.Run("user with deleted_at", func(t *testing.T) {
		deletedAt := time.Now()
		user := &entity.User{
			ID:        1,
			Username:  "deleted",
			Email:     "deleted@example.com",
			Password:  "pass",
			Role:      entity.RoleUser,
			DeletedAt: gorm.DeletedAt{Time: deletedAt, Valid: true},
		}

		doc := mapper.ToDocument(user)
		assert.NotNil(t, doc.DeletedAt)
		assert.Equal(t, deletedAt.Unix(), doc.DeletedAt.Unix())
	})
}

func TestUserMapper_ToEntity(t *testing.T) {
	mapper := NewUserMapper()

	t.Run("nil document", func(t *testing.T) {
		entity := mapper.ToEntity(nil)
		assert.Nil(t, entity)
	})

	t.Run("valid document", func(t *testing.T) {
		now := time.Now()
		doc := &document.UserDocument{
			NumericID:  1,
			Username:   "testuser",
			Email:      testEmail,
			Password:   "hashedpass",
			FirstName:  "Test",
			LastName:   "User",
			Role:       "ADMIN",
			IsActive:   true,
			IsVerified: true,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		user := mapper.ToEntity(doc)
		assert.NotNil(t, user)
		assert.Equal(t, uint(1), user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, testEmail, user.Email)
		assert.Equal(t, entity.RoleAdmin, user.Role)
		assert.True(t, user.IsActive)
		assert.False(t, user.DeletedAt.Valid)
	})

	t.Run("document with deleted_at", func(t *testing.T) {
		deletedAt := time.Now()
		doc := &document.UserDocument{
			NumericID: 1,
			Username:  "deleted",
			Email:     "deleted@example.com",
			Password:  "pass",
			Role:      "USER",
			DeletedAt: &deletedAt,
		}

		user := mapper.ToEntity(doc)
		assert.True(t, user.DeletedAt.Valid)
		assert.Equal(t, deletedAt.Unix(), user.DeletedAt.Time.Unix())
	})
}

func TestUserMapper_ToEntities(t *testing.T) {
	mapper := NewUserMapper()

	t.Run("nil slice", func(t *testing.T) {
		entities := mapper.ToEntities(nil)
		assert.Nil(t, entities)
	})

	t.Run("valid slice", func(t *testing.T) {
		docs := []*document.UserDocument{
			{NumericID: 1, Username: "user1", Email: "user1@example.com", Password: "pass", Role: "USER"},
			{NumericID: 2, Username: "user2", Email: "user2@example.com", Password: "pass", Role: "ADMIN"},
		}

		entities := mapper.ToEntities(docs)
		assert.Len(t, entities, 2)
		assert.Equal(t, "user1", entities[0].Username)
		assert.Equal(t, "user2", entities[1].Username)
	})
}

func TestUserMapper_ToDocuments(t *testing.T) {
	mapper := NewUserMapper()

	t.Run("nil slice", func(t *testing.T) {
		docs := mapper.ToDocuments(nil)
		assert.Nil(t, docs)
	})

	t.Run("valid slice", func(t *testing.T) {
		users := []*entity.User{
			{ID: 1, Username: "user1", Email: "user1@example.com", Password: "pass", Role: entity.RoleUser},
			{ID: 2, Username: "user2", Email: "user2@example.com", Password: "pass", Role: entity.RoleAdmin},
		}

		docs := mapper.ToDocuments(users)
		assert.Len(t, docs, 2)
		assert.Equal(t, "user1", docs[0].Username)
		assert.Equal(t, "user2", docs[1].Username)
	})
}

func TestRefreshTokenMapper(t *testing.T) {
	mapper := NewRefreshTokenMapper()

	t.Run("ToDocument nil", func(t *testing.T) {
		doc := mapper.ToDocument(nil)
		assert.Nil(t, doc)
	})

	t.Run("ToDocument valid", func(t *testing.T) {
		now := time.Now()
		token := &entity.RefreshToken{
			ID:        1,
			UserID:    10,
			Token:     testToken,
			ExpiresAt: now.Add(24 * time.Hour),
			Revoked:   false,
			CreatedAt: now,
		}

		doc := mapper.ToDocument(token)
		assert.NotNil(t, doc)
		assert.Equal(t, uint(1), doc.NumericID)
		assert.Equal(t, uint(10), doc.UserID)
		assert.Equal(t, testToken, doc.Token)
		assert.False(t, doc.Revoked)
	})

	t.Run("ToEntity nil", func(t *testing.T) {
		entity := mapper.ToEntity(nil)
		assert.Nil(t, entity)
	})

	t.Run("ToEntity valid", func(t *testing.T) {
		now := time.Now()
		doc := &document.RefreshTokenDocument{
			NumericID: 1,
			UserID:    10,
			Token:     testToken,
			ExpiresAt: now.Add(24 * time.Hour),
			Revoked:   false,
			CreatedAt: now,
		}

		token := mapper.ToEntity(doc)
		assert.NotNil(t, token)
		assert.Equal(t, uint(1), token.ID)
		assert.Equal(t, uint(10), token.UserID)
		assert.Equal(t, testToken, token.Token)
	})

	t.Run("ToEntities", func(t *testing.T) {
		docs := []*document.RefreshTokenDocument{
			{NumericID: 1, Token: "token1"},
			{NumericID: 2, Token: "token2"},
		}
		entities := mapper.ToEntities(docs)
		assert.Len(t, entities, 2)
	})

	t.Run("ToDocuments", func(t *testing.T) {
		tokens := []*entity.RefreshToken{
			{ID: 1, Token: "token1"},
			{ID: 2, Token: "token2"},
		}
		docs := mapper.ToDocuments(tokens)
		assert.Len(t, docs, 2)
	})
}

func TestPluginMapper(t *testing.T) {
	mapper := NewPluginMapper()

	t.Run("ToDocument nil", func(t *testing.T) {
		doc := mapper.ToDocument(nil)
		assert.Nil(t, doc)
	})

	t.Run("ToDocument valid", func(t *testing.T) {
		now := time.Now()
		enabledAt := now.Add(-1 * time.Hour)
		plugin := &entity.Plugin{
			ID:          1,
			Key:         "test-plugin",
			Name:        "Test Plugin",
			Description: "A test plugin",
			Version:     "1.0.0",
			Author:      "Test Author",
			Type:        entity.PluginTypeService,
			State:       entity.PluginStateEnabled,
			Config:      `{"key": "value"}`,
			Checksum:    "abc123",
			Path:        "/plugins/test",
			InstalledAt: now,
			EnabledAt:   &enabledAt,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		doc := mapper.ToDocument(plugin)
		assert.NotNil(t, doc)
		assert.Equal(t, "test-plugin", doc.Key)
		assert.Equal(t, "SERVICE", doc.Type)
		assert.Equal(t, "ENABLED", doc.State)
		assert.NotNil(t, doc.EnabledAt)
	})

	t.Run("ToEntity nil", func(t *testing.T) {
		entity := mapper.ToEntity(nil)
		assert.Nil(t, entity)
	})

	t.Run("ToEntity valid", func(t *testing.T) {
		now := time.Now()
		doc := &document.PluginDocument{
			NumericID:   1,
			Key:         "test-plugin",
			Name:        "Test Plugin",
			Version:     "1.0.0",
			Type:        "SERVICE",
			State:       "ENABLED",
			InstalledAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		plugin := mapper.ToEntity(doc)
		assert.NotNil(t, plugin)
		assert.Equal(t, "test-plugin", plugin.Key)
		assert.Equal(t, entity.PluginTypeService, plugin.Type)
		assert.Equal(t, entity.PluginStateEnabled, plugin.State)
	})

	t.Run("ToEntities", func(t *testing.T) {
		docs := []*document.PluginDocument{
			{NumericID: 1, Key: "plugin1"},
			{NumericID: 2, Key: "plugin2"},
		}
		entities := mapper.ToEntities(docs)
		assert.Len(t, entities, 2)
	})

	t.Run("ToDocuments", func(t *testing.T) {
		plugins := []*entity.Plugin{
			{ID: 1, Key: "plugin1"},
			{ID: 2, Key: "plugin2"},
		}
		docs := mapper.ToDocuments(plugins)
		assert.Len(t, docs, 2)
	})
}

func TestPluginExtensionMapper(t *testing.T) {
	mapper := NewPluginExtensionMapper()

	t.Run("ToDocument nil", func(t *testing.T) {
		doc := mapper.ToDocument(nil)
		assert.Nil(t, doc)
	})

	t.Run("ToDocument valid", func(t *testing.T) {
		now := time.Now()
		ext := &entity.PluginExtension{
			ID:        1,
			PluginID:  10,
			Name:      "Test Extension",
			Type:      entity.PluginTypeRestEndpoint,
			Path:      "/api/test",
			Handler:   "TestHandler",
			Config:    `{"enabled": true}`,
			CreatedAt: now,
		}

		doc := mapper.ToDocument(ext)
		assert.NotNil(t, doc)
		assert.Equal(t, uint(1), doc.NumericID)
		assert.Equal(t, uint(10), doc.PluginID)
		assert.Equal(t, "Test Extension", doc.Name)
		assert.Equal(t, "REST_ENDPOINT", doc.Type)
	})

	t.Run("ToEntity nil", func(t *testing.T) {
		entity := mapper.ToEntity(nil)
		assert.Nil(t, entity)
	})

	t.Run("ToEntity valid", func(t *testing.T) {
		now := time.Now()
		doc := &document.PluginExtensionDocument{
			NumericID: 1,
			PluginID:  10,
			Name:      "Test Extension",
			Type:      "REST_ENDPOINT",
			Path:      "/api/test",
			Handler:   "TestHandler",
			CreatedAt: now,
		}

		ext := mapper.ToEntity(doc)
		assert.NotNil(t, ext)
		assert.Equal(t, uint(1), ext.ID)
		assert.Equal(t, uint(10), ext.PluginID)
		assert.Equal(t, entity.PluginTypeRestEndpoint, ext.Type)
	})

	t.Run("ToEntity with deleted_at", func(t *testing.T) {
		deletedAt := time.Now()
		doc := &document.PluginExtensionDocument{
			NumericID: 1,
			PluginID:  10,
			Name:      "Deleted Extension",
			Type:      "SERVICE",
			DeletedAt: &deletedAt,
		}

		ext := mapper.ToEntity(doc)
		assert.True(t, ext.DeletedAt.Valid)
	})

	t.Run("ToEntities", func(t *testing.T) {
		docs := []*document.PluginExtensionDocument{
			{NumericID: 1, Name: "ext1"},
			{NumericID: 2, Name: "ext2"},
		}
		entities := mapper.ToEntities(docs)
		assert.Len(t, entities, 2)
	})

	t.Run("ToDocuments", func(t *testing.T) {
		extensions := []*entity.PluginExtension{
			{ID: 1, Name: "ext1"},
			{ID: 2, Name: "ext2"},
		}
		docs := mapper.ToDocuments(extensions)
		assert.Len(t, docs, 2)
	})
}
