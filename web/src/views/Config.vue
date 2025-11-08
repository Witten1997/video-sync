<template>
  <div class="config">
    <el-card v-loading="loading">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="基本设置" name="basic">
          <el-form :model="config" label-width="180px">
            <el-form-item label="同步间隔（秒）">
              <el-input-number v-model="config.sync.interval" :min="60" />
            </el-form-item>
            <el-form-item label="下载基础路径">
              <el-input v-model="config.paths.download_base" />
            </el-form-item>
            <el-form-item label="UP主信息路径">
              <el-input v-model="config.paths.upper_path" />
            </el-form-item>
            <el-form-item label="视频名称模板">
              <el-input v-model="config.template.video_name" />
              <span class="help-text">
                可用变量: {{ videoNameHelp }}
              </span>
            </el-form-item>
            <el-form-item label="分P名称模板">
              <el-input v-model="config.template.page_name" />
              <span class="help-text">
                可用变量: {{ pageNameHelp }}
              </span>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="B站认证" name="bilibili">
          <el-form :model="config.bilibili.credential" label-width="180px">
            <el-form-item label="SESSDATA">
              <el-input v-model="config.bilibili.credential.sessdata" type="password" show-password />
            </el-form-item>
            <el-form-item label="bili_jct">
              <el-input v-model="config.bilibili.credential.bili_jct" />
            </el-form-item>
            <el-form-item label="buvid3">
              <el-input v-model="config.bilibili.credential.buvid3" />
            </el-form-item>
            <el-form-item label="DedeUserID">
              <el-input v-model="config.bilibili.credential.dedeuserid" />
            </el-form-item>
            <el-form-item label="ac_time_value">
              <el-input v-model="config.bilibili.credential.ac_time_value" />
            </el-form-item>

            <!-- 认证验证状态 -->
            <el-form-item v-if="credentialValidation.show" label="验证状态">
              <el-alert
                :type="credentialValidation.valid ? 'success' : 'error'"
                :title="credentialValidation.message"
                :closable="false"
                show-icon
              >
                <template v-if="credentialValidation.valid && credentialValidation.userInfo" #default>
                  <div class="user-info">
                    <p><strong>用户名:</strong> {{ credentialValidation.userInfo.uname }}</p>
                    <p><strong>UID:</strong> {{ credentialValidation.userInfo.mid }}</p>
                    <p><strong>等级:</strong> Lv{{ credentialValidation.userInfo.level }}</p>
                    <p v-if="credentialValidation.userInfo.vip_status === 1"><strong>会员状态:</strong> 大会员</p>
                  </div>
                </template>
              </el-alert>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="视频设置" name="video">
          <el-form label-width="180px">
            <!-- 视频质量设置 -->
            <el-divider content-position="left">视频质量</el-divider>

            <el-form-item label="最高分辨率">
              <el-select v-model="config.quality.max_resolution">
                <el-option label="8K" value="8K" />
                <el-option label="杜比视界" value="DOLBY" />
                <el-option label="HDR真彩" value="HDR" />
                <el-option label="4K超清" value="4K" />
                <el-option label="1080P60" value="1080P60" />
                <el-option label="1080P高清" value="1080P+" />
                <el-option label="1080P" value="1080P" />
                <el-option label="720P" value="720P" />
                <el-option label="480P" value="480P" />
                <el-option label="360P" value="360P" />
              </el-select>
            </el-form-item>

            <el-form-item label="视频编码格式">
              <el-select v-model="config.quality.codec_priority" multiple placeholder="请选择编码格式（按优先级排序）">
                <el-option label="AVC (H.264)" value="AVC" />
                <el-option label="HEVC (H.265)" value="HEVC" />
                <el-option label="AV1" value="AV1" />
              </el-select>
              <div style="font-size: 12px; color: #909399; margin-top: 4px;">
                排在前面的编码格式优先级更高，可拖拽调整顺序
              </div>
            </el-form-item>

            <el-form-item label="音频质量">
              <el-select v-model="config.quality.audio_quality">
                <el-option label="Hi-RES无损" value="30251" />
                <el-option label="杜比全景声" value="30250" />
                <el-option label="192K" value="30280" />
                <el-option label="132K" value="30232" />
                <el-option label="64K" value="30216" />
              </el-select>
            </el-form-item>

            <el-form-item label="启用CDN排序">
              <el-switch v-model="config.quality.cdn_sort" />
            </el-form-item>

            <!-- 下载选项 -->
            <el-divider content-position="left">下载选项</el-divider>

            <el-form-item label="跳过封面下载">
              <el-switch v-model="config.download.skip_poster" />
            </el-form-item>
            <el-form-item label="跳过NFO元数据">
              <el-switch v-model="config.download.skip_video_nfo" />
            </el-form-item>
            <el-form-item label="跳过UP主信息">
              <el-switch v-model="config.download.skip_upper" />
            </el-form-item>
            <el-form-item label="跳过弹幕">
              <el-switch v-model="config.download.skip_danmaku" />
            </el-form-item>
            <el-form-item label="跳过字幕">
              <el-switch v-model="config.download.skip_subtitle" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="弹幕设置" name="danmaku">
          <el-form :model="config.danmaku" label-width="220px">
            <el-form-item label="弹幕持续时间（秒）">
              <el-input-number v-model="config.danmaku.duration" :min="1" :max="30" :step="0.5" />
              <span style="margin-left: 10px; font-size: 12px; color: #909399;">
                弹幕在屏幕上的停留时间
              </span>
            </el-form-item>

            <el-form-item label="字体名称">
              <el-input v-model="config.danmaku.font_name" placeholder="Microsoft YaHei" />
            </el-form-item>

            <el-form-item label="字体大小">
              <el-input-number v-model="config.danmaku.font_size" :min="10" :max="100" />
            </el-form-item>

            <el-form-item label="宽度比例">
              <el-input-number v-model="config.danmaku.width_ratio" :min="0.5" :max="3" :step="0.1" />
              <span style="margin-left: 10px; font-size: 12px; color: #909399;">
                弹幕显示宽度相对于视频宽度的比例
              </span>
            </el-form-item>

            <el-form-item label="水平间距（像素）">
              <el-input-number v-model="config.danmaku.horizontal_gap" :min="0" :max="200" />
              <span style="margin-left: 10px; font-size: 12px; color: #909399;">
                弹幕之间的水平间隔
              </span>
            </el-form-item>

            <el-form-item label="轨道大小（像素）">
              <el-input-number v-model="config.danmaku.lane_size" :min="20" :max="100" />
              <span style="margin-left: 10px; font-size: 12px; color: #909399;">
                每条弹幕轨道的高度
              </span>
            </el-form-item>

            <el-form-item label="滚动弹幕高度百分比（%）">
              <el-input-number v-model="config.danmaku.float_percentage" :min="0" :max="100" :step="5" />
              <span style="margin-left: 10px; font-size: 12px; color: #909399;">
                滚动弹幕可使用的屏幕高度百分比
              </span>
            </el-form-item>

            <el-form-item label="底部弹幕高度百分比（%）">
              <el-input-number v-model="config.danmaku.bottom_percentage" :min="0" :max="100" :step="5" />
              <span style="margin-left: 10px; font-size: 12px; color: #909399;">
                底部弹幕可使用的屏幕高度百分比
              </span>
            </el-form-item>

            <el-form-item label="透明度（0-255）">
              <el-input-number v-model="config.danmaku.opacity" :min="0" :max="255" />
              <span style="margin-left: 10px; font-size: 12px; color: #909399;">
                0为完全透明，255为完全不透明
              </span>
            </el-form-item>

            <el-form-item label="描边宽度">
              <el-input-number v-model="config.danmaku.outline_width" :min="0" :max="5" :step="0.5" />
              <span style="margin-left: 10px; font-size: 12px; color: #909399;">
                弹幕文字的描边宽度
              </span>
            </el-form-item>

            <el-form-item label="时间偏移（秒）">
              <el-input-number v-model="config.danmaku.time_offset" :min="-60" :max="60" :step="0.1" />
              <span style="margin-left: 10px; font-size: 12px; color: #909399;">
                调整弹幕出现的时间偏移
              </span>
            </el-form-item>

            <el-form-item label="粗体显示">
              <el-switch v-model="config.danmaku.bold" />
            </el-form-item>

            <el-form-item label="强制使用自定义颜色">
              <el-switch v-model="config.danmaku.force_custom_color" />
              <span style="margin-left: 10px; font-size: 12px; color: #909399;">
                启用后所有弹幕将使用统一的自定义颜色
              </span>
            </el-form-item>

            <el-form-item label="自定义弹幕颜色">
              <el-color-picker v-model="config.danmaku.custom_color" show-alpha :predefine="predefineColors" />
              <span style="margin-left: 10px; font-size: 12px; color: #909399;">
                选择弹幕颜色（仅在启用强制自定义颜色时生效）
              </span>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="高级设置" name="advanced">
          <el-form :model="config.advanced" label-width="180px">
            <el-form-item label="视频并发数">
              <el-input-number v-model="config.advanced.concurrent_limit.video" :min="1" :max="10" />
            </el-form-item>
            <el-form-item label="分P并发数">
              <el-input-number v-model="config.advanced.concurrent_limit.page" :min="1" :max="10" />
            </el-form-item>
            <el-form-item label="频率限制时间窗口（毫秒）">
              <el-input-number v-model="config.advanced.rate_limit.duration_ms" :min="100" :max="5000" />
            </el-form-item>
            <el-form-item label="频率限制请求数">
              <el-input-number v-model="config.advanced.rate_limit.limit" :min="1" :max="20" />
            </el-form-item>
            <el-form-item label="NFO时间类型">
              <el-radio-group v-model="config.advanced.nfo_time_type">
                <el-radio label="favtime">收藏时间</el-radio>
                <el-radio label="pubtime">发布时间</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="工具管理" name="tools">
          <el-form label-width="180px">
            <el-divider content-position="left">yt-dlp 版本管理</el-divider>

            <el-form-item label="运行平台">
              <el-tag type="info" size="large">
                {{ ytdlpVersion.platform || 'unknown' }}
              </el-tag>
            </el-form-item>

            <el-form-item label="当前版本">
              <div class="version-info">
                <el-tag v-if="ytdlpVersion.current_version" type="info" size="large">
                  {{ ytdlpVersion.current_version }}
                </el-tag>
                <el-text v-else type="info">加载中...</el-text>
              </div>
            </el-form-item>

            <el-form-item label="最新版本">
              <div class="version-info">
                <el-tag v-if="ytdlpVersion.latest_version" :type="ytdlpVersion.has_update ? 'success' : 'info'" size="large">
                  {{ ytdlpVersion.latest_version }}
                </el-tag>
                <el-text v-else type="info">检查中...</el-text>
                <el-tag v-if="ytdlpVersion.has_update" type="warning" size="small" style="margin-left: 10px">
                  有新版本可用
                </el-tag>
              </div>
            </el-form-item>

            <el-form-item label="更新方式" v-if="ytdlpVersion.update_method">
              <el-text type="info" size="small">
                {{ ytdlpVersion.update_method }}
              </el-text>
            </el-form-item>

            <el-form-item label="操作">
              <el-space>
                <el-button @click="checkYtdlpVersion" :loading="ytdlpLoading">
                  <el-icon><Refresh /></el-icon>
                  检查更新
                </el-button>
                <el-button
                  type="primary"
                  @click="updateYtdlp"
                  :loading="ytdlpUpdating"
                  :disabled="!ytdlpVersion.has_update && !ytdlpForceUpdate"
                >
                  <el-icon><Upload /></el-icon>
                  {{ ytdlpVersion.has_update ? '立即更新' : '已是最新版本' }}
                </el-button>
              </el-space>
              <div style="font-size: 12px; color: #909399; margin-top: 8px;">
                更新过程可能需要1-2分钟，请耐心等待
              </div>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>

      <div class="actions">
        <el-button @click="loadData">重置</el-button>
        <el-button type="primary" @click="handleSave">保存配置</el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Upload } from '@element-plus/icons-vue'
