<template>
  <div class="sync-logs">
    <div class="page-header">
      <h2>同步日志</h2>
      <p class="page-description">查看视频源同步历史记录</p>
    </div>

    <!-- 统计信息 -->
    <el-card class="stats-card" v-loading="statsLoading">
      <template #header>
        <div class="card-header">
          <span>统计信息</span>
          <el-select
            v-model="statsPeriod"
            size="small"
            style="width: 120px"
            @change="fetchStats"
          >
            <el-option label="最近1天" value="1d" />
            <el-option label="最近7天" value="7d" />
            <el-option label="最近30天" value="30d" />
            <el-option label="全部" value="all" />
          </el-select>
        </div>
      </template>

      <div class="stats-content" v-if="stats">
        <el-row :gutter="24">
          <el-col :span="6">
            <el-statistic
              title="总同步次数"
              :value="stats.total_syncs"
            >
              <template #prefix>
                <el-icon><Refresh /></el-icon>
              </template>
            </el-statistic>
          </el-col>
          <el-col :span="6">
            <el-statistic
              title="成功次数"
              :value="stats.successful_syncs"
              :value-style="{ color: '#67c23a' }"
            >
              <template #prefix>
                <el-icon><CircleCheck /></el-icon>
              </template>
            </el-statistic>
          </el-col>
          <el-col :span="6">
            <el-statistic
              title="失败次数"
              :value="stats.failed_syncs"
              :value-style="{ color: '#f56c6c' }"
            >
              <template #prefix>
                <el-icon><CircleClose /></el-icon>
              </template>
            </el-statistic>
          </el-col>
          <el-col :span="6">
            <el-statistic
              title="成功率"
              :value="stats.success_rate"
              suffix="%"
              :precision="2"
              :value-style="{ color: getSuccessRateColor(stats.success_rate) }"
            >
              <template #prefix>
                <el-icon><TrendCharts /></el-icon>
              </template>
            </el-statistic>
          </el-col>
        </el-row>

        <el-divider />

        <el-row :gutter="24">
          <el-col :span="8">
            <el-statistic
              title="发现视频数"
              :value="stats.total_videos_found"
            />
          </el-col>
          <el-col :span="8">
            <el-statistic
              title="新增视频数"
              :value="stats.total_videos_new"
              :value-style="{ color: '#409eff' }"
            />
          </el-col>
          <el-col :span="8">
            <el-statistic
              title="已入队数"
              :value="stats.total_videos_queued"
              :value-style="{ color: '#67c23a' }"
            />
          </el-col>
        </el-row>
      </div>
    </el-card>

    <!-- 日志列表 -->
    <el-card class="logs-card">
      <template #header>
        <div class="card-header">
          <span>同步日志</span>
          <el-button
            text
            @click="fetchLogs"
            :loading="logsLoading"
            :icon="Refresh"
          >
            刷新
          </el-button>
        </div>
      </template>

      <!-- 过滤条件 -->
      <div class="filters">
        <el-form :inline="true" :model="filters" size="small">
          <el-form-item label="触发方式">
            <el-select v-model="filters.trigger_type" @change="handleFilterChange">
              <el-option label="全部" value="all" />
              <el-option label="自动触发" value="auto" />
              <el-option label="手动触发" value="manual" />
            </el-select>
          </el-form-item>

          <el-form-item label="状态">
            <el-select v-model="filters.status" @change="handleFilterChange">
              <el-option label="全部" value="all" />
              <el-option label="运行中" value="running" />
              <el-option label="已完成" value="completed" />
              <el-option label="失败" value="failed" />
              <el-option label="已取消" value="cancelled" />
            </el-select>
          </el-form-item>

          <el-form-item label="排序">
            <el-select v-model="filters.sort_by" @change="handleFilterChange">
              <el-option label="开始时间" value="start_at" />
              <el-option label="耗时" value="duration_ms" />
              <el-option label="视频数" value="videos_found" />
            </el-select>
          </el-form-item>

          <el-form-item>
            <el-select v-model="filters.sort_order" @change="handleFilterChange">
              <el-option label="降序" value="desc" />
              <el-option label="升序" value="asc" />
            </el-select>
          </el-form-item>
        </el-form>
      </div>

      <!-- 日志表格 -->
      <el-table
        :data="logs"
        v-loading="logsLoading"
        stripe
        @row-click="handleRowClick"
        style="cursor: pointer"
      >
        <el-table-column prop="task_id" label="任务ID" width="200" />

        <el-table-column label="触发方式" width="100">
          <template #default="{ row }">
            <el-tag :type="row.trigger_type === 'auto' ? 'success' : 'primary'" size="small">
              {{ row.trigger_type === 'auto' ? '自动' : '手动' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="开始时间" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.start_at) }}
          </template>
        </el-table-column>

        <el-table-column label="耗时" width="120">
          <template #default="{ row }">
            {{ formatDuration(row.duration_ms) }}
          </template>
        </el-table-column>

        <el-table-column label="视频源" width="120" align="center">
          <template #default="{ row }">
            <span>{{ row.sources_scanned }}/{{ row.sources_total }}</span>
            <el-text
              v-if="row.sources_failed > 0"
              type="danger"
              size="small"
              style="margin-left: 4px"
            >
              ({{ row.sources_failed }}失败)
            </el-text>
          </template>
        </el-table-column>

        <el-table-column label="发现/新增" width="120" align="center">
          <template #default="{ row }">
            <span>{{ row.videos_found }}/{{ row.videos_new }}</span>
          </template>
        </el-table-column>

        <el-table-column label="已过滤/已入队" width="140" align="center">
          <template #default="{ row }">
            <span>{{ row.videos_filtered }}/{{ row.videos_queued }}</span>
          </template>
        </el-table-column>

        <el-table-column label="任务统计" width="140" align="center">
          <template #default="{ row }">
            <el-text type="success">{{ row.tasks_completed }}</el-text>
            /
            <el-text>{{ row.tasks_created }}</el-text>
            <el-text
              v-if="row.tasks_failed > 0"
              type="danger"
              size="small"
              style="margin-left: 4px"
            >
              ({{ row.tasks_failed }}失败)
            </el-text>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.page_size"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 日志详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="同步日志详情"
      width="80%"
      :close-on-click-modal="false"
    >
      <div v-if="selectedLog" v-loading="detailLoading">
        <!-- 基本信息 -->
        <el-descriptions :column="2" border>
          <el-descriptions-item label="任务ID">
            {{ selectedLog.task_id }}
          </el-descriptions-item>
          <el-descriptions-item label="触发方式">
            <el-tag :type="selectedLog.trigger_type === 'auto' ? 'success' : 'primary'" size="small">
              {{ selectedLog.trigger_type === 'auto' ? '自动触发' : '手动触发' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusType(selectedLog.status)">
              {{ getStatusText(selectedLog.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="开始时间">
            {{ formatDateTime(selectedLog.start_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="结束时间">
            {{ selectedLog.end_at ? formatDateTime(selectedLog.end_at) : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="耗时">
            {{ formatDuration(selectedLog.duration_ms) }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- 统计信息 -->
        <el-divider content-position="left">统计信息</el-divider>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-card shadow="never">
              <el-statistic title="视频源扫描" :value="selectedLog.sources_scanned">
                <template #suffix>/ {{ selectedLog.sources_total }}</template>
              </el-statistic>
            </el-card>
          </el-col>
          <el-col :span="8">
            <el-card shadow="never">
              <el-statistic title="发现视频" :value="selectedLog.videos_found" />
            </el-card>
          </el-col>
          <el-col :span="8">
            <el-card shadow="never">
              <el-statistic title="新增视频" :value="selectedLog.videos_new" />
            </el-card>
          </el-col>
        </el-row>

        <el-row :gutter="16" style="margin-top: 16px">
          <el-col :span="8">
            <el-card shadow="never">
              <el-statistic title="已过滤" :value="selectedLog.videos_filtered" />
            </el-card>
          </el-col>
          <el-col :span="8">
            <el-card shadow="never">
              <el-statistic title="已入队" :value="selectedLog.videos_queued" />
            </el-card>
          </el-col>
          <el-col :span="8">
            <el-card shadow="never">
              <el-statistic title="任务完成率">
                <template #default>
                  {{ getCompletionRate(selectedLog) }}%
                </template>
              </el-statistic>
            </el-card>
          </el-col>
        </el-row>

        <!-- 错误信息 -->
        <div v-if="selectedLog.error_message">
          <el-divider content-position="left">错误信息</el-divider>
          <el-alert
            :title="selectedLog.error_message"
            type="error"
            :closable="false"
            show-icon
          />
        </div>

        <!-- 视频源扫描详情 -->
        <div v-if="selectedLog.source_scans && selectedLog.source_scans.length > 0">
          <el-divider content-position="left">视频源扫描详情</el-divider>
          <el-table :data="selectedLog.source_scans" stripe max-height="400">
            <el-table-column prop="source_name" label="视频源" width="200" />
            <el-table-column label="类型" width="100">
              <template #default="{ row }">
                <el-tag size="small">{{ getSourceTypeText(row.source_type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="row.success ? 'success' : 'danger'" size="small">
                  {{ row.success ? '成功' : '失败' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="耗时" width="100">
              <template #default="{ row }">
                {{ formatDuration(row.duration_ms) }}
              </template>
            </el-table-column>
            <el-table-column label="发现/新增" width="120">
              <template #default="{ row }">
                {{ row.videos_found }}/{{ row.videos_new }}
              </template>
            </el-table-column>
            <el-table-column label="已过滤/已入队" width="140">
              <template #default="{ row }">
                {{ row.videos_filtered }}/{{ row.videos_queued }}
              </template>
            </el-table-column>
            <el-table-column prop="error_message" label="错误信息" min-width="200">
              <template #default="{ row }">
                <el-text v-if="row.error_message" type="danger" size="small">
                  {{ row.error_message }}
                </el-text>
                <span v-else>-</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <template #footer>
        <el-button @click="detailDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Refresh,
  CircleCheck,
  CircleClose,
  TrendCharts
} from '@element-plus/icons-vue'
import {
  getSyncLogs,
  getSyncLog,
  getSyncStats
} from '@/api/scheduler'
import type { SyncLog, SyncStats, PageParams } from '@/types'

// 统计信息
const stats = ref<SyncStats | null>(null)
const statsLoading = ref(false)
const statsPeriod = ref<'1d' | '7d' | '30d' | 'all'>('7d')

// 日志列表
const logs = ref<SyncLog[]>([])
const logsLoading = ref(false)
const filters = ref<PageParams & {
  trigger_type?: string
  status?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}>({
  trigger_type: 'all',
  status: 'all',
  sort_by: 'start_at',
  sort_order: 'desc'
})
const pagination = ref({
  page: 1,
  page_size: 20,
  total: 0
})

// 日志详情
const selectedLog = ref<SyncLog | null>(null)
const detailDialogVisible = ref(false)
const detailLoading = ref(false)

// 获取统计信息
const fetchStats = async () => {
  try {
    statsLoading.value = true
    const res = await getSyncStats(statsPeriod.value)
    stats.value = res
  } catch (error: any) {
    ElMessage.error(error.message || '获取统计信息失败')
  } finally {
    statsLoading.value = false
  }
}

// 获取日志列表
const fetchLogs = async () => {
  try {
    logsLoading.value = true
    const params = {
      page: pagination.value.page,
      page_size: pagination.value.page_size,
      trigger_type: filters.value.trigger_type,
      status: filters.value.status,
      sort_by: filters.value.sort_by,
      sort_order: filters.value.sort_order
    }
    const res = await getSyncLogs(params)
    logs.value = res.items
    pagination.value.total = res.total
  } catch (error: any) {
    ElMessage.error(error.message || '获取日志列表失败')
  } finally {
    logsLoading.value = false
  }
}

// 获取日志详情
const fetchLogDetail = async (id: number) => {
  try {
    detailLoading.value = true
    const res = await getSyncLog(id)
    selectedLog.value = res
  } catch (error: any) {
    ElMessage.error(error.message || '获取日志详情失败')
  } finally {
    detailLoading.value = false
  }
}

// 过滤变化
const handleFilterChange = () => {
  pagination.value.page = 1
  fetchLogs()
}

// 分页变化
const handleSizeChange = () => {
  pagination.value.page = 1
  fetchLogs()
}

const handleCurrentChange = () => {
  fetchLogs()
}

// 行点击
const handleRowClick = (row: SyncLog) => {
  detailDialogVisible.value = true
  fetchLogDetail(row.id)
}

// 辅助函数
const getStatusType = (status: string): '' | 'success' | 'info' | 'warning' | 'danger' => {
  const map: Record<string, '' | 'success' | 'info' | 'warning' | 'danger'> = {
    running: '',
    completed: 'success',
    failed: 'danger',
    cancelled: 'warning'
  }
  return map[status] || 'info'
}

const getStatusText = (status: string): string => {
  const map: Record<string, string> = {
    running: '运行中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消'
  }
  return map[status] || status
}

const getSourceTypeText = (type: string): string => {
  const map: Record<string, string> = {
    favorite: '收藏夹',
    submission: 'UP主投稿',
    collection: '合集',
    watch_later: '稍后再看'
  }
  return map[type] || type
}

const getSuccessRateColor = (rate: number): string => {
  if (rate >= 95) return '#67c23a'
  if (rate >= 80) return '#e6a23c'
  return '#f56c6c'
}

const getCompletionRate = (log: SyncLog): string => {
  if (log.tasks_created === 0) return '0.00'
  return ((log.tasks_completed / log.tasks_created) * 100).toFixed(2)
}

const formatDateTime = (dateStr: string): string => {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const formatDuration = (ms: number): string => {
  if (!ms || ms === 0) return '-'
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}秒`
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  if (minutes < 60) return `${minutes}分${remainingSeconds}秒`
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return `${hours}小时${remainingMinutes}分`
}

onMounted(() => {
  fetchStats()
  fetchLogs()
})
</script>

<style scoped lang="scss">
.sync-logs {
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

  .stats-card {
    margin-bottom: 24px;

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-weight: 600;
    }

    .stats-content {
      :deep(.el-statistic) {
        text-align: center;

        .el-statistic__head {
          font-size: 14px;
          color: #909399;
          margin-bottom: 8px;
        }

        .el-statistic__content {
          font-size: 28px;
          font-weight: 600;
        }
      }
    }
  }

  .logs-card {
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-weight: 600;
    }

    .filters {
      margin-bottom: 16px;
      padding: 16px;
      background-color: #f5f7fa;
      border-radius: 4px;

      .el-form-item {
        margin-bottom: 0;
      }
    }

    .pagination {
      display: flex;
      justify-content: center;
      margin-top: 20px;
      padding-top: 16px;
      border-top: 1px solid #ebeef5;
    }
  }
}
</style>
