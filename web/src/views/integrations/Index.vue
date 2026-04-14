<template>
  <div class="integrations">
    <div class="page-header">
      <h2>平台集成</h2>
      <p>管理第三方平台的连接与配置</p>
    </div>

    <div class="platform-grid">
      <!-- Telegram -->
      <el-card class="platform-card" shadow="hover" @click="router.push({ name: 'TelegramIntegration' })">
        <div class="platform-card-body">
          <div class="platform-icon telegram-icon">
            <span class="material-icons-round">send</span>
          </div>
          <div class="platform-info">
            <div class="platform-name">Telegram</div>
            <div class="platform-desc">通过 Telegram Bot 提交下载链接、接收通知</div>
          </div>
          <div class="platform-status">
            <el-tag :type="telegramStatusType" size="small">{{ telegramStatusText }}</el-tag>
          </div>
        </div>
      </el-card>

      <!-- 飞书 - 即将支持 -->
      <el-card class="platform-card platform-card-disabled" shadow="never">
        <div class="platform-card-body">
          <div class="platform-icon feishu-icon">
            <span class="material-icons-round">chat</span>
          </div>
          <div class="platform-info">
            <div class="platform-name">飞书</div>
            <div class="platform-desc">通过飞书机器人提交下载链接、接收通知</div>
          </div>
          <div class="platform-status">
            <el-tag type="info" size="small">即将支持</el-tag>
          </div>
        </div>
      </el-card>

      <!-- QQ - 即将支持 -->
      <el-card class="platform-card platform-card-disabled" shadow="never">
        <div class="platform-card-body">
          <div class="platform-icon qq-icon">
            <span class="material-icons-round">forum</span>
          </div>
          <div class="platform-info">
            <div class="platform-name">QQ</div>
            <div class="platform-desc">通过 QQ 机器人提交下载链接、接收通知</div>
          </div>
          <div class="platform-status">
            <el-tag type="info" size="small">即将支持</el-tag>
          </div>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getTelegramStatus } from '@/api/telegram'
import type { TelegramRuntimeStatus } from '@/types'

defineOptions({
  name: 'Integrations'
})

const router = useRouter()

const telegramStatus = ref<TelegramRuntimeStatus>({
  enabled: false,
  running: false,
  mode: 'polling',
  bot_name: '',
  last_update_id: 0,
  last_poll_at: null,
  last_error: '',
  last_error_at: null
})

const telegramStatusType = computed(() => {
  if (!telegramStatus.value.enabled) return 'info'
  return telegramStatus.value.running ? 'success' : 'warning'
})

const telegramStatusText = computed(() => {
  if (!telegramStatus.value.enabled) return '未启用'
  return telegramStatus.value.running ? '运行中' : '已停止'
})

onMounted(async () => {
  try {
    telegramStatus.value = await getTelegramStatus()
  } catch {
    // ignore
  }
})
</script>

<style scoped>
.integrations {
  padding: 32px;
}

.page-header {
  margin-bottom: 24px;
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

.platform-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 16px;
}

.platform-card {
  cursor: pointer;
  transition: all 0.2s;
}

.platform-card:hover {
  border-color: #3b82f6;
}

.platform-card-disabled {
  cursor: default;
  opacity: 0.6;
}

.platform-card-disabled:hover {
  border-color: #e2e8f0;
}

.platform-card-body {
  display: flex;
  align-items: center;
  gap: 16px;
}

.platform-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.platform-icon .material-icons-round {
  font-size: 24px;
  color: #fff;
}

.telegram-icon {
  background: #2AABEE;
}

.feishu-icon {
  background: #3370FF;
}

.qq-icon {
  background: #12B7F5;
}

.platform-info {
  flex: 1;
  min-width: 0;
}

.platform-name {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin-bottom: 4px;
}

.platform-desc {
  font-size: 13px;
  color: #94a3b8;
}

.platform-status {
  flex-shrink: 0;
}
</style>
