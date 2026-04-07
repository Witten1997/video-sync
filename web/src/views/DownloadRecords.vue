<template>
  <div class="download-records" style="padding: 32px;">
    <div style="margin-bottom: 24px; display: flex; justify-content: space-between; align-items: flex-start;">
      <div>
        <h2 style="margin: 0 0 8px; font-size: 1.25rem; font-weight: 700; color: #1e293b;">下载管理</h2>
        <p style="margin: 0; font-size: 0.875rem; color: #64748b;">查看和管理所有下载记录</p>
      </div>
      <el-button type="primary" @click="showDownloadDialog">
        <el-icon><Link /></el-icon>
        通过URL下载
      </el-button>
    </div>

    <!-- URL下载对话框 -->
    <el-dialog
      v-model="downloadDialogVisible"
      title="通过URL下载视频"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form :model="downloadForm" label-width="100px">
        <el-form-item label="视频链接">
          <el-input
            v-model="downloadForm.url"
            type="textarea"
            :rows="3"
            placeholder="请输入视频链接，支持B站、YouTube等平台"
          />
        </el-form-item>
        <el-form-item>
          <el-alert
            title="支持的链接格式"
            type="info"
            :closable="false"
            show-icon
          >
            <template #default>
              <ul style="margin: 0; padding-left: 20px;">
                <li>B站: https://www.bilibili.com/video/BV1xx411c7XD</li>
                <li>B站短链: https://b23.tv/BV1xx411c7XD</li>
                <li>YouTube: https://www.youtube.com/watch?v=xxxxx</li>
                <li>其他yt-dlp支持的平台链接</li>
              </ul>
            </template>
          </el-alert>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="downloadDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="downloadLoading"
          :disabled="!downloadForm.url"
          @click="handleDownloadByURL"
        >
          开始下载
        </el-button>
      </template>
    </el-dialog>

    <!-- 筛选栏 -->
    <el-card style="margin-bottom: 16px;">
      <div style="display: flex; gap: 12px; align-items: center; flex-wrap: wrap;">
        <el-select v-model="filters.status" placeholder="状态" clearable style="width: 140px;" @change="loadRecords">
          <el-option label="全部" value="" />
          <el-option label="等待中" value="pending" />
          <el-option label="下载中" value="downloading" />
          <el-option label="已完成" value="completed" />
          <el-option label="失败" value="failed" />
        </el-select>
        <el-select v-model="filters.source_type" placeholder="视频源类型" clearable style="width: 160px;" @change="loadRecords">
          <el-option label="全部" value="" />
          <el-option label="收藏夹" value="favorite" />
          <el-option label="合集" value="collection" />
          <el-option label="UP主投稿" value="submission" />
          <el-option label="稍后再看" value="watch_later" />
          <el-option label="URL下载" value="url" />
        </el-select>
        <el-input
          v-model="filters.keyword"
          placeholder="搜索视频标题"
          clearable
          style="width: 200px;"
          @keyup.enter="loadRecords"
          @clear="loadRecords"
        >
          <template #prefix>
            <span class="material-icons-round" style="font-size: 16px;">search</span>
          </template>
        </el-input>
      </div>
    </el-card>

    <!-- 表格 -->
    <el-card>
      <div v-if="selectedIds.length > 0" style="margin-bottom: 12px; display: flex; align-items: center; gap: 12px;">
        <span style="font-size: 13px; color: #64748b;">已选 {{ selectedIds.length }} 项</span>
        <el-button type="primary" size="small" @click="handleBatchRetry">批量重试</el-button>
        <el-popconfirm :title="`确定删除选中的 ${selectedIds.length} 条记录？`" @confirm="handleBatchDelete">
          <template #reference>
            <el-button type="danger" size="small">批量删除</el-button>
          </template>
        </el-popconfirm>
      </div>
      <el-table :data="records" v-loading="loading" style="width: 100%;" @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="45" />
        <el-table-column label="视频" min-width="280">
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 12px;">
              <img
                v-if="row.video?.cover"
                :src="getProxiedImageUrl(row.video.cover)"
                style="width: 80px; height: 45px; object-fit: cover; border-radius: 6px; flex-shrink: 0;"
              />
              <div style="min-width: 0;">
                <div style="font-size: 13px; font-weight: 500; color: #1e293b; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
                  {{ row.video?.name || '-' }}
                </div>
                <div style="font-size: 11px; color: #94a3b8; margin-top: 2px;">
                  {{ row.video?.upper_name || '' }}
                </div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="视频源" prop="source_name" width="150">
          <template #default="{ row }">
            <span style="font-size: 13px; color: #64748b;">{{ row.source_name || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="下载进度" width="200">
          <template #default="{ row }">
            <SegmentedProgress v-if="row.file_details?.files" :files="row.file_details.files" />
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">{{ getStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="160">
          <template #default="{ row }">
            <span style="font-size: 12px; color: #94a3b8;">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'failed' || row.status === 'completed'"
              type="primary" link size="small"
              @click="handleRetry(row)"
            >重试</el-button>
            <el-popconfirm title="确定删除？" @confirm="handleDelete(row)">
              <template #reference>
                <el-button type="danger" link size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div style="display: flex; justify-content: flex-end; margin-top: 16px;">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @current-change="loadRecords"
          @size-change="loadRecords"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Link } from '@element-plus/icons-vue'
import { getDownloadRecords, getDownloadRecord, retryDownloadRecord, deleteDownloadRecord, batchDeleteDownloadRecords, batchRetryDownloadRecords } from '@/api/download-records'
import { downloadVideoByURL } from '@/api/video'
import SegmentedProgress from '@/components/SegmentedProgress.vue'
import type { DownloadRecord } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { getProxiedImageUrl } from '@/utils/image'
import { useRoute } from 'vue-router'

const records = ref<DownloadRecord[]>([])
const loading = ref(false)
const selectedIds = ref<number[]>([])
const selectedRows = ref<DownloadRecord[]>([])
const route = useRoute()
const filters = reactive({ status: '', source_type: '', keyword: '', record_id: '' })
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })

// URL下载对话框
const downloadDialogVisible = ref(false)
const downloadLoading = ref(false)
const downloadForm = ref({ url: '' })

const showDownloadDialog = () => {
  downloadForm.value.url = ''
  downloadDialogVisible.value = true
}

const handleDownloadByURL = async () => {
  const url = downloadForm.value.url.trim()
  if (!url) {
    ElMessage.warning('请输入视频链接')
    return
  }
  downloadLoading.value = true
  try {
    const result = await downloadVideoByURL(url)
    ElMessage.success(result.message || '下载任务已创建')
    downloadDialogVisible.value = false
    downloadForm.value.url = ''
    setTimeout(() => loadRecords(), 1000)
  } catch (error: any) {
    const errorMsg = error?.response?.data?.message || error?.message || '下载失败'
    ElMessage.error(errorMsg)
  } finally {
    downloadLoading.value = false
  }
}

let ws: WebSocket | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null

// 进度更新批量合并：收集 WebSocket 消息，用 rAF 批量应用
let pendingProgressUpdates: Map<string, { record_id: number; file_name: string; status: string; progress: number; size: number }> = new Map()
let pendingStatusUpdates: Map<number, string> = new Map()
let rafId: number | null = null

const flushUpdates = () => {
  rafId = null

  // 批量应用进度更新
  if (pendingProgressUpdates.size > 0) {
    for (const update of pendingProgressUpdates.values()) {
      const record = records.value.find(r => r.id === update.record_id)
      if (record?.file_details?.files) {
        const file = record.file_details.files.find(f => f.name === update.file_name)
        if (file) {
          file.status = update.status as typeof file.status
          file.progress = update.progress
          file.size = update.size
        }
        if (record.status === 'pending') {
          record.status = 'downloading'
        }
      }
    }
    pendingProgressUpdates.clear()
  }

  // 批量应用状态更新
  if (pendingStatusUpdates.size > 0) {
    for (const [record_id, status] of pendingStatusUpdates) {
      const record = records.value.find(r => r.id === record_id)
      if (record) {
        record.status = status as DownloadRecord['status']
        if (status === 'completed') {
          record.file_details.files.forEach(f => {
            if (f.status !== 'failed' && f.status !== 'skipped') {
              f.status = 'completed'
              f.progress = 100
            }
          })
        }
      }
    }
    pendingStatusUpdates.clear()
  }
}

const scheduleFlush = () => {
  if (rafId === null) {
    rafId = requestAnimationFrame(flushUpdates)
  }
}

const applyRouteFilters = () => {
  const recordID = route.query.record_id
  filters.record_id = typeof recordID === 'string' ? recordID : ''
}

