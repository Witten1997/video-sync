<template>
  <el-card class="task-card" :class="`status-${task.status}`">
    <div class="task-content">
      <!-- 视频封面 -->
      <div class="task-cover">
        <el-image
          :src="coverUrl"
          fit="cover"
          lazy
        >
          <template #error>
            <div class="image-slot">
              <el-icon><Picture /></el-icon>
            </div>
          </template>
        </el-image>
        <div class="type-badge">
          <el-tag :type="getTypeColor(task.type)" size="small">
            {{ getTypeText(task.type) }}
          </el-tag>
        </div>
      </div>

      <!-- 任务信息 -->
      <div class="task-info">
        <div class="task-header">
          <h4 class="task-title" :title="task.video.name">
            {{ task.video.name }}
          </h4>
          <el-tag
            :type="getStatusColor(task.status)"
            size="small"
            effect="dark"
          >
            {{ getStatusText(task.status) }}
          </el-tag>
        </div>

        <div class="task-meta">
          <div class="meta-item">
            <el-icon><User /></el-icon>
            <span>{{ task.video.upper_name }}</span>
          </div>
          <div class="meta-item" v-if="task.page">
            <el-icon><VideoCamera /></el-icon>
            <span>P{{ task.page.pid }}: {{ task.page.name }}</span>
          </div>
          <div class="meta-item">
            <el-icon><Clock /></el-icon>
            <span>{{ formatTime(task.created_at) }}</span>
          </div>
        </div>

        <!-- 进度条（仅运行中显示） -->
        <div class="task-progress" v-if="task.status === 'running'">
          <el-progress
            :percentage="progress"
            :status="progress === 100 ? 'success' : undefined"
            :stroke-width="6"
          />
          <div class="progress-info">
            <span class="speed">{{ downloadSpeed }}</span>
            <span class="eta">{{ eta }}</span>
          </div>
        </div>

        <!-- 错误信息 -->
        <div class="task-error" v-if="task.status === 'failed' && task.error_msg">
          <el-alert
            :title="task.error_msg"
            type="error"
            :closable="false"
            show-icon
          />
        </div>

        <!-- 重试信息 -->
        <div class="task-retry" v-if="task.retry_count > 0">
          <el-text type="warning" size="small">
            已重试 {{ task.retry_count }}/{{ task.max_retries }} 次
          </el-text>
        </div>

        <!-- 操作按钮 -->
        <div class="task-actions">
          <el-button-group>
            <el-button
              v-if="canPause"
              size="small"
              :icon="VideoPause"
              @click="handlePause"
            >
              暂停
            </el-button>
            <el-button
              v-if="canResume"
              size="small"
              type="primary"
              :icon="VideoPlay"
              @click="handleResume"
            >
              继续
            </el-button>
            <el-button
              v-if="canRetry"
              size="small"
              type="warning"
              :icon="Refresh"
              @click="handleRetry"
            >
              重试
            </el-button>
            <el-button
              v-if="canCancel"
              size="small"
              type="danger"
              :icon="Close"
              @click="handleCancel"
            >
              取消
            </el-button>
            <el-button
              v-if="canRemove"
              size="small"
              :icon="Delete"
              @click="handleRemove"
            >
              移除
            </el-button>
          </el-button-group>

          <!-- 优先级标签 -->
          <el-tag
            v-if="task.priority > 0"
            size="small"
            :type="getPriorityColor(task.priority)"
          >
            优先级: {{ task.priority }}
          </el-tag>
        </div>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  Picture,
  User,
  VideoCamera,
  Clock,
  VideoPause,
  VideoPlay,
  Refresh,
  Close,
  Delete
} from '@element-plus/icons-vue'
import type { Task } from '@/types'
import { getProxiedImageUrl } from '@/utils/image'

interface Props {
  task: Task
  progress?: number
  downloadSpeed?: string
  eta?: string
}

interface Emits {
  (e: 'pause', taskId: string): void
  (e: 'resume', taskId: string): void
  (e: 'retry', taskId: string): void
  (e: 'cancel', taskId: string): void
  (e: 'remove', taskId: string): void
}

