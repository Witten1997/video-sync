# Download Manager 下载管理器模块文档

下载管理器、队列和并发控制模块，提供完整的任务调度、队列管理和并发控制功能。

## 模块概述

该模块在 yt-dlp 下载器封装的基础上，实现了完整的下载管理系统，包括：

- ✅ **任务队列**: 基于优先级的任务队列
- ✅ **并发控制**: 视频和分P级别的并发控制
- ✅ **任务调度**: 自动任务调度和执行
- ✅ **状态管理**: 完整的任务状态追踪
- ✅ **事件系统**: 异步事件通知机制
- ✅ **重试机制**: 自动重试失败的任务

## 文件结构

```
internal/downloader/
├── progress.go          (9.4 KB)  - 进度追踪数据结构
├── ytdlp.go            (7.7 KB)  - yt-dlp 命令封装
├── downloader.go       (13 KB)   - B站视频下载器
├── task.go             (6.0 KB)  - 下载任务定义
├── queue.go            (3.7 KB)  - 优先级任务队列
├── concurrency.go      (4.5 KB)  - 并发控制器
├── manager.go          (16 KB)   - 下载管理器
├── downloader_test.go  (7.5 KB)  - 下载器单元测试
├── manager_test.go     (9.9 KB)  - 管理器单元测试
├── README.md           (11 KB)   - 原始文档
└── MANAGER.md          (本文档)  - 管理器文档
```

## 核心组件

### 1. DownloadTask (task.go)

下载任务的定义和管理。

**任务类型**:
- `TaskTypeVideo`: 视频任务（下载所有分P）
- `TaskTypePage`: 分P任务（下载单个分P）
- `TaskTypeCollection`: 合集任务（批量视频）

**任务状态**:
- `TaskStatusPending`: 待处理
- `TaskStatusQueued`: 已入队
- `TaskStatusRunning`: 运行中
- `TaskStatusPaused`: 已暂停（预留）
- `TaskStatusCompleted`: 已完成
- `TaskStatusFailed`: 失败
- `TaskStatusCancelled`: 已取消

**任务优先级**:
- `PriorityLow` = 0
- `PriorityNormal` = 5
- `PriorityHigh` = 10
- `PriorityUrgent` = 15

**使用示例**:
```go
// 创建视频任务
video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}
task := NewDownloadTask(TaskTypeVideo, video, nil, "./downloads")
task.Priority = PriorityHigh
task.MaxRetries = 3

// 获取任务状态
status := task.GetStatus()

// 取消任务
task.Cancel()

// 检查是否可以重试
if task.CanRetry() {
    task.IncrementRetry()
}
```

### 2. TaskQueue (queue.go)

基于堆的优先级队列实现。

**特性**:
- 优先级排序（高优先级优先）
- 同优先级按创建时间排序（FIFO）
- 线程安全
- O(log n) 入队和出队操作
- 支持任务查找和移除

**使用示例**:
```go
queue := NewTaskQueue()

// 入队
queue.Enqueue(task1)
queue.Enqueue(task2)

// 出队（获取最高优先级任务）
task := queue.Dequeue()

// 查找任务
if queue.Contains(taskID) {
    task := queue.Get(taskID)
}

// 移除任务
removed := queue.Remove(taskID)

// 更新优先级
queue.UpdatePriority(taskID, PriorityUrgent)

// 获取队列大小
size := queue.Size()
```

### 3. ConcurrencyController (concurrency.go)

并发控制器，管理视频和分P级别的并发。

**组件**:
- `Semaphore`: 信号量实现
- `ConcurrencyController`: 两级并发控制

**使用示例**:
```go
// 创建控制器（最多2个视频并发，4个分P并发）
cc := NewConcurrencyController(2, 4)

// 获取视频级别许可
ctx := context.Background()
err := cc.AcquireVideo(ctx)
defer cc.ReleaseVideo()

// 获取分P级别许可
err = cc.AcquirePage(ctx)
defer cc.ReleasePage()

// 检查是否可以启动新任务
if cc.CanStartVideo() {
    // 可以启动视频下载
}

// 获取统计信息
stats := cc.GetStats()
fmt.Printf("视频: %d/%d, 分P: %d/%d\n",
    stats.VideoUsed, stats.VideoTotal,
    stats.PageUsed, stats.PageTotal)
```

