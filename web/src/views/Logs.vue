<template>
  <div class="logs">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>实时日志</span>
          <div class="header-actions">
            <el-select
              v-model="logLevel"
              placeholder="日志级别"
              style="width: 120px; margin-right: 10px"
            >
              <el-option label="全部" value="" />
              <el-option label="DEBUG" value="debug" />
              <el-option label="INFO" value="info" />
              <el-option label="WARN" value="warn" />
              <el-option label="ERROR" value="error" />
            </el-select>
            <el-button @click="clearLogs">
              <el-icon><Delete /></el-icon>
              清空
            </el-button>
            <el-button :type="isConnected ? 'danger' : 'primary'" @click="toggleConnection">
              <el-icon>
                <component :is="isConnected ? 'VideoPause' : 'VideoPlay'" />
              </el-icon>
              {{ isConnected ? '停止' : '开始' }}
            </el-button>
          </div>
        </div>
      </template>

      <div ref="logContainer" class="log-container">
        <div
          v-for="(log, index) in filteredLogs"
          :key="index"
          :class="['log-item', `log-${log.level}`]"
        >
          <span class="log-time">{{ log.time }}</span>
          <span class="log-level">[{{ log.level.toUpperCase() }}]</span>
          <span class="log-message">{{ log.message }}</span>
        </div>
        <el-empty v-if="filteredLogs.length === 0" description="暂无日志" />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'

defineOptions({
  name: 'Logs'
})

interface LogEntry {
  time: string
  level: string
  message: string
}

const logContainer = ref<HTMLElement>()
const logs = ref<LogEntry[]>([])
const logLevel = ref('')
const isConnected = ref(false)
let ws: WebSocket | null = null

// 过滤后的日志
const filteredLogs = computed(() => {
  if (!logLevel.value) {
    return logs.value
  }
  return logs.value.filter(log => log.level === logLevel.value)
})

// 连接 WebSocket
const connectWebSocket = () => {
  try {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    ws = new WebSocket(`${protocol}//${host}/api/ws`)

    ws.onopen = () => {
      isConnected.value = true
      ElMessage.success('日志连接已建立')
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        logs.value.push({
          time: dayjs().format('HH:mm:ss'),
          level: data.level || 'info',
          message: data.message || event.data
        })

        // 限制日志数量
        if (logs.value.length > 1000) {
          logs.value = logs.value.slice(-1000)
        }

        // 自动滚动到底部
        nextTick(() => {
          if (logContainer.value) {
            logContainer.value.scrollTop = logContainer.value.scrollHeight
          }
        })
      } catch (error) {
        // 如果不是 JSON，直接添加
        logs.value.push({
          time: dayjs().format('HH:mm:ss'),
          level: 'info',
          message: event.data
        })
      }
    }

    ws.onerror = () => {
      ElMessage.error('日志连接错误')
    }

    ws.onclose = () => {
      isConnected.value = false
      ElMessage.warning('日志连接已断开')
    }
  } catch (error) {
    ElMessage.error('无法连接日志服务')
    console.error('WebSocket连接失败:', error)
  }
}

// 断开 WebSocket
const disconnectWebSocket = () => {
  if (ws) {
    ws.close()
    ws = null
  }
  isConnected.value = false
}

// 切换连接状态
const toggleConnection = () => {
  if (isConnected.value) {
    disconnectWebSocket()
  } else {
    connectWebSocket()
  }
}

// 清空日志
const clearLogs = () => {
  logs.value = []
}

// 组件卸载时断开连接
onUnmounted(() => {
  disconnectWebSocket()
})
</script>

<style scoped>
.logs {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.log-container {
  height: calc(100vh - 280px);
  overflow-y: auto;
  background: #1e1e1e;
  padding: 15px;
  border-radius: 4px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.log-item {
  padding: 4px 0;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-time {
  color: #858585;
  margin-right: 8px;
}

.log-level {
  margin-right: 8px;
  font-weight: bold;
}

.log-debug .log-level {
  color: #b5cea8;
}

.log-info .log-level {
  color: #4fc3f7;
}

.log-warn .log-level {
  color: #ffa726;
}

.log-error .log-level {
  color: #ef5350;
}

.log-message {
  color: #d4d4d4;
}
</style>
