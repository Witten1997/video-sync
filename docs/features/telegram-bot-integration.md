# Telegram 机器人视频下载接入方案

## 1. 背景与目标

当前项目已经具备以下基础能力：

- Web 端支持“通过 URL 下载视频”
- 服务端已经区分 B 站链接和非 B 站链接
- 非 B 站链接通过 `yt-dlp` 创建下载任务
- 下载任务统一进入 `DownloadManager` 队列执行
- 下载记录、进度、完成状态已经有数据库和事件机制支撑

本次扩展目标是接入 Telegram Bot，使用户向机器人发送视频链接后，系统自动解析链接并调用现有 URL 下载能力完成下载。

本次方案的首个完整里程碑同时包含：

- 服务端 Telegram 接入链路
- Telegram 请求日志与消息回执
- Web 管理端中的 Telegram 配置与可观测性

## 2. 当前项目可复用能力

从现有代码结构看，Telegram 接入可以直接复用以下模块：

- URL 下载入口：
  - `internal/api/handler_video.go`
  - 已存在 `handleDownloadByURL`
- B 站 URL 下载逻辑：
  - `handleBilibiliDownloadByURL`
- 非 B 站 URL 下载逻辑：
  - `handleYtdlpDownloadByURL`
- 通用下载任务创建与入队：
  - `internal/downloader/manager.go`
  - `PrepareAndAddVideoTask`
  - `PrepareAndAddYtdlpTask`
- 下载队列与并发控制：
  - `internal/downloader/queue.go`
  - `internal/downloader/task.go`
- 下载记录：
  - `internal/database/models/download_record.go`
- URL 下载独立目录：
  - `internal/config/paths.go`
  - `PathsConfig.URLDownloadBase()`

结论：

- 下载核心已经存在，Telegram 不应该直接调用下载器或 shell
- 最合理的做法是把 `handler_video.go` 中“按 URL 创建下载任务”的逻辑下沉为共享应用服务
- Web API 和 Telegram Bot 应共同调用同一套 URL 提交服务

## 3. 推荐接入模式

### 3.1 一期推荐：Long Polling

推荐首个版本采用 Telegram Bot Long Polling，而不是 Webhook。

原因：

- 当前项目偏向本地 / 自托管 / NAS 场景，Polling 不要求公网 HTTPS 回调地址
- 不需要额外暴露 Telegram Webhook 路由
- 不需要改动当前 `Gin` 路由鉴权豁免逻辑
- 更适合作为低风险落地路径

### 3.2 二期可选：Webhook

Webhook 作为后续增强项，适用于满足以下条件的部署环境：

- 服务具备公网可访问 HTTPS 地址
- 已有稳定的反向代理和证书
- 对更低延迟与更少轮询开销有明确需求

如启用 Webhook，需要新增 Telegram 专用公开回调路由，并校验 Telegram Secret Token。

为避免首个完整里程碑范围膨胀，`webhook_url`、`webhook_secret` 及其对应的 Web API / 前端配置页读写逻辑统一延后到 Phase 5；Phase 1-4 仅覆盖 Polling 模式所需配置与展示。

## 4. 总体架构设计

推荐将 Telegram 设计为独立“接入层”，而不是另一套下载实现。

```text
Telegram User / Web User
    ->
Entry Adapter
    - Web API handler
    - Telegram update processor
    ->
URLDownloadService
    - normalize request
    - resolve source type
    - apply business idempotency
    - create/reuse video and download record
    - enqueue task
    ->
DownloadManager
    - queue
    - execution
    - events
    ->
Telegram notification + Web visibility
```

职责划分：

- `Web API handler`
  - 参数绑定
  - 调用 `URLDownloadService`
  - 返回 HTTP 响应
- `Telegram update processor`
  - 获取更新
  - 做 Telegram 适配层幂等
  - 解析命令和 URL
  - 调用 `URLDownloadService`
  - 发送 / 编辑 Telegram 消息
- `URLDownloadService`
  - 统一处理“传入 URL 后如何建任务”
  - 判断 B 站或非 B 站
  - 统一执行下载提交幂等与复用策略
  - 创建 / 复用视频、下载记录、任务
- `DownloadManager`
  - 继续负责排队、并发、执行、事件通知
- `Web visibility`
  - 提供 Telegram 配置、状态、请求日志的 API 与界面

