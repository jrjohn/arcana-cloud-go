package websocket

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/jrjohn/arcana-cloud-go/internal/security"
)

// WebSocketConfig holds WebSocket configuration
type WebSocketConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	Path              string        `mapstructure:"path"`
	AllowedOrigins    []string      `mapstructure:"allowed_origins"`
	ReadBufferSize    int           `mapstructure:"read_buffer_size"`
	WriteBufferSize   int           `mapstructure:"write_buffer_size"`
	HandshakeTimeout  time.Duration `mapstructure:"handshake_timeout"`
	EnableCompression bool          `mapstructure:"enable_compression"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
}

// DefaultWebSocketConfig returns default configuration
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		Enabled:           false,
		Path:              "/ws",
		AllowedOrigins:    []string{"*"},
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		HandshakeTimeout:  10 * time.Second,
		EnableCompression: true,
		HeartbeatInterval: 30 * time.Second,
	}
}

// Handler handles WebSocket connections
type Handler struct {
	config      *WebSocketConfig
	hub         *Hub
	upgrader    websocket.Upgrader
	jwtProvider *security.JWTProvider
	logger      *zap.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(
	config *WebSocketConfig,
	hub *Hub,
	jwtProvider *security.JWTProvider,
	logger *zap.Logger,
) *Handler {
	h := &Handler{
		config:      config,
		hub:         hub,
		jwtProvider: jwtProvider,
		logger:      logger,
	}

	h.upgrader = websocket.Upgrader{
		ReadBufferSize:    config.ReadBufferSize,
		WriteBufferSize:   config.WriteBufferSize,
		HandshakeTimeout:  config.HandshakeTimeout,
		EnableCompression: config.EnableCompression,
		CheckOrigin:       h.checkOrigin,
	}

	return h
}

// RegisterRoutes registers WebSocket routes
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET(h.config.Path, h.handleWebSocket)
	router.GET(h.config.Path+"/status", h.handleStatus)
}

// handleWebSocket handles WebSocket upgrade requests
func (h *Handler) handleWebSocket(c *gin.Context) {
	// Authenticate user (optional - can be done via query param or header)
	var userID uint
	var username string

	// Try to get token from query parameter
	token := c.Query("token")
	if token == "" {
		// Try Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token = parts[1]
			}
		}
	}

	// Validate token if provided
	if token != "" && h.jwtProvider != nil {
		claims, err := h.jwtProvider.ValidateAccessToken(token)
		if err == nil {
			userID = claims.UserID
			username = claims.Username
		}
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection",
			zap.Error(err),
		)
		return
	}

	// Create client
	client := NewClient(h.hub, conn, userID, username, h.logger)

	// Register client
	h.hub.register <- client

	// Send welcome message
	client.Send(&Message{
		Type: MessageTypeEvent,
		Event: "connected",
		Data: map[string]interface{}{
			"clientId": client.ID,
			"userId":   userID,
			"username": username,
		},
		Timestamp: time.Now(),
	})

	// Start read/write pumps
	go client.WritePump()
	go client.ReadPump()
}

// handleStatus returns WebSocket hub status
func (h *Handler) handleStatus(c *gin.Context) {
	metrics := h.hub.GetMetrics()

	c.JSON(http.StatusOK, gin.H{
		"enabled":            h.config.Enabled,
		"activeConnections":  metrics.ActiveConnections,
		"totalConnections":   metrics.TotalConnections,
		"totalMessages":      metrics.TotalMessages,
		"totalBroadcasts":    metrics.TotalBroadcasts,
		"activeRooms":        metrics.TotalRooms,
		"onlineUsers":        len(h.hub.GetOnlineUsers()),
	})
}

// checkOrigin checks if the origin is allowed
func (h *Handler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	for _, allowed := range h.config.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}

	return false
}

// StartHeartbeat starts a goroutine that sends heartbeats
func (h *Handler) StartHeartbeat() {
	if h.config.HeartbeatInterval <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(h.config.HeartbeatInterval)
		defer ticker.Stop()

		for range ticker.C {
			h.hub.SendHeartbeat()
		}
	}()
}

// GetHub returns the hub
func (h *Handler) GetHub() *Hub {
	return h.hub
}
