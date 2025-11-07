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

// WebSocket 消息处理
const handleWebSocketMessage = (event: MessageEvent) => {
  try {
    const message = JSON.parse(event.data)

    // 根据消息类型更新组件
    switch (message.type) {
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

onMounted(() => {
  // TODO: 连接 WebSocket 并监听消息
  // const ws = useWebSocket()
  // ws.addEventListener('message', handleWebSocketMessage)
})

onUnmounted(() => {
  // TODO: 清理 WebSocket 监听
  // const ws = useWebSocket()
  // ws.removeEventListener('message', handleWebSocketMessage)
})
</script>

<style scoped lang="scss">
.task-manager {
  padding: 24px;

  .page-header {
    margin-bottom: 24px;

    h2 {
      margin: 0 0 8px 0;
      font-size: 24px;
      font-weight: 600;
      color: #303133;
    }

    .page-description {
      margin: 0;
      font-size: 14px;
      color: #909399;
    }
  }

  .scheduler-control {
    margin-bottom: 24px;
  }
}
</style>
