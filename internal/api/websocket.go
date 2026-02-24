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
	priority   chan WebSocketMessage // 高优先级通道，用于关键状态变更消息
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
	hub           *WebSocketHub
	lastBroadcast time.Time
	mu            sync.Mutex
}

// NewWebSocketLogHook 创建 WebSocket 日志钩子
func NewWebSocketLogHook(hub *WebSocketHub) *WebSocketLogHook {
	return &WebSocketLogHook{hub: hub}
}

// OnLog 日志回调 - 节流100ms，减少日志消息对通道的压力
func (h *WebSocketLogHook) OnLog(entry utils.LogEntry) {
	if h.hub == nil {
		return
	}
	h.mu.Lock()
	now := time.Now()
	if now.Sub(h.lastBroadcast) < 100*time.Millisecond {
		h.mu.Unlock()
		return
	}
	h.lastBroadcast = now
	h.mu.Unlock()

	h.hub.Broadcast(WebSocketMessage{
		Type:      "log",
		Data:      entry,
		Timestamp: entry.Timestamp,
	})
}

// NewWebSocketHub 创建新的 WebSocket Hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan WebSocketMessage, 1024),
		priority:   make(chan WebSocketMessage, 256),
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
		// 优先处理高优先级消息
		select {
		case message := <-h.priority:
			h.broadcastToClients(message)
			continue
		default:
		}

		select {
		case message := <-h.priority:
			h.broadcastToClients(message)

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
			h.broadcastToClients(message)
		}
	}
}

// broadcastToClients 向所有客户端发送消息
func (h *WebSocketHub) broadcastToClients(message WebSocketMessage) {
	h.mu.RLock()
	var toUnregister []*WebSocketClient
	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			toUnregister = append(toUnregister, client)
		}
	}
	h.mu.RUnlock()

	if len(toUnregister) > 0 {
		h.mu.Lock()
		for _, client := range toUnregister {
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				utils.Info("WebSocket 客户端已断开(send满): %s", client.id)
			}
		}
		h.mu.Unlock()
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

	for client := range h.clients {
		client.conn.Close()
		close(client.send)
		delete(h.clients, client)
	}

	utils.Info("WebSocket Hub 已停止")
}

// Broadcast 广播普通消息
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
	}
}

// BroadcastPriority 广播高优先级消息（不会被丢弃）
func (h *WebSocketHub) BroadcastPriority(message WebSocketMessage) {
	h.mu.RLock()
	if !h.running {
		h.mu.RUnlock()
		return
	}
	h.mu.RUnlock()

	select {
	case h.priority <- message:
	default:
		// 高优先级通道也满了，阻塞等待发送，确保不丢失
		h.priority <- message
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
			}
			break
		}
	}
}
