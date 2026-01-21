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
	Hub  *EventHub
	Conn *websocket.Conn
	Send chan []byte
}

// EventHub manages connected clients for a specific event
type EventHub struct {
	EventID    string
	Clients    map[*Client]bool
	Broadcast  chan []byte
	Register   chan *Client
	Unregister chan *Client
	mu         sync.Mutex
}

var (
	eventHubs   = make(map[string]*EventHub)
	eventHubsMu sync.RWMutex
)

func newEventHub(eventID string) *EventHub {
	return &EventHub{
		EventID:    eventID,
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
	}
}

func (h *EventHub) run() {
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
		eventID := c.Param("eventId")

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return
		}

		eventHubsMu.Lock()
		hub, ok := eventHubs[eventID]
		if !ok {
			hub = newEventHub(eventID)
			eventHubs[eventID] = hub
			go hub.run()
		}
		eventHubsMu.Unlock()

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

// BroadcastEventUpdate sends a message to all clients watching a specific event
func BroadcastEventUpdate(eventID string, message interface{}) {
	eventHubsMu.RLock()
	hub, ok := eventHubs[eventID]
	eventHubsMu.RUnlock()

	if ok {
		data, err := json.Marshal(message)
		if err == nil {
			hub.Broadcast <- data
		}
	}
}
