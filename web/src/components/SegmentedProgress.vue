<template>
  <div class="segmented-progress">
    <el-tooltip
      v-for="(file, index) in files"
      :key="file.name"
      :content="getTooltip(file)"
      placement="top"
    >
      <div
        class="segment"
        :style="{ width: segmentWidth }"
        :class="{ 'segment-gap': index > 0 }"
      >
        <div class="segment-bg">
          <div
            class="segment-fill"
            :class="getStatusClass(file.status)"
            :style="{ width: file.progress + '%' }"
          />
        </div>
      </div>
    </el-tooltip>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FileDetail } from '@/types'

const props = defineProps<{
  files: FileDetail[]
}>()

const segmentWidth = computed(() => {
  return `calc(${100 / props.files.length}% - ${props.files.length > 1 ? 1 : 0}px)`
})

const getStatusClass = (status: string) => {
  switch (status) {
    case 'downloading': return 'fill-blue'
    case 'completed':
    case 'succeeded': return 'fill-green'
    case 'failed': return 'fill-red'
    case 'skipped': return 'fill-gray'
    case 'pending': return 'fill-blue'
    default: return 'fill-gray'
  }
}

const getTooltip = (file: FileDetail) => {
  const statusMap: Record<string, string> = {
    pending: '等待中',
    downloading: '下载中',
    completed: '已完成',
    succeeded: '已完成',
    failed: '失败',
    skipped: '已跳过'
  }
  const status = statusMap[file.status] || file.status
  const size = file.size > 0 ? formatSize(file.size) : '-'
  return `${file.label}: ${status} (${size})`
}

const formatSize = (bytes: number) => {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / 1024 / 1024).toFixed(1) + ' MB'
  return (bytes / 1024 / 1024 / 1024).toFixed(2) + ' GB'
}
</script>

<style scoped>
.segmented-progress {
  display: flex;
  align-items: center;
  height: 20px;
  min-width: 120px;
}
.segment {
  height: 100%;
  flex-shrink: 0;
}
.segment-gap {
  margin-left: 1px;
}
.segment-bg {
  width: 100%;
  height: 100%;
  background: #f1f5f9;
  border-radius: 4px;
  overflow: hidden;
}
.segment-fill {
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s ease;
}
.fill-blue { background: #3b82f6; }
.fill-green { background: #22c55e; }
.fill-red { background: #ef4444; }
.fill-gray { background: #cbd5e1; }
</style>
