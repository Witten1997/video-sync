<template>
  <Teleport to="body">
    <div v-if="visible" class="iv-overlay" @wheel.prevent="onWheel">
      <div class="iv-header">
        <button class="iv-btn" @click="close" title="返回 (Esc)">
          <el-icon :size="20"><ArrowLeft /></el-icon>
        </button>
        <div class="iv-counter">{{ index + 1 }} / {{ items.length }}</div>
        <div class="iv-actions">
          <button class="iv-btn" @click="zoomIn" title="放大 (+)"><el-icon :size="18"><ZoomIn /></el-icon></button>
          <button class="iv-btn" @click="zoomOut" title="缩小 (-)"><el-icon :size="18"><ZoomOut /></el-icon></button>
          <button class="iv-btn" @click="rotateLeft" title="向左旋转"><el-icon :size="18"><RefreshLeft /></el-icon></button>
          <button class="iv-btn" @click="rotateRight" title="向右旋转"><el-icon :size="18"><RefreshRight /></el-icon></button>
          <button v-if="current?.kind === 'live_photo'" class="iv-btn" :class="{ active: isPlayingLive }" @click="toggleLive" title="播放 Live (Space)">
            <el-icon :size="18"><VideoPlay /></el-icon>
          </button>
          <button class="iv-btn" :class="{ active: showInfo }" @click="toggleInfo" title="信息 (i)">
            <el-icon :size="18"><InfoFilled /></el-icon>
          </button>
          <button class="iv-btn iv-danger" @click="onDelete" title="删除"><el-icon :size="18"><Delete /></el-icon></button>
        </div>
      </div>

      <div class="iv-body">
        <div class="iv-canvas" @click.self="close">
          <button class="iv-nav iv-nav-left" :disabled="index <= 0" @click="prev" title="上一张 (←)">
            <el-icon :size="28"><ArrowLeft /></el-icon>
          </button>

          <div class="iv-stage" @click.self="close">
            <video
              v-if="isPlayingLive && current"
              ref="videoEl"
              :src="`/api/pages/${current.id}/live-video`"
              autoplay
              controls
              :style="mediaStyle"
              class="iv-media"
              @ended="isPlayingLive = false"
            />
            <img
              v-else-if="resolvedUrl"
              :src="resolvedUrl"
              :alt="current?.name"
              :style="mediaStyle"
              class="iv-media"
              draggable="false"
            />
            <div v-else class="iv-loading">加载中...</div>
          </div>

          <button class="iv-nav iv-nav-right" :disabled="index >= items.length - 1" @click="next" title="下一张 (→)">
            <el-icon :size="28"><ArrowRight /></el-icon>
          </button>
        </div>

        <aside v-if="showInfo" class="iv-info">
          <h3 class="iv-info-title">信息</h3>
          <dl class="iv-info-list">
            <dt>名称</dt><dd>{{ current?.name || '-' }}</dd>
            <dt>分辨率</dt><dd>{{ current?.width || 0 }} × {{ current?.height || 0 }}</dd>
            <dt>大小</dt><dd>{{ formatBytes(current?.file_size) }}</dd>
            <dt>路径</dt><dd class="iv-info-path" :title="current?.file_path">{{ current?.file_path || '-' }}</dd>
            <dt>修改时间</dt><dd>{{ formatDate(current?.modified_at) }}</dd>
          </dl>
        </aside>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import {
  ArrowLeft, ArrowRight, ZoomIn, ZoomOut, RefreshLeft, RefreshRight,
  VideoPlay, InfoFilled, Delete,
} from '@element-plus/icons-vue'
import { isHeicLikeUrl, resolveHeicImageUrl } from '@/utils/heic'
import { deletePage } from '@/api/video'

export interface ViewerItem {
  id: number
  pid: number
  kind: 'image' | 'live_photo'
  fullUrl: string
  name: string
  width: number
  height: number
  file_path?: string
  file_size?: number
  modified_at?: string
}

const props = defineProps<{
  visible: boolean
  items: ViewerItem[]
  index: number
}>()

const emit = defineEmits<{
  (e: 'update:visible', v: boolean): void
  (e: 'update:index', i: number): void
  (e: 'deleted', id: number): void
}>()

const zoom = ref(1)
const rotation = ref(0)
const showInfo = ref(false)
const isPlayingLive = ref(false)
const resolvedUrl = ref('')
const videoEl = ref<HTMLVideoElement | null>(null)

const current = computed(() => props.items[props.index])

const mediaStyle = computed(() => ({
  transform: `scale(${zoom.value}) rotate(${rotation.value}deg)`,
  transition: 'transform 0.2s ease',
}))

function resetTransform() {
  zoom.value = 1
  rotation.value = 0
  isPlayingLive.value = false
}

async function loadCurrent() {
  resetTransform()
  const item = current.value
  if (!item) {
    resolvedUrl.value = ''
    return
  }
  if (isHeicLikeUrl(item.fullUrl)) {
    resolvedUrl.value = ''
    try {
      resolvedUrl.value = await resolveHeicImageUrl(item.fullUrl)
    } catch {
      resolvedUrl.value = item.fullUrl
    }
  } else {
    resolvedUrl.value = item.fullUrl
  }
}

