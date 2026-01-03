package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/entity"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestRouter() *gin.Engine {
	return gin.New()
}

func newTestJWTProvider() *security.JWTProvider {
	cfg := &config.JWTConfig{
		Secret:               "test-secret-key-for-testing",
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	return security.NewJWTProvider(cfg)
}

func newTestSecurityService(provider *security.JWTProvider) *security.SecurityService {
	return security.NewSecurityService(provider)
}

// RequestID Tests
func TestRequestID(t *testing.T) {
	router := newTestRouter()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		requestID := GetRequestID(c)
		c.String(http.StatusOK, requestID)
	})

	t.Run("generates new request ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		// Check response header
		headerID := w.Header().Get(RequestIDHeader)
		if headerID == "" {
			t.Error("RequestID header not set")
		}

		// Check response body matches header
		if w.Body.String() != headerID {
			t.Errorf("Body = %v, header = %v", w.Body.String(), headerID)
		}
	})

	t.Run("uses provided request ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(RequestIDHeader, "custom-request-id")
		router.ServeHTTP(w, req)

		headerID := w.Header().Get(RequestIDHeader)
		if headerID != "custom-request-id" {
			t.Errorf("RequestID = %v, want custom-request-id", headerID)
		}
	})
}

func TestGetRequestID(t *testing.T) {
	t.Run("request ID exists", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(RequestIDKey, "test-id")

		result := GetRequestID(c)
		if result != "test-id" {
			t.Errorf("GetRequestID() = %v, want test-id", result)
		}
	})

	t.Run("request ID not exists", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		result := GetRequestID(c)
		if result != "" {
			t.Errorf("GetRequestID() = %v, want empty", result)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(RequestIDKey, 123)

		result := GetRequestID(c)
		if result != "" {
			t.Errorf("GetRequestID() = %v, want empty", result)
		}
	})
}

func TestRequestIDConstants(t *testing.T) {
	if RequestIDHeader != "X-Request-ID" {
		t.Errorf("RequestIDHeader = %v, want X-Request-ID", RequestIDHeader)
	}
	if RequestIDKey != "request_id" {
		t.Errorf("RequestIDKey = %v, want request_id", RequestIDKey)
	}
}

// CORS Tests
func TestDefaultCORSConfig(t *testing.T) {
	cfg := DefaultCORSConfig()

	if len(cfg.AllowOrigins) == 0 || cfg.AllowOrigins[0] != "*" {
		t.Error("AllowOrigins should include *")
	}
	if len(cfg.AllowMethods) == 0 {
		t.Error("AllowMethods should not be empty")
	}
	if len(cfg.AllowHeaders) == 0 {
		t.Error("AllowHeaders should not be empty")
	}
	if !cfg.AllowCredentials {
		t.Error("AllowCredentials should be true")
	}
	if cfg.MaxAge != 12*time.Hour {
		t.Errorf("MaxAge = %v, want %v", cfg.MaxAge, 12*time.Hour)
	}
}

