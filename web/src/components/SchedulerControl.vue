<template>
  <el-card class="scheduler-control">
    <template #header>
      <div class="card-header">
        <span>调度器控制</span>
        <el-button
          text
          @click="refreshStatus"
          :loading="loading"
          :icon="Refresh"
        />
      </div>
    </template>

    <div class="control-content">
      <!-- 状态指示器 -->
      <div class="status-section">
        <div class="status-indicator">
          <el-tag
            :type="status?.is_running ? 'success' : 'info'"
            size="large"
            effect="dark"
          >
            <el-icon class="status-icon">
              <component :is="status?.is_running ? VideoPlay : VideoPause" />
            </el-icon>
            {{ status?.is_running ? '运行中' : '已停止' }}
          </el-tag>
        </div>

        <div class="status-info">
          <div class="info-item">
            <span class="label">同步间隔：</span>
            <span class="value">{{ formatInterval(status?.interval || 0) }}</span>
          </div>
          <div class="info-item" v-if="status?.last_run_at">
            <span class="label">上次运行：</span>
            <span class="value">{{ formatTime(status.last_run_at) }}</span>
          </div>
          <div class="info-item" v-if="status?.next_run_at">
            <span class="label">下次运行：</span>
            <span class="value">{{ formatTime(status.next_run_at) }}</span>
          </div>
          <div class="info-item" v-if="status?.current_sync_id">
            <span class="label">当前任务：</span>
            <span class="value">{{ status.current_sync_id }}</span>
          </div>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="action-buttons">
        <el-button
          v-if="!status?.is_running"
          type="success"
          :icon="VideoPlay"
          @click="handleStart"
          :loading="actionLoading"
        >
          启动调度器
        </el-button>
        <el-button
          v-else
          type="danger"
          :icon="VideoPause"
          @click="handleStop"
          :loading="actionLoading"
        >
          停止调度器
        </el-button>

        <el-button
          type="primary"
          :icon="Refresh"
          @click="handleTrigger"
          :loading="triggerLoading"
          :disabled="!!status?.current_sync_id"
        >
          立即同步
        </el-button>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  VideoPlay,
  VideoPause,
  Refresh
} from '@element-plus/icons-vue'
import {
  getSchedulerStatus,
  startScheduler,
  stopScheduler,
  triggerSync
} from '@/api/scheduler'
import type { SchedulerStatus } from '@/types'

const status = ref<SchedulerStatus | null>(null)
const loading = ref(false)
const actionLoading = ref(false)
const triggerLoading = ref(false)

// 获取状态
const fetchStatus = async () => {
  try {
    loading.value = true
    const res = await getSchedulerStatus()
    status.value = res
  } catch (error: any) {
    ElMessage.error(error.message || '获取调度器状态失败')
  } finally {
    loading.value = false
  }
}

// 刷新状态
const refreshStatus = () => {
  fetchStatus()
}

// 启动调度器
const handleStart = async () => {
  try {
    actionLoading.value = true
    await startScheduler()
    ElMessage.success('调度器已启动')
    await fetchStatus()
  } catch (error: any) {
    ElMessage.error(error.message || '启动调度器失败')
  } finally {
    actionLoading.value = false
  }
}

// 停止调度器
const handleStop = async () => {
  try {
    actionLoading.value = true
    await stopScheduler()
    ElMessage.success('调度器已停止')
    await fetchStatus()
  } catch (error: any) {
    ElMessage.error(error.message || '停止调度器失败')
  } finally {
    actionLoading.value = false
  }
}

// 手动触发同步
const handleTrigger = async () => {
  try {
    triggerLoading.value = true
    const res = await triggerSync()
    ElMessage.success(`同步任务已触发: ${res.sync_id}`)
    await fetchStatus()
  } catch (error: any) {
    ElMessage.error(error.message || '触发同步失败')
  } finally {
    triggerLoading.value = false
  }
}

// 格式化间隔时间
const formatInterval = (seconds: number): string => {
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟`
  return `${Math.floor(seconds / 3600)}小���`
}

// 格式化时间
const formatTime = (timeStr: string): string => {
  const date = new Date(timeStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  // 如果是未来时间（下次运行）
  if (diff < 0) {
    const futureDiff = -diff
    const minutes = Math.floor(futureDiff / 1000 / 60)
    if (minutes < 60) return `还有${minutes}分钟`
    const hours = Math.floor(minutes / 60)
    return `还有${hours}小时${minutes % 60}分钟`
  }

  // 如果是过去时间（上次运行）
  const minutes = Math.floor(diff / 1000 / 60)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  const days = Math.floor(hours / 24)
  return `${days}天前`
}

onMounted(() => {
  fetchStatus()
})

// 导出方法供父组件调用
defineExpose({
  refreshStatus
})
</script>

<style scoped>
.scheduler-control {
  margin-bottom: 24px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
  font-size: 14px;
}

.control-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.status-section {
  display: flex;
  gap: 32px;
  align-items: flex-start;
}

.status-indicator .status-icon {
  margin-right: 4px;
}

.status-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-item {
  display: flex;
  align-items: center;
  font-size: 14px;
}

.info-item .label {
  color: #64748b;
  margin-right: 8px;
  min-width: 80px;
}

.info-item .value {
  color: #1e293b;
  font-weight: 500;
}

.action-buttons {
  display: flex;
  gap: 12px;
  padding-top: 12px;
  border-top: 1px solid #f1f5f9;
}
</style>
