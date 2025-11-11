<template>
  <el-card class="task-queue">
    <template #header>
      <div class="card-header">
        <span>任务队列</span>
        <div class="header-actions">
          <el-button
            text
            @click="refreshTasks"
            :loading="loading"
            :icon="Refresh"
          >
            刷新
          </el-button>
          <el-dropdown @command="handleBatchAction">
            <el-button type="primary" size="small">
              批量操作
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="pause-all">暂停全部</el-dropdown-item>
                <el-dropdown-item command="resume-all">继续全部</el-dropdown-item>
                <el-dropdown-item command="cancel-all" divided>取消全部</el-dropdown-item>
                <el-dropdown-item command="clear-completed">清除已完成</el-dropdown-item>
                <el-dropdown-item command="clear-failed">清除失败</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </template>

    <!-- 统计信息 -->
    <div class="queue-stats" v-if="summary">
      <el-statistic
        title="队列中"
        :value="summary.queued"
        :value-style="{ color: '#409eff' }"
      >
        <template #prefix>
          <el-icon><List /></el-icon>
        </template>
      </el-statistic>
      <el-statistic
        title="运行中"
        :value="summary.running"
        :value-style="{ color: '#67c23a' }"
      >
        <template #prefix>
          <el-icon><VideoPlay /></el-icon>
        </template>
      </el-statistic>
      <el-statistic
        title="已完成"
        :value="summary.completed"
        :value-style="{ color: '#909399' }"
      >
        <template #prefix>
          <el-icon><Select /></el-icon>
        </template>
      </el-statistic>
      <el-statistic
        title="失败"
        :value="summary.failed"
        :value-style="{ color: '#f56c6c' }"
      >
        <template #prefix>
          <el-icon><CircleClose /></el-icon>
        </template>
      </el-statistic>
      <el-statistic
        title="总计"
        :value="summary.total"
      >
        <template #prefix>
          <el-icon><Files /></el-icon>
        </template>
      </el-statistic>
    </div>

    <el-divider />

    <!-- 过滤和排序 -->
    <div class="queue-filters">
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <el-tab-pane label="全部" name="all" />
        <el-tab-pane label="队列中" name="queued" />
        <el-tab-pane label="运行中" name="running" />
        <el-tab-pane label="已暂停" name="paused" />
        <el-tab-pane label="已完成" name="completed" />
        <el-tab-pane label="失败" name="failed" />
      </el-tabs>

      <div class="filter-controls">
        <el-select
          v-model="sortBy"
          placeholder="排序方式"
          size="small"
          style="width: 140px"
          @change="handleSortChange"
        >
          <el-option label="创建时间" value="created_at" />
          <el-option label="优先级" value="priority" />
          <el-option label="状态" value="status" />
        </el-select>

        <el-select
          v-model="sortOrder"
          size="small"
          style="width: 100px"
          @change="handleSortChange"
        >
          <el-option label="降序" value="desc" />
          <el-option label="升序" value="asc" />
        </el-select>
      </div>
    </div>

    <!-- 任务列表 -->
    <div class="task-list" v-loading="loading">
      <div v-if="filteredTasks.length === 0" class="empty-state">
        <el-empty
          :description="getEmptyDescription()"
          :image-size="120"
        />
      </div>

      <TaskCard
        v-for="task in paginatedTasks"
        :key="task.id"
        :task="task"
        :progress="getTaskProgress(task.id)"
        :download-speed="getTaskSpeed(task.id)"
        :eta="getTaskEta(task.id)"
        @pause="handlePause"
        @resume="handleResume"
        @retry="handleRetry"
        @cancel="handleCancel"
        @remove="handleRemove"
      />
    </div>

    <!-- 分页 -->
    <div class="queue-pagination" v-if="filteredTasks.length > pageSize">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        :total="filteredTasks.length"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Refresh,
  ArrowDown,
  List,
  VideoPlay,
  Select,
  CircleClose,
  Files
} from '@element-plus/icons-vue'
import TaskCard from './TaskCard.vue'
import type { Task, TasksSummary } from '@/types'
import { getTasksSummary, getTasks } from '@/api/scheduler'

