package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestSecurityService() *SecurityService {
	cfg := &config.JWTConfig{
		Secret:              "test-secret",
		AccessTokenDuration: 3600,
		Issuer:              "test",
	}
	provider := NewJWTProvider(cfg)
	return NewSecurityService(provider)
}

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, w
}

func TestNewSecurityService(t *testing.T) {
	cfg := &config.JWTConfig{
		Secret: "secret",
		Issuer: "test",
	}
	provider := NewJWTProvider(cfg)
	service := NewSecurityService(provider)

	if service == nil {
		t.Fatal("NewSecurityService() returned nil")
	}
	if service.jwtProvider != provider {
		t.Error("jwtProvider not set correctly")
	}
}

func TestSecurityService_GetCurrentUser(t *testing.T) {
	service := newTestSecurityService()

	t.Run("user exists", func(t *testing.T) {
		c, _ := newTestContext()
		user := &entity.User{ID: 1, Username: "testuser"}
		c.Set(ContextKeyUser, user)

		result := service.GetCurrentUser(c)
		if result == nil {
			t.Fatal("GetCurrentUser() returned nil")
		}
		if result.ID != 1 {
			t.Errorf("User ID = %v, want 1", result.ID)
		}
	})

	t.Run("user not exists", func(t *testing.T) {
		c, _ := newTestContext()

		result := service.GetCurrentUser(c)
		if result != nil {
			t.Errorf("GetCurrentUser() = %v, want nil", result)
		}
	})

	t.Run("wrong type in context", func(t *testing.T) {
		c, _ := newTestContext()
		c.Set(ContextKeyUser, "not a user")

		result := service.GetCurrentUser(c)
		if result != nil {
			t.Errorf("GetCurrentUser() = %v, want nil", result)
		}
	})
}

func TestSecurityService_GetCurrentUserID(t *testing.T) {
	service := newTestSecurityService()

	t.Run("claims exist", func(t *testing.T) {
		c, _ := newTestContext()
		claims := &UserClaims{UserID: 42}
		c.Set(ContextKeyClaims, claims)

		result := service.GetCurrentUserID(c)
		if result != 42 {
			t.Errorf("GetCurrentUserID() = %v, want 42", result)
		}
	})

	t.Run("no claims", func(t *testing.T) {
		c, _ := newTestContext()

		result := service.GetCurrentUserID(c)
		if result != 0 {
			t.Errorf("GetCurrentUserID() = %v, want 0", result)
		}
	})
}

func TestSecurityService_GetCurrentClaims(t *testing.T) {
	service := newTestSecurityService()

	t.Run("claims exist", func(t *testing.T) {
		c, _ := newTestContext()
		claims := &UserClaims{UserID: 1, Username: "test"}
		c.Set(ContextKeyClaims, claims)

		result := service.GetCurrentClaims(c)
		if result == nil {
			t.Fatal("GetCurrentClaims() returned nil")
		}
		if result.UserID != 1 {
			t.Errorf("UserID = %v, want 1", result.UserID)
		}
	})

	t.Run("claims not exist", func(t *testing.T) {
		c, _ := newTestContext()

		result := service.GetCurrentClaims(c)
		if result != nil {
			t.Errorf("GetCurrentClaims() = %v, want nil", result)
		}
	})

	t.Run("wrong type in context", func(t *testing.T) {
		c, _ := newTestContext()
		c.Set(ContextKeyClaims, "not claims")

		result := service.GetCurrentClaims(c)
		if result != nil {
			t.Errorf("GetCurrentClaims() = %v, want nil", result)
		}
	})
}

func TestSecurityService_SetCurrentUser(t *testing.T) {
	service := newTestSecurityService()
	c, _ := newTestContext()

	user := &entity.User{ID: 1, Username: "testuser"}
	service.SetCurrentUser(c, user)

	// Verify it was set
	result := service.GetCurrentUser(c)
	if result == nil {
		t.Fatal("SetCurrentUser() did not set user")
	}
	if result.ID != user.ID {
		t.Errorf("User ID = %v, want %v", result.ID, user.ID)
	}
}

func TestSecurityService_SetCurrentClaims(t *testing.T) {
	service := newTestSecurityService()
	c, _ := newTestContext()

	claims := &UserClaims{UserID: 1, Username: "test", Role: entity.RoleAdmin}
	service.SetCurrentClaims(c, claims)

	// Verify it was set
	result := service.GetCurrentClaims(c)
	if result == nil {
		t.Fatal("SetCurrentClaims() did not set claims")
	}
	if result.UserID != claims.UserID {
		t.Errorf("UserID = %v, want %v", result.UserID, claims.UserID)
	}
	if result.Role != claims.Role {
		t.Errorf("Role = %v, want %v", result.Role, claims.Role)
	}
}

