# 媒体库集成

## 支持的媒体服务器

- **Emby**
- **Jellyfin**
- **Plex**
- 其他支持NFO的媒体服务器

## 元数据格式

### NFO文件（Kodi标准）

```xml
<?xml version="1.0" encoding="UTF-8"?>
<movie>
  <title>视频标题</title>
  <plot>视频简介</plot>
  <aired>2024-01-01</aired>
  <director>UP主名称</director>
  <actor>
    <name>UP主名称</name>
    <thumb>metadata/people/UP主名称/folder.jpg</thumb>
  </actor>
  <thumb>poster.jpg</thumb>
</movie>
```

### 文件命名规范

- **视频文件**：`视频标题.mp4`
- **NFO文件**：`视频标题.nfo`
- **封面图**：`poster.jpg`
- **字幕文件**：`视频标题.ass`

## 配置媒体服务器

### Emby/Jellyfin

1. 添加媒体库
2. 路径设置为 `downloads/bilibili`
3. 内容类型选择「电影」
4. 启用「使用本地元数据」
5. 扫描库

### Plex

1. 添加库 → 电影
2. 路径设置为 `downloads/bilibili`
3. 高级 → 扫描器：Personal Media
4. 代理：Local Media Assets
5. 扫描库

## 弹幕显示

### 配置步骤

1. 确保 `download.skip_danmaku: false`
2. 下载完成后会生成 `.ass` 字幕文件
3. 媒体服务器播放时选择字幕轨道

### 弹幕样式

系统自动转换B站弹幕为ASS格式，支持：
- 滚动弹幕
- 顶部弹幕
- 底部弹幕
- 颜色和字体

## 元数据刷新

### 自动刷新
下载新视频后，媒体服务器会自动检测并导入。

### 手动刷新
- Emby/Jellyfin：扫描媒体库
- Plex：刷新库

## 目录挂载

### Docker部署
确保 `docker-compose.yml` 已正确挂载：

```yaml
volumes:
  - ./downloads:/downloads
  - ./metadata:/metadata
```

### 媒体服务器访问
将相同目录挂载到媒体服务器容器：

```yaml
# Emby/Jellyfin
volumes:
  - /path/to/downloads/bilibili:/media/bilibili
  - /path/to/metadata:/media/metadata
```