import { getConfig, updateConfig, validateBilibiliCredential } from '@/api/config'
import { getYtdlpVersionInfo, updateYtdlpVersion } from '@/api/ytdlp'

defineOptions({
  name: 'Config'
})
import type { Config } from '@/types'

// 帮助文本
const videoNameHelp = '{{bvid}}, {{title}}, {{upper_name}}, {{pubtime}}'
const pageNameHelp = '{{ptitle}}, {{pid}} + 视频名称所有变量'

// 预定义颜色
const predefineColors = ref([
  '#FFFFFF', // 白色
  '#000000', // 黑色
  '#FF0000', // 红色
  '#00FF00', // 绿色
  '#0000FF', // 蓝色
  '#FFFF00', // 黄色
  '#FF00FF', // 洋红
  '#00FFFF', // 青色
])

const loading = ref(false)
const activeTab = ref('basic')

// yt-dlp 版本管理
const ytdlpVersion = ref({
  current_version: '',
  latest_version: '',
  has_update: false,
  platform: '',
  update_method: ''
})
const ytdlpLoading = ref(false)
const ytdlpUpdating = ref(false)
const ytdlpForceUpdate = ref(false)

// 认证验证状态
const credentialValidation = ref<{
  show: boolean
  valid: boolean
  message: string
  userInfo?: {
    mid: number
    uname: string
    face: string
    sign: string
    level: number
    vip_type: number
    vip_status: number
  }
}>({
  show: false,
  valid: false,
  message: ''
})