interface Props {
  autoRefresh?: boolean
  refreshInterval?: number
}

const props = withDefaults(defineProps<Props>(), {
  autoRefresh: true,
  refreshInterval: 5000
})

// 状态
const loading = ref(false)
const tasks = ref<Task[]>([])
const summary = ref<TasksSummary | null>(null)
const activeTab = ref('all')
const sortBy = ref('created_at')
const sortOrder = ref<'asc' | 'desc'>('desc')
const currentPage = ref(1)
const pageSize = ref(20)

// 任务进度信息（这些应该从 WebSocket 或轮询获取）
const taskProgress = ref<Record<string, number>>({})
const taskSpeed = ref<Record<string, string>>({})
const taskEta = ref<Record<string, string>>({})

let refreshTimer: number | null = null

// 获取任务进度
const getTaskProgress = (taskId: string) => taskProgress.value[taskId] || 0
const getTaskSpeed = (taskId: string) => taskSpeed.value[taskId] || ''
const getTaskEta = (taskId: string) => taskEta.value[taskId] || ''

// 过滤任务
const filteredTasks = computed(() => {
  let filtered = tasks.value

  // 按状态过滤
  if (activeTab.value !== 'all') {
    filtered = filtered.filter(task => task.status === activeTab.value)
  }

  // 排序
  filtered = [...filtered].sort((a, b) => {
    let aVal: any, bVal: any

    switch (sortBy.value) {
      case 'created_at':
        aVal = new Date(a.created_at).getTime()
        bVal = new Date(b.created_at).getTime()
        break
      case 'priority':
        aVal = a.priority
        bVal = b.priority
        break
      case 'status':
        aVal = a.status
        bVal = b.status
        break
      default:
        return 0
    }

    return sortOrder.value === 'desc' ? bVal - aVal : aVal - bVal
  })

  return filtered
})

// 分页任务
const paginatedTasks = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredTasks.value.slice(start, end)
})

// 空状态描述
const getEmptyDescription = () => {
  switch (activeTab.value) {
    case 'queued':
      return '暂无队列中的任务'
    case 'running':
      return '暂无运行中的任务'
    case 'paused':
      return '暂无已暂停的任务'
    case 'completed':
      return '暂无已完成的任务'
    case 'failed':
      return '暂无失败的任务'
    default:
      return '暂无任务'
  }
}

// 获取任务列表
const fetchTasks = async () => {
  try {
    loading.value = true
    const params = {
      status: activeTab.value === 'all' ? undefined : activeTab.value,
      sort_by: sortBy.value,
      sort_order: sortOrder.value
    }
    const res = await getTasks(params)
    tasks.value = res || []
  } catch (error: any) {
    ElMessage.error(error.message || '获取任务列表失败')
  } finally {
    loading.value = false
  }
}

// 获取任务统计
const fetchSummary = async () => {
  try {
    const res = await getTasksSummary()
    summary.value = res
  } catch (error: any) {
    ElMessage.error(error.message || '获取任务统计失败')
  }
}

// 刷新任务
const refreshTasks = async () => {
  await Promise.all([fetchTasks(), fetchSummary()])
}

// Tab 切换
const handleTabChange = async () => {
  currentPage.value = 1
  await fetchTasks()
}

// 排序改变
const handleSortChange = () => {
  currentPage.value = 1
}

// 分页改变
const handleSizeChange = () => {
  currentPage.value = 1
}

const handleCurrentChange = () => {
  // 页码改变时无需额外操作
}

// 任务操作
const handlePause = async (taskId: string) => {
  try {
    // TODO: 实现暂停任务的 API
    // await pauseTask(taskId)
    ElMessage.success('任务已暂停')
    await refreshTasks()
  } catch (error: any) {
    ElMessage.error(error.message || '暂停任务失败')
  }
}

