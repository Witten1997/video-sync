<template>
  <div class="telegram-requests">
    <div class="page-header">
      <div class="page-header-left">
        <el-button text @click="router.push({ name: 'TelegramIntegration' })">
          <span class="material-icons-round" style="font-size: 20px">arrow_back</span>
        </el-button>
        <div>
          <h2>Telegram 请求日志</h2>
          <p>查看最近的 Telegram 提交请求，以及它们关联的下载记录。</p>
        </div>
      </div>
      <el-button type="primary" @click="loadRows">刷新</el-button>
    </div>

    <el-card class="filter-card">
      <el-form inline>
        <el-form-item label="状态">
          <el-select v-model="filters.status" clearable placeholder="全部" style="width: 140px">
            <el-option label="待处理" value="pending" />
            <el-option label="已入队" value="queued" />
            <el-option label="已完成" value="completed" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-form-item>
        <el-form-item label="Chat ID">
          <el-input v-model="filters.chat_id" clearable placeholder="输入 Chat ID" style="width: 160px" />
        </el-form-item>
        <el-form-item label="用户 ID">
          <el-input v-model="filters.user_id" clearable placeholder="输入用户 ID" style="width: 160px" />
        </el-form-item>
        <el-form-item label="任务 ID">
          <el-input v-model="filters.task_id" clearable placeholder="输入任务 ID" style="width: 180px" />
        </el-form-item>
        <el-form-item label="记录 ID">
          <el-input v-model="filters.record_id" clearable placeholder="输入记录 ID" style="width: 160px" />
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="filters.keyword" clearable placeholder="URL 或错误信息" style="width: 220px" @keyup.enter="handleSearch" />
        </el-form-item>
        <el-form-item>
          <el-space>
            <el-button type="primary" @click="handleSearch">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-space>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card>
      <el-table :data="rows" v-loading="loading" style="width: 100%">
        <el-table-column prop="created_at" label="时间" min-width="160">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="Chat / 用户" min-width="180">
          <template #default="{ row }">
            <div>{{ row.chat_id }}</div>
            <div class="subtext">{{ row.user_id }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="raw_url" label="URL" min-width="320" show-overflow-tooltip />
        <el-table-column prop="task_id" label="任务 ID" min-width="160" show-overflow-tooltip />
        <el-table-column label="记录 ID" width="120">
          <template #default="{ row }">
            <el-button
              v-if="row.record_id"
              type="primary"
              link
              @click="goToDownloadRecords(row.record_id)"
            >
              {{ row.record_id }}
            </el-button>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="error_message" label="错误信息" min-width="220" show-overflow-tooltip />
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.page_size"
          :total="pagination.total"
          :page-sizes="[20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @current-change="loadRows"
          @size-change="handlePageSizeChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getTelegramRequests } from '@/api/telegram'
import type { TelegramRequestLog } from '@/types'

defineOptions({
  name: 'TelegramRequests'
})

const router = useRouter()
const loading = ref(false)
const rows = ref<TelegramRequestLog[]>([])
const filters = reactive({
  status: '',
  chat_id: '',
  user_id: '',
  task_id: '',
  record_id: '',
  keyword: ''
})
const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

const loadRows = async () => {
  loading.value = true
  try {
    const data = await getTelegramRequests({
      page: pagination.page,
      page_size: pagination.page_size,
      ...filters
    })
    rows.value = data.items || []
    pagination.total = data.total || 0
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  loadRows()
}

const handleReset = () => {
  filters.status = ''
  filters.chat_id = ''
  filters.user_id = ''
  filters.task_id = ''
  filters.record_id = ''
  filters.keyword = ''
  pagination.page = 1
  loadRows()
}

const handlePageSizeChange = () => {
  pagination.page = 1
  loadRows()
}

const goToDownloadRecords = (recordID: number | null) => {
  if (!recordID) {
    return
  }

  router.push({
    name: 'DownloadRecords',
    query: { record_id: String(recordID) }
  })
}

const formatDateTime = (value: string) => {
  if (!value) {
    return '-'
  }

  return new Date(value).toLocaleString('zh-CN')
}

const statusTagType = (status: string) => {
  const mapping: Record<string, string> = {
    pending: 'info',
    queued: 'warning',
    completed: 'success',
    failed: 'danger'
  }

  return mapping[status] || 'info'
}

const statusText = (status: string) => {
  const mapping: Record<string, string> = {
    pending: '待处理',
    queued: '已入队',
    completed: '已完成',
    failed: '失败'
  }

  return mapping[status] || status
}

onMounted(loadRows)
</script>

<style scoped>
.telegram-requests {
  padding: 32px;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}

.page-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.page-header h2 {
  margin: 0 0 8px;
  font-size: 1.25rem;
  font-weight: 700;
  color: #1e293b;
}

.page-header p {
  margin: 0;
  color: #64748b;
}

.filter-card {
  margin-bottom: 16px;
}

.subtext {
  font-size: 12px;
  color: #94a3b8;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