const config = ref<Config>({
  server: {
    bind_address: '0.0.0.0:8080',
    auth_token: ''
  },
  database: {
    host: 'localhost',
    port: 5432,
    user: 'bili_sync',
    password: '',
    dbname: 'bili_sync',
    sslmode: 'disable',
    max_open_conns: 25,
    max_idle_conns: 5,
    conn_max_lifetime: 300
  },
  sync: {
    interval: 3600,
    scan_only: false
  },
  paths: {
    download_base: '/downloads/bilibili',
    upper_path: '/metadata/people'
  },
  template: {
    video_name: '{{title}}',
    page_name: '{{title}}',
    time_format: '%Y-%m-%d'
  },
  bilibili: {
    credential: {
      sessdata: '',
      bili_jct: '',
      buvid3: '',
      dedeuserid: '',
      ac_time_value: ''
    }
  },
  quality: {
    max_resolution: '1080P+',
    codec_priority: ['AVC', 'HEVC', 'AV1'],
    audio_quality: '30280',
    cdn_sort: false
  },
  download: {
    skip_poster: false,
    skip_video_nfo: false,
    skip_upper: false,
    skip_danmaku: false,
    skip_subtitle: false
  },
  danmaku: {
    duration: 12,
    font_name: 'Microsoft YaHei',
    font_size: 38,
    width_ratio: 1.5,
    horizontal_gap: 30,
    lane_size: 0,
    float_percentage: 0.5,
    bottom_percentage: 0.25,
    opacity: 180,
    outline_width: 1.5,
    time_offset: 0,
    bold: false,
    custom_color: '#FFFFFF',
    force_custom_color: false
  },
  advanced: {
    concurrent_limit: {
      video: 3,
      page: 2
    },
    rate_limit: {
      duration_ms: 250,
      limit: 4
    },
    nfo_time_type: 'favtime',
    ytdlp_extra_args: []
  },
  logging: {
    level: 'info',
    file: '',
    max_size_mb: 100,
    max_backups: 3,
    max_age_days: 30
  }
})

