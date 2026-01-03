package websocket

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// Hub maintains active clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Clients by user ID
	userClients map[uint]map[*Client]bool

	// Clients by room
	roomClients map[string]map[*Client]bool

	// Inbound messages from clients
	broadcast chan *Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Room operations
	joinRoom  chan *RoomOperation
	leaveRoom chan *RoomOperation

	// Mutex for thread-safe access
	mutex sync.RWMutex

	// Logger
	logger *zap.Logger

	// Metrics
	metrics *HubMetrics
}

// HubMetrics holds hub metrics
type HubMetrics struct {
	TotalConnections    int64
	ActiveConnections   int64
	TotalMessages       int64
	TotalBroadcasts     int64
	TotalRooms          int
	mutex               sync.RWMutex
}

// RoomOperation represents a room join/leave operation
type RoomOperation struct {
	Client *Client
	Room   string
}

// NewHub creates a new hub
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		userClients: make(map[uint]map[*Client]bool),
		roomClients: make(map[string]map[*Client]bool),
		broadcast:   make(chan *Message, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		joinRoom:    make(chan *RoomOperation),
		leaveRoom:   make(chan *RoomOperation),
		logger:      logger,
		metrics:     &HubMetrics{},
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case op := <-h.joinRoom:
			h.handleJoinRoom(op)

		case op := <-h.leaveRoom:
			h.handleLeaveRoom(op)

		case message := <-h.broadcast:
			h.handleBroadcast(message)
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients[client] = true

	// Add to user clients
	if client.UserID > 0 {
		if _, ok := h.userClients[client.UserID]; !ok {
			h.userClients[client.UserID] = make(map[*Client]bool)
		}
		h.userClients[client.UserID][client] = true
	}

	h.metrics.mutex.Lock()
	h.metrics.TotalConnections++
	h.metrics.ActiveConnections++
	h.metrics.mutex.Unlock()

	h.logger.Debug("Client registered",
		zap.String("client_id", client.ID),
		zap.Uint("user_id", client.UserID),
	)
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)

		// Remove from user clients
		if client.UserID > 0 {
			if clients, ok := h.userClients[client.UserID]; ok {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.userClients, client.UserID)
				}
			}
		}

		// Remove from all rooms
		for room, clients := range h.roomClients {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.roomClients, room)
			}
		}

		h.metrics.mutex.Lock()
		h.metrics.ActiveConnections--
		h.metrics.mutex.Unlock()

		h.logger.Debug("Client unregistered",
			zap.String("client_id", client.ID),
		)
	}
}

// handleJoinRoom handles a room join operation
func (h *Hub) handleJoinRoom(op *RoomOperation) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, ok := h.roomClients[op.Room]; !ok {
		h.roomClients[op.Room] = make(map[*Client]bool)
	}
	h.roomClients[op.Room][op.Client] = true
	op.Client.Rooms[op.Room] = true

	h.metrics.mutex.Lock()
	h.metrics.TotalRooms = len(h.roomClients)
	h.metrics.mutex.Unlock()

	h.logger.Debug("Client joined room",
		zap.String("client_id", op.Client.ID),
		zap.String("room", op.Room),
	)
}

// handleLeaveRoom handles a room leave operation
func (h *Hub) handleLeaveRoom(op *RoomOperation) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if clients, ok := h.roomClients[op.Room]; ok {
		delete(clients, op.Client)
		if len(clients) == 0 {
			delete(h.roomClients, op.Room)
		}
	}
	delete(op.Client.Rooms, op.Room)

	h.metrics.mutex.Lock()
	h.metrics.TotalRooms = len(h.roomClients)
	h.metrics.mutex.Unlock()

	h.logger.Debug("Client left room",
		zap.String("client_id", op.Client.ID),
		zap.String("room", op.Room),
	)
}

// handleBroadcast handles a broadcast message
func (h *Hub) handleBroadcast(message *Message) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	h.metrics.mutex.Lock()
	h.metrics.TotalBroadcasts++
	h.metrics.mutex.Unlock()

	var targets map[*Client]bool

	switch {
	case message.Room != "":
		// Send to room
		targets = h.roomClients[message.Room]
	case message.UserID > 0:
		// Send to specific user
		targets = h.userClients[message.UserID]
	default:
		// Send to all
		targets = h.clients
	}

	for client := range targets {
		select {
		case client.send <- message:
			h.metrics.mutex.Lock()
			h.metrics.TotalMessages++
			h.metrics.mutex.Unlock()
		default:
			// Client's send buffer is full, skip
			h.logger.Warn("Client send buffer full",
				zap.String("client_id", client.ID),
			)
		}
	}
}

// Broadcast sends a message to all clients
func (h *Hub) Broadcast(message *Message) {
	h.broadcast <- message
}

// BroadcastToRoom sends a message to all clients in a room
func (h *Hub) BroadcastToRoom(room string, message *Message) {
	message.Room = room
	h.broadcast <- message
}

// BroadcastToUser sends a message to all clients of a specific user
func (h *Hub) BroadcastToUser(userID uint, message *Message) {
	message.UserID = userID
	h.broadcast <- message
}

// JoinRoom adds a client to a room
func (h *Hub) JoinRoom(client *Client, room string) {
	h.joinRoom <- &RoomOperation{Client: client, Room: room}
}

// LeaveRoom removes a client from a room
func (h *Hub) LeaveRoom(client *Client, room string) {
	h.leaveRoom <- &RoomOperation{Client: client, Room: room}
}

// GetClientCount returns the number of active clients
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// GetRoomCount returns the number of active rooms
func (h *Hub) GetRoomCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.roomClients)
}

// GetRoomClientCount returns the number of clients in a room
func (h *Hub) GetRoomClientCount(room string) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if clients, ok := h.roomClients[room]; ok {
		return len(clients)
	}
	return 0
}

// GetMetrics returns hub metrics
func (h *Hub) GetMetrics() HubMetrics {
	h.metrics.mutex.RLock()
	defer h.metrics.mutex.RUnlock()
	return *h.metrics
}

// IsUserOnline checks if a user is online
func (h *Hub) IsUserOnline(userID uint) bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	clients, ok := h.userClients[userID]
	return ok && len(clients) > 0
}

// GetOnlineUsers returns a list of online user IDs
func (h *Hub) GetOnlineUsers() []uint {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	users := make([]uint, 0, len(h.userClients))
	for userID := range h.userClients {
		users = append(users, userID)
	}
	return users
}

// SendHeartbeat sends heartbeat to all clients
func (h *Hub) SendHeartbeat() {
	h.Broadcast(&Message{
		Type:      MessageTypePing,
		Timestamp: time.Now(),
	})
}