func TestCORS(t *testing.T) {
	cfg := DefaultCORSConfig()
	router := newTestRouter()
	router.Use(CORS(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	t.Run("regular request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}

		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Error("CORS header not set")
		}
	})

	t.Run("OPTIONS preflight", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusNoContent)
		}

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Allow-Methods header not set")
		}
	})

	t.Run("no origin header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

func TestCORS_SpecificOrigins(t *testing.T) {
	cfg := CORSConfig{
		AllowOrigins:     []string{"http://allowed.com"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}

	router := newTestRouter()
	router.Use(CORS(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	t.Run("allowed origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://allowed.com")
		router.ServeHTTP(w, req)

		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin != "http://allowed.com" {
			t.Errorf("Allow-Origin = %v, want http://allowed.com", origin)
		}
	})

	t.Run("disallowed origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://notallowed.com")
		router.ServeHTTP(w, req)

		// Request still succeeds but without CORS headers
		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

// Logger Middleware Tests
func TestLogger(t *testing.T) {
	logger := zap.NewNop()
	router := newTestRouter()
	router.Use(RequestID())
	router.Use(Logger(logger))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test?query=value", nil)
	req.Header.Set("User-Agent", "test-agent")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestLogger_StatusCodes(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name   string
		status int
	}{
		{"success", http.StatusOK},
		{"client error", http.StatusBadRequest},
		{"not found", http.StatusNotFound},
		{"server error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newTestRouter()
			router.Use(Logger(logger))
			router.GET("/test", func(c *gin.Context) {
				c.String(tt.status, "response")
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			router.ServeHTTP(w, req)

			if w.Code != tt.status {
				t.Errorf("Status = %v, want %v", w.Code, tt.status)
			}
		})
	}
}

// Recovery Middleware Tests
func TestRecovery(t *testing.T) {
	logger := zap.NewNop()
	router := newTestRouter()
	router.Use(Recovery(logger))
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})
	router.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	t.Run("recovers from panic", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusInternalServerError)
		}
	})

	t.Run("normal request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ok", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

// Auth Middleware Tests
func TestAuthMiddleware_Authenticate(t *testing.T) {
	provider := newTestJWTProvider()
	secService := newTestSecurityService(provider)
	authMiddleware := NewAuthMiddleware(provider, secService)

	router := newTestRouter()
	router.Use(authMiddleware.Authenticate())
	router.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	t.Run("valid token", func(t *testing.T) {
		user := &entity.User{ID: 1, Username: "test", Email: "test@test.com", Role: entity.RoleUser}
		token, _ := provider.GenerateAccessToken(user)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}
	})

	t.Run("missing auth header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("case insensitive bearer", func(t *testing.T) {
		user := &entity.User{ID: 1, Username: "test", Email: "test@test.com", Role: entity.RoleUser}
		token, _ := provider.GenerateAccessToken(user)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "BEARER "+token)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

func TestAuthMiddleware_OptionalAuth(t *testing.T) {
	provider := newTestJWTProvider()
	secService := newTestSecurityService(provider)
	authMiddleware := NewAuthMiddleware(provider, secService)

	router := newTestRouter()
	router.Use(authMiddleware.OptionalAuth())
	router.GET("/optional", func(c *gin.Context) {
		claims := secService.GetCurrentClaims(c)
		if claims != nil {
			c.String(http.StatusOK, "authenticated")
		} else {
			c.String(http.StatusOK, "anonymous")
		}
	})

	t.Run("with valid token", func(t *testing.T) {
		user := &entity.User{ID: 1, Username: "test", Email: "test@test.com", Role: entity.RoleUser}
		token, _ := provider.GenerateAccessToken(user)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		if w.Body.String() != "authenticated" {
			t.Errorf("Body = %v, want authenticated", w.Body.String())
		}
	})

	t.Run("without token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		router.ServeHTTP(w, req)

		if w.Body.String() != "anonymous" {
			t.Errorf("Body = %v, want anonymous", w.Body.String())
		}
	})

	t.Run("with invalid token format", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/optional", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		router.ServeHTTP(w, req)

		// Should still work but as anonymous
		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

func TestAuthMiddleware_RequireRole(t *testing.T) {
	provider := newTestJWTProvider()
	secService := newTestSecurityService(provider)
	authMiddleware := NewAuthMiddleware(provider, secService)

	router := newTestRouter()
	router.Use(authMiddleware.Authenticate())
	router.GET("/admin", authMiddleware.RequireRole(entity.RoleAdmin), func(c *gin.Context) {
		c.String(http.StatusOK, "admin")
	})

	t.Run("admin user", func(t *testing.T) {
		user := &entity.User{ID: 1, Username: "admin", Email: "admin@test.com", Role: entity.RoleAdmin}
		token, _ := provider.GenerateAccessToken(user)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}
	})

	t.Run("non-admin user", func(t *testing.T) {
		user := &entity.User{ID: 1, Username: "user", Email: "user@test.com", Role: entity.RoleUser}
		token, _ := provider.GenerateAccessToken(user)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusForbidden)
		}
	})
}

func TestAuthMiddleware_RequireAdmin(t *testing.T) {
	provider := newTestJWTProvider()
	secService := newTestSecurityService(provider)
	authMiddleware := NewAuthMiddleware(provider, secService)

	router := newTestRouter()
	router.Use(authMiddleware.Authenticate())
	router.GET("/admin-only", authMiddleware.RequireAdmin(), func(c *gin.Context) {
		c.String(http.StatusOK, "admin only")
	})

	t.Run("admin access", func(t *testing.T) {
		user := &entity.User{ID: 1, Username: "admin", Email: "admin@test.com", Role: entity.RoleAdmin}
		token, _ := provider.GenerateAccessToken(user)

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/admin-only", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
		}
	})
}

// Helper function tests
func TestJoinStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{"empty", []string{}, ""},
		{"single", []string{"one"}, "one"},
		{"multiple", []string{"one", "two", "three"}, "one, two, three"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinStrings(tt.input)
			if result != tt.expected {
				t.Errorf("joinStrings() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Benchmarks
func BenchmarkRequestID(b *testing.B) {
	router := newTestRouter()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkCORS(b *testing.B) {
	cfg := DefaultCORSConfig()
	router := newTestRouter()
	router.Use(CORS(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkAuthenticate(b *testing.B) {
	provider := newTestJWTProvider()
	secService := newTestSecurityService(provider)
	authMiddleware := NewAuthMiddleware(provider, secService)

	router := newTestRouter()
	router.Use(authMiddleware.Authenticate())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	user := &entity.User{ID: 1, Username: "test", Email: "test@test.com", Role: entity.RoleUser}
	token, _ := provider.GenerateAccessToken(user)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