### 4. DownloadManager (manager.go)

下载管理器核心，负责任务调度和执行。

**功能**:
- 任务队列管理
- 自动任务调度
- 并发控制
- 状态追踪
- 事件通知
- 失败重试

**使用示例**:
```go
// 创建管理器
cfg := config.LoadConfig("config.yaml")
biliClient := bilibili.NewClient(cfg)
manager, err := NewDownloadManager(cfg, biliClient)
if err != nil {
    log.Fatal(err)
}

// 启动管理器
err = manager.Start()
if err != nil {
    log.Fatal(err)
}
defer manager.Stop()

// 添加事件处理器
manager.AddEventHandler(func(event ManagerEvent) {
    switch event.Type {
    case EventTaskStarted:
        fmt.Printf("任务开始: %s\n", event.Task.ID)
    case EventTaskCompleted:
        fmt.Printf("任务完成: %s\n", event.Task.ID)
    case EventTaskFailed:
        fmt.Printf("任务失败: %s - %s\n", event.Task.ID, event.Message)
    case EventTaskProgress:
        fmt.Printf("进度更新: %.2f%%\n", event.Progress.Progress)
    }
})

// 添加任务
video := &models.Video{ID: 1, BVid: "BV1xx411c7mD", Name: "测试视频"}
task, err := manager.AddVideoTask(video, "./downloads", PriorityNormal)

// 获取统计信息
stats := manager.GetStats()
fmt.Printf("队列: %d, 运行中: %d, 已完成: %d, 失败: %d\n",
    stats.QueuedTasks, stats.RunningTasks,
    stats.CompletedTasks, stats.FailedTasks)

// 取消任务
err = manager.CancelTask(task.ID)

// 重试失败的任务
err = manager.RetryTask(task.ID)
```

## 事件系统

### 事件类型

```go
type ManagerEventType string

const (
    EventTaskAdded     = "task_added"
    EventTaskStarted   = "task_started"
    EventTaskCompleted = "task_completed"
    EventTaskFailed    = "task_failed"
    EventTaskCancelled = "task_cancelled"
    EventTaskRetrying  = "task_retrying"
    EventTaskProgress  = "task_progress"
)
```

### 事件结构

```go
type ManagerEvent struct {
    Type      ManagerEventType `json:"type"`
    Task      *DownloadTask    `json:"task"`
    Progress  *SubTaskProgress `json:"progress,omitempty"`
    Message   string           `json:"message,omitempty"`
    Timestamp time.Time        `json:"timestamp"`
}
```

### 事件处理示例

```go
manager.AddEventHandler(func(event ManagerEvent) {
    log.Printf("[%s] %s: %s",
        event.Type,
        event.Task.ID,
        event.Message)

    // 可以将事件推送到前端
    if websocketConn != nil {
        websocketConn.WriteJSON(event)
    }

    // 可以记录到数据库
    if event.Type == EventTaskCompleted {
        db.UpdateTaskStatus(event.Task.ID, "completed")
    }
})
```

## API 接口

### 管理器控制

```go
// 启动管理器
Start() error

// 停止管理器
Stop() error

// 检查是否运行
IsRunning() bool
```

### 任务管理

```go
// 添加任务
AddTask(task *DownloadTask) error
AddVideoTask(video *models.Video, outputDir string, priority TaskPriority) (*DownloadTask, error)
AddPageTask(video *models.Video, page *models.Page, outputDir string, priority TaskPriority) (*DownloadTask, error)

// 取消任务
CancelTask(taskID string) error

// 重试任务
RetryTask(taskID string) error

// 暂停/恢复任务（预留接口，暂未实现）
PauseTask(taskID string) error
ResumeTask(taskID string) error
```

### 任务查询

```go
// 获取单个任务
GetTask(taskID string) *DownloadTask

// 获取所有任务
GetAllTasks() []*DownloadTask

// 获取队列中的任务
GetQueuedTasks() []*DownloadTask

// 获取运行中的任务
GetRunningTasks() []*DownloadTask

// 获取已完成的任务
GetCompletedTasks() []*DownloadTask

// 获取统计信息
GetStats() ManagerStats

// 清理已完成的任务
ClearCompletedTasks() int
```

