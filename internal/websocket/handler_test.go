package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestNewHandler creates a handler and checks it's not nil
func TestNewHandler(t *testing.T) {
	cfg := DefaultWebSocketConfig()
	hub := NewHub(zap.NewNop())
	handler := NewHandler(cfg, hub, nil, zap.NewNop())
	assert.NotNil(t, handler)
	assert.Equal(t, hub, handler.GetHub())
}

// TestHandler_GetHub returns the hub
func TestHandler_GetHub(t *testing.T) {
	cfg := DefaultWebSocketConfig()
	hub := NewHub(zap.NewNop())
	handler := NewHandler(cfg, hub, nil, zap.NewNop())
	assert.Equal(t, hub, handler.GetHub())
}

// TestHandler_RegisterRoutes registers routes
func TestHandler_RegisterRoutes(t *testing.T) {
	cfg := DefaultWebSocketConfig()
	hub := NewHub(zap.NewNop())
	handler := NewHandler(cfg, hub, nil, zap.NewNop())

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))

	routes := router.Routes()
	assert.Greater(t, len(routes), 0)
}

// TestHandler_CheckOrigin_Wildcard allows all origins
func TestHandler_CheckOrigin_Wildcard(t *testing.T) {
	cfg := DefaultWebSocketConfig()
	hub := NewHub(zap.NewNop())
	handler := NewHandler(cfg, hub, nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://example.com")

	assert.True(t, handler.checkOrigin(req))
}

// TestHandler_CheckOrigin_EmptyOrigin allows empty origin
func TestHandler_CheckOrigin_EmptyOrigin(t *testing.T) {
	cfg := &WebSocketConfig{
		AllowedOrigins: []string{"https://allowed.com"},
	}
	hub := NewHub(zap.NewNop())
	handler := &Handler{config: cfg, hub: hub}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	// No Origin header
	assert.True(t, handler.checkOrigin(req))
}

// TestHandler_CheckOrigin_AllowedOrigin allows specific origin
func TestHandler_CheckOrigin_AllowedOrigin(t *testing.T) {
	cfg := &WebSocketConfig{
		AllowedOrigins: []string{"https://allowed.com", "https://other.com"},
	}
	hub := NewHub(zap.NewNop())
	handler := &Handler{config: cfg, hub: hub}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://allowed.com")
	assert.True(t, handler.checkOrigin(req))

	req2 := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req2.Header.Set("Origin", "https://other.com")
	assert.True(t, handler.checkOrigin(req2))
}

// TestHandler_CheckOrigin_DeniedOrigin denies unknown origin
func TestHandler_CheckOrigin_DeniedOrigin(t *testing.T) {
	cfg := &WebSocketConfig{
		AllowedOrigins: []string{"https://allowed.com"},
	}
	hub := NewHub(zap.NewNop())
	handler := &Handler{config: cfg, hub: hub}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "https://evil.com")
	assert.False(t, handler.checkOrigin(req))
}

// TestHandler_handleStatus returns hub status
func TestHandler_handleStatus(t *testing.T) {
	cfg := DefaultWebSocketConfig()
	hub := NewHub(zap.NewNop())
	handler := NewHandler(cfg, hub, nil, zap.NewNop())

	router := gin.New()
	router.GET("/ws/status", func(c *gin.Context) {
		handler.handleStatus(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/ws/status", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestHandler_StartHeartbeat_ZeroInterval does not start goroutine
func TestHandler_StartHeartbeat_ZeroInterval(t *testing.T) {
	cfg := &WebSocketConfig{HeartbeatInterval: 0}
	hub := NewHub(zap.NewNop())
	handler := &Handler{config: cfg, hub: hub}

	// Should not panic
	assert.NotPanics(t, func() {
		handler.StartHeartbeat()
	})
}
