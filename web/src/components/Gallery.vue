<template>
  <div class="gallery">
    <div class="gallery-grid">
      <div
        v-for="(item, idx) in items"
        :key="item.id"
        class="gallery-item"
        :style="itemStyle(item)"
        @click="handleItemClick(item, idx)"
      >
        <div class="gallery-media">
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
        </div>

        <div class="gallery-badge" :class="badgeClass(item.kind)">
          <span v-if="item.kind === 'video'">视频</span>
          <span v-else-if="item.kind === 'live_photo'">Live</span>
          <span v-else>图</span>
        </div>

        <div v-if="item.kind === 'video'" class="gallery-play-overlay">
          <el-icon :size="40"><VideoPlay /></el-icon>
        </div>

        <div class="gallery-index">{{ item.pid }}</div>
      </div>
    </div>

    <el-image-viewer
      v-if="previewVisible"
      :url-list="previewList"
      :initial-index="previewIndex"
      :hide-on-click-modal="true"
      :z-index="3000"
      @close="previewVisible = false"
    />

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
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Picture, VideoPlay } from '@element-plus/icons-vue'
import type { Page } from '@/types'
import { decodeHeicToObjectUrl, isHeicLikeUrl } from '@/utils/heic'

interface GalleryItem {
  id: number
  pid: number
  kind: 'image' | 'video' | 'live_photo'
  sourceUrl: string
  thumbUrl: string
  fullUrl: string
  name: string
  width: number
  height: number
}

const props = defineProps<{
  pages: Page[]
}>()

const GALLERY_ITEM_HEIGHT = 300
const items = ref<GalleryItem[]>([])
const previewVisible = ref(false)
const previewIndex = ref(0)
const videoDialogVisible = ref(false)
const currentVideoUrl = ref('')
const currentVideoName = ref('')
const objectUrls = new Set<string>()
const heicUrlCache = new Map<string, Promise<string>>()
let buildVersion = 0

function buildURL(filePath?: string): string {
  if (!filePath) return ''
  const cleaned = filePath.replace(/^\/+/, '')
  return `/downloads/${cleaned}`
}

async function resolveMediaUrl(url: string): Promise<string> {
  if (!url || !isHeicLikeUrl(url)) {
    return url
  }

  let task = heicUrlCache.get(url)
  if (!task) {
    task = decodeHeicToObjectUrl(url).then((objectUrl) => {
      objectUrls.add(objectUrl)
      return objectUrl
    })
    heicUrlCache.set(url, task)
  }

  try {
    return await task
  } catch (error) {
    console.error('HEIC 图片解码失败:', error)
    return url
  }
}

async function rebuildItems() {
  const version = ++buildVersion
  const nextItems = props.pages
    .filter((page) => !!page.file_path)
    .map((page) => {
      const kind = (page.kind || 'image') as GalleryItem['kind']
      const sourceUrl = buildURL(page.file_path)
      const isHeic = kind !== 'video' && isHeicLikeUrl(sourceUrl)

      return {
        id: page.id,
        pid: page.pid,
        kind,
        sourceUrl,
        thumbUrl: kind === 'video' ? (page.image || '') : (isHeic ? '' : sourceUrl),
        fullUrl: sourceUrl,
        name: page.name || `media-${page.pid}`,
        width: page.width || 0,
        height: page.height || 0,
      }
    })

  items.value = nextItems

  for (const item of nextItems) {
    if (version !== buildVersion) {
      return
    }
    if (item.kind === 'video' || !isHeicLikeUrl(item.sourceUrl)) {
      continue
    }

    void resolveMediaUrl(item.sourceUrl).then((resolvedUrl) => {
      if (version !== buildVersion) {
        return
      }

      const target = items.value.find((current) => current.id === item.id)
      if (!target) {
        return
      }

      target.thumbUrl = resolvedUrl
      target.fullUrl = resolvedUrl
    })
  }
}

const previewList = computed(() => {
  return items.value
    .filter((item) => item.kind !== 'video')
    .map((item) => item.fullUrl)
})

function findPreviewIndex(target: GalleryItem): number {
  let n = 0
  for (const item of items.value) {
    if (item.kind === 'video') continue
    if (item.id === target.id) return n
    n++
  }
  return 0
}

async function handleItemClick(item: GalleryItem, _idx: number) {
  if (item.kind === 'video') {
    currentVideoUrl.value = item.fullUrl
    currentVideoName.value = item.name
    videoDialogVisible.value = true
    return
  }

  if (isHeicLikeUrl(item.sourceUrl) && item.fullUrl === item.sourceUrl) {
    const resolvedUrl = await resolveMediaUrl(item.sourceUrl)
    item.thumbUrl = resolvedUrl
    item.fullUrl = resolvedUrl
  }

  previewIndex.value = findPreviewIndex(item)
  previewVisible.value = true
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

function itemStyle(item: GalleryItem) {
  const ratio = item.width > 0 && item.height > 0 ? item.width / item.height : 1
  return {
    width: `${Math.max(Math.round(GALLERY_ITEM_HEIGHT * ratio), 120)}px`,
  }
}

watch(
  () => props.pages,
  () => {
    void rebuildItems()
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  for (const url of objectUrls) {
    URL.revokeObjectURL(url)
  }
})
</script>

<style scoped>
.gallery {
  width: 100%;
}

.gallery-grid {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 8px;
}

@media (min-width: 1024px) {
  .gallery-grid {
    gap: 8px;
  }
}

@media (min-width: 1400px) {
  .gallery-grid {
    gap: 8px;
  }
}

.gallery-item {
  flex: 0 0 auto;
  position: relative;
  overflow: hidden;
  border-radius: 8px;
  background: #f1f5f9;
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.gallery-media {
  height: 300px;
}

.gallery-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.gallery-thumb {
  width: 100%;
  height: 100%;
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