### 事件处理

```go
// 添加事件处理器
AddEventHandler(handler EventHandler)
```

## 配置要求

下载管理器需要以下配置项：

```yaml
advanced:
  concurrent_limit:
    video: 2    # 视频级别并发数
    page: 4     # 分P级别并发数
  rate_limit:
    duration_ms: 1000  # 速率限制窗口
    limit: 10          # 请求限制数
```

## 完整使用示例

### 示例 1: 基础使用

```go
package main

import (
    "fmt"
    "time"

    "bili-download/internal/bilibili"
    "bili-download/internal/config"
    "bili-download/internal/database/models"
    "bili-download/internal/downloader"
)

func main() {
    // 加载配置
    cfg, _ := config.LoadConfig("./config.yaml")

    // 创建B站客户端
    biliClient := bilibili.NewClient(cfg)

    // 创建下载管理器
    manager, _ := downloader.NewDownloadManager(cfg, biliClient)

    // 启动管理器
    manager.Start()
    defer manager.Stop()

    // 添加任务
    video := &models.Video{
        ID:   1,
        BVid: "BV1xx411c7mD",
        Name: "测试视频",
        Pages: []models.Page{
            {ID: 1, PID: 1, CID: 123456, Name: "第一集"},
            {ID: 2, PID: 2, CID: 123457, Name: "第二集"},
        },
    }

    task, _ := manager.AddVideoTask(video, "./downloads", downloader.PriorityNormal)
    fmt.Printf("任务已添加: %s\n", task.ID)

    // 等待任务完成
    time.Sleep(10 * time.Second)

    // 查看统计
    stats := manager.GetStats()
    fmt.Printf("统计信息: %+v\n", stats)
}
```

### 示例 2: 带事件处理

```go
package main

import (
    "fmt"

    "bili-download/internal/downloader"
)

func main() {
    manager, _ := downloader.NewDownloadManager(cfg, biliClient)
    manager.Start()
    defer manager.Stop()

    // 添加事件处理器
    manager.AddEventHandler(func(event downloader.ManagerEvent) {
        switch event.Type {
        case downloader.EventTaskAdded:
            fmt.Printf("[添加] %s\n", event.Task.ID)

        case downloader.EventTaskStarted:
            fmt.Printf("[开始] %s - %s\n", event.Task.ID, event.Task.Video.Name)

        case downloader.EventTaskProgress:
            if event.Progress != nil {
                fmt.Printf("[进度] %s: %.2f%%\n",
                    event.Progress.Name,
                    event.Progress.Progress)
            }

        case downloader.EventTaskCompleted:
            fmt.Printf("[完成] %s - %s\n", event.Task.ID, event.Message)

        case downloader.EventTaskFailed:
            fmt.Printf("[失败] %s - %s\n", event.Task.ID, event.Message)

        case downloader.EventTaskRetrying:
            fmt.Printf("[重试] %s - %s\n", event.Task.ID, event.Message)

        case downloader.EventTaskCancelled:
            fmt.Printf("[取消] %s\n", event.Task.ID)
        }
    })

    // 添加任务...
}
```

### 示例 3: 批量任务管理

```go
package main

import (
    "fmt"

    "bili-download/internal/downloader"
)

func main() {
    manager, _ := downloader.NewDownloadManager(cfg, biliClient)
    manager.Start()
    defer manager.Stop()

    // 批量添加任务
    videos := []*models.Video{
        {ID: 1, BVid: "BV1xx411c7mD", Name: "视频1"},
        {ID: 2, BVid: "BV2xx411c7mD", Name: "视频2"},
        {ID: 3, BVid: "BV3xx411c7mD", Name: "视频3"},
    }

    for _, video := range videos {
        // 高优先级任务
        priority := downloader.PriorityNormal
        if video.ID == 1 {
            priority = downloader.PriorityHigh
        }

        task, err := manager.AddVideoTask(video, "./downloads", priority)
        if err != nil {
            fmt.Printf("添加任务失败: %v\n", err)
            continue
        }
        fmt.Printf("已添加: %s (优先级: %d)\n", task.ID, task.Priority)
    }

    // 监控任务
    for {
        stats := manager.GetStats()
        fmt.Printf("\r队列: %d | 运行: %d | 完成: %d | 失败: %d",
            stats.QueuedTasks,
            stats.RunningTasks,
            stats.CompletedTasks,
            stats.FailedTasks)

        if stats.RunningTasks == 0 && stats.QueuedTasks == 0 {
            break
        }

        time.Sleep(1 * time.Second)
    }

    fmt.Println("\n所有任务完成")
}
```

