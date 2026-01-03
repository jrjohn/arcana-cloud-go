package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jrjohn/arcana-cloud-go/internal/config"
	"github.com/jrjohn/arcana-cloud-go/internal/domain/service"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/request"
	"github.com/jrjohn/arcana-cloud-go/internal/dto/response"
	"github.com/jrjohn/arcana-cloud-go/internal/jobs"
	"github.com/jrjohn/arcana-cloud-go/internal/middleware"
	"github.com/jrjohn/arcana-cloud-go/internal/security"
	"github.com/jrjohn/arcana-cloud-go/internal/testutil/mocks"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter() *gin.Engine {
	return gin.New()
}

func setupSecurityService(t *testing.T) (*security.SecurityService, *security.JWTProvider) {
	jwtConfig := &config.JWTConfig{
		Secret:               "test-secret-key-for-testing-purposes-only",
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "test",
	}
	jwtProvider := security.NewJWTProvider(jwtConfig)
	return security.NewSecurityService(jwtProvider), jwtProvider
}

func setupAuthMiddleware(t *testing.T, jwtProvider *security.JWTProvider, securityService *security.SecurityService) *middleware.AuthMiddleware {
	return middleware.NewAuthMiddleware(jwtProvider, securityService)
}

// Auth Controller Tests
func TestNewAuthController(t *testing.T) {
	authService := mocks.NewMockAuthService()
	securityService, _ := setupSecurityService(t)

	controller := NewAuthController(authService, securityService)
	if controller == nil {
		t.Fatal("NewAuthController() returned nil")
	}
}

func TestAuthController_Register_Success(t *testing.T) {
	authService := mocks.NewMockAuthService()
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/register", controller.Register)

	body := `{"username":"testuser","email":"test@example.com","password":"password123","first_name":"Test","last_name":"User"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Register() status = %v, want %v", w.Code, http.StatusCreated)
	}
}

func TestAuthController_Register_ValidationError(t *testing.T) {
	authService := mocks.NewMockAuthService()
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/register", controller.Register)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Register() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestAuthController_Register_UserExists(t *testing.T) {
	authService := mocks.NewMockAuthService()
	authService.RegisterFunc = func(_ context.Context, _ *request.RegisterRequest) (*response.AuthResponse, error) {
		return nil, service.ErrUserAlreadyExists
	}
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/register", controller.Register)

	body := `{"username":"testuser","email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Register() status = %v, want %v", w.Code, http.StatusConflict)
	}
}