// 深度合并配置，保持响应式
const deepAssign = (target: any, source: any) => {
  if (!source) return target

  Object.keys(source).forEach(key => {
    const sourceValue = source[key]
    const targetValue = target[key]

    if (sourceValue && typeof sourceValue === 'object' && !Array.isArray(sourceValue)) {
      // 如果是对象，递归合并
      if (targetValue && typeof targetValue === 'object') {
        deepAssign(targetValue, sourceValue)
      } else {
        target[key] = sourceValue
      }
    } else {
      // 直接赋值（包括数组、基本类型、null）
      target[key] = sourceValue
    }
  })

  return target
}

// 加载配置
const loadData = async () => {
  loading.value = true
  try {
    const data = await getConfig()
    // 使用深度合并保持响应式
    if (data) {
      deepAssign(config.value, data)

      // 如果B站认证信息存在，自动验证
      if (data.bilibili?.credential?.sessdata) {
        await validateCredential()
      }
    }
  } catch (error) {
    console.error('加载配置失败:', error)
    ElMessage.error('加载配置失败')
  } finally {
    loading.value = false
  }
}

// 根据当前激活的标签获取要提交的配置
const getCurrentTabConfig = () => {
  const tabConfigMap: Record<string, any> = {
    basic: {
      sync: config.value.sync,
      paths: config.value.paths,
      template: config.value.template
    },
    bilibili: {
      bilibili: config.value.bilibili
    },
    video: {
      quality: config.value.quality,
      download: config.value.download
    },
    danmaku: {
      danmaku: config.value.danmaku
    },
    advanced: {
      advanced: config.value.advanced
    }
  }

  return tabConfigMap[activeTab.value] || {}
}

