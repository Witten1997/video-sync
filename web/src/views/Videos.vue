<template>
  <div class="videos">
    <div class="toolbar">
      <el-input
        v-model="searchKeyword"
        placeholder="搜索视频标题或BV号"
        style="width: 300px"
        clearable
        @clear="handleSearch"
      >
        <template #append>
          <el-button :icon="Search" @click="handleSearch" />
        </template>
      </el-input>
      <el-select
        v-model="filterSourceType"
        placeholder="筛选视频源类型"
        clearable
        style="width: 180px"
        @change="handleSearch"
      >
        <el-option label="收藏夹" value="favorite" />
        <el-option label="稍后再看" value="watch_later" />
        <el-option label="合集" value="collection" />
        <el-option label="UP主投稿" value="submission" />
        <el-option label="URL下载" value="url" />
      </el-select>
      <el-button @click="loadData">
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>
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
            placeholder="请输入B站视频链接，支持以下格式：&#10;https://www.bilibili.com/video/BVxxxxxxxxxx&#10;https://b23.tv/xxxxxxxx&#10;BVxxxxxxxxxx"
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
                <li>标准链接: https://www.bilibili.com/video/BV1xx411c7XD</li>
                <li>短链接: https://b23.tv/BV1xx411c7XD</li>
                <li>直接BV号: BV1xx411c7XD</li>
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

    <el-table
      v-loading="loading"
      :data="videos"
      border
      stripe
      style="width: 100%"
    >
      <el-table-column label="封面" width="150">
        <template #default="{ row }">
          <el-image
            :src="getProxiedImageUrl(row.cover)"
            fit="cover"
            style="width: 120px; height: 68px; border-radius: 4px"
            lazy
          >
            <template #error>
              <div class="image-slot">
                <el-icon><Picture /></el-icon>
              </div>
            </template>
          </el-image>
        </template>
      </el-table-column>
      <el-table-column prop="name" label="标题" min-width="300" show-overflow-tooltip />
      <el-table-column prop="bvid" label="BV号" width="150" />
      <el-table-column prop="upper_name" label="UP主" width="150" />
      <el-table-column label="分P" width="80" align="center">
        <template #default="{ row }">
          {{ row.single_page ? '单P' : '多P' }}
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100" align="center">
        <template #default="{ row }">
          <el-tag v-if="!row.valid" type="danger">无效</el-tag>
          <el-tag v-else-if="row.download_status === 0" type="info">待下载</el-tag>
          <el-tag v-else-if="isDownloadComplete(row.download_status)" type="success">已完成</el-tag>
          <el-tag v-else type="warning">下载中</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="发布时间" width="180">
        <template #default="{ row }">
          {{ formatTime(row.pubtime) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button text type="primary" size="small" @click="handleViewDetail(row)">
            <el-icon><View /></el-icon>
            详情
          </el-button>
          <el-button text type="primary" size="small" @click="handleRedownload(row)">
            <el-icon><Download /></el-icon>
            重新下载
          </el-button>
          <el-button text type="danger" size="small" @click="handleDelete(row)">
            <el-icon><Delete /></el-icon>
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handlePageChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Link } from '@element-plus/icons-vue'
import { getVideos, deleteVideo, redownloadVideo, downloadVideoByURL } from '@/api/video'
import { getProxiedImageUrl } from '@/utils/image'
import type { Video } from '@/types'
import dayjs from 'dayjs'

defineOptions({
  name: 'Videos'
})

const router = useRouter()
const loading = ref(false)
const videos = ref<Video[]>([])
const searchKeyword = ref('')
const filterSourceType = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

// URL下载对话框
const downloadDialogVisible = ref(false)
const downloadLoading = ref(false)
const downloadForm = ref({
  url: ''
})

// 显示下载对话框
const showDownloadDialog = () => {
  downloadForm.value.url = ''
  downloadDialogVisible.value = true
}

// 通过URL下载视频
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

    // 刷新列表以显示新添加的视频
    setTimeout(() => {
      loadData()
    }, 1000)
  } catch (error: any) {
    const errorMsg = error?.response?.data?.message || error?.message || '下载失败'
    ElMessage.error(errorMsg)
    console.error('通过URL下载失败:', error)
  } finally {
    downloadLoading.value = false
  }
}

// 加载视频列表
const loadData = async () => {
  loading.value = true
  try {
    const result = await getVideos({
      page: currentPage.value,
      page_size: pageSize.value,
      keyword: searchKeyword.value,
      source_type: filterSourceType.value
    })
    videos.value = result.items || []
    total.value = result.total
  } catch (error) {
    console.error('加载视频列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  currentPage.value = 1
  loadData()
}

// 页码变化
const handlePageChange = () => {
  loadData()
}

// 每页数量变化
const handleSizeChange = () => {
  currentPage.value = 1
  loadData()
}

// 查看详情
const handleViewDetail = (row: Video) => {
  router.push(`/videos/${row.id}`)
}

// 重新下载
const handleRedownload = async (row: Video) => {
  try {
    await ElMessageBox.confirm(
      `确定要重新下载视频 "${row.name}" 吗？`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await redownloadVideo(row.id)
    ElMessage.success('下载任务已创建')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('重新下载失败:', error)
    }
  }
}

// 删除视频
const handleDelete = async (row: Video) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除视频 "${row.name}" 吗？此操作将删除数据库记录和本地文件。`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await deleteVideo(row.id)
    ElMessage.success('删除成功')
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除失败:', error)
    }
  }
}

// 检查是否下载完成（简单判断）
const isDownloadComplete = (status: number) => {
  return status !== 0
}

// 格式化时间
const formatTime = (time: string) => {
  return dayjs(time).format('YYYY-MM-DD HH:mm')
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.videos {
  padding: 20px;
}

.toolbar {
  margin-bottom: 20px;
  display: flex;
  gap: 10px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.image-slot {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: #f5f7fa;
  color: #909399;
  font-size: 30px;
}
</style>