func TestSecurityService_IsAuthenticated(t *testing.T) {
	service := newTestSecurityService()

	t.Run("authenticated", func(t *testing.T) {
		c, _ := newTestContext()
		claims := &UserClaims{UserID: 1}
		c.Set(ContextKeyClaims, claims)

		if !service.IsAuthenticated(c) {
			t.Error("IsAuthenticated() = false, want true")
		}
	})

	t.Run("not authenticated", func(t *testing.T) {
		c, _ := newTestContext()

		if service.IsAuthenticated(c) {
			t.Error("IsAuthenticated() = true, want false")
		}
	})
}

func TestSecurityService_HasRole(t *testing.T) {
	service := newTestSecurityService()

	tests := []struct {
		name       string
		claimsRole entity.UserRole
		checkRole  entity.UserRole
		expected   bool
	}{
		{
			name:       "has admin role",
			claimsRole: entity.RoleAdmin,
			checkRole:  entity.RoleAdmin,
			expected:   true,
		},
		{
			name:       "has user role",
			claimsRole: entity.RoleUser,
			checkRole:  entity.RoleUser,
			expected:   true,
		},
		{
			name:       "user checking for admin",
			claimsRole: entity.RoleUser,
			checkRole:  entity.RoleAdmin,
			expected:   false,
		},
		{
			name:       "admin checking for user",
			claimsRole: entity.RoleAdmin,
			checkRole:  entity.RoleUser,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestContext()
			claims := &UserClaims{UserID: 1, Role: tt.claimsRole}
			c.Set(ContextKeyClaims, claims)

			result := service.HasRole(c, tt.checkRole)
			if result != tt.expected {
				t.Errorf("HasRole() = %v, want %v", result, tt.expected)
			}
		})
	}

	t.Run("no claims", func(t *testing.T) {
		c, _ := newTestContext()

		if service.HasRole(c, entity.RoleUser) {
			t.Error("HasRole() should return false when no claims")
		}
	})
}

func TestSecurityService_IsAdmin(t *testing.T) {
	service := newTestSecurityService()

	t.Run("is admin", func(t *testing.T) {
		c, _ := newTestContext()
		claims := &UserClaims{UserID: 1, Role: entity.RoleAdmin}
		c.Set(ContextKeyClaims, claims)

		if !service.IsAdmin(c) {
			t.Error("IsAdmin() = false, want true")
		}
	})

	t.Run("not admin", func(t *testing.T) {
		c, _ := newTestContext()
		claims := &UserClaims{UserID: 1, Role: entity.RoleUser}
		c.Set(ContextKeyClaims, claims)

		if service.IsAdmin(c) {
			t.Error("IsAdmin() = true, want false")
		}
	})

	t.Run("no claims", func(t *testing.T) {
		c, _ := newTestContext()

		if service.IsAdmin(c) {
			t.Error("IsAdmin() should return false when no claims")
		}
	})
}

func TestContextKeys(t *testing.T) {
	if ContextKeyUser != "current_user" {
		t.Errorf("ContextKeyUser = %v, want current_user", ContextKeyUser)
	}
	if ContextKeyClaims != "current_claims" {
		t.Errorf("ContextKeyClaims = %v, want current_claims", ContextKeyClaims)
	}
}

// Benchmarks
func BenchmarkSecurityService_GetCurrentClaims(b *testing.B) {
	service := newTestSecurityService()
	c, _ := newTestContext()
	claims := &UserClaims{UserID: 1}
	c.Set(ContextKeyClaims, claims)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.GetCurrentClaims(c)
	}
}

func BenchmarkSecurityService_IsAuthenticated(b *testing.B) {
	service := newTestSecurityService()
	c, _ := newTestContext()
	claims := &UserClaims{UserID: 1}
	c.Set(ContextKeyClaims, claims)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.IsAuthenticated(c)
	}
}

func BenchmarkSecurityService_HasRole(b *testing.B) {
	service := newTestSecurityService()
	c, _ := newTestContext()
	claims := &UserClaims{UserID: 1, Role: entity.RoleAdmin}
	c.Set(ContextKeyClaims, claims)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.HasRole(c, entity.RoleAdmin)
	}
}
