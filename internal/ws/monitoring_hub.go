package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	sendBufferSize = 256
)

// MonitoringPayload is pushed to pengawas/admin dashboards.
type MonitoringPayload struct {
	ID               string             `json:"id"`
	StudentID        string             `json:"student_id"`
	RoomID           *string            `json:"room_id,omitempty"`
	Locked           bool               `json:"locked"`
	BlockedFromExam  bool               `json:"blocked_from_exam"`
	UpdatedAt        time.Time          `json:"updated_at"`
	ForceLogoutAt    *time.Time         `json:"force_logout_at,omitempty"`
	LastAppVersion   string             `json:"app_version,omitempty"`
	Monitoring       MonitoringSnapshot `json:"monitoring"`
	Room             MonitoringRoom     `json:"room"`
}

// MonitoringSnapshot mirrors the monitoring block returned by the REST API.
type MonitoringSnapshot struct {
	ID              string     `json:"id"`
	AppVersion      string     `json:"app_version"`
	Locked          bool       `json:"locked"`
	BlockedFromExam bool       `json:"blocked_from_exam"`
	ForceLogoutAt   *time.Time `json:"force_logout_at,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

// MonitoringRoom mirrors the room object in REST responses.
type MonitoringRoom struct {
	ID       string `json:"id"`
	RoomName string `json:"room_name"`
}

type monitoringMessage struct {
	roomID  *string
	payload []byte
}

// MonitoringHub handles websocket clients who listen for student status updates.
type MonitoringHub struct {
	register   chan *monitoringClient
	unregister chan *monitoringClient
	broadcast  chan monitoringMessage
	clients    map[*monitoringClient]struct{}
}

func NewMonitoringHub() *MonitoringHub {
	return &MonitoringHub{
		register:   make(chan *monitoringClient),
		unregister: make(chan *monitoringClient),
		broadcast:  make(chan monitoringMessage, 256),
		clients:    make(map[*monitoringClient]struct{}),
	}
}

func (h *MonitoringHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = struct{}{}
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				client.conn.Close()
			}
		case msg := <-h.broadcast:
			for client := range h.clients {
				if !client.allowAll {
					if msg.roomID == nil {
						continue
					}
					if _, ok := client.allowedRooms[*msg.roomID]; !ok {
						continue
					}
				}
				select {
				case client.send <- msg.payload:
				default:
					delete(h.clients, client)
					close(client.send)
					client.conn.Close()
				}
			}
		}
	}
}

// Broadcast pushes payload to all relevant clients (room-scoped if provided).
func (h *MonitoringHub) Broadcast(payload MonitoringPayload) {
	if h == nil {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ws: failed to marshal payload: %v", err)
		return
	}
	h.broadcast <- monitoringMessage{
		roomID:  payload.RoomID,
		payload: data,
	}
}

type monitoringClient struct {
	hub          *MonitoringHub
	conn         *websocket.Conn
	send         chan []byte
	allowedRooms map[string]struct{}
	allowAll     bool
}

func newMonitoringClient(hub *MonitoringHub, conn *websocket.Conn, allowed map[string]struct{}, allowAll bool) *monitoringClient {
	return &monitoringClient{
		hub:          hub,
		conn:         conn,
		send:         make(chan []byte, sendBufferSize),
		allowedRooms: allowed,
		allowAll:     allowAll,
	}
}

func (c *monitoringClient) readPump() {
	defer func() {
		c.hub.unregister <- c
	}()
	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (c *monitoringClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(msg); err != nil {
				return
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
