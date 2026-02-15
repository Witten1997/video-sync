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
      <el-button
        v-if="selectedVideos.length > 0"
        type="primary"
        @click="handleBatchRedownload"
      >
        <el-icon><Download /></el-icon>
        批量重新下载 ({{ selectedVideos.length }})
      </el-button>
      <el-button
        v-if="selectedVideos.length > 0"
        type="danger"
        @click="handleBatchDelete"
      >
        <el-icon><Delete /></el-icon>
        批量删除 ({{ selectedVideos.length }})
      </el-button>
      <div class="toolbar-right">
        <el-radio-group v-model="viewMode" size="small">
          <el-radio-button label="list">
            <el-icon><List /></el-icon>
            列表
          </el-radio-button>
          <el-radio-button label="grid">
            <el-icon><Grid /></el-icon>
            卡片
          </el-radio-button>
        </el-radio-group>
        <el-button @click="loadData">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button type="primary" @click="showDownloadDialog">
          <el-icon><Link /></el-icon>
          通过URL下载
        </el-button>
      </div>
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

    <!-- 列表视图 -->
    <el-table
      v-if="viewMode === 'list'"
      v-loading="loading"
      :data="videos"
      border
      stripe
      style="width: 100%"
      @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" width="55" align="center" />
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
      <el-table-column label="操作" width="280" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="isDownloadComplete(row.download_status) && row.valid"
            text
            type="success"
            size="small"
            @click="handlePlay(row)"
          >
            <el-icon><VideoPlay /></el-icon>
            播放
          </el-button>
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

    <!-- 卡片视图 -->
    <div v-else class="grid-view" v-loading="loading">
      <div v-for="item in videos" :key="item.id" class="grid-item">
        <el-card :body-style="{ padding: '0px' }" shadow="hover">
          <div class="grid-cover-wrapper">
            <el-image
              :src="getProxiedImageUrl(item.cover)"
              fit="cover"
              class="grid-cover"
              lazy
              @click="handleViewDetail(item)"
              style="cursor: pointer"
            >
              <template #error>
                <div class="image-slot-grid">
                  <el-icon><Picture /></el-icon>
                </div>
              </template>
            </el-image>
            <!-- 播放按钮悬浮层 -->
            <div
              v-if="isDownloadComplete(item.download_status) && item.valid"
              class="play-overlay"
              @click="handlePlay(item)"
            >
              <el-icon :size="48" class="play-icon">
                <VideoPlay />
              </el-icon>
            </div>
          </div>
          <div class="grid-content">
            <div class="grid-title" :title="item.name" @click="handleViewDetail(item)">
              {{ item.name }}
            </div>
            <div class="grid-info">
              <el-text size="small" type="info">{{ item.upper_name }}</el-text>
            </div>
            <div class="grid-meta">
              <el-tag size="small" type="info">{{ item.bvid }}</el-tag>
              <el-tag size="small">{{ item.single_page ? '单P' : '多P' }}</el-tag>
            </div>
            <div class="grid-status">
              <el-tag v-if="!item.valid" type="danger" size="small">无效</el-tag>
              <el-tag v-else-if="item.download_status === 0" type="info" size="small">待下载</el-tag>
              <el-tag v-else-if="isDownloadComplete(item.download_status)" type="success" size="small">已完成</el-tag>
              <el-tag v-else type="warning" size="small">下载中</el-tag>
            </div>
            <div class="grid-time">
              <el-text size="small" type="info">{{ formatTime(item.pubtime) }}</el-text>
            </div>
            <div class="grid-actions">
              <el-button-group>
                <el-button size="small" @click="handleViewDetail(item)">
                  <el-icon><View /></el-icon>
                  详情
                </el-button>
                <el-button size="small" @click="handleRedownload(item)">
                  <el-icon><Download /></el-icon>
                </el-button>
                <el-button size="small" type="danger" @click="handleDelete(item)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-button-group>
            </div>
          </div>
        </el-card>
      </div>
    </div>

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

    <!-- 视频播放器 -->
    <VideoPlayer
      v-model:visible="playerVisible"
      :video="currentPlayingVideo"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Link, List, Grid, VideoPlay, View, Download, Delete, Refresh, Picture } from '@element-plus/icons-vue'
import { getVideos, deleteVideo, redownloadVideo, downloadVideoByURL, getVideoPages } from '@/api/video'
import { getProxiedImageUrl } from '@/utils/image'
import type { Video } from '@/types'
import dayjs from 'dayjs'
import VideoPlayer from '@/components/VideoPlayer.vue'

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
const viewMode = ref<'list' | 'grid'>('grid')
const selectedVideos = ref<Video[]>([])

