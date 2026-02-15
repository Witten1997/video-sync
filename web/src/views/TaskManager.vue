<template>
  <div class="task-manager">
    <div class="page-header">
      <h2>任务管理</h2>
      <p class="page-description">管理下载任务和调度器</p>
    </div>

    <!-- 调度器控制 -->
    <SchedulerControl ref="schedulerControlRef" />

    <!-- 任务队列 -->
    <TaskQueue ref="taskQueueRef" :auto-refresh="true" :refresh-interval="5000" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import SchedulerControl from '@/components/SchedulerControl.vue'
import TaskQueue from '@/components/TaskQueue.vue'

const schedulerControlRef = ref<InstanceType<typeof SchedulerControl>>()
const taskQueueRef = ref<InstanceType<typeof TaskQueue>>()

let ws: WebSocket | null = null

// WebSocket 消息处理
const handleWebSocketMessage = (event: MessageEvent) => {
  try {
    const message = JSON.parse(event.data)

    // 根据消息类型更新组件
    switch (message.type) {
      case 'config_updated':
        // 配置更新时刷新调度器状态
        console.log('收到配置更新事件，刷新调度器状态')
        schedulerControlRef.value?.refreshStatus()
        break

      case 'scheduler_started':
      case 'scheduler_stopped':
      case 'sync_started':
      case 'sync_completed':
      case 'sync_failed':
        // 刷新调度器状态
        schedulerControlRef.value?.refreshStatus()
        break

      case 'task_created':
      case 'task_started':
      case 'task_completed':
      case 'task_failed':
      case 'task_cancelled':
        // 刷新任务队列
        taskQueueRef.value?.refreshTasks()
        break

      case 'task_progress':
        // 更新任务进度（可以添加更精细的处理）
        taskQueueRef.value?.refreshTasks()
        break
    }
  } catch (error) {
    console.error('Failed to parse WebSocket message:', error)
  }
}

// 连接 WebSocket
const connectWebSocket = () => {
  try {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    ws = new WebSocket(`${protocol}//${host}/api/ws`)

    ws.onopen = () => {
      console.log('WebSocket connected')
    }

    ws.onmessage = handleWebSocketMessage

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    ws.onclose = () => {
      console.log('WebSocket disconnected')
      // 5秒后尝试重连
      setTimeout(() => {
        if (!ws || ws.readyState === WebSocket.CLOSED) {
          connectWebSocket()
        }
      }, 5000)
    }
  } catch (error) {
    console.error('Failed to connect WebSocket:', error)
  }
}

// 断开 WebSocket
const disconnectWebSocket = () => {
  if (ws) {
    ws.close()
    ws = null
  }
}

onMounted(() => {
  connectWebSocket()
})

onUnmounted(() => {
  disconnectWebSocket()
})
</script>

<style scoped>
.task-manager {
  padding: 32px;
}
.page-header {
  margin-bottom: 24px;
}
.page-header h2 {
  margin: 0 0 8px 0;
  font-size: 1.25rem;
  font-weight: 700;
  color: #1e293b;
}
.page-description {
  margin: 0;
  font-size: 0.875rem;
  color: #64748b;
}
</style>
