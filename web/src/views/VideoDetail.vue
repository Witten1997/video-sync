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
            <el-image :src="video.cover" fit="cover" style="width: 100%; border-radius: 8px" />
          </el-col>
          <el-col :span="16">
            <h2>{{ video.name }}</h2>
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
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getVideo, getVideoPages } from '@/api/video'
import type { Video, Page } from '@/types'
import dayjs from 'dayjs'

defineOptions({
  name: 'VideoDetail'
})

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const video = ref<Video | null>(null)
const pages = ref<Page[]>([])

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
  padding: 20px;
}

.page-title {
  font-size: 18px;
  font-weight: 500;
}

.detail-card {
  margin-top: 20px;
}

.info-desc {
  margin-top: 20px;
}

.pages-card {
  margin-top: 20px;
}
</style>
