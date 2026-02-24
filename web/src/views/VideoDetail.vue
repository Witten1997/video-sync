<template>
  <div class="video-detail">
    <el-page-header @back="goBack" title="返回">
      <template #content>
        <span class="page-title">视频详情</span>
      </template>
    </el-page-header>

    <el-card v-loading="loading" class="detail-card">
      <template v-if="video">
        <el-row :gutter="20">
          <el-col :span="8">
            <el-image :src="getProxiedImageUrl(video.cover, false)" fit="cover" style="width: 100%; border-radius: 8px" />
          </el-col>
          <el-col :span="16">
            <div class="title-actions">
              <h2>{{ video.name }}</h2>
              <el-button
                v-if="isDownloadComplete(video.download_status) && video.valid"
                type="primary"
                size="large"
                :icon="VideoPlay"
                @click="handlePlay"
              >
                播放视频
              </el-button>
            </div>
            <el-descriptions :column="2" border class="info-desc">
              <el-descriptions-item label="BV号">{{ video.bvid }}</el-descriptions-item>
              <el-descriptions-item label="UP主">{{ video.upper_name }}</el-descriptions-item>
              <el-descriptions-item label="发布时间">
                {{ formatTime(video.pubtime) }}
              </el-descriptions-item>
              <el-descriptions-item label="收藏时间">
                {{ formatTime(video.favtime) }}
              </el-descriptions-item>
              <el-descriptions-item label="分P数量">
                {{ video.single_page ? '单P' : `多P (${pages.length})` }}
              </el-descriptions-item>
              <el-descriptions-item label="状态">
                <el-tag v-if="!video.valid" type="danger">无效</el-tag>
                <el-tag v-else-if="video.download_status === 0" type="info">待下载</el-tag>
                <el-tag v-else type="success">已下载</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="简介" :span="2">
                {{ video.intro || '暂无简介' }}
              </el-descriptions-item>
              <el-descriptions-item label="标签" :span="2">
                <el-tag
                  v-for="tag in video.tags"
                  :key="tag"
                  style="margin-right: 5px"
                  size="small"
                >
                  {{ tag }}
                </el-tag>
              </el-descriptions-item>
            </el-descriptions>
          </el-col>
        </el-row>
      </template>
    </el-card>

    <el-card v-if="!video?.single_page" class="pages-card">
      <template #header>
        <span>分P列表</span>
      </template>
      <el-table :data="pages" border stripe>
        <el-table-column prop="pid" label="P序号" width="100" />
        <el-table-column prop="name" label="标题" min-width="300" />
        <el-table-column label="时长" width="120">
          <template #default="{ row }">
            {{ formatDuration(row.duration) }}
          </template>
        </el-table-column>
        <el-table-column label="分辨率" width="120">
          <template #default="{ row }">
            {{ row.width }} × {{ row.height }}
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.download_status === 0" type="info">待下载</el-tag>
            <el-tag v-else type="success">已完成</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" align="center" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="isDownloadComplete(row.download_status)"
              type="primary"
              size="small"
              :icon="VideoPlay"
              @click="handlePlayPage(row)"
            >
              播放
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 视频播放器 -->
    <VideoPlayer
      v-model:visible="playerVisible"
      :video="currentPlayingVideo"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { VideoPlay } from '@element-plus/icons-vue'
import { getVideo, getVideoPages } from '@/api/video'
import { getProxiedImageUrl } from '@/utils/image'
import type { Video, Page } from '@/types'
import dayjs from 'dayjs'
import VideoPlayer from '@/components/VideoPlayer.vue'

defineOptions({
  name: 'VideoDetail'
})

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const video = ref<Video | null>(null)
const pages = ref<Page[]>([])

// 播放器相关
const playerVisible = ref(false)
const currentPlayingVideo = ref<Video | null>(null)

const videoId = Number(route.params.id)

// 加载视频详情
const loadData = async () => {
  loading.value = true
  try {
    video.value = await getVideo(videoId)
    if (!video.value.single_page) {
      pages.value = await getVideoPages(videoId)
    }
  } catch (error) {
    console.error('加载视频详情失败:', error)
  } finally {
    loading.value = false
  }
}

// 返回
const goBack = () => {
  router.back()
}

// 检查是否下载完成
const isDownloadComplete = (status: number) => {
  return status !== 0
}

// 播放视频（单P）
const handlePlay = () => {
  if (!video.value) return

  if (!isDownloadComplete(video.value.download_status) || !video.value.valid) {
    ElMessage.warning('该视频尚未下载完成，无法播放')
    return
  }

  const videoWithPages = { ...video.value, pages: pages.value }
  currentPlayingVideo.value = videoWithPages
  playerVisible.value = true
}

// 播放指定分集（多P）
const handlePlayPage = (page: Page) => {
  if (!video.value) return

  if (!isDownloadComplete(page.download_status)) {
    ElMessage.warning('该分集尚未下载完成，无法播放')
    return
  }

  const videoWithPages = { ...video.value, pages: pages.value }
  currentPlayingVideo.value = videoWithPages
  playerVisible.value = true

  // 延迟切换到指定分集
  setTimeout(() => {
    // VideoPlayer 组件会自动加载第一个已下载的分集
    // 这里可以扩展为直接播放指定分集
  }, 100)
}

// 格式化时间
const formatTime = (time: string) => {
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

// 格式化时长
const formatDuration = (seconds: number) => {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60

  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
  }
  return `${minutes}:${String(secs).padStart(2, '0')}`
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.video-detail {
  padding: 32px;
}

.page-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: #1e293b;
}

.detail-card {
  margin-top: 24px;
}

.title-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.title-actions h2 {
  margin: 0;
  flex: 1;
  font-size: 20px;
  font-weight: 700;
  color: #1e293b;
}

.info-desc {
  margin-top: 24px;
}

.pages-card {
  margin-top: 24px;
}
</style>