const loadRecords = async () => {
  loading.value = true
  try {
    const data = await getDownloadRecords({
      page: pagination.page,
      page_size: pagination.pageSize,
      ...filters
    })
    records.value = data.items || []
    pagination.total = data.total
  } finally {
    loading.value = false
  }
}

// 检查是否有正在下载的记录，有则定时轮询保底
const hasActiveDownloads = () => records.value.some(r => r.status === 'downloading' || r.status === 'pending')

const startPollIfNeeded = () => {
  stopPoll()
  if (hasActiveDownloads()) {
    pollTimer = setInterval(async () => {
      if (!hasActiveDownloads()) {
        stopPoll()
        return
      }
      // 静默刷新，不显示 loading
      try {
        const data = await getDownloadRecords({
          page: pagination.page,
          page_size: pagination.pageSize,
          ...filters
        })
        records.value = data.items || []
        pagination.total = data.total
      } catch (_) {}
    }, 10000)
  }
}

const stopPoll = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

const handleRetry = async (row: DownloadRecord) => {
  await retryDownloadRecord(row.id)
  ElMessage.success('重试任务已创建')
  loadRecords()
}

const handleDelete = async (row: DownloadRecord) => {
  await deleteDownloadRecord(row.id)
  ElMessage.success('删除成功')
  loadRecords()
}

const handleSelectionChange = (rows: DownloadRecord[]) => {
  selectedIds.value = rows.map(r => r.id)
  selectedRows.value = rows
}

const handleBatchRetry = async () => {
  const retryableIds = selectedRows.value
    .filter(r => r.status === 'failed' || r.status === 'completed')
    .map(r => r.id)
  if (retryableIds.length === 0) {
    ElMessage.warning('选中记录中没有可重试的项')
    return
  }
  const data = await batchRetryDownloadRecords(retryableIds)
  ElMessage.success(`已重试 ${data.retried} 条记录`)
  selectedIds.value = []
  selectedRows.value = []
  loadRecords()
}

const handleBatchDelete = async () => {
  await batchDeleteDownloadRecords(selectedIds.value)
  ElMessage.success('批量删除成功')
  selectedIds.value = []
  selectedRows.value = []
  loadRecords()
}

const getStatusType = (status: string) => {
  const map: Record<string, string> = { pending: 'info', downloading: '', completed: 'success', failed: 'danger' }
  return map[status] || 'info'
}

const getStatusLabel = (status: string) => {
  const map: Record<string, string> = { pending: '等待中', downloading: '下载中', completed: '已完成', failed: '失败' }
  return map[status] || status
}

const formatTime = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

// WebSocket
const connectWebSocket = () => {
  const authStore = useAuthStore()
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  ws = new WebSocket(`${protocol}//${window.location.host}/api/ws?token=${encodeURIComponent(authStore.token)}`)

  ws.onmessage = async (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'download_record_created') {
        const { id } = msg.data
        if (id && !records.value.find(r => r.id === id)) {
          try {
            const record = await getDownloadRecord(id)
            records.value.unshift(record)
            pagination.total++
            startPollIfNeeded()
          } catch (e) {}
        }
      } else if (msg.type === 'download_progress') {
        const { record_id, file_name, status, progress, size } = msg.data
        // 合并到待更新队列，同一 record+file 只保留最新值
        pendingProgressUpdates.set(`${record_id}-${file_name}`, { record_id, file_name, status, progress, size })
        scheduleFlush()
      } else if (msg.type === 'download_status') {
        const { record_id, status } = msg.data
        pendingStatusUpdates.set(record_id, status)
        scheduleFlush()
        // 状态变更后检查是否还需要轮询
        setTimeout(startPollIfNeeded, 100)
      }
    } catch (e) {}
  }

  ws.onclose = () => {
    setTimeout(() => {
      if (!ws || ws.readyState === WebSocket.CLOSED) connectWebSocket()
    }, 5000)
  }
}

onMounted(() => {
  applyRouteFilters()
  loadRecords().then(startPollIfNeeded)
  connectWebSocket()
})

watch(() => route.query.record_id, () => {
  applyRouteFilters()
  pagination.page = 1
  loadRecords().then(startPollIfNeeded)
})

onUnmounted(() => {
  ws?.close()
  ws = null
  stopPoll()
  if (rafId !== null) {
    cancelAnimationFrame(rafId)
    rafId = null
  }
})
</script>
