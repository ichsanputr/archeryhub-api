package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
)

// Client represents a connected spectator
type Client struct {
	Hub  *TournamentHub
	Conn *websocket.Conn
	Send chan []byte
}

// TournamentHub manages connected clients for a specific tournament
type TournamentHub struct {
	TournamentID string
	Clients      map[*Client]bool
	Broadcast    chan []byte
	Register     chan *Client
	Unregister   chan *Client
	mu           sync.Mutex
}

var (
	hubs   = make(map[string]*TournamentHub)
	hubsMu sync.RWMutex
)

func newHub(tournamentID string) *TournamentHub {
	return &TournamentHub{
		TournamentID: tournamentID,
		Broadcast:    make(chan []byte),
		Register:     make(chan *Client),
		Unregister:   make(chan *Client),
		Clients:      make(map[*Client]bool),
	}
}

func (h *TournamentHub) run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
		case message := <-h.Broadcast:
			h.mu.Lock()
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// LiveUpdatesWebSocket handles websocket connections for real-time updates
func LiveUpdatesWebSocket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return
		}

		hubsMu.Lock()
		hub, ok := hubs[tournamentID]
		if !ok {
			hub = newHub(tournamentID)
			hubs[tournamentID] = hub
			go hub.run()
		}
		hubsMu.Unlock()

		client := &Client{Hub: hub, Conn: conn, Send: make(chan []byte, 256)}
		client.Hub.Register <- client

		// Allow collection of memory referenced by the caller by doing all work in new goroutines.
		go client.writePump()
		go client.readPump()
	}
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

// BroadcastTournamentUpdate sends a message to all clients watching a specific tournament
func BroadcastTournamentUpdate(tournamentID string, message interface{}) {
	hubsMu.RLock()
	hub, ok := hubs[tournamentID]
	hubsMu.RUnlock()

	if ok {
		data, err := json.Marshal(message)
		if err == nil {
			hub.Broadcast <- data
		}
	}
}
