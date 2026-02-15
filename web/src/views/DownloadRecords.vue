<template>
  <div class="download-records" style="padding: 32px;">
    <div style="margin-bottom: 24px;">
      <h2 style="margin: 0 0 8px; font-size: 1.25rem; font-weight: 700; color: #1e293b;">下载管理</h2>
      <p style="margin: 0; font-size: 0.875rem; color: #64748b;">查看和管理所有下载记录</p>
    </div>

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
      <el-table :data="records" v-loading="loading" style="width: 100%;">
        <el-table-column label="视频" min-width="280">
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 12px;">
              <img
                v-if="row.video?.cover"
                :src="'/api/image-proxy?url=' + encodeURIComponent(row.video.cover)"
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
              v-if="row.status === 'failed'"
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
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getDownloadRecords, retryDownloadRecord, deleteDownloadRecord } from '@/api/download-records'
import SegmentedProgress from '@/components/SegmentedProgress.vue'
import type { DownloadRecord } from '@/types'

const records = ref<DownloadRecord[]>([])
const loading = ref(false)
const filters = reactive({ status: '', source_type: '', keyword: '' })
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
let ws: WebSocket | null = null

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
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  ws = new WebSocket(`${protocol}//${window.location.host}/api/ws`)

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'download_progress') {
        const { record_id, file_name, status, progress, size } = msg.data
        const record = records.value.find(r => r.id === record_id)
        if (record?.file_details?.files) {
          const file = record.file_details.files.find(f => f.name === file_name)
          if (file) {
            file.status = status
            file.progress = progress
            file.size = size
          }
          if (record.status === 'pending') {
            record.status = 'downloading'
          }
        }
      } else if (msg.type === 'download_status') {
        const { record_id, status } = msg.data
        const record = records.value.find(r => r.id === record_id)
        if (record) {
          record.status = status
          if (status === 'completed') {
            record.file_details.files.forEach(f => {
              if (f.status === 'downloading') {
                f.status = 'completed'
                f.progress = 100
              }
            })
          }
        }
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
  loadRecords()
  connectWebSocket()
})

onUnmounted(() => {
  ws?.close()
  ws = null
})
</script>
