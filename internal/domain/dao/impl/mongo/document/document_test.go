package document

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUserDocument(t *testing.T) {
	t.Run("CollectionName", func(t *testing.T) {
		doc := UserDocument{}
		assert.Equal(t, "users", doc.CollectionName())
	})

	t.Run("IsDeleted false", func(t *testing.T) {
		doc := UserDocument{
			Username: "test",
		}
		assert.False(t, doc.IsDeleted())
	})

	t.Run("IsDeleted true", func(t *testing.T) {
		deletedAt := time.Now()
		doc := UserDocument{
			Username:  "test",
			DeletedAt: &deletedAt,
		}
		assert.True(t, doc.IsDeleted())
	})
}

func TestRefreshTokenDocument(t *testing.T) {
	t.Run("CollectionName", func(t *testing.T) {
		doc := RefreshTokenDocument{}
		assert.Equal(t, "refresh_tokens", doc.CollectionName())
	})

	t.Run("IsExpired false", func(t *testing.T) {
		doc := RefreshTokenDocument{
			Token:     "test-token",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		assert.False(t, doc.IsExpired())
	})

	t.Run("IsExpired true", func(t *testing.T) {
		doc := RefreshTokenDocument{
			Token:     "test-token",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		assert.True(t, doc.IsExpired())
	})

	t.Run("IsValid true", func(t *testing.T) {
		doc := RefreshTokenDocument{
			Token:     "test-token",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Revoked:   false,
		}
		assert.True(t, doc.IsValid())
	})

	t.Run("IsValid false - revoked", func(t *testing.T) {
		doc := RefreshTokenDocument{
			Token:     "test-token",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Revoked:   true,
		}
		assert.False(t, doc.IsValid())
	})

	t.Run("IsValid false - expired", func(t *testing.T) {
		doc := RefreshTokenDocument{
			Token:     "test-token",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
			Revoked:   false,
		}
		assert.False(t, doc.IsValid())
	})

	t.Run("IsDeleted", func(t *testing.T) {
		doc := RefreshTokenDocument{}
		assert.False(t, doc.IsDeleted())

		deletedAt := time.Now()
		doc.DeletedAt = &deletedAt
		assert.True(t, doc.IsDeleted())
	})
}

func TestPluginDocument(t *testing.T) {
	t.Run("CollectionName", func(t *testing.T) {
		doc := PluginDocument{}
		assert.Equal(t, "plugins", doc.CollectionName())
	})

	t.Run("IsEnabled true", func(t *testing.T) {
		doc := PluginDocument{
			Key:   "test-plugin",
			State: "ENABLED",
		}
		assert.True(t, doc.IsEnabled())
	})

	t.Run("IsEnabled false", func(t *testing.T) {
		doc := PluginDocument{
			Key:   "test-plugin",
			State: "INSTALLED",
		}
		assert.False(t, doc.IsEnabled())
	})

	t.Run("IsDeleted", func(t *testing.T) {
		doc := PluginDocument{}
		assert.False(t, doc.IsDeleted())

		deletedAt := time.Now()
		doc.DeletedAt = &deletedAt
		assert.True(t, doc.IsDeleted())
	})
}

func TestPluginExtensionDocument(t *testing.T) {
	t.Run("CollectionName", func(t *testing.T) {
		doc := PluginExtensionDocument{}
		assert.Equal(t, "plugin_extensions", doc.CollectionName())
	})

	t.Run("IsDeleted", func(t *testing.T) {
		doc := PluginExtensionDocument{}
		assert.False(t, doc.IsDeleted())

		deletedAt := time.Now()
		doc.DeletedAt = &deletedAt
		assert.True(t, doc.IsDeleted())
	})
}