## 5. 关键设计原则

### 5.1 不让 Telegram 层反向调用 HTTP API

不建议 Telegram 模块通过 HTTP 再调用 `/api/videos/download-by-url`，原因如下：

- 会引入额外鉴权和序列化开销
- 内部调用外部接口不利于错误定位
- 后续回执、幂等、上下文透传会更复杂

正确方式是提炼共享服务，供 Web API 和 Telegram 模块共同调用。

### 5.2 明确区分两层幂等

本方案必须区分两类幂等，不得混在一个表或一个模块里处理：

- Telegram 适配层幂等：
  - `update_id`
  - `chat_id`
  - `message_id`
- URL 提交服务幂等：
  - 已存在视频是否复用
  - 已存在下载记录是否复用
  - 已存在下载任务是否复用或拒绝重复创建

Telegram replay 防护不等于下载业务幂等，两者必须分层实现。

### 5.3 `telegram_request_logs` 的里程碑 1 建模规则

首个完整里程碑明确只支持“一条消息提交一个 URL”：

- 仅以下两类“URL 提交请求”会创建 `telegram_request_logs`：
  - 直接发送单个 URL
  - `/download <url>`，且仅解析出一个 URL
- `/start`、`/help` 不创建请求日志
- `/status`、`/status <task_id>` 属于查询命令，不创建下载请求日志
- 一条消息解析出 0 个 URL 或超过 1 个 URL 时，直接返回固定拒绝文案，不创建 `telegram_request_logs`

因此在里程碑 1 中：

- 每个被接受的 URL 提交请求只创建一条 `telegram_request_logs`
- `message_id` 与该条下载请求日志是 1:1 关系
- `reply_message_id` 指向该请求对应的唯一受理回执消息

这样可以避免以下字段语义不清：

- `status`
- `reply_message_id`
- `task_id`
- `record_id`
- `video_id`

### 5.4 Polling offset 必须持久化并定义提交时机

Long Polling 方案必须明确：

- offset 的持久化位置
- offset 在何时提交
- 进程重启后的恢复逻辑
- 崩溃后重复投递如何通过幂等层消除影响

推荐策略：

- 将最后处理完成的 `update_id` 持久化到数据库运行状态表
- 仅在该条更新处理完成后推进 offset；若是 URL 提交请求，则要求请求日志写入完成后再推进 offset
- 重启后从 `last_update_id + 1` 开始拉取

### 5.5 首个完整里程碑包含 Web 配置与可观测性

本方案的首个完整里程碑不是“只有 Telegram 后端接入”，还应包括：

- Web 端 Telegram 配置入口
- Telegram 运行状态接口与状态卡
- Telegram 请求日志查询页面
- Telegram 请求与下载记录之间的跳转关联

### 5.6 敏感信息不能明文返回前端

首个完整里程碑中，以下字段不得通过 `/api/config` 或其他 Web 接口明文返回：

- `bot_token`

要求：

- 配置读取时对密钥字段做脱敏或直接置空
- 配置保存时支持“留空表示不覆盖原值”
- 日志与错误回执中不得打印敏感密钥

说明：

- `webhook_secret` 属于 Phase 5 Webhook 增强项，不属于当前里程碑的配置模型、Web API 或前端表单范围
- 后续如引入 `webhook_secret`，沿用与 `bot_token` 相同的脱敏、留空不覆盖、日志不回显规则

## 6. 共享 URL 下载服务设计

建议新增：

- `internal/service/url_download.go`

核心接口建议如下：

```go
type SubmitURLRequest struct {
    URL               string
    TriggerChannel    string
    RequesterID       string
    RequesterName     string
    CorrelationID     string
    AllowExistingTask bool
}

type SubmitURLResult struct {
    VideoID         uint
    RecordID        uint
    TaskID          string
    VideoName       string
    SourceType      string
    IsExistingVideo bool
    IsExistingTask  bool
}

type URLDownloadService interface {
    Submit(ctx context.Context, req SubmitURLRequest) (*SubmitURLResult, error)
}
```

该服务负责承接当前 `handler_video.go` 中的以下逻辑：

- `handleDownloadByURL`
- `handleBilibiliDownloadByURL`
- `handleYtdlpDownloadByURL`

改造后：

- Web API 只做参数绑定和 HTTP 响应
- Telegram 模块直接调用 `URLDownloadService.Submit`
- 后续如接入更多入口，也统一调用该服务