func TestAuthController_Register_InternalError(t *testing.T) {
	authService := mocks.NewMockAuthService()
	authService.RegisterFunc = func(_ context.Context, _ *request.RegisterRequest) (*response.AuthResponse, error) {
		return nil, errors.New("internal error")
	}
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/register", controller.Register)

	body := `{"username":"testuser","email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Register() status = %v, want %v", w.Code, http.StatusInternalServerError)
	}
}

func TestAuthController_Login_Success(t *testing.T) {
	authService := mocks.NewMockAuthService()
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/login", controller.Login)

	body := `{"username_or_email":"testuser","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Login() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestAuthController_Login_InvalidCredentials(t *testing.T) {
	authService := mocks.NewMockAuthService()
	authService.LoginFunc = func(_ context.Context, _ *request.LoginRequest) (*response.AuthResponse, error) {
		return nil, service.ErrInvalidCredentials
	}
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/login", controller.Login)

	body := `{"username_or_email":"testuser","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Login() status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthController_Login_UserInactive(t *testing.T) {
	authService := mocks.NewMockAuthService()
	authService.LoginFunc = func(_ context.Context, _ *request.LoginRequest) (*response.AuthResponse, error) {
		return nil, service.ErrUserInactive
	}
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/login", controller.Login)

	body := `{"username_or_email":"testuser","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Login() status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthController_Login_ValidationError(t *testing.T) {
	authService := mocks.NewMockAuthService()
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/login", controller.Login)

	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Login() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestAuthController_Login_InternalError(t *testing.T) {
	authService := mocks.NewMockAuthService()
	authService.LoginFunc = func(_ context.Context, _ *request.LoginRequest) (*response.AuthResponse, error) {
		return nil, errors.New("internal error")
	}
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/login", controller.Login)

	body := `{"username_or_email":"testuser","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Login() status = %v, want %v", w.Code, http.StatusInternalServerError)
	}
}

func TestAuthController_RefreshToken_Success(t *testing.T) {
	authService := mocks.NewMockAuthService()
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/refresh", controller.RefreshToken)

	body := `{"refresh_token":"valid-refresh-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("RefreshToken() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestAuthController_RefreshToken_InvalidToken(t *testing.T) {
	authService := mocks.NewMockAuthService()
	authService.RefreshTokenFunc = func(_ context.Context, _ *request.RefreshTokenRequest) (*response.AuthResponse, error) {
		return nil, service.ErrInvalidToken
	}
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/refresh", controller.RefreshToken)

	body := `{"refresh_token":"invalid-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("RefreshToken() status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthController_RefreshToken_ValidationError(t *testing.T) {
	authService := mocks.NewMockAuthService()
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/refresh", controller.RefreshToken)

	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("RefreshToken() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestAuthController_RefreshToken_InternalError(t *testing.T) {
	authService := mocks.NewMockAuthService()
	authService.RefreshTokenFunc = func(_ context.Context, _ *request.RefreshTokenRequest) (*response.AuthResponse, error) {
		return nil, errors.New("internal error")
	}
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/refresh", controller.RefreshToken)

	body := `{"refresh_token":"valid-token"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("RefreshToken() status = %v, want %v", w.Code, http.StatusInternalServerError)
	}
}

func TestAuthController_Logout(t *testing.T) {
	authService := mocks.NewMockAuthService()
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/logout", controller.Logout)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Logout() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestAuthController_Logout_NoToken(t *testing.T) {
	authService := mocks.NewMockAuthService()
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/logout", controller.Logout)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Logout() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestAuthController_LogoutAll(t *testing.T) {
	authService := mocks.NewMockAuthService()
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	router.POST("/auth/logout-all", func(c *gin.Context) {
		// Simulate authenticated user
		c.Set(security.ContextKeyClaims, &security.UserClaims{UserID: 1})
		controller.LogoutAll(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/logout-all", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("LogoutAll() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestAuthController_RegisterRoutes(t *testing.T) {
	authService := mocks.NewMockAuthService()
	securityService, _ := setupSecurityService(t)
	controller := NewAuthController(authService, securityService)

	router := setupTestRouter()
	controller.RegisterRoutes(router.Group("/api/v1"))

	// Test that routes are registered
	routes := router.Routes()
	if len(routes) == 0 {
		t.Error("RegisterRoutes() should register routes")
	}
}

// User Controller Tests
func TestNewUserController(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)

	controller := NewUserController(userService, securityService, authMiddleware)
	if controller == nil {
		t.Fatal("NewUserController() returned nil")
	}
}

func TestUserController_List_Success(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.GET("/users", controller.List)

	req := httptest.NewRequest(http.MethodGet, "/users?page=1&size=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("List() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestUserController_List_Error(t *testing.T) {
	userService := mocks.NewMockUserService()
	userService.ListFunc = func(_ context.Context, _, _ int) (*response.PagedResponse[response.UserResponse], error) {
		return nil, errors.New("database error")
	}
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.GET("/users", controller.List)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("List() status = %v, want %v", w.Code, http.StatusInternalServerError)
	}
}

func TestUserController_GetCurrentUser_Success(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.GET("/users/me", func(c *gin.Context) {
		c.Set(security.ContextKeyClaims, &security.UserClaims{UserID: 1})
		controller.GetCurrentUser(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetCurrentUser() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestUserController_GetCurrentUser_NotAuthenticated(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.GET("/users/me", controller.GetCurrentUser)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("GetCurrentUser() status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestUserController_GetCurrentUser_NotFound(t *testing.T) {
	userService := mocks.NewMockUserService()
	userService.GetByIDFunc = func(_ context.Context, _ uint) (*response.UserResponse, error) {
		return nil, service.ErrUserNotFound
	}
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.GET("/users/me", func(c *gin.Context) {
		c.Set(security.ContextKeyClaims, &security.UserClaims{UserID: 1})
		controller.GetCurrentUser(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetCurrentUser() status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestUserController_GetCurrentUser_InternalError(t *testing.T) {
	userService := mocks.NewMockUserService()
	userService.GetByIDFunc = func(_ context.Context, _ uint) (*response.UserResponse, error) {
		return nil, errors.New("internal error")
	}
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.GET("/users/me", func(c *gin.Context) {
		c.Set(security.ContextKeyClaims, &security.UserClaims{UserID: 1})
		controller.GetCurrentUser(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("GetCurrentUser() status = %v, want %v", w.Code, http.StatusInternalServerError)
	}
}

func TestUserController_UpdateCurrentUser_Success(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.PUT("/users/me", func(c *gin.Context) {
		c.Set(security.ContextKeyClaims, &security.UserClaims{UserID: 1})
		controller.UpdateCurrentUser(c)
	})

	body := `{"first_name":"Updated","last_name":"Name"}`
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("UpdateCurrentUser() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestUserController_UpdateCurrentUser_NotAuthenticated(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.PUT("/users/me", controller.UpdateCurrentUser)

	body := `{"first_name":"Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("UpdateCurrentUser() status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestUserController_UpdateCurrentUser_ValidationError(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.PUT("/users/me", func(c *gin.Context) {
		c.Set(security.ContextKeyClaims, &security.UserClaims{UserID: 1})
		controller.UpdateCurrentUser(c)
	})

	body := `{invalid}`
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("UpdateCurrentUser() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_UpdateCurrentUser_NotFound(t *testing.T) {
	userService := mocks.NewMockUserService()
	userService.UpdateFunc = func(_ context.Context, _ uint, _ *request.UpdateProfileRequest) (*response.UserResponse, error) {
		return nil, service.ErrUserNotFound
	}
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.PUT("/users/me", func(c *gin.Context) {
		c.Set(security.ContextKeyClaims, &security.UserClaims{UserID: 1})
		controller.UpdateCurrentUser(c)
	})

	body := `{"first_name":"Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("UpdateCurrentUser() status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestUserController_UpdateCurrentUser_EmailConflict(t *testing.T) {
	userService := mocks.NewMockUserService()
	userService.UpdateFunc = func(_ context.Context, _ uint, _ *request.UpdateProfileRequest) (*response.UserResponse, error) {
		return nil, service.ErrUserAlreadyExists
	}
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.PUT("/users/me", func(c *gin.Context) {
		c.Set(security.ContextKeyClaims, &security.UserClaims{UserID: 1})
		controller.UpdateCurrentUser(c)
	})

	body := `{"email":"existing@example.com"}`
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("UpdateCurrentUser() status = %v, want %v", w.Code, http.StatusConflict)
	}
}

func TestUserController_UpdateCurrentUser_InternalError(t *testing.T) {
	userService := mocks.NewMockUserService()
	userService.UpdateFunc = func(_ context.Context, _ uint, _ *request.UpdateProfileRequest) (*response.UserResponse, error) {
		return nil, errors.New("internal error")
	}
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.PUT("/users/me", func(c *gin.Context) {
		c.Set(security.ContextKeyClaims, &security.UserClaims{UserID: 1})
		controller.UpdateCurrentUser(c)
	})

	body := `{"first_name":"Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("UpdateCurrentUser() status = %v, want %v", w.Code, http.StatusInternalServerError)
	}
}

func TestUserController_ChangePassword_Success(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.PUT("/users/me/password", func(c *gin.Context) {
		c.Set(security.ContextKeyClaims, &security.UserClaims{UserID: 1})
		controller.ChangePassword(c)
	})

	body := `{"old_password":"oldpassword","new_password":"newpassword123"}`
	req := httptest.NewRequest(http.MethodPut, "/users/me/password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ChangePassword() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestUserController_ChangePassword_NotAuthenticated(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.PUT("/users/me/password", controller.ChangePassword)

	body := `{"old_password":"oldpass","new_password":"newpass"}`
	req := httptest.NewRequest(http.MethodPut, "/users/me/password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ChangePassword() status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestUserController_ChangePassword_InvalidCredentials(t *testing.T) {
	userService := mocks.NewMockUserService()
	userService.ChangePasswordFunc = func(_ context.Context, _ uint, _ *request.ChangePasswordRequest) error {
		return service.ErrInvalidCredentials
	}
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.PUT("/users/me/password", func(c *gin.Context) {
		c.Set(security.ContextKeyClaims, &security.UserClaims{UserID: 1})
		controller.ChangePassword(c)
	})

	body := `{"old_password":"wrongpass","new_password":"newpass"}`
	req := httptest.NewRequest(http.MethodPut, "/users/me/password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("ChangePassword() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_GetByID_Success(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.GET("/users/:id", controller.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetByID() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestUserController_GetByID_InvalidID(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.GET("/users/:id", controller.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/users/invalid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("GetByID() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_GetByUsername_Success(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.GET("/users/username/:username", controller.GetByUsername)

	req := httptest.NewRequest(http.MethodGet, "/users/username/testuser", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetByUsername() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestUserController_Delete_Success(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.DELETE("/users/:id", controller.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Delete() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestUserController_Delete_InvalidID(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.DELETE("/users/:id", controller.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/users/invalid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Delete() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestUserController_Delete_Error(t *testing.T) {
	userService := mocks.NewMockUserService()
	userService.DeleteFunc = func(_ context.Context, _ uint) error {
		return errors.New("delete error")
	}
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	router.DELETE("/users/:id", controller.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Delete() status = %v, want %v", w.Code, http.StatusInternalServerError)
	}
}

// Plugin Controller Tests
func TestNewPluginController(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)

	controller := NewPluginController(pluginService, authMiddleware)
	if controller == nil {
		t.Fatal("NewPluginController() returned nil")
	}
}

func TestPluginController_List_Success(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.GET("/plugins", controller.List)

	req := httptest.NewRequest(http.MethodGet, "/plugins?page=1&size=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("List() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestPluginController_GetByKey_Success(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.GET("/plugins/:key", controller.GetByKey)

	req := httptest.NewRequest(http.MethodGet, "/plugins/test-plugin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetByKey() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestPluginController_GetByKey_NotFound(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	pluginService.GetByKeyFunc = func(_ context.Context, _ string) (*response.PluginDetailResponse, error) {
		return nil, service.ErrPluginNotFound
	}
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.GET("/plugins/:key", controller.GetByKey)

	req := httptest.NewRequest(http.MethodGet, "/plugins/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetByKey() status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestPluginController_Install_Success(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.POST("/plugins/install", controller.Install)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("name", "Test Plugin")
	writer.WriteField("version", "1.0.0")
	writer.WriteField("type", "SERVICE")
	part, _ := writer.CreateFormFile("file", "plugin.so")
	part.Write([]byte("fake plugin data"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/plugins/install", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Install() status = %v, want %v", w.Code, http.StatusCreated)
	}
}

func TestPluginController_Install_NoFile(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.POST("/plugins/install", controller.Install)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("name", "Test Plugin")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/plugins/install", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Install() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestPluginController_Install_MissingFields(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.POST("/plugins/install", controller.Install)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "plugin.so")
	part.Write([]byte("fake plugin data"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/plugins/install", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Install() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestPluginController_Enable_Success(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.POST("/plugins/:key/enable", controller.Enable)

	req := httptest.NewRequest(http.MethodPost, "/plugins/test-plugin/enable", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Enable() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestPluginController_Enable_InvalidState(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	pluginService.EnableFunc = func(_ context.Context, _ string) (*response.PluginResponse, error) {
		return nil, service.ErrPluginInvalidState
	}
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.POST("/plugins/:key/enable", controller.Enable)

	req := httptest.NewRequest(http.MethodPost, "/plugins/test-plugin/enable", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Enable() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestPluginController_Disable_Success(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.POST("/plugins/:key/disable", controller.Disable)

	req := httptest.NewRequest(http.MethodPost, "/plugins/test-plugin/disable", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Disable() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestPluginController_Uninstall_Success(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.DELETE("/plugins/:key", controller.Uninstall)

	req := httptest.NewRequest(http.MethodDelete, "/plugins/test-plugin", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Uninstall() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestPluginController_GetHealth_Success(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.GET("/plugins/health", controller.GetHealth)

	req := httptest.NewRequest(http.MethodGet, "/plugins/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetHealth() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestPluginController_GetReadiness(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.GET("/plugins/health/ready", controller.GetReadiness)

	req := httptest.NewRequest(http.MethodGet, "/plugins/health/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetReadiness() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestPluginController_GetLiveness(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	router.GET("/plugins/health/live", controller.GetLiveness)

	req := httptest.NewRequest(http.MethodGet, "/plugins/health/live", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetLiveness() status = %v, want %v", w.Code, http.StatusOK)
	}
}

// Job Controller Tests
func TestNewJobController(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)

	controller := NewJobController(jobService, nil, authMiddleware)
	if controller == nil {
		t.Fatal("NewJobController() returned nil")
	}
}

func TestJobController_EnqueueJob_Success(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.POST("/jobs", controller.EnqueueJob)

	body := `{"type":"test-job","payload":{"key":"value"},"priority":"normal"}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("EnqueueJob() status = %v, want %v", w.Code, http.StatusCreated)
	}
}

func TestJobController_EnqueueJob_WithDelay(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.POST("/jobs", controller.EnqueueJob)

	body := `{"type":"test-job","payload":{"key":"value"},"delay_seconds":60}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("EnqueueJob() status = %v, want %v", w.Code, http.StatusCreated)
	}
}

func TestJobController_EnqueueJob_WithScheduledAt(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.POST("/jobs", controller.EnqueueJob)

	scheduledAt := time.Now().Add(time.Hour).Format(time.RFC3339)
	body := `{"type":"test-job","payload":{"key":"value"},"scheduled_at":"` + scheduledAt + `"}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("EnqueueJob() status = %v, want %v", w.Code, http.StatusCreated)
	}
}

func TestJobController_EnqueueJob_InvalidScheduledAt(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.POST("/jobs", controller.EnqueueJob)

	body := `{"type":"test-job","payload":{"key":"value"},"scheduled_at":"invalid-date"}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("EnqueueJob() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestJobController_EnqueueJob_InvalidPayload(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.POST("/jobs", controller.EnqueueJob)

	body := `{"type":"test-job","payload":"invalid-not-json-object"}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Note: "invalid-not-json-object" is valid JSON string, so this should work
	// Let's test with truly invalid JSON
	body = `{"type":"test-job","payload":{invalid}}`
	req = httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("EnqueueJob() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestJobController_EnqueueJob_AllPriorities(t *testing.T) {
	priorities := []string{"low", "normal", "high", "critical", "unknown"}

	for _, priority := range priorities {
		t.Run(priority, func(t *testing.T) {
			jobService := mocks.NewMockJobService()
			securityService, jwtProvider := setupSecurityService(t)
			authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
			controller := NewJobController(jobService, nil, authMiddleware)

			router := setupTestRouter()
			router.POST("/jobs", controller.EnqueueJob)

			body := `{"type":"test-job","payload":{},"priority":"` + priority + `"}`
			req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("EnqueueJob() with priority %s status = %v, want %v", priority, w.Code, http.StatusCreated)
			}
		})
	}
}

func TestJobController_EnqueueJob_WithTags(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.POST("/jobs", controller.EnqueueJob)

	body := `{"type":"test-job","payload":{},"tags":["tag1","tag2"]}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("EnqueueJob() status = %v, want %v", w.Code, http.StatusCreated)
	}
}

func TestJobController_EnqueueJob_WithUniqueKey(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.POST("/jobs", controller.EnqueueJob)

	body := `{"type":"test-job","payload":{},"unique_key":"unique-123"}`
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("EnqueueJob() status = %v, want %v", w.Code, http.StatusCreated)
	}
}

func TestJobController_GetJob_Success(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.GET("/jobs/:id", controller.GetJob)

	req := httptest.NewRequest(http.MethodGet, "/jobs/job-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetJob() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestJobController_GetJob_NotFound(t *testing.T) {
	jobService := mocks.NewMockJobService()
	jobService.GetJobFunc = func(_ context.Context, _ string) (*jobs.JobPayload, error) {
		return nil, errors.New("not found")
	}
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.GET("/jobs/:id", controller.GetJob)

	req := httptest.NewRequest(http.MethodGet, "/jobs/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetJob() status = %v, want %v", w.Code, http.StatusNotFound)
	}
}

func TestJobController_CancelJob_Success(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.DELETE("/jobs/:id", controller.CancelJob)

	req := httptest.NewRequest(http.MethodDelete, "/jobs/job-123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CancelJob() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestJobController_RetryJob_Success(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.POST("/jobs/:id/retry", controller.RetryJob)

	req := httptest.NewRequest(http.MethodPost, "/jobs/job-123/retry", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("RetryJob() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestJobController_GetQueueStats_Success(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.GET("/jobs/queues", controller.GetQueueStats)

	req := httptest.NewRequest(http.MethodGet, "/jobs/queues", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetQueueStats() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestJobController_GetDashboard(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.GET("/jobs/dashboard", controller.GetDashboard)

	req := httptest.NewRequest(http.MethodGet, "/jobs/dashboard", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetDashboard() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestJobController_GetDLQJobs_Success(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.GET("/jobs/dlq", controller.GetDLQJobs)

	req := httptest.NewRequest(http.MethodGet, "/jobs/dlq?limit=50", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetDLQJobs() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestJobController_GetDLQJobs_InvalidLimit(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.GET("/jobs/dlq", controller.GetDLQJobs)

	// Test with limit = 0 (should default to 100)
	req := httptest.NewRequest(http.MethodGet, "/jobs/dlq?limit=0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetDLQJobs() status = %v, want %v", w.Code, http.StatusOK)
	}

	// Test with limit > 1000 (should default to 100)
	req = httptest.NewRequest(http.MethodGet, "/jobs/dlq?limit=2000", nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetDLQJobs() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestJobController_RetryDLQJob_Success(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.POST("/jobs/dlq/:id/retry", controller.RetryDLQJob)

	req := httptest.NewRequest(http.MethodPost, "/jobs/dlq/job-123/retry", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("RetryDLQJob() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestJobController_PurgeDLQ_Success(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.DELETE("/jobs/dlq", controller.PurgeDLQ)

	req := httptest.NewRequest(http.MethodDelete, "/jobs/dlq", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("PurgeDLQ() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestJobController_GetScheduledJobs_NoScheduler(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	router.GET("/jobs/scheduled", controller.GetScheduledJobs)

	req := httptest.NewRequest(http.MethodGet, "/jobs/scheduled", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetScheduledJobs() status = %v, want %v", w.Code, http.StatusOK)
	}

	// Verify empty array is returned
	var resp response.ApiResponse[[]response.ScheduledJobResponse]
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Data) != 0 {
		t.Errorf("GetScheduledJobs() should return empty array when scheduler is nil")
	}
}

func TestJobController_RegisterRoutes(t *testing.T) {
	jobService := mocks.NewMockJobService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewJobController(jobService, nil, authMiddleware)

	router := setupTestRouter()
	controller.RegisterRoutes(router.Group("/api/v1"))

	routes := router.Routes()
	if len(routes) == 0 {
		t.Error("RegisterRoutes() should register routes")
	}
}

func TestUserController_RegisterRoutes(t *testing.T) {
	userService := mocks.NewMockUserService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewUserController(userService, securityService, authMiddleware)

	router := setupTestRouter()
	controller.RegisterRoutes(router.Group("/api/v1"))

	routes := router.Routes()
	if len(routes) == 0 {
		t.Error("RegisterRoutes() should register routes")
	}
}

func TestPluginController_RegisterRoutes(t *testing.T) {
	pluginService := mocks.NewMockPluginService()
	securityService, jwtProvider := setupSecurityService(t)
	authMiddleware := setupAuthMiddleware(t, jwtProvider, securityService)
	controller := NewPluginController(pluginService, authMiddleware)

	router := setupTestRouter()
	controller.RegisterRoutes(router.Group("/api/v1"))

	routes := router.Routes()
	if len(routes) == 0 {
		t.Error("RegisterRoutes() should register routes")
	}
}
