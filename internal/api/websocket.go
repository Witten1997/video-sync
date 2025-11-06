package api

import (
	"sync"
	"time"

	"bili-download/internal/utils"

	"github.com/gorilla/websocket"
)

// WebSocketHub WebSocket 连接中心
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan WebSocketMessage
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
	running    bool
}

// WebSocketClient WebSocket 客户端
type WebSocketClient struct {
	hub  *WebSocketHub
	conn *websocket.Conn
	send chan WebSocketMessage
	id   string
}

// WebSocketMessage WebSocket 消息
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// WebSocketLogHook WebSocket 日志钩子
type WebSocketLogHook struct {
	hub *WebSocketHub
}

// NewWebSocketLogHook 创建 WebSocket 日志钩子
func NewWebSocketLogHook(hub *WebSocketHub) *WebSocketLogHook {
	return &WebSocketLogHook{hub: hub}
}

// OnLog 日志回调
func (h *WebSocketLogHook) OnLog(entry utils.LogEntry) {
	if h.hub != nil {
		h.hub.Broadcast(WebSocketMessage{
			Type:      "log",
			Data:      entry,
			Timestamp: entry.Timestamp,
		})
	}
}

// NewWebSocketHub 创建新的 WebSocket Hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan WebSocketMessage, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		running:    false,
	}
}

// Run 运行 Hub
func (h *WebSocketHub) Run() {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.mu.Unlock()

	utils.Info("WebSocket Hub 已启动")

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			utils.Info("WebSocket 客户端已连接: %s", client.id)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				utils.Info("WebSocket 客户端已断开: %s", client.id)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// 发送失败，关闭客户端
					h.mu.RUnlock()
					h.unregister <- client
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Stop 停止 Hub
func (h *WebSocketHub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}

	h.running = false

	// 关闭所有客户端
	for client := range h.clients {
		client.conn.Close()
		close(client.send)
		delete(h.clients, client)
	}

	utils.Info("WebSocket Hub 已停止")
}

// Broadcast 广播消息
func (h *WebSocketHub) Broadcast(message WebSocketMessage) {
	h.mu.RLock()
	if !h.running {
		h.mu.RUnlock()
		return
	}
	h.mu.RUnlock()

	select {
	case h.broadcast <- message:
	default:
		// 消息队列已满，不记录日志避免循环
	}
}

// GetClientCount 获取客户端数量
func (h *WebSocketHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// writePump 写入循环
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub 关闭了通道
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump 读取循环
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// 避免在日志中记录错误，防止循环
			}
			break
		}
	}
}