## 7. Telegram 模块拆分方案

建议新增目录：

- `internal/telegram/service.go`
- `internal/telegram/client.go`
- `internal/telegram/poller.go`
- `internal/telegram/parser.go`
- `internal/telegram/access.go`
- `internal/telegram/notifier.go`
- `internal/telegram/store.go`

职责建议如下：

- `service.go`
  - 启动 / 停止机器人
  - 编排 Polling、日志、通知、状态
- `client.go`
  - 封装 Telegram Bot API 调用
  - 发送消息、编辑消息、获取更新
- `poller.go`
  - Long Polling 拉取更新
  - 推进并持久化 `offset`
- `parser.go`
  - 提取消息中的 URL
  - 识别 `/start`、`/help`、`/download`、`/status`
  - 仅将“直接发送单个 URL”与 `/download <url>` 路由到 URL 提交流程
- `access.go`
  - 白名单校验
  - 简单限流
- `notifier.go`
  - 将下载任务状态映射为 Telegram 回执
- `store.go`
  - 持久化运行状态和请求日志

## 8. 配置设计

建议在配置中新增 `telegram` 段：

```yaml
telegram:
  enabled: false
  bot_token: ""
  mode: "polling"
  poll_timeout_seconds: 30
  allowed_chat_ids: []
  allowed_user_ids: []
  allowed_chat_types:
    - "private"
  max_urls_per_message: 1
  notify_on_accept: true
  notify_on_complete: true
  notify_on_fail: true
```

首个完整里程碑仅实现 Polling 所需配置。`webhook_url` 与 `webhook_secret` 延后到 Phase 5，当前不进入配置结构、配置接口和前端配置页。

对应需要修改：

- `internal/config/config.go`
- `internal/config/validator.go`
- `configs/config.example.yaml`

校验规则建议：

- `enabled=true` 时 `bot_token` 必填
- `mode` 首个版本仅启用 `polling`
- `poll_timeout_seconds` 范围建议 `10-60`
- `max_urls_per_message` 在首个完整里程碑固定为 `1`
- `allowed_chat_types` 首个版本默认仅允许 `private`

## 9. 数据模型设计

### 9.1 运行状态表

建议新增轻量运行状态表，用于持久化 Polling 进度和运行态信息。

建议字段：

- `bot_name`
- `last_update_id`
- `last_poll_at`
- `last_error`
- `last_error_at`

用途：

- offset 恢复
- Web 状态卡展示
- 故障排查

### 9.2 请求日志表：`telegram_request_logs`

建议新增 `telegram_request_logs`，在首个完整里程碑中按“每个被接受的 URL 提交请求一行”建模。

建议字段：

- `update_id`
- `chat_id`
- `message_id`
- `user_id`
- `raw_text`
- `raw_url`
- `url_hash`
- `status`
- `video_id`
- `record_id`
- `task_id`
- `reply_message_id`
- `error_message`
- `created_at`
- `updated_at`

字段语义补充：

- 该表只记录下载提交路径，不记录 `/start`、`/help`、`/status`
- 仅当消息被识别为有效 URL 提交且恰好解析出 1 个 URL 时才创建行
- `reply_message_id` 保存该请求的首条“已受理 / 已入队”回执消息 ID，后续完成 / 失败通知优先编辑该消息
- 若编辑失败而改为补发新消息，`reply_message_id` 仍保持为首条受理回执消息 ID，不切换到补发消息；补发消息仅作为兜底通知，不作为主关联键

用途：

- URL 提交路径的 Telegram 适配层幂等与审计
- 请求与下载记录关联
- 故障排查
- Web 请求日志页面展示

## 10. 消息处理流程

```text
接收 Telegram 更新
  ->
校验 bot 是否启用
  ->
校验 chat type / chat id / user id
  ->
校验 Telegram 适配层幂等
  ->
解析命令或提取 URL
  ->
若是 /start 或 /help，则直接回复帮助信息，不创建日志
  ->
若是 /status 查询，则直接回复查询结果，不创建下载请求日志
  ->
若是 URL 提交请求且恰好解析出 1 个 URL，则创建一条 telegram_request_log
  ->
调用 URLDownloadService.Submit()
  ->
写入 record_id / task_id / video_id
  ->
发送“已受理 / 已入队”回执
  ->
监听下载完成 / 失败事件并编辑同一条回执消息
```

