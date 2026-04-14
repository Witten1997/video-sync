<template>
  <div class="telegram-integration">
    <div class="page-header">
      <div class="page-header-left">
        <el-button text @click="router.push({ name: 'Integrations' })">
          <span class="material-icons-round" style="font-size: 20px">arrow_back</span>
        </el-button>
        <div>
          <h2>Telegram</h2>
          <p>配置 Telegram Bot 连接、权限与通知</p>
        </div>
      </div>
      <el-space>
        <el-button @click="goToTelegramRequests">请求日志</el-button>
        <el-button @click="loadData({ validateCredential: false })">重置</el-button>
        <el-button type="primary" @click="handleSave">保存配置</el-button>
      </el-space>
    </div>

    <div class="telegram-content" v-loading="loading">
      <!-- 运行状态 -->
      <el-card class="telegram-status-card" shadow="never">
        <template #header>
          <div class="telegram-status-header">
            <span>运行状态</span>
            <el-button link type="primary" @click="loadTelegramRuntimeStatus">刷新</el-button>
          </div>
        </template>

        <div class="telegram-status-grid">
          <div class="telegram-status-item">
            <span class="label">状态</span>
            <el-tag :type="telegramStatusTagType">{{ telegramStatus.running ? '运行中' : '已停止' }}</el-tag>
          </div>
          <div class="telegram-status-item">
            <span class="label">已启用</span>
            <span>{{ telegramStatus.enabled ? '是' : '否' }}</span>
          </div>
          <div class="telegram-status-item">
            <span class="label">模式</span>
            <span>{{ formatTelegramMode(telegramStatus.mode) }}</span>
          </div>
          <div class="telegram-status-item">
            <span class="label">机器人名称</span>
            <span>{{ telegramStatus.bot_name || '-' }}</span>
          </div>
          <div class="telegram-status-item">
            <span class="label">最近更新 ID</span>
            <span>{{ telegramStatus.last_update_id || 0 }}</span>
          </div>
          <div class="telegram-status-item">
            <span class="label">{{ telegramLastActivityLabel }}</span>
            <span>{{ formatStatusTime(telegramStatus.last_poll_at) }}</span>
          </div>
          <div class="telegram-status-item telegram-status-item-wide">
            <span class="label">最近错误</span>
            <span>{{ telegramStatus.last_error || '-' }}</span>
          </div>
          <div class="telegram-status-item telegram-status-item-wide">
            <span class="label">最近错误时间</span>
            <span>{{ formatStatusTime(telegramStatus.last_error_at) }}</span>
          </div>
        </div>

        <el-divider content-position="left">运维操作</el-divider>
        <div class="telegram-operator-actions">
          <el-button
            :loading="telegramReconnectLoading"
            :disabled="!telegramStatus.running"
            @click="handleTelegramReconnect"
          >
            重连
          </el-button>
          <el-input
            v-model="telegramTestSend.chat_id"
            class="telegram-action-field telegram-chat-id-field"
            placeholder="目标 Chat ID，默认取第一个允许的 Chat ID"
          />
          <el-input
            v-model="telegramTestSend.message"
            class="telegram-action-field"
            placeholder="可选测试消息"
          />
          <el-button type="primary" :loading="telegramTestSendLoading" @click="handleTelegramTestSend">
            测试发送
          </el-button>
        </div>
        <span class="help-text">{{ telegramOperatorHelpText }}</span>
      </el-card>

      <!-- 配置表单 -->
      <el-card shadow="never">
        <template #header>
          <span>连接配置</span>
        </template>
        <el-form :model="config.telegram" label-width="180px">
          <el-form-item label="启用 Telegram">
            <el-switch v-model="config.telegram.enabled" />
          </el-form-item>
          <el-form-item label="机器人 Token">
            <el-input
              v-model="config.telegram.bot_token"
              type="password"
              show-password
              :placeholder="telegramBotTokenPlaceholder"
            />
            <div class="telegram-secret-help">
              <span class="help-text">后端不会把已保存的 Token 明文返回给浏览器。</span>
              <el-tag v-if="config.telegram.bot_token_configured" type="success" size="small">已保存</el-tag>
              <el-tag v-else type="info" size="small">未配置</el-tag>
            </div>
          </el-form-item>
          <el-form-item label="运行模式">
            <el-select v-model="config.telegram.mode" style="width: 180px">
              <el-option label="轮询（Polling）" value="polling" />
              <el-option label="回调（Webhook）" value="webhook" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="config.telegram.mode === 'polling'" label="轮询超时时间（秒）">
            <el-input-number v-model="config.telegram.poll_timeout_seconds" :min="10" :max="60" />
          </el-form-item>
          <el-form-item v-if="config.telegram.mode === 'webhook'" label="Webhook 地址">
            <el-input
              v-model="config.telegram.webhook_url"
              placeholder="https://example.com/telegram/webhook"
            />
            <span class="help-text">填写可公网访问的 HTTPS 回调地址，并指向本服务的 `/telegram/webhook`。</span>
          </el-form-item>
          <el-form-item v-if="config.telegram.mode === 'webhook'" label="Webhook 密钥">
            <el-input
              v-model="config.telegram.webhook_secret"
              type="password"
              show-password
              :placeholder="telegramWebhookSecretPlaceholder"
            />
            <div class="telegram-secret-help">
              <span class="help-text">仅支持 1-256 位字母、数字、下划线或连字符。后端会校验 `X-Telegram-Bot-Api-Secret-Token`，且不会把已保存密钥明文返回给浏览器。</span>
              <el-tag v-if="config.telegram.webhook_secret_configured" type="success" size="small">已保存</el-tag>
              <el-tag v-else type="info" size="small">未配置</el-tag>
            </div>
          </el-form-item>
          <el-form-item label="单条消息最大 URL 数">
            <el-input-number v-model="config.telegram.max_urls_per_message" :min="1" :max="1" />
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 权限配置 -->
      <el-card shadow="never">
        <template #header>
          <span>权限配置</span>
        </template>
        <el-form :model="config.telegram" label-width="180px">
          <el-form-item label="允许的 Chat ID">
            <el-input
              v-model="telegramAllowedChatIDsText"
              type="textarea"
              :rows="3"
              placeholder="每行一个 Chat ID"
            />
          </el-form-item>
          <el-form-item label="允许的用户 ID">
            <el-input
              v-model="telegramAllowedUserIDsText"
              type="textarea"
              :rows="3"
              placeholder="每行一个用户 ID"
            />
          </el-form-item>
          <el-form-item label="允许的聊天类型">
            <el-select v-model="config.telegram.allowed_chat_types" multiple style="width: 320px">
              <el-option label="私聊" value="private" />
              <el-option label="群组" value="group" />
              <el-option label="超级群组" value="supergroup" />
            </el-select>
            <span class="help-text">私聊保持现有的直接 URL 提交流程。群组和超级群组消息只处理 `/download@botname`、`/status@botname` 或以 `@botname` 开头的消息。</span>
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 通知配置 -->
      <el-card shadow="never">
        <template #header>
          <span>通知配置</span>
        </template>
        <el-form :model="config.telegram" label-width="180px">
          <el-form-item label="受理时通知">
            <el-switch v-model="config.telegram.notify_on_accept" />
          </el-form-item>
          <el-form-item label="完成时通知">
            <el-switch v-model="config.telegram.notify_on_complete" />
          </el-form-item>
          <el-form-item label="失败时通知">
            <el-switch v-model="config.telegram.notify_on_fail" />
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 待批准会话 -->
      <el-card class="telegram-status-card" shadow="never">
        <template #header>
          <div class="telegram-status-header">
            <span>待批准会话</span>
            <el-button link type="primary" @click="loadTelegramAccessCandidates">刷新</el-button>
          </div>
        </template>

        <el-empty
          v-if="!telegramAccessCandidatesLoading && telegramAccessCandidates.length === 0"
          description="暂无待批准会话"
        />

        <el-table
          v-else
          :data="telegramAccessCandidates"
          v-loading="telegramAccessCandidatesLoading"
          style="width: 100%"
        >
          <el-table-column label="Chat / User" min-width="180">
            <template #default="{ row }">
              <div>{{ row.chat_id }}</div>
              <div class="subtext">{{ row.user_id }}</div>
            </template>
          </el-table-column>
          <el-table-column label="身份" min-width="180">
            <template #default="{ row }">
              <div>{{ formatTelegramCandidateName(row) }}</div>
              <div class="subtext">{{ row.username ? `@${row.username}` : row.chat_type }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="last_message" label="最近消息" min-width="260" show-overflow-tooltip />
          <el-table-column label="最近出现" min-width="180">
            <template #default="{ row }">
              {{ formatStatusTime(row.last_seen_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" min-width="240" fixed="right">
            <template #default="{ row }">
              <el-space wrap>
                <el-button size="small" @click="handleApproveTelegramAccessCandidate(row, 'chat')">
                  加入 Chat
                </el-button>
                <el-button size="small" @click="handleApproveTelegramAccessCandidate(row, 'user')">
                  加入 User
                </el-button>
                <el-button type="primary" size="small" @click="handleApproveTelegramAccessCandidate(row, 'both')">
                  全部批准
                </el-button>
              </el-space>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { getConfig, updateConfig } from '@/api/config'
import {
  approveTelegramAccessCandidate,
  getTelegramAccessCandidates,
  getTelegramStatus,
  reconnectTelegram,
  sendTelegramTestMessage
} from '@/api/telegram'
import type { TelegramAccessCandidate, TelegramRuntimeStatus } from '@/types'

defineOptions({
  name: 'TelegramIntegration'
})

const router = useRouter()
const loading = ref(false)

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
const telegramReconnectLoading = ref(false)
const telegramTestSendLoading = ref(false)
const telegramAccessCandidatesLoading = ref(false)
const telegramTestSend = ref({
  chat_id: '',
  message: ''
})
const telegramAccessCandidates = ref<TelegramAccessCandidate[]>([])

const config = ref({
  telegram: {
    enabled: false,
    bot_token: '',
    bot_token_configured: false,
    mode: 'polling',
    poll_timeout_seconds: 30,
    webhook_url: '',
    webhook_secret: '',
    webhook_secret_configured: false,
    allowed_chat_ids: [] as number[],
    allowed_user_ids: [] as number[],
    allowed_chat_types: ['private'] as string[],
    max_urls_per_message: 1,
    notify_on_accept: true,
    notify_on_complete: true,
    notify_on_fail: true
  }
})

const parseNumberList = (value: string) => {
  return value
    .split(/[\n,]/)
    .map(item => item.trim())
    .filter(Boolean)
    .map(item => Number(item))
    .filter(item => Number.isFinite(item))
}

const telegramAllowedChatIDsText = computed({
  get: () => config.value.telegram.allowed_chat_ids.join('\n'),
  set: (value: string) => {
    config.value.telegram.allowed_chat_ids = parseNumberList(value)
  }
})

const telegramAllowedUserIDsText = computed({
  get: () => config.value.telegram.allowed_user_ids.join('\n'),
  set: (value: string) => {
    config.value.telegram.allowed_user_ids = parseNumberList(value)
  }
})

const formatStatusTime = (value?: string | null) => {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN')
}

const telegramStatusTagType = computed(() => {
  if (!telegramStatus.value.enabled) return 'info'
  return telegramStatus.value.running ? 'success' : 'warning'
})

const formatTelegramMode = (mode?: string | null) => {
  if (!mode) return '-'
  const mapping: Record<string, string> = {
    polling: '轮询（Polling）',
    webhook: '回调（Webhook）'
  }
  return mapping[mode] || mode
}

const telegramLastActivityLabel = computed(() => {
  return telegramStatus.value.mode === 'webhook' ? '最近投递时间' : '最近轮询时间'
})

const telegramOperatorHelpText = computed(() => {
  if (config.value.telegram.mode === 'webhook') {
    return '重连会重新应用当前保存的 Webhook 注册配置。测试发送使用当前保存的 Bot Token。'
  }
  return '重连会按当前保存的运行配置重启轮询循环。测试发送使用当前保存的 Bot Token。'
})

const telegramBotTokenPlaceholder = computed(() => {
  return config.value.telegram.bot_token_configured ? '已保存，留空则保持当前 Token 不变' : '请输入 Bot Token'
})

const telegramWebhookSecretPlaceholder = computed(() => {
  return config.value.telegram.webhook_secret_configured ? '已保存，留空则保持当前 Webhook 密钥不变' : '请输入 Webhook 密钥'
})

const loadTelegramRuntimeStatus = async () => {
  try {
    telegramStatus.value = await getTelegramStatus()
  } catch (error) {
    console.error('load telegram status failed:', error)
  }
}

const loadTelegramAccessCandidates = async () => {
  telegramAccessCandidatesLoading.value = true
  try {
    telegramAccessCandidates.value = await getTelegramAccessCandidates()
  } catch (error) {
    console.error('load telegram access candidates failed:', error)
  } finally {
    telegramAccessCandidatesLoading.value = false
  }
}

const goToTelegramRequests = () => {
  router.push({ name: 'TelegramRequests' })
}

const handleTelegramReconnect = async () => {
  telegramReconnectLoading.value = true
  try {
    const result = await reconnectTelegram()
    ElMessage.success(result.message || '已提交 Telegram 重连请求')
    await loadTelegramRuntimeStatus()
  } catch (error: any) {
    const errorMsg = error?.response?.data?.message || error?.message || 'Telegram 重连失败'
    ElMessage.error(errorMsg)
  } finally {
    telegramReconnectLoading.value = false
  }
}

const resolveTelegramTestChatID = () => {
  const directValue = telegramTestSend.value.chat_id.trim()
  if (directValue) {
    const parsed = Number(directValue)
    if (!Number.isInteger(parsed) || parsed === 0) {
      throw new Error('目标 Chat ID 必须是非 0 整数')
    }
    return parsed
  }
  const fallbackChatID = config.value.telegram.allowed_chat_ids[0]
  if (Number.isInteger(fallbackChatID) && fallbackChatID !== 0) {
    return fallbackChatID
  }
  throw new Error('请先填写目标 Chat ID，或至少配置一个允许的 Chat ID')
}

const handleTelegramTestSend = async () => {
  telegramTestSendLoading.value = true
  try {
    const chatID = resolveTelegramTestChatID()
    const result = await sendTelegramTestMessage({
      chat_id: chatID,
      message: telegramTestSend.value.message.trim() || undefined
    })
    telegramTestSend.value.chat_id = String(chatID)
    ElMessage.success(`测试消息已发送到 ${result.chat_id}（消息 #${result.message_id}）`)
    await loadTelegramRuntimeStatus()
  } catch (error: any) {
    const errorMsg = error?.response?.data?.message || error?.message || 'Telegram 测试发送失败'
    ElMessage.error(errorMsg)
  } finally {
    telegramTestSendLoading.value = false
  }
}

const formatTelegramCandidateName = (candidate: TelegramAccessCandidate) => {
  const parts = [candidate.first_name, candidate.last_name].filter(Boolean)
  if (parts.length > 0) return parts.join(' ')
  if (candidate.username) return `@${candidate.username}`
  return '-'
}

const handleApproveTelegramAccessCandidate = async (
  candidate: TelegramAccessCandidate,
  mode: 'chat' | 'user' | 'both'
) => {
  try {
    await approveTelegramAccessCandidate(candidate.id, {
      approve_chat_id: mode === 'chat' || mode === 'both',
      approve_user_id: mode === 'user' || mode === 'both'
    })
    ElMessage.success('已加入 Telegram 白名单')
    await loadData({ validateCredential: false })
    await loadTelegramRuntimeStatus()
    await loadTelegramAccessCandidates()
  } catch (error: any) {
    const errorMsg = error?.response?.data?.message || error?.message || '批准 Telegram 会话失败'
    ElMessage.error(errorMsg)
  }
}

const loadData = async (_options: { validateCredential?: boolean } = {}) => {
  loading.value = true
  try {
    const data = await getConfig()
    if (data?.telegram) {
      const tg = data.telegram
      config.value.telegram = {
        enabled: tg.enabled ?? false,
        bot_token: '',
        bot_token_configured: tg.bot_token_configured ?? false,
        mode: tg.mode ?? 'polling',
        poll_timeout_seconds: tg.poll_timeout_seconds ?? 30,
        webhook_url: tg.webhook_url ?? '',
        webhook_secret: '',
        webhook_secret_configured: tg.webhook_secret_configured ?? false,
        allowed_chat_ids: tg.allowed_chat_ids ?? [],
        allowed_user_ids: tg.allowed_user_ids ?? [],
        allowed_chat_types: tg.allowed_chat_types ?? ['private'],
        max_urls_per_message: tg.max_urls_per_message ?? 1,
        notify_on_accept: tg.notify_on_accept ?? true,
        notify_on_complete: tg.notify_on_complete ?? true,
        notify_on_fail: tg.notify_on_fail ?? true
      }
    }
  } catch (error) {
    console.error('加载配置失败:', error)
    ElMessage.error('加载配置失败')
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  loading.value = true
  try {
    const result = await updateConfig({ telegram: config.value.telegram })
    if (result.restart_needed) {
      ElMessage.warning(`已保存，需要重启: ${result.requires_restart.join(', ')}`)
    } else {
      ElMessage.success(result.message || '保存成功')
    }
    await loadData({ validateCredential: false })
    await loadTelegramRuntimeStatus()
    await loadTelegramAccessCandidates()
  } catch (error) {
    console.error('保存配置失败:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadData()
  loadTelegramRuntimeStatus()
  loadTelegramAccessCandidates()
})
</script>

<style scoped>
.telegram-integration {
  padding: 32px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}

.page-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.page-header h2 {
  margin: 0 0 4px;
  font-size: 1.25rem;
  font-weight: 700;
  color: #1e293b;
}

.page-header p {
  margin: 0;
  color: #64748b;
  font-size: 13px;
}

.telegram-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.help-text {
  font-size: 12px;
  color: #94a3b8;
  display: block;
  margin-top: 5px;
}

.telegram-status-card {
  border-color: #e2e8f0;
}

.telegram-status-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.telegram-status-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 20px;
}

.telegram-status-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.telegram-status-item-wide {
  grid-column: 1 / -1;
}

.telegram-status-item .label {
  font-size: 12px;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.telegram-secret-help {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.telegram-operator-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.telegram-action-field {
  flex: 1;
  min-width: 220px;
}

.telegram-chat-id-field {
  max-width: 280px;
}

.subtext {
  font-size: 12px;
  color: #94a3b8;
}
</style>
