package api

import (
	"context"
	"strings"
	"sync"
	"time"

	"bili-download/internal/scheduler"
	"bili-download/internal/utils"

	"github.com/gin-gonic/gin"
)

// SystemAlert 系统告警
type SystemAlert struct {
	Key       string      `json:"key"`       // 唯一键，例如 "version_update"、"bili_credential_invalid"
	Type      string      `json:"type"`      // 告警类型：version_update / credential_invalid
	Title     string      `json:"title"`     // 告警标题
	Message   string      `json:"message"`   // 告警描述
	Severity  string      `json:"severity"`  // info / warning / error
	Action    string      `json:"action"`    // 跳转动作，例如 "/config?tab=version" 或 "/config?tab=bilibili"
	CreatedAt time.Time   `json:"created_at"`
	Data      interface{} `json:"data,omitempty"`
}

// alertsStore 系统告警存储
type alertsStore struct {
	mu                  sync.RWMutex
	items               map[string]SystemAlert
	telegramLastNotify  map[string]time.Time // 各告警类型上次发送 Telegram 的时间，用于节流
}

func newAlertsStore() *alertsStore {
	return &alertsStore{
		items:              make(map[string]SystemAlert),
		telegramLastNotify: make(map[string]time.Time),
	}
}

// Set 设置告警；如果同 key 已存在且内容相同则不更新（避免重复 WS 推送）
func (a *alertsStore) Set(alert SystemAlert) (changed bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if old, ok := a.items[alert.Key]; ok {
		if old.Title == alert.Title && old.Message == alert.Message && old.Action == alert.Action {
			return false
		}
	}
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now()
	}
	a.items[alert.Key] = alert
	return true
}

// Clear 清除指定 key 的告警
func (a *alertsStore) Clear(key string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.items[key]; !ok {
		return false
	}
	delete(a.items, key)
	return true
}

// List 列出所有告警
func (a *alertsStore) List() []SystemAlert {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]SystemAlert, 0, len(a.items))
	for _, item := range a.items {
		result = append(result, item)
	}
	return result
}

// ShouldNotifyTelegram 节流：同 key 的告警在指定时间窗口内只允许发送一次
func (a *alertsStore) ShouldNotifyTelegram(key string, throttle time.Duration) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	last, ok := a.telegramLastNotify[key]
	if ok && time.Since(last) < throttle {
		return false
	}
	a.telegramLastNotify[key] = time.Now()
	return true
}

// handleListSystemAlerts 列出当前活跃告警
func (s *Server) handleListSystemAlerts(c *gin.Context) {
	respondSuccess(c, gin.H{
		"items": s.alerts.List(),
	})
}

// pushAlert 设置告警并通过 WebSocket 推送
func (s *Server) pushAlert(alert SystemAlert) {
	if !s.alerts.Set(alert) {
		return
	}
	if s.websocketHub != nil {
		s.websocketHub.BroadcastPriority(WebSocketMessage{
			Type:      "system_alert",
			Data:      alert,
			Timestamp: time.Now(),
		})
	}
}

// clearAlert 清除告警并通过 WebSocket 推送清除事件
func (s *Server) clearAlert(key string) {
	if !s.alerts.Clear(key) {
		return
	}
	if s.websocketHub != nil {
		s.websocketHub.Broadcast(WebSocketMessage{
			Type:      "system_alert_cleared",
			Data:      gin.H{"key": key},
			Timestamp: time.Now(),
		})
	}
}

// notifyTelegramAdmins 给所有已配置的 Telegram chat 发送通知；throttleKey 不为空时按 key 节流
func (s *Server) notifyTelegramAdmins(text, throttleKey string, throttle time.Duration) {
	if s.config == nil || !s.config.Telegram.Enabled {
		return
	}
	if strings.TrimSpace(s.config.Telegram.BotToken) == "" {
		return
	}
	chatIDs := s.config.Telegram.AllowedChatIDs
	if len(chatIDs) == 0 {
		return
	}
	if throttleKey != "" && !s.alerts.ShouldNotifyTelegram(throttleKey, throttle) {
		return
	}

	client := s.newTelegramClient()
	if client == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		for _, chatID := range chatIDs {
			if _, err := client.SendMessage(ctx, chatID, text, 0); err != nil {
				utils.Warn("发送 Telegram 告警失败 chat_id=%d: %v", chatID, err)
			}
		}
	}()
}

// handleCredentialInvalidEvent 处理调度器上报的 B 站凭据失效事件
func (s *Server) handleCredentialInvalidEvent(event scheduler.Event) {
	dataMap, _ := event.Data.(map[string]interface{})
	if dataMap == nil {
		dataMap = map[string]interface{}{}
	}
	sourceName, _ := dataMap["source_name"].(string)
	platform, _ := dataMap["platform"].(string)
	if platform == "" {
		platform = "bilibili"
	}

	alert := SystemAlert{
		Key:      "bili_credential_invalid",
		Type:     "credential_invalid",
		Title:    "B 站登录凭据失效",
		Message:  "调度器同步时检测到 B 站凭据失效，请重新登录以恢复同步。",
		Severity: "warning",
		Action:   "/config?tab=bilibili",
		Data:     dataMap,
	}
	s.pushAlert(alert)

	text := "⚠️ video-sync 告警：B 站登录凭据已失效，调度器同步任务无法继续。\n" +
		"请前往后台 系统配置 → B 站账号 重新登录。"
	if sourceName != "" {
		text += "\n触发视频源：" + sourceName
	}
	// 节流：同 key 至少间隔 6 小时再次通过 Telegram 提醒
	s.notifyTelegramAdmins(text, "bili_credential_invalid", 6*time.Hour)
}