## 任务生命周期

```
┌─────────────┐
│   Pending   │  任务创建
└──────┬──────┘
       │
       │ 入队
       ▼
┌─────────────┐
│   Queued    │  在队列中等待
└──────┬──────┘
       │
       │ 调度器选中
       ▼
┌─────────────┐
│   Running   │  正在执行
└──────┬──────┘
       │
       ├─────────┐
       │         │
       ▼         ▼
┌──────────┐  ┌──────────┐
│Completed │  │  Failed  │
└──────────┘  └────┬─────┘
                   │
                   │ 可重试？
                   ▼
              ┌─────────┐
              │ Retrying│ ─┐
              └─────────┘  │
                   ▲        │
                   └────────┘
```

## 性能特征

### 队列操作

- 入队: O(log n)
- 出队: O(log n)
- 查找: O(1)
- 移除: O(log n)
- 更新优先级: O(log n)

### 并发控制

- 获取许可: O(1)
- 释放许可: O(1)
- 带超时获取: O(1) + 等待时间

### 内存占用

- 每个任务: ~200-300 字节
- 队列开销: O(n)
- 并发控制: O(1)

## 测试

### 运行所有测试

```bash
cd internal/downloader
go test -v
```

### 运行特定测试

```bash
# 队列测试
go test -v -run TestTaskQueue

# 并发测试
go test -v -run TestConcurrency

# 任务测试
go test -v -run TestDownloadTask
```

### 性能测试

```bash
go test -bench=. -benchmem
```

## 已知限制

1. **暂停/恢复功能**: 接口已预留但未实现
2. **任务持久化**: 当前仅在内存中，重启后丢失
3. **优先级动态调整**: 可以调整但不会影响正在执行的任务
4. **速率限制**: 配置项存在但未在管理器中实现

## 后续开发计划

### 高优先级

1. **任务持久化**
   - 将任务状态保存到数据库
   - 支持管理器重启后恢复任务
   - 任务历史记录

2. **速率限制实现**
   - API 调用速率限制
   - 下载速度限制
   - 自适应限速

### 中优先级

3. **暂停/恢复功能**
   - 支持暂停正在下载的任务
   - 断点续传
   - 批量暂停/恢复

4. **任务依赖**
   - 任务间依赖关系
   - 串行任务链
   - 条件执行

### 低优先级

5. **高级调度策略**
   - 时间窗口调度
   - 资源感知调度
   - 负载均衡

6. **Web Dashboard**
   - 任务可视化管理
   - 实时进度展示
   - 统计图表

## 常见问题

### Q: 如何调整并发数？

A: 在配置文件中修改：
```yaml
advanced:
  concurrent_limit:
    video: 3  # 改为3个视频并发
    page: 6   # 改为6个分P并发
```

### Q: 如何处理任务失败？

A: 管理器会自动重试失败的任务（默认3次）。如果仍然失败，可以手动重试：
```go
err := manager.RetryTask(taskID)
```

### Q: 如何获取实时进度？

A: 使用事件处理器：
```go
manager.AddEventHandler(func(event ManagerEvent) {
    if event.Type == EventTaskProgress && event.Progress != nil {
        fmt.Printf("Progress: %.2f%%\n", event.Progress.Progress)
    }
})
```

### Q: 如何取消所有任务？

A: 停止管理器会自动取消所有运行中的任务：
```go
manager.Stop()
```

### Q: 任务ID是如何生成的？

A: 任务ID根据任务类型自动生成：
- 视频任务: `video-{videoID}`
- 分P任务: `page-{videoID}-{pageID}`
- 合集任务: `collection-{videoID}-{timestamp}`

## 贡献指南

欢迎提交 Issue 和 Pull Request！

## 许可证

与主项目保持一致