const props = withDefaults(defineProps<Props>(), {
  progress: 0,
  downloadSpeed: '',
  eta: ''
})

const emit = defineEmits<Emits>()

// 获取代理后的封面图片URL
const coverUrl = computed(() => getProxiedImageUrl(props.task.video.cover))

// 任务类型文本
const getTypeText = (type: string): string => {
  const map: Record<string, string> = {
    video: '视频',
    page: '分P',
    collection: '合集'
  }
  return map[type] || type
}

// 任务类型颜色
const getTypeColor = (type: string): '' | 'success' | 'info' | 'warning' => {
  const map: Record<string, '' | 'success' | 'info' | 'warning'> = {
    video: 'success',
    page: 'info',
    collection: 'warning'
  }
  return map[type] || ''
}

// 状态文本
const getStatusText = (status: string): string => {
  const map: Record<string, string> = {
    pending: '等待中',
    queued: '队列中',
    running: '下载中',
    paused: '已暂停',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消'
  }
  return map[status] || status
}

// 状态颜色
const getStatusColor = (status: string): '' | 'success' | 'info' | 'warning' | 'danger' => {
  const map: Record<string, '' | 'success' | 'info' | 'warning' | 'danger'> = {
    pending: 'info',
    queued: 'info',
    running: '',
    paused: 'warning',
    completed: 'success',
    failed: 'danger',
    cancelled: 'info'
  }
  return map[status] || 'info'
}

// 优先级颜色
const getPriorityColor = (priority: number): '' | 'success' | 'warning' | 'danger' => {
  if (priority >= 8) return 'danger'
  if (priority >= 5) return 'warning'
  return 'success'
}

// 格式化时间
const formatTime = (timeStr: string): string => {
  const date = new Date(timeStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 1000 / 60)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  const days = Math.floor(hours / 24)
  return `${days}天前`
}

// 按钮显示逻辑
const canPause = computed(() => props.task.status === 'running')
const canResume = computed(() => props.task.status === 'paused')
const canRetry = computed(() =>
  props.task.status === 'failed' && props.task.retry_count < props.task.max_retries
)
const canCancel = computed(() =>
  ['pending', 'queued', 'running', 'paused'].includes(props.task.status)
)
const canRemove = computed(() =>
  ['completed', 'failed', 'cancelled'].includes(props.task.status)
)

// 操作处理
const handlePause = () => emit('pause', props.task.id)
const handleResume = () => emit('resume', props.task.id)
const handleRetry = () => emit('retry', props.task.id)
const handleCancel = () => emit('cancel', props.task.id)
const handleRemove = () => emit('remove', props.task.id)
</script>

<style scoped>
.task-card {
  margin-bottom: 16px;
  transition: box-shadow 0.3s;
}

.task-card:hover {
  box-shadow: 0 4px 12px 0 rgba(0, 0, 0, 0.08);
}

.task-card.status-running {
  border-left: 4px solid #3b82f6;
}

.task-card.status-completed {
  border-left: 4px solid #22c55e;
}

.task-card.status-failed {
  border-left: 4px solid #ef4444;
}

.task-card.status-paused {
  border-left: 4px solid #f59e0b;
}

.task-content {
  display: flex;
  gap: 16px;
}

.task-cover {
  position: relative;
  flex-shrink: 0;
  width: 160px;
  height: 90px;
  border-radius: 8px;
  overflow: hidden;
}

.task-cover .el-image {
  width: 100%;
  height: 100%;
}

.image-slot {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  height: 100%;
  background-color: #f8fafc;
  color: #94a3b8;
  font-size: 32px;
}

.type-badge {
  position: absolute;
  top: 4px;
  right: 4px;
}

.task-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}

.task-title {
  flex: 1;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-meta {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #64748b;
}

.meta-item .el-icon {
  color: #94a3b8;
}

.meta-item span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  margin-top: 4px;
  font-size: 12px;
  color: #94a3b8;
}

.progress-info .speed {
  font-weight: 500;
  color: #3b82f6;
}

.task-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 8px;
  border-top: 1px solid #f1f5f9;
}
</style>