支持输入形式：

- 直接发送单个 URL
- `/download <url>`，且仅允许单个 URL
- `/status`
- `/status <task_id>`
- `/start`
- `/help`

首个完整里程碑对多 URL 消息的规则：

- 一条消息中解析出超过 1 个 URL 时，直接拒绝并提示“单条消息仅支持一个 URL”
- 拒绝后不创建下载任务，不写入 `telegram_request_logs`

首个版本不做：

- 群聊命令处理
- `@botname` 提及触发
- 文件上传触发下载

## 11. 回执与通知设计

Telegram 不适合高频推送下载进度。

建议仅处理以下状态：

- 已接收
- 已入队
- 下载完成
- 下载失败

通知策略建议：

- 受理时发送一条回执消息
- 后续优先编辑同一条消息，而不是连续发送新消息
- 仅当编辑失败时再补发新消息，且补发消息不回写覆盖 `reply_message_id`
- 首个完整里程碑中，一条被接受的消息只对应一条回执消息，`reply_message_id` 也只指向这条消息

## 12. 安全与风控设计

### 12.1 白名单

首个版本仅支持私聊白名单：

- `allowed_chat_types`
- `allowed_chat_ids`
- `allowed_user_ids`

默认行为：

- 不在白名单则拒绝处理
- 返回固定拒绝文案，不暴露系统内部信息

校验顺序明确为：

- 先校验 `allowed_chat_types`
- 再校验 `allowed_chat_ids`
- 最后校验 `allowed_user_ids`
- 若 `allowed_chat_ids` 与 `allowed_user_ids` 同时配置，则必须同时命中，两者是 AND 关系而不是择一通过

### 12.2 频控

首个版本可采用内存级限流，例如：

- 每个 `chat_id` 每分钟最多 5 条消息
- 每条消息最多 1 个 URL

并在文档中明确：

- 重启后限流计数会清零
- 多实例部署下不保证全局一致

### 12.3 输入校验

明确限制：

- 非 URL 文本不触发下载
- 超长文本直接拒绝
- 空消息忽略，不进入“0 URL 固定拒绝文案”分支
- 首个版本仅支持文本链接，不支持文件上传触发下载

## 13. Web 管理端设计

首个完整里程碑需要提供以下 Web 能力：

### 13.1 配置页

在现有配置页中新增 Telegram 配置区块，支持：

- 启用 / 禁用
- `bot_token` 写入
- Polling 参数
- 白名单配置
- 通知开关

要求：

- `bot_token` 不回显明文
- 留空表示不覆盖已有密钥
- Webhook 相关字段属于 Phase 5，不在当前配置页展示或编辑

### 13.2 状态卡

暴露 Telegram 运行状态：

- `enabled`
- `running`
- `mode`
- `last_poll_at`
- `last_update_id`
- `last_error`
- `last_error_at`

### 13.3 请求日志页

提供 Telegram 请求日志查询页面，支持：

- 按状态筛选
- 查看原始 URL
- 查看关联 `task_id`
- 查看关联 `record_id`
- 跳转到下载记录

## 14. 分阶段实施计划

### Phase 1：抽共享 URL 下载服务

目标：

- 将 Web URL 下载逻辑改为调用 `URLDownloadService`
- 不改变现有外部 API 行为

### Phase 2：Telegram 最小提交链路

目标：

- 私聊白名单
- Long Polling
- URL 提交闭环

### Phase 3：请求日志与通知闭环

目标：

- `telegram_request_logs`
- `/status`
- 下载完成 / 失败通知
- `record_id` 关联

### Phase 4：Web 管理端支持

目标：

- Telegram 配置页
- 运行状态卡
- 请求日志页

### Phase 5：后续增强

目标：

- Webhook
- 群聊支持
- `@botname` 提及
- 更强的分布式限流
- 机器人测试发送与运维操作

## 15. 实施结论

首个完整里程碑的正确交付顺序为：

1. 先抽 `URLDownloadService`
2. 再接 Telegram Polling 提交链路
3. 再补请求日志与通知闭环
4. 最后补 Web 配置与可观测性

本方案的关键不是单独接入 Telegram，而是建立稳定的共享 URL 提交服务边界，以及可运营、可观测的 Telegram 产品面。
