<template>
  <div class="gallery">
    <!-- 九宫格 -->
    <div class="gallery-grid">
      <div
        v-for="(item, idx) in items"
        :key="item.id"
        class="gallery-item"
        @click="handleItemClick(item, idx)"
      >
        <!-- 缩略图：图片 / Live Photo / 视频海报（视频用第一张图占位） -->
        <img
          v-if="item.thumbUrl"
          :src="item.thumbUrl"
          :alt="`media-${item.pid}`"
          loading="lazy"
          class="gallery-thumb"
        />
        <div v-else class="gallery-thumb-placeholder">
          <el-icon :size="32"><Picture /></el-icon>
        </div>

        <!-- 类型角标 -->
        <div class="gallery-badge" :class="badgeClass(item.kind)">
          <span v-if="item.kind === 'video'">视频</span>
          <span v-else-if="item.kind === 'live_photo'">Live</span>
          <span v-else>图</span>
        </div>

        <!-- 视频播放图标遮罩 -->
        <div v-if="item.kind === 'video'" class="gallery-play-overlay">
          <el-icon :size="40"><VideoPlay /></el-icon>
        </div>

        <!-- 序号 -->
        <div class="gallery-index">{{ item.pid }}</div>
      </div>
    </div>

    <!-- 图片预览（el-image 的预览能力） -->
    <el-image-viewer
      v-if="previewVisible"
      :url-list="previewList"
      :initial-index="previewIndex"
      :hide-on-click-modal="true"
      :z-index="3000"
      @close="previewVisible = false"
    />

    <!-- 视频弹窗播放 -->
    <el-dialog
      v-model="videoDialogVisible"
      :title="currentVideoName"
      width="80%"
      top="6vh"
      destroy-on-close
      @close="videoDialogVisible = false"
    >
      <video
        v-if="currentVideoUrl"
        :src="currentVideoUrl"
        controls
        autoplay
        style="width: 100%; max-height: 75vh; background: #000;"
      />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Picture, VideoPlay } from '@element-plus/icons-vue'
import type { Page } from '@/types'

interface GalleryItem {
  id: number
  pid: number
  kind: 'image' | 'video' | 'live_photo'
  thumbUrl: string  // 缩略图 URL（图片/LivePhoto 用本身；视频暂用空）
  fullUrl: string   // 完整 URL（图片放大用；视频播放用）
  name: string
}

const props = defineProps<{
  pages: Page[]
}>()

// 把 page.file_path 转成 /downloads/{file_path}（已是相对 download_base 的路径）
function buildURL(filePath?: string): string {
  if (!filePath) return ''
  const cleaned = filePath.replace(/^\/+/, '')
  return `/downloads/${cleaned}`
}

const items = computed<GalleryItem[]>(() => {
  return props.pages
    .filter((p) => !!p.file_path)
    .map((p) => {
      const kind = (p.kind || 'image') as GalleryItem['kind']
      const url = buildURL(p.file_path)
      // 视频缩略图暂用 page.image，没有则空
      const thumbUrl = kind === 'video' ? (p.image || '') : url
      return {
        id: p.id,
        pid: p.pid,
        kind,
        thumbUrl,
        fullUrl: url,
        name: p.name || `media-${p.pid}`,
      }
    })
})

// 仅图片/Live Photo 进预览列表（视频走弹窗）
const previewList = computed(() => {
  return items.value
    .filter((it) => it.kind !== 'video')
    .map((it) => it.fullUrl)
})

// 在 previewList 中查找指定 item 的索引
function findPreviewIndex(target: GalleryItem): number {
  let n = 0
  for (const it of items.value) {
    if (it.kind === 'video') continue
    if (it.id === target.id) return n
    n++
  }
  return 0
}

const previewVisible = ref(false)
const previewIndex = ref(0)

const videoDialogVisible = ref(false)
const currentVideoUrl = ref('')
const currentVideoName = ref('')

function handleItemClick(item: GalleryItem, _idx: number) {
  if (item.kind === 'video') {
    currentVideoUrl.value = item.fullUrl
    currentVideoName.value = item.name
    videoDialogVisible.value = true
  } else {
    previewIndex.value = findPreviewIndex(item)
    previewVisible.value = true
  }
}

function badgeClass(kind: string) {
  switch (kind) {
    case 'video':
      return 'badge-video'
    case 'live_photo':
      return 'badge-live'
    default:
      return 'badge-image'
  }
}
</script>

<style scoped>
.gallery {
  width: 100%;
}

.gallery-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

@media (min-width: 1024px) {
  .gallery-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

@media (min-width: 1400px) {
  .gallery-grid {
    grid-template-columns: repeat(5, 1fr);
  }
}

.gallery-item {
  position: relative;
  aspect-ratio: 1 / 1;
  overflow: hidden;
  border-radius: 8px;
  background: #f1f5f9;
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.gallery-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.gallery-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.gallery-thumb-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  background: linear-gradient(135deg, #e2e8f0, #cbd5e1);
}

.gallery-badge {
  position: absolute;
  top: 6px;
  right: 6px;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 600;
  color: #fff;
  border-radius: 4px;
  z-index: 2;
}

.badge-image {
  background: rgba(59, 130, 246, 0.85);
}

.badge-video {
  background: rgba(239, 68, 68, 0.85);
}

.badge-live {
  background: rgba(245, 158, 11, 0.9);
}

.gallery-play-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.3);
  color: #fff;
  z-index: 1;
}

.gallery-index {
  position: absolute;
  bottom: 6px;
  left: 6px;
  padding: 1px 6px;
  font-size: 11px;
  color: #fff;
  background: rgba(0, 0, 0, 0.55);
  border-radius: 3px;
  z-index: 2;
}
</style>