// URL下载对话框
const downloadDialogVisible = ref(false)
const downloadLoading = ref(false)
const downloadForm = ref({
  url: ''
})

// 播放器相关
const playerVisible = ref(false)
const currentPlayingVideo = ref<Video | null>(null)

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

// 播放视频
const handlePlay = async (row: Video) => {
  // 检查视频是否已下载完成
  if (!isDownloadComplete(row.download_status) || !row.valid) {
    ElMessage.warning('该视频尚未下载完成，无法播放')
    return
  }

  try {
    // 如果是多P视频，需要加载分P列表
    if (!row.single_page) {
      const pages = await getVideoPages(row.id)
      row.pages = pages
    }

    currentPlayingVideo.value = row
    playerVisible.value = true
  } catch (error) {
    console.error('加载视频信息失败:', error)
    ElMessage.error('加载视频信息失败')
  }
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

// 处理选择变化
const handleSelectionChange = (selection: Video[]) => {
  selectedVideos.value = selection
}

// 批量重新下载
const handleBatchRedownload = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要重新下载选中的 ${selectedVideos.value.length} 个视频吗？`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    let successCount = 0
    for (const video of selectedVideos.value) {
      try {
        await redownloadVideo(video.id)
        successCount++
      } catch (error) {
        console.error(`重新下载视频 ${video.name} 失败:`, error)
      }
    }
    ElMessage.success(`已创建 ${successCount} 个下载任务`)
    selectedVideos.value = []
  } catch (error) {
    if (error !== 'cancel') {
      console.error('批量重新下载失败:', error)
    }
  }
}

// 批量删除
const handleBatchDelete = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedVideos.value.length} 个视频吗？此操作将删除数据库记录和本地文件。`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    let successCount = 0
    for (const video of selectedVideos.value) {
      try {
        await deleteVideo(video.id)
        successCount++
      } catch (error) {
        console.error(`删除视频 ${video.name} 失败:`, error)
      }
    }
    ElMessage.success(`已删除 ${successCount} 个视频`)
    selectedVideos.value = []
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('批量删除失败:', error)
    }
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.videos {
  padding: 32px;
}

.toolbar {
  margin-bottom: 24px;
  display: flex;
  gap: 12px;
  align-items: center;
}

.toolbar-right {
  margin-left: auto;
  display: flex;
  gap: 12px;
  align-items: center;
}

.pagination {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}

.image-slot {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: #f8fafc;
  color: #94a3b8;
  font-size: 30px;
}

/* 卡片视图样式 */
.grid-view {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 20px;
  margin-top: 20px;
}

.grid-item {
  height: 100%;
}

.grid-item :deep(.el-card) {
  border-radius: 16px;
  border: 1px solid #f1f5f9;
  box-shadow: 0 1px 2px 0 rgb(0 0 0 / 0.05);
  overflow: hidden;
  transition: box-shadow 0.2s ease, transform 0.2s ease;
}

.grid-item :deep(.el-card:hover) {
  box-shadow: 0 4px 12px 0 rgb(0 0 0 / 0.08);
  transform: translateY(-2px);
}

.grid-cover-wrapper {
  position: relative;
  overflow: hidden;
  border-radius: 16px 16px 0 0;
}

.grid-cover {
  width: 100%;
  height: 160px;
  object-fit: cover;
  display: block;
  transition: transform 0.3s ease;
}

.grid-cover-wrapper:hover .grid-cover {
  transform: scale(1.05);
}

.play-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.4);
  opacity: 0;
  transition: opacity 0.3s ease;
  cursor: pointer;
  z-index: 1;
}

.grid-cover-wrapper:hover .play-overlay {
  opacity: 1;
}

.play-icon {
  color: #fff;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.5));
  transition: transform 0.2s ease;
}

.play-overlay:hover .play-icon {
  transform: scale(1.1);
}

.image-slot-grid {
  width: 100%;
  height: 160px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f8fafc;
  color: #94a3b8;
  font-size: 40px;
}

.grid-content {
  padding: 14px;
}

.grid-title {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
  margin-bottom: 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  line-height: 1.4;
  min-height: 40px;
  cursor: pointer;
}

.grid-title:hover {
  color: #3b82f6;
}

.grid-info {
  margin-bottom: 8px;
  color: #64748b;
}

.grid-meta {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.grid-status {
  margin-bottom: 8px;
}

.grid-time {
  margin-bottom: 12px;
  color: #94a3b8;
}

.grid-actions {
  display: flex;
  justify-content: center;
}

.grid-actions .el-button-group {
  width: 100%;
}

.grid-actions .el-button {
  flex: 1;
}
</style>