const handleResume = async (taskId: string) => {
  try {
    // TODO: 实现继续任务的 API
    // await resumeTask(taskId)
    ElMessage.success('任务已继续')
    await refreshTasks()
  } catch (error: any) {
    ElMessage.error(error.message || '继续任务失败')
  }
}

const handleRetry = async (taskId: string) => {
  try {
    // TODO: 实现重试任务的 API
    // await retryTask(taskId)
    ElMessage.success('任务已重试')
    await refreshTasks()
  } catch (error: any) {
    ElMessage.error(error.message || '重试任务失败')
  }
}

const handleCancel = async (taskId: string) => {
  try {
    await ElMessageBox.confirm('确定要取消此任务吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })

    // TODO: 实现取消任务的 API
    // await cancelTask(taskId)
    ElMessage.success('任务已取消')
    await refreshTasks()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '取消任务失败')
    }
  }
}

const handleRemove = async (taskId: string) => {
  try {
    await ElMessageBox.confirm('确定要移除此任务吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })

    // TODO: 实现移除任务的 API
    // await removeTask(taskId)
    ElMessage.success('任务已移除')
    await refreshTasks()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '移除任务失败')
    }
  }
}

// 批量操作
const handleBatchAction = async (command: string) => {
  try {
    switch (command) {
      case 'pause-all':
        await ElMessageBox.confirm('确定要暂停所有运行中的任务吗？', '提示', {
          type: 'warning'
        })
        // TODO: 实现批量暂停
        ElMessage.success('已暂停所有任务')
        break
      case 'resume-all':
        await ElMessageBox.confirm('确定要继续所有已暂停的任务吗？', '提示', {
          type: 'warning'
        })
        // TODO: 实现批量继续
        ElMessage.success('已继续所有任务')
        break
      case 'cancel-all':
        await ElMessageBox.confirm('确定要取消所有进行中的任务吗？此操作不可恢复！', '警告', {
          type: 'error'
        })
        // TODO: 实现批量取消
        ElMessage.success('已取消所有任务')
        break
      case 'clear-completed':
        await ElMessageBox.confirm('确定要清除所有已完成的任务吗？', '提示', {
          type: 'warning'
        })
        // TODO: 实现清除已完成
        ElMessage.success('已清除已完成任务')
        break
      case 'clear-failed':
        await ElMessageBox.confirm('确定要清除所有失败的任务吗？', '提示', {
          type: 'warning'
        })
        // TODO: 实现清除失败
        ElMessage.success('已清除失败任务')
        break
    }
    await refreshTasks()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '操作失败')
    }
  }
}

// 启动自动刷新
const startAutoRefresh = () => {
  if (props.autoRefresh && !refreshTimer) {
    refreshTimer = window.setInterval(() => {
      refreshTasks()
    }, props.refreshInterval)
  }
}

// 停止自动刷新
const stopAutoRefresh = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

onMounted(() => {
  refreshTasks()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})

// 导出方法供父组件调用
defineExpose({
  refreshTasks
})
</script>

<style scoped lang="scss">
.task-queue {
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-weight: 600;

    .header-actions {
      display: flex;
      gap: 8px;
    }
  }

  .queue-stats {
    display: flex;
    justify-content: space-around;
    gap: 24px;
    padding: 16px 0;

    :deep(.el-statistic) {
      text-align: center;

      .el-statistic__head {
        font-size: 13px;
        color: #909399;
        margin-bottom: 8px;
      }

      .el-statistic__content {
        font-size: 24px;
        font-weight: 600;
      }
    }
  }

  .queue-filters {
    margin-bottom: 20px;

    .el-tabs {
      margin-bottom: 16px;
    }

    .filter-controls {
      display: flex;
      gap: 12px;
      align-items: center;
    }
  }

  .task-list {
    min-height: 200px;

    .empty-state {
      padding: 40px 0;
    }
  }

  .queue-pagination {
    display: flex;
    justify-content: center;
    margin-top: 24px;
    padding-top: 16px;
    border-top: 1px solid #ebeef5;
  }
}
</style>