function close() {
  emit('update:visible', false)
}

function prev() {
  if (props.index > 0) emit('update:index', props.index - 1)
}

function next() {
  if (props.index < props.items.length - 1) emit('update:index', props.index + 1)
}

function zoomIn() { zoom.value = Math.min(zoom.value * 1.2, 8) }
function zoomOut() { zoom.value = Math.max(zoom.value / 1.2, 0.1) }
function rotateLeft() { rotation.value -= 90 }
function rotateRight() { rotation.value += 90 }
function toggleInfo() { showInfo.value = !showInfo.value }
function toggleLive() {
  if (current.value?.kind !== 'live_photo') return
  isPlayingLive.value = !isPlayingLive.value
}

function onWheel(e: WheelEvent) {
  if (e.deltaY < 0) zoomIn()
  else zoomOut()
}

async function onDelete() {
  const item = current.value
  if (!item) return
  try {
    await ElMessageBox.confirm(`确定删除「${item.name}」？将同时删除本地文件和数据库记录。`, '删除确认', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }
  try {
    await deletePage(item.id)
    ElMessage.success('已删除')
    const deletedId = item.id
    emit('deleted', deletedId)
    if (props.items.length === 0) {
      close()
      return
    }
    if (props.index >= props.items.length) {
      emit('update:index', props.items.length - 1)
    } else {
      void loadCurrent()
    }
  } catch (err: any) {
    ElMessage.error('删除失败: ' + (err?.message || err))
  }
}

function onKey(e: KeyboardEvent) {
  if (!props.visible) return
  const tag = (e.target as HTMLElement)?.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA') return
  switch (e.key) {
    case 'Escape': close(); break
    case 'ArrowLeft': prev(); break
    case 'ArrowRight': next(); break
    case ' ':
      if (current.value?.kind === 'live_photo') { e.preventDefault(); toggleLive() }
      break
    case 'i': case 'I': toggleInfo(); break
    case '+': case '=': zoomIn(); break
    case '-': case '_': zoomOut(); break
  }
}

function formatBytes(n?: number): string {
  if (!n || n <= 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}

function formatDate(s?: string): string {
  if (!s) return '-'
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  return d.toLocaleString()
}

watch(() => props.visible, (v) => {
  if (v) {
    document.body.style.overflow = 'hidden'
    window.addEventListener('keydown', onKey)
    void loadCurrent()
  } else {
    document.body.style.overflow = ''
    window.removeEventListener('keydown', onKey)
  }
})

watch(() => props.index, () => {
  if (props.visible) void loadCurrent()
})

onBeforeUnmount(() => {
  document.body.style.overflow = ''
  window.removeEventListener('keydown', onKey)
})
</script>

<style scoped>
.iv-overlay {
  position: fixed;
  inset: 0;
  z-index: 3000;
  background: rgba(0, 0, 0, 0.92);
  display: flex;
  flex-direction: column;
  user-select: none;
}

.iv-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: rgba(0, 0, 0, 0.4);
  color: #fff;
  flex: 0 0 auto;
}

.iv-counter {
  font-size: 13px;
  color: #ddd;
}

.iv-actions {
  display: flex;
  gap: 6px;
}

.iv-btn {
  background: transparent;
  border: 1px solid transparent;
  color: #fff;
  width: 36px;
  height: 36px;
  border-radius: 6px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, border-color 0.15s;
}

.iv-btn:hover {
  background: rgba(255, 255, 255, 0.12);
}

.iv-btn.active {
  background: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.3);
}

.iv-danger:hover {
  background: rgba(239, 68, 68, 0.6);
}

.iv-body {
  flex: 1;
  display: flex;
  min-height: 0;
}

.iv-canvas {
  flex: 1;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.iv-stage {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.iv-media {
  max-width: 95%;
  max-height: 95%;
  object-fit: contain;
  background: #000;
}

.iv-loading {
  color: #aaa;
  font-size: 14px;
}

.iv-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  background: rgba(0, 0, 0, 0.4);
  color: #fff;
  border: none;
  width: 44px;
  height: 64px;
  border-radius: 6px;
  cursor: pointer;
  z-index: 2;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.iv-nav:hover:not(:disabled) {
  background: rgba(0, 0, 0, 0.65);
}

.iv-nav:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.iv-nav-left { left: 16px; }
.iv-nav-right { right: 16px; }

.iv-info {
  width: 320px;
  flex: 0 0 320px;
  background: #1f2937;
  color: #e5e7eb;
  padding: 18px 20px;
  overflow-y: auto;
  border-left: 1px solid rgba(255, 255, 255, 0.08);
}

.iv-info-title {
  margin: 0 0 14px;
  font-size: 14px;
  font-weight: 600;
  color: #fff;
}

.iv-info-list {
  margin: 0;
  font-size: 13px;
  display: grid;
  grid-template-columns: 72px 1fr;
  gap: 10px 12px;
}

.iv-info-list dt {
  color: #94a3b8;
  font-weight: 500;
}

.iv-info-list dd {
  margin: 0;
  color: #e5e7eb;
  word-break: break-all;
}

.iv-info-path {
  font-family: 'Consolas', 'Menlo', monospace;
  font-size: 12px;
  line-height: 1.5;
}
</style>
