# Bug 修复总结

## 修复日期
2025-11-03

## 问题描述
服务启动时出现多个编译错误，主要涉及字段名大小写不一致和方法名不匹配的问题。

## 错误列表

### 1. 方法名错误
```
s.biliClient.GetVideoInfo undefined
```

**修复**: 将 `GetVideoInfo` 改为 `GetVideoDetail`

**影响文件**:
- `internal/api/handler_video.go`
- `internal/workflow/fetch.go`

### 2. 字段名大小写错误

#### Video 模型字段
```
unknown field BVID, Pubtime, Favtime, Ctime
```

**正确字段名**: `BVid`, `PubTime`, `FavTime`, `CTime`

**影响文件**:
- `internal/api/handler_video.go`
- `internal/workflow/refresh.go`
- `internal/workflow/download.go`

#### VideoPage 字段
```
page.Cid undefined (should be CID)
```

**正确字段名**: `CID` (全大写)

**影响文件**:
- `internal/api/handler_video.go`

### 3. 不存在的方法
```
video.SetTags undefined
```

**修复**: `Video.Tags` 是 `[]string` 类型，直接赋值即可，不需要 `SetTags` 方法

**影响文件**:
- `internal/api/handler_video.go`
- `internal/workflow/fetch.go`

### 4. Page 模型字段
```
unknown field Ctime in struct literal of type models.Page
```

**修复**: Page 模型没有 `Ctime` 字段，只有 `CreatedAt`，移除该字段赋值

**影响文件**:
- `internal/api/handler_video.go`
- `internal/workflow/fetch.go`

### 5. 接口方法名
```
adapter.FetchVideos undefined
```

**修复**: 根据 `adapter.VideoSource` 接口，应使用 `Scan(ctx, opts)` 方法

**影响文件**:
- `internal/workflow/refresh.go`

## 详细修复内容

### internal/api/handler_video.go

1. 添加 `time` 包导入
2. 修改方法调用:
   - `GetVideoInfo` → `GetVideoDetail`
   - `GetVideoTags` 单独调用获取标签
3. 修正字段名:
   - `BVID` → `BVid`
   - `Pubtime` → `PubTime`
   - `Favtime` → `FavTime`
   - `Ctime` → `CTime`
   - `page.Cid` → `page.CID`
4. 修正时间字段，使用 `time.Unix()` 转换时间戳
5. 移除 `video.SetTags` 调用，直接赋值 `video.Tags`
6. 移除 Page 的 `Ctime` 字段

### internal/workflow/fetch.go

1. 修改方法调用:
   - `GetVideoInfo` → `GetVideoDetail`
   - 添加 `GetVideoTags` 调用
2. 修正字段名:
   - `video.BVID` → `video.BVid`
   - `videoInfo.Tid` 类型已是 `int`，不需要类型转换
3. 修正 Tags 赋值方式
4. 移除 Page 的 `Ctime` 字段

### internal/workflow/refresh.go

1. 修正字段名和映射:
   - `video.BVID` → `video.BVid`
   - `video.BVID` → `video.BVid`
   - `video.UpperID` → `video.Owner.Mid`
   - `video.UpperName` → `video.Owner.Name`
   - `video.UpperFace` → `video.Owner.Face`
   - `video.PublishTime` → `video.PubDate`
   - `video.FavoriteTime` → `video.AddTime`
   - `Pubtime` → `PubTime`
   - `Favtime` → `FavTime`
   - `Ctime` → `CTime`
2. 修改接口调用:
   - `adapter.FetchVideos(ctx)` → `adapter.Scan(ctx, nil)`

### internal/workflow/download.go

使用 sed 批量替换:
- `video.BVID` → `video.BVid`
- `video.Pubtime` → `video.PubTime`
- `video.Favtime` → `video.FavTime`

## 数据结构对应关系

### bilibili.VideoDetail → models.Video

```go
BVid        → BVid
Title       → Name
Desc        → Intro
Pic         → Cover
Owner.Mid   → UpperID
Owner.Name  → UpperName
Owner.Face  → UpperFace
Tid         → Category
PubDate     → PubTime (使用 time.Unix 转换)
CTime       → CTime (使用 time.Unix 转换)
```

### bilibili.VideoPage → models.Page

```go
CID             → CID (注意大写)
Page            → PID
Part            → Name
Duration        → Duration
Dimension.Width → Width
Dimension.Height → Height
FirstFrame      → Image
```

### adapter.VideoInfo → models.Video

```go
BVid            → BVid
Title           → Name
Description     → Intro
Cover           → Cover
Owner.Mid       → UpperID
Owner.Name      → UpperName
Owner.Face      → UpperFace
PubDate         → PubTime
AddTime         → FavTime
```

## 编译结果

✅ 所有错误已修复，项目编译成功

```bash
cd /d/Code/bili-download
go build -o build/bili-download.exe ./cmd/server
# 编译成功，无错误
```

## 注意事项

1. **字段命名规范**:
   - 首字母大写的字段名遵循 Go 的导出规则
   - 缩写词通常全大写（如 `BVid`, `CID`, `ID`）
   - 时间相关字段使用驼峰命名（如 `PubTime`, `FavTime`, `CTime`）

2. **时间类型转换**:
   - Bilibili API 返回的时间是 Unix 时间戳（int64）
   - 需要使用 `time.Unix(timestamp, 0)` 转换为 `time.Time`

3. **接口一致性**:
   - 确保使用正确的接口方法名
   - 适配器接口使用 `Scan` 而不是 `FetchVideos`

4. **标签处理**:
   - `Video.Tags` 是 `[]string` 类型，直接赋值
   - 不要使用不存在的 `SetTags` 方法

## 相关文件

- [x] internal/api/handler_video.go
- [x] internal/workflow/fetch.go
- [x] internal/workflow/refresh.go
- [x] internal/workflow/download.go

## 测试建议

1. 测试通过 URL 下载视频功能
2. 测试视频源扫描功能
3. 测试视频详情获取功能
4. 验证数据库字段映射正确性
