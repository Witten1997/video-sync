<template>
  <div class="config">
    <el-card v-loading="loading">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="基本设置" name="basic">
          <el-form :model="config" label-width="180px">
            <el-form-item label="同步间隔（秒）">
              <el-input-number v-model="config.sync.interval" :min="60" />
            </el-form-item>
            <el-form-item label="启用网络代理">
              <el-switch v-model="config.proxy.enabled" />
            </el-form-item>
            <el-form-item label="HTTP 代理地址">
              <el-input v-model="config.proxy.url" placeholder="例如: http://127.0.0.1:7890" />
              <span class="help-text">
                启用后，B站接口请求、二维码登录、图片代理、版本检查、升级下载和 yt-dlp 下载都会走该代理
              </span>
            </el-form-item>
            <el-form-item label="下载基础路径">
              <el-input v-model="config.paths.download_base" />
            </el-form-item>
            <el-form-item label="URL下载路径">
              <el-input v-model="config.paths.url_download_path" placeholder="例如: url/youtube，留空则直接使用下载基础路径" />
              <span class="help-text">
                实际 URL 下载目录 = 下载基础路径 + URL下载路径
              </span>
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
          <!-- 认证方式选择 -->
          <el-radio-group v-model="authMethod" style="margin-bottom: 20px">
            <el-radio-button value="qrcode">扫码登录</el-radio-button>
            <el-radio-button value="manual">手动输入</el-radio-button>
          </el-radio-group>

          <!-- 扫码登录 -->
          <div v-if="authMethod === 'qrcode'" class="qrcode-login">
            <div v-if="!qrcode.url" class="qrcode-placeholder">
              <el-button
                type="primary"
                :loading="qrcode.loading"
                :icon="qrcode.loading ? '' : 'Refresh'"
                @click="handleGenerateQRCode"
              >
                {{ qrcode.loading ? '生成中...' : '生成二维码' }}
              </el-button>
              <p class="tip">点击按钮生成二维码，使用 B站 APP 扫描登录</p>
            </div>

            <div v-else class="qrcode-container">
              <div class="qrcode-wrapper">
                <canvas ref="qrcodeCanvas" />

                <!-- 状态遮罩 -->
                <div v-if="qrcode.status !== 86101" class="qrcode-mask">
                  <!-- 已扫码未确认 -->
                  <div v-if="qrcode.status === 86090" class="status-box status-scanned">
                    <el-icon :size="48" color="#67c23a"><SuccessFilled /></el-icon>
                    <p>已扫码，请在手机上确认</p>
                  </div>
                  <!-- 登录成功 -->
                  <div v-else-if="qrcode.status === 0" class="status-box status-success">
                    <el-icon :size="48" color="#67c23a"><CircleCheckFilled /></el-icon>
                    <p>登录成功！</p>
                    <p class="sub">页面即将刷新...</p>
                  </div>
                  <!-- 二维码失效 -->
                  <div v-else-if="qrcode.status === 86038" class="status-box status-expired">
                    <el-icon :size="48" color="#f56c6c"><CircleCloseFilled /></el-icon>
                    <p>二维码已失效</p>
                    <el-button type="primary" size="small" @click="handleGenerateQRCode">
                      重新生成
                    </el-button>
                  </div>
                </div>
              </div>

              <div class="qrcode-info">
                <el-alert
                  :type="qrcode.status === 86101 ? 'info' : qrcode.status === 86090 ? 'warning' : 'success'"
                  :closable="false"
                  show-icon
                >
                  <template #title>
                    <div class="alert-title">{{ qrcode.message }}</div>
                  </template>
                  <div v-if="qrcode.status === 86101">
                    请使用 B站 APP 扫描二维码
                  </div>
                  <div v-else-if="qrcode.status === 86090">
                    请在手机上点击"确认登录"
                  </div>
                  <div v-else-if="qrcode.status === 0">
                    凭据已保存到服务器配置
                  </div>
                </el-alert>

                <div class="qrcode-footer">
                  <div class="countdown">
                    <el-icon><Clock /></el-icon>
                    <span>有效期: {{ qrcode.remainingTime }}秒</span>
                  </div>
                  <el-button text @click="handleCancelQRCode">取消</el-button>
                </div>
              </div>
            </div>
          </div>

          <!-- 手动输入认证 -->
          <el-form v-else :model="config.bilibili.credential" label-width="180px">
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
            <el-form-item label="失败重试次数">
              <el-input-number v-model="config.advanced.max_retry_count" :min="0" :max="10" />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="Telegram" name="telegram">
          <div class="telegram-panel">
            <el-card class="telegram-status-card" shadow="never">
              <template #header>
                <div class="telegram-status-header">
                  <span>运行状态</span>
                  <el-space>
                    <el-button link type="primary" @click="goToTelegramRequests">查看请求日志</el-button>
                    <el-button link type="primary" @click="loadTelegramRuntimeStatus">刷新</el-button>
                  </el-space>
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

        <el-tab-pane label="版本信息" name="version">
          <el-form label-width="180px">
            <el-divider content-position="left">程序版本</el-divider>

            <el-form-item label="当前版本">
              <el-tag type="info" size="large">
                {{ appVersion.current_version || '加载中...' }}
              </el-tag>
            </el-form-item>

            <el-form-item label="构建时间" v-if="appVersion.build_time">
              <el-text type="info">{{ appVersion.build_time }}</el-text>
            </el-form-item>

            <el-form-item label="最新版本">
              <div class="version-info">
                <el-tag v-if="appVersion.new_version" :type="appVersion.has_update ? 'success' : 'info'" size="large">
                  {{ appVersion.new_version }}
                </el-tag>
                <el-text v-else type="info">未检查</el-text>
                <el-tag v-if="appVersion.has_update" type="warning" size="small" style="margin-left: 10px">
                  有新版本可用
                </el-tag>
              </div>
            </el-form-item>

            <el-form-item label="检查时间" v-if="appVersion.checked_at">
              <el-text type="info">{{ appVersion.checked_at }}</el-text>
            </el-form-item>

            <el-form-item label="更新日志" v-if="appVersion.changelog">
              <el-input
                type="textarea"
                :model-value="appVersion.changelog"
                :rows="8"
                readonly
                style="width: 100%;"
              />
            </el-form-item>

            <el-form-item label="操作">
              <el-space>
                <el-button @click="handleCheckAppVersion" :loading="appVersionLoading">
                  <el-icon><Refresh /></el-icon>
                  检查更新
                </el-button>
                <el-button
                  type="primary"
                  @click="handleAppUpgrade"
                  :loading="appUpgrading"
                  :disabled="!appVersion.has_update"
                >
                  <el-icon><Upload /></el-icon>
                  {{ appVersion.has_update ? '立即更新' : '已是最新版本' }}
                </el-button>
              </el-space>
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
import { computed, ref, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Upload, Clock, SuccessFilled, CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue'
import { getConfig, updateConfig, validateBilibiliCredential, generateQRCode, pollQRCodeStatus } from '@/api/config'
import {
  approveTelegramAccessCandidate,
  getTelegramAccessCandidates,
  getTelegramStatus,
  reconnectTelegram,
  sendTelegramTestMessage
} from '@/api/telegram'
import { getYtdlpVersionInfo, updateYtdlpVersion } from '@/api/ytdlp'
import { getVersionInfo, checkVersion, doUpgrade } from '@/api/version'
import { useRouter } from 'vue-router'
import QRCode from 'qrcode'

defineOptions({
  name: 'Config'
})
import type { Config, TelegramAccessCandidate, TelegramRuntimeStatus } from '@/types'

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
const telegramReconnectLoading = ref(false)
const telegramTestSendLoading = ref(false)
const telegramAccessCandidatesLoading = ref(false)
const telegramTestSend = ref({
  chat_id: '',
  message: ''
})
const telegramAccessCandidates = ref<TelegramAccessCandidate[]>([])

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

// 程序版本管理
const appVersion = ref({
  current_version: '',
  build_time: '',
  has_update: false,
  new_version: '',
  changelog: '',
  checked_at: '',
  download_url: ''
})
const appVersionLoading = ref(false)
const appUpgrading = ref(false)

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

// ==================== 二维码登录状态 ====================

// 认证方式（qrcode: 扫码登录, manual: 手动输入）
const authMethod = ref<'qrcode' | 'manual'>('qrcode')

// 二维码canvas元素引用
const qrcodeCanvas = ref<HTMLCanvasElement>()

// 二维码状态
const qrcode = ref({
  loading: false,
  url: '',
  qrcode_key: '',
  status: null as number | null,
  message: '',
  remainingTime: 180
})

// 轮询和倒计时定时器
let pollTimer: number | null = null
let countdownTimer: number | null = null

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
  proxy: {
    enabled: false,
    url: ''
  },
  sync: {
    interval: 3600,
    scan_only: false
  },
  paths: {
    download_base: '/downloads/bilibili',
    url_download_path: '',
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
    ytdlp_extra_args: [],
    max_retry_count: 3
  },
  logging: {
    level: 'info',
    file: '',
    max_size_mb: 100,
    max_backups: 3,
    max_age_days: 30
  },
  telegram: {
    enabled: false,
    bot_token: '',
    bot_token_configured: false,
    mode: 'polling',
    poll_timeout_seconds: 30,
    webhook_url: '',
    webhook_secret: '',
    webhook_secret_configured: false,
    allowed_chat_ids: [],
    allowed_user_ids: [],
    allowed_chat_types: ['private'],
    max_urls_per_message: 1,
    notify_on_accept: true,
    notify_on_complete: true,
    notify_on_fail: true
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
  if (!value) {
    return '-'
  }

  return new Date(value).toLocaleString('zh-CN')
}

const telegramStatusTagType = computed(() => {
  if (!telegramStatus.value.enabled) {
    return 'info'
  }
  return telegramStatus.value.running ? 'success' : 'warning'
})

const formatTelegramMode = (mode?: string | null) => {
  if (!mode) {
    return '-'
  }

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
  if (parts.length > 0) {
    return parts.join(' ')
  }
  if (candidate.username) {
    return `@${candidate.username}`
  }
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

// 加载配置
const loadData = async (options: { validateCredential?: boolean } = {}) => {
  loading.value = true
  try {
    const data = await getConfig()
    // 使用深度合并保持响应式
    if (data) {
      deepAssign(config.value, data)
      config.value.telegram.bot_token = ''
      config.value.telegram.webhook_secret = ''

      // 如果B站认证信息存在，自动验证
      if (options.validateCredential !== false && data.bilibili?.credential?.sessdata) {
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
      proxy: config.value.proxy,
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
    },
    telegram: {
      telegram: config.value.telegram
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
    const result = await updateConfig(configToSubmit)
    if (result.restart_needed) {
      ElMessage.warning(`Saved. Restart required: ${result.requires_restart.join(', ')}`)
    } else {
      ElMessage.success(result.message || '保存成功')
    }

    // 如果保存的是B站认证配置，自动验证
    if (activeTab.value === 'bilibili') {
      await validateCredential()
    }
    if (activeTab.value === 'telegram') {
      await loadData({ validateCredential: false })
      await loadTelegramRuntimeStatus()
      await loadTelegramAccessCandidates()
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

// ==================== 程序版本管理 ====================

const loadAppVersion = async () => {
  try {
    const data = await getVersionInfo()
    appVersion.value = data
  } catch (error) {
    console.error('获取版本信息失败:', error)
  }
}

const handleCheckAppVersion = async () => {
  appVersionLoading.value = true
  try {
    const data = await checkVersion()
    appVersion.value = data
    if (data.has_update) {
      ElMessage.success(`发现新版本 ${data.new_version}`)
    } else {
      ElMessage.info('已是最新版本')
    }
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.message || '检查版本失败')
  } finally {
    appVersionLoading.value = false
  }
}

const handleAppUpgrade = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要升级到 ${appVersion.value.new_version} 吗？升级完成后服务将自动重启。`,
      '确认升级',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }
    )

    appUpgrading.value = true
    await doUpgrade(appVersion.value.new_version)

    ElMessage.success('升级成功，服务正在重启，请稍后刷新页面...')

    // 10秒后自动刷新
    setTimeout(() => {
      window.location.reload()
    }, 10000)
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error?.response?.data?.message || '升级失败')
    }
  } finally {
    appUpgrading.value = false
  }
}

// ==================== 二维码登录处理函数 ====================

// 生成二维码
const handleGenerateQRCode = async () => {
  qrcode.value.loading = true

  try {
    const data = await generateQRCode()

    qrcode.value = {
      loading: false,
      url: data.url,
      qrcode_key: data.qrcode_key,
      status: 86101,
      message: '等待扫码',
      remainingTime: data.expires_in
    }

    // 等待 Vue 更新 DOM
    await nextTick()

    // 生成二维码图片
    if (qrcodeCanvas.value) {
      await QRCode.toCanvas(qrcodeCanvas.value, data.url, {
        width: 256,
        margin: 2,
        color: {
          dark: '#000000',
          light: '#FFFFFF'
        }
      })
    } else {
      console.error('Canvas 元素未找到')
      ElMessage.error('二维码显示失败，请刷新页面重试')
      return
    }

    // 开始轮询
    startPolling()
    // 开始倒计时
    startCountdown()
  } catch (error: any) {
    console.error('生成二维码失败:', error)
    ElMessage.error(error?.response?.data?.message || '生成二维码失败')
    qrcode.value.loading = false
  }
}

// 开始轮询二维码状态
const startPolling = () => {
  const poll = async () => {
    try {
      const data = await pollQRCodeStatus(qrcode.value.qrcode_key)

      qrcode.value.status = data.status
      qrcode.value.message = data.message

      // 登录成功或失效，停止轮询
      if (data.status === 0 || data.status === 86038) {
        stopPolling()
        stopCountdown()

        if (data.status === 0) {
          ElMessage.success('登录成功！凭据已保存')
          // 2秒后刷新页面
          setTimeout(() => {
            window.location.reload()
          }, 2000)
        }
      }
    } catch (error: any) {
      console.error('轮询失败:', error)
    }
  }

  // 立即执行一次
  poll()
  // 每2秒轮询一次
  pollTimer = window.setInterval(poll, 2000)
}

// 停止轮询
const stopPolling = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// 开始倒计时
const startCountdown = () => {
  countdownTimer = window.setInterval(() => {
    qrcode.value.remainingTime--
    if (qrcode.value.remainingTime <= 0) {
      qrcode.value.status = 86038
      qrcode.value.message = '二维码已失效'
      stopPolling()
      stopCountdown()
    }
  }, 1000)
}

// 停止倒计时
const stopCountdown = () => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

// 取消二维码登录
const handleCancelQRCode = () => {
  stopPolling()
  stopCountdown()
  qrcode.value = {
    loading: false,
    url: '',
    qrcode_key: '',
    status: null,
    message: '',
    remainingTime: 180
  }
}

onMounted(() => {
  loadData()
  loadTelegramRuntimeStatus()
  loadTelegramAccessCandidates()
  checkYtdlpVersion()
  loadAppVersion()
})

// 组件卸载时清理定时器
onUnmounted(() => {
  stopPolling()
  stopCountdown()
})
</script>

<style scoped>
.config {
  padding: 32px;
}

.help-text {
  font-size: 12px;
  color: #94a3b8;
  display: block;
  margin-top: 5px;
}

.actions {
  margin-top: 32px;
  text-align: right;
  border-top: 1px solid #f1f5f9;
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
  gap: 12px;
}

.telegram-panel {
  display: flex;
  flex-direction: column;
  gap: 20px;
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

/* ==================== 二维码登录样式 ==================== */

.qrcode-login {
  max-width: 600px;
}

.qrcode-placeholder {
  text-align: center;
  padding: 60px 20px;
}

.qrcode-placeholder .tip {
  margin-top: 15px;
  font-size: 14px;
  color: #94a3b8;
}

.qrcode-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.qrcode-wrapper {
  position: relative;
  display: inline-block;
}

.qrcode-wrapper canvas {
  display: block;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
}

.qrcode-mask {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.95);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
}

.status-box {
  text-align: center;
  padding: 20px;
}

.status-box p {
  margin: 12px 0 0 0;
  font-size: 16px;
  font-weight: 500;
  color: #1e293b;
}

.status-box .sub {
  margin-top: 8px;
  font-size: 14px;
  font-weight: normal;
  color: #94a3b8;
}

.status-box .el-button {
  margin-top: 15px;
}

.qrcode-info {
  width: 100%;
  max-width: 400px;
}

.alert-title {
  font-size: 14px;
  font-weight: 500;
}

.qrcode-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 15px;
  padding-top: 15px;
  border-top: 1px solid #f1f5f9;
}

.countdown {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #475569;
}

.countdown .el-icon {
  font-size: 16px;
  color: #94a3b8;
}

</style>
