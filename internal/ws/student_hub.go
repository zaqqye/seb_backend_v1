package ws

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

type StudentMessage struct {
	Type            string     `json:"type"`
	Locked          bool       `json:"locked,omitempty"`
	BlockedFromExam bool       `json:"blocked_from_exam,omitempty"`
	ForceLogoutAt   *time.Time `json:"force_logout_at,omitempty"`
	AppVersion      string     `json:"app_version,omitempty"`
	Message         string     `json:"message,omitempty"`
}

type studentNotification struct {
	studentID string
	payload   []byte
}

type StudentHub struct {
	register   chan *studentClient
	unregister chan *studentClient
	notify     chan studentNotification
	clients    map[string]*studentClient
}

func NewStudentHub() *StudentHub {
	return &StudentHub{
		register:   make(chan *studentClient),
		unregister: make(chan *studentClient),
		notify:     make(chan studentNotification, 256),
		clients:    make(map[string]*studentClient),
	}
}

func (h *StudentHub) Run() {
	for {
		select {
		case client := <-h.register:
			if existing, ok := h.clients[client.userID]; ok {
				existing.conn.Close()
			}
			h.clients[client.userID] = client
		case client := <-h.unregister:
			if stored, ok := h.clients[client.userID]; ok && stored == client {
				delete(h.clients, client.userID)
			}
		case msg := <-h.notify:
			if client, ok := h.clients[msg.studentID]; ok {
				select {
				case client.send <- msg.payload:
				default:
					client.conn.Close()
					delete(h.clients, msg.studentID)
				}
			}
		}
	}
}

func (h *StudentHub) Notify(studentID string, message StudentMessage) {
	if h == nil {
		return
	}
	data, err := json.Marshal(message)
	if err != nil {
		return
	}
	h.notify <- studentNotification{
		studentID: studentID,
		payload:   data,
	}
}

type studentClient struct {
	hub    *StudentHub
	conn   *websocket.Conn
	send   chan []byte
	userID string
}

func newStudentClient(hub *StudentHub, conn *websocket.Conn, userID string) *studentClient {
	return &studentClient{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 64),
		userID: userID,
	}
}

func (c *studentClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
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

func (c *studentClient) writePump() {
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
