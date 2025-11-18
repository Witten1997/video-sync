<template>
  <el-dialog
    v-model="dialogVisible"
    :title="currentVideoTitle"
    width="80%"
    :before-close="handleClose"
    class="video-player-dialog"
    append-to-body
    destroy-on-close
    top="5vh"
  >
    <div class="video-player-container">
      <!-- 视频播放器 -->
      <div class="player-wrapper">
        <video
          ref="videoPlayerRef"
          class="video-js vjs-big-play-centered"
        ></video>
      </div>

      <!-- 多P选集列表 -->
      <div v-if="pages && pages.length > 1" class="episode-list">
        <div class="episode-header">
          <span class="episode-title">选集列表 ({{ pages.length }}P)</span>
        </div>
        <el-scrollbar height="300px">
          <div class="episode-items">
            <div
              v-for="page in pages"
              :key="page.id"
              :class="['episode-item', { active: currentPage?.id === page.id, disabled: !isDownloadComplete(page.download_status) }]"
              @click="switchEpisode(page)"
            >
              <div class="episode-info">
                <span class="episode-number">P{{ page.pid }}</span>
                <span class="episode-name">{{ page.name }}</span>
              </div>
              <div class="episode-meta">
                <span v-if="isDownloadComplete(page.download_status)" class="episode-duration">
                  {{ formatDuration(page.duration) }}
                </span>
                <el-tag v-else size="small" type="info">未下载</el-tag>
              </div>
            </div>
          </div>
        </el-scrollbar>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import videojs from 'video.js'
import 'video.js/dist/video-js.css'
import type { Video, Page } from '@/types'
import { ElMessage } from 'element-plus'

interface Props {
  visible: boolean
  video: Video | null
}

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  video: null
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const videoPlayerRef = ref<HTMLVideoElement>()
let player: any = null
const currentPage = ref<Page | null>(null)
const pages = ref<Page[]>([])

// 创建可写的 computed 用于 v-model
const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value)
})

const currentVideoTitle = computed(() => {
  if (!props.video) return ''
  if (currentPage.value) {
    return `${props.video.name} - P${currentPage.value.pid}: ${currentPage.value.name}`
  }
  return props.video.name
})

// 检查是否下载完成
const isDownloadComplete = (status: number): boolean => {
  return status !== 0
}

// 格式化时长
const formatDuration = (seconds: number): string => {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60

  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
  }
  return `${minutes}:${String(secs).padStart(2, '0')}`
}

