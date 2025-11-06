package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该限制
	},
}

// handleWebSocket 处理 WebSocket 连接
func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		respondInternalError(c, fmt.Errorf("WebSocket 升级失败: %w", err))
		return
	}

	// 创建客户端
	client := &WebSocketClient{
		hub:  s.websocketHub,
		conn: conn,
		send: make(chan WebSocketMessage, 256),
		id:   fmt.Sprintf("%s-%d", c.ClientIP(), time.Now().Unix()),
	}

	// 注册客户端
	client.hub.register <- client

	// 启动读写循环
	go client.writePump()
	go client.readPump()
}
