package routes

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/domain"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// wsClient represents one connected WebSocket reviewer.
type wsClient struct {
	hub  *ReviewHub
	conn *websocket.Conn
	send chan []byte
}

// ReviewHub broadcasts new ReviewItems to all connected WebSocket clients.
type ReviewHub struct {
	mu       sync.RWMutex
	clients  map[*wsClient]bool
	notifyCh <-chan *domain.ReviewItem
}

// NewReviewHub creates a hub that reads from notifyCh and broadcasts to clients.
func NewReviewHub(notifyCh <-chan *domain.ReviewItem) *ReviewHub {
	return &ReviewHub{
		clients:  make(map[*wsClient]bool),
		notifyCh: notifyCh,
	}
}

// Run must be started in a goroutine from main. It loops until notifyCh is
// closed.
func (h *ReviewHub) Run() {
	for item := range h.notifyCh {
		payload, err := json.Marshal(item)
		if err != nil {
			slog.Error("ws hub marshal", "err", err)
			continue
		}
		h.broadcast(payload)
	}
}

func (h *ReviewHub) register(c *wsClient) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
}

func (h *ReviewHub) unregister(c *wsClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.send)
}

func (h *ReviewHub) broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- msg:
		default:
			// Slow client: drop message rather than block the hub.
		}
	}
}

// HandleReviewWS upgrades the connection and streams new ReviewItems.
// Requires withAuth in the route chain — only admin and doctor may connect.
func (h *ReviewHub) HandleReviewWS(w http.ResponseWriter, r *http.Request) {
	role := auth.RoleFromCtx(r.Context())
	if role != domain.RoleAdmin && role != domain.RoleDoctor {
		http.Error(w, "insufficient permissions", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade", "err", err)
		return
	}

	c := &wsClient{hub: h, conn: conn, send: make(chan []byte, 256)}
	h.register(c)

	go c.writePump()
	go c.readPump()
}

func (c *wsClient) writePump() {
	defer func() {
		c.conn.Close()
		c.hub.unregister(c)
	}()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *wsClient) readPump() {
	defer c.conn.Close()
	for {
		// Discard incoming messages; we're server-push only.
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}