// 构建视频URL
const buildVideoUrl = (video: Video, page?: Page): string => {
  if (!video.path) {
    throw new Error('视频路径不存在')
  }

  // 视频文件名处理函数（与后端 utils.Filenamify 保持一致）
  const filenamify = (str: string): string => {
    // 替换不允许的字符为下划线（与后端一致）
    let name = str.replace(/[<>:"/\\|?*\x00-\x1f]/g, '_')

    // 去除首尾空格和点
    name = name.replace(/^[\s.]+|[\s.]+$/g, '')

    // 限制长度（与后端保持一致）
    if (name.length > 200) {
      name = name.substring(0, 200)
    }

    // 如果结果为空，使用默认名称
    if (!name) {
      name = 'unnamed'
    }

    return name
  }

  // video.path 现在已经是相对路径（如 "test/Deja Vu"）
  const videoFolder = video.path

  let fileName: string
  if (page) {
    // 多P视频
    fileName = `${filenamify(video.name)}-${filenamify(page.name)}.mp4`
  } else {
    // 单P视频
    fileName = `${filenamify(video.name)}.mp4`
  }

  // 构建完整URL
  return `/downloads/${videoFolder}/${fileName}`
}

// 初始化播放器
const initPlayer = () => {
  if (!videoPlayerRef.value) return

  player = videojs(videoPlayerRef.value, {
    controls: true,
    autoplay: false,
    preload: 'auto',
    fluid: true,
    aspectRatio: '16:9',
    playbackRates: [0.5, 0.75, 1, 1.25, 1.5, 2],
    controlBar: {
      children: [
        'playToggle',
        'currentTimeDisplay',
        'timeDivider',
        'durationDisplay',
        'progressControl',
        'volumePanel',
        'playbackRateMenuButton',
        'fullscreenToggle'
      ]
    },
    language: 'zh-CN',
    languages: {
      'zh-CN': {
        'Play': '播放',
        'Pause': '暂停',
        'Current Time': '当前时间',
        'Duration': '时长',
        'Remaining Time': '剩余时间',
        'Loaded': '已加载',
        'Progress': '进度',
        'Fullscreen': '全屏',
        'Exit Fullscreen': '退出全屏',
        'Mute': '静音',
        'Unmute': '取消静音',
        'Playback Rate': '播放速度',
        'Subtitles': '字幕',
        'subtitles off': '关闭字幕',
        'Captions': '内嵌字幕',
        'captions off': '关闭内嵌字幕',
        'Chapters': '节目段落',
        'You aborted the media playback': '视频播放被终止',
        'A network error caused the media download to fail part-way.': '网络错误导致视频下载中途失败',
        'The media could not be loaded, either because the server or network failed or because the format is not supported.': '视频因格式不支持或者服务器或网络的问题无法加载',
        'The media playback was aborted due to a corruption problem or because the media used features your browser did not support.': '由于视频文件损坏或是该视频使用了你的浏览器不支持的功能，播放终止',
        'No compatible source was found for this media.': '无法找到此视频兼容的源'
      }
    }
  })

  // 监听错误事件
  player.on('error', () => {
    const error = player.error()
    console.error('Video.js error:', error)
    ElMessage.error(`播放失败: ${error?.message || '未知错误'}`)
  })
}

// 加载视频
const loadVideo = async (video: Video, page?: Page) => {
  if (!player) return

  try {
    const videoUrl = buildVideoUrl(video, page)
    console.log('Loading video:', videoUrl)

    player.src({
      type: 'video/mp4',
      src: videoUrl
    })

    player.load()
    currentPage.value = page || null

    // 尝试自动播放
    setTimeout(() => {
      player.play().catch((err: Error) => {
        console.warn('Auto-play failed:', err)
      })
    }, 100)
  } catch (error: any) {
    ElMessage.error(error.message || '加载视频失败')
  }
}

// 切换分集
const switchEpisode = (page: Page) => {
  if (!isDownloadComplete(page.download_status)) {
    ElMessage.warning('该分集尚未下载完成')
    return
  }

  if (props.video) {
    loadVideo(props.video, page)
  }
}

// 关闭对话框
const handleClose = () => {
  if (player) {
    player.pause()
  }
  emit('update:visible', false)
}

// 监听对话框显示状态
watch(() => props.visible, (newVal) => {
  if (newVal && props.video) {
    // 对话框打开时初始化播放器
    setTimeout(() => {
      initPlayer()

      // 判断是单P还是多P视频
      if (props.video.single_page) {
        // 单P视频：直接播放，不传 page 参数
        if (isDownloadComplete(props.video.download_status)) {
          pages.value = []
          loadVideo(props.video)
        } else {
          ElMessage.warning('该视频尚未下载完成')
        }
      } else {
        // 多P视频：加载分P列表
        if (props.video.pages && props.video.pages.length > 0) {
          pages.value = props.video.pages
          // 查找第一个已下载的分集
          const firstDownloadedPage = pages.value.find(p => isDownloadComplete(p.download_status))
          if (firstDownloadedPage) {
            loadVideo(props.video, firstDownloadedPage)
          } else {
            ElMessage.warning('该视频的所有分集尚未下载完成')
          }
        } else {
          ElMessage.warning('该视频没有可播放的分集')
        }
      }
    }, 100)
  } else if (!newVal && player) {
    // 对话框关闭时销毁播放器
    player.dispose()
    player = null
    currentPage.value = null
    pages.value = []
  }
})

// 组件卸载时清理
onBeforeUnmount(() => {
  if (player) {
    player.dispose()
  }
})
</script>

<style scoped lang="scss">
.video-player-dialog {
  :deep(.el-dialog) {
    max-height: 90vh;
    display: flex;
    flex-direction: column;
  }

  :deep(.el-dialog__body) {
    padding: 10px;
    overflow-y: auto;
    flex: 1;
  }
}

.video-player-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.player-wrapper {
  width: 100%;
  max-width: 100%;
  background: #000;
  border-radius: 4px;

  // Video.js播放器容器
  :deep(.video-js) {
    width: 100%;
    max-height: 70vh;
  }
}

.episode-list {
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  overflow: hidden;

  .episode-header {
    padding: 12px 16px;
    background: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color);

    .episode-title {
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }

  .episode-items {
    padding: 8px;
  }

  .episode-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    margin-bottom: 8px;
    border-radius: 4px;
    background: var(--el-fill-color-lighter);
    cursor: pointer;
    transition: all 0.2s;

    &:hover:not(.disabled) {
      background: var(--el-fill-color);
      transform: translateX(4px);
    }

    &.active {
      background: var(--el-color-primary-light-9);
      border-left: 3px solid var(--el-color-primary);
      padding-left: 13px;

      .episode-number {
        color: var(--el-color-primary);
        font-weight: 600;
      }
    }

    &.disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    .episode-info {
      display: flex;
      align-items: center;
      gap: 12px;
      flex: 1;
      overflow: hidden;

      .episode-number {
        flex-shrink: 0;
        font-weight: 500;
        color: var(--el-text-color-secondary);
        min-width: 40px;
      }

      .episode-name {
        color: var(--el-text-color-primary);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }

    .episode-meta {
      flex-shrink: 0;
      display: flex;
      align-items: center;
      gap: 8px;

      .episode-duration {
        color: var(--el-text-color-secondary);
        font-size: 14px;
      }
    }
  }
}

// 响应式设计
@media (max-width: 768px) {
  .video-player-dialog {
    width: 100% !important;

    :deep(.el-dialog) {
      margin: 0 !important;
      max-height: 100vh;
    }
  }

  .player-wrapper {
    :deep(.video-js) {
      max-height: 60vh;
    }
  }

  .episode-list {
    .episode-item {
      flex-direction: column;
      align-items: flex-start;
      gap: 8px;

      .episode-meta {
        width: 100%;
        justify-content: space-between;
      }
    }
  }
}
</style>