// 保存配置
const handleSave = async () => {
  loading.value = true
  try {
    // 只提交当前标签的配置
    const configToSubmit = getCurrentTabConfig()
    await updateConfig(configToSubmit)
    ElMessage.success('保存成功')

    // 如果保存的是B站认证配置，自动验证
    if (activeTab.value === 'bilibili') {
      await validateCredential()
    }
  } catch (error) {
    console.error('保存配置失败:', error)
  } finally {
    loading.value = false
  }
}

// 验证B站认证信息
const validateCredential = async () => {
  credentialValidation.value.show = false

  try {
    const result = await validateBilibiliCredential()

    credentialValidation.value = {
      show: true,
      valid: true,
      message: result.message || '认证信息有效',
      userInfo: result.user_info
    }

    ElMessage.success('认证信息验证成功')
  } catch (error: any) {
    const errorMsg = error?.response?.data?.message || error?.message || '认证验证失败'

    credentialValidation.value = {
      show: true,
      valid: false,
      message: errorMsg,
      userInfo: undefined
    }

    ElMessage.error(errorMsg)
  }
}

// 检查 yt-dlp 版本
const checkYtdlpVersion = async () => {
  ytdlpLoading.value = true
  try {
    const data = await getYtdlpVersionInfo()
    ytdlpVersion.value = data
    if (data.has_update) {
      ElMessage.success(`发现新版本 ${data.latest_version}`)
    } else {
      ElMessage.info('已是最新版本')
    }
  } catch (error: any) {
    console.error('检查版本失败:', error)
    ElMessage.error(error?.response?.data?.message || '检查版本失败')
  } finally {
    ytdlpLoading.value = false
  }
}

// 更新 yt-dlp
const updateYtdlp = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要更新 yt-dlp 吗？更新过程可能需要1-2分钟。',
      '确认更新',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    ytdlpUpdating.value = true
    const result = await updateYtdlpVersion()

    ElMessage.success({
      message: `更新成功！版本: ${result.old_version} → ${result.current_version}`,
      duration: 5000
    })

    // 更新完成后重新检查版本
    await checkYtdlpVersion()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('更新失败:', error)
      ElMessage.error(error?.response?.data?.message || '更新失败')
    }
  } finally {
    ytdlpUpdating.value = false
  }
}

onMounted(() => {
  loadData()
  checkYtdlpVersion()
})
</script>

<style scoped>
.config {
  padding: 20px;
}

.help-text {
  font-size: 12px;
  color: #909399;
  display: block;
  margin-top: 5px;
}

.actions {
  margin-top: 30px;
  text-align: right;
  border-top: 1px solid #ebeef5;
  padding-top: 20px;
}

.user-info {
  margin-top: 10px;
}

.user-info p {
  margin: 5px 0;
  font-size: 14px;
}

.version-info {
  display: flex;
  align-items: center;
  gap: 10px;
}
</style>
