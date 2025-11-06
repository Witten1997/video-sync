# bili-download

专为 NAS 用户设计的哔哩哔哩视频同步工具，支持自动化下载、管理和组织 B站 视频，与媒体服务器（Emby/Jellyfin）无缝集成。

## 特性

- 🎯 **多种视频源支持**：收藏夹、稍后再看、视频合集、UP主投稿
- 📦 **自动化同步**：定时扫描和下载，无需手动操作
- 🎬 **媒体库友好**：自动生成 NFO 元数据，支持 Emby/Jellyfin/Kodi
- 🚀 **高效下载**：基于 yt-dlp，支持断点续传和并发下载
- 💬 **弹幕支持**：自动下载并转换为 ASS 格式
- 🎨 **Web 管理界面**：可视化管理视频源和下载任务
- 🐳 **容器化部署**：支持 Docker 和 Docker Compose
- 📊 **PostgreSQL 数据库**：高性能数据存储和查询

## 技术栈

- **后端**: Go 1.23.3
- **数据库**: PostgreSQL
- **前端**: Vue 3 + Vite
- **下载引擎**: yt-dlp

## 快速开始

### 前置要求

- Go 1.23.3+
- PostgreSQL 12+
- yt-dlp

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/your-org/bili-download.git
cd bili-download

# 安装依赖
make install-deps

# 编译项目
make build

# 运行
./build/bili-download
```

### 使用 Docker

```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run
```

### 使用 Docker Compose

```bash
# 启动服务（包含 PostgreSQL）
docker-compose up -d
```

## 配置

首次运行时会在 `configs/` 目录下自动生成默认配置文件 `config.yaml`。

关键配置项：

```yaml
# 数据库配置
database:
  host: "localhost"
  port: 5432
  user: "bili_sync"
  password: "your_password"
  dbname: "bili_sync"

# B站认证
bilibili:
  credential:
    sessdata: ""
    bili_jct: ""
    buvid3: ""

# 下载路径
paths:
  download_base: "/downloads/bilibili"
  upper_path: "/metadata/people"
```

详细配置说明请参考 [配置文档](./docs/config.md)。

## 环境变量

支持通过环境变量覆盖数据库配置：

- `DB_HOST`: PostgreSQL 主机地址
- `DB_PORT`: PostgreSQL 端口（默认 5432）
- `DB_USER`: 数据库用户名
- `DB_PASSWORD`: 数据库密码
- `DB_NAME`: 数据库名称
- `DB_SSLMODE`: SSL 模式（默认 disable）

## 开发

### 项目结构

```
bili-download/
├── cmd/                    # 程序入口
│   └── server/
├── internal/               # 内部包
│   ├── adapter/           # 视频源适配器
│   ├── api/               # HTTP API
│   ├── bilibili/          # B站 API 客户端
│   ├── config/            # 配置管理
│   ├── database/          # 数据库
│   ├── downloader/        # 下载器
│   └── ...
├── web/                   # 前端代码
├── configs/               # 配置文件
└── scripts/               # 脚本
```

### 开发命令

```bash
# 开发模式运行（热重载）
make dev

# 运行测试
make test

# 生成测试覆盖率报告
make test-coverage

# 代码格式化
make fmt

# 代码检查
make lint

# 多平台构建
make build-all
```

### 数据库迁移

TODO: 数据库迁移说明

## 使用指南

### 添加视频源

1. 访问 Web 界面：http://localhost:8080
2. 进入"视频源管理"页面
3. 点击"添加视频源"
4. 输入收藏夹 ID、UP主 ID 或合集 URL
5. 配置下载选项和过滤规则

### 配置 B站 凭据

1. 在浏览器中登录 B站
2. 打开开发者工具，查看 Cookie
3. 复制以下字段到配置文件：
   - SESSDATA
   - bili_jct
   - buvid3
   - DedeUserID
   - ac_time_value

### 媒体服务器集成

下载的视频会按照以下结构组织：

```
downloads/
└── 收藏夹名称/
    └── 视频标题/
        ├── 视频标题.mp4
        ├── 视频标题.nfo
        ├── 视频标题-poster.jpg
        └── 视频标题.zh-CN.default.ass
```

在 Emby/Jellyfin 中添加媒体库时，将 `downloads/` 目录添加为电影或电视剧库即可。

## API 文档

TODO: API 文档链接

## 常见问题

### Q: 如何获取收藏夹 ID？

A: 打开收藏夹页面，URL 中的数字即为收藏夹 ID。例如：`https://space.bilibili.com/xxx/favlist?fid=123456789`，其中 `123456789` 就是收藏夹 ID。

### Q: 下载失败怎么办？

A: 检查以下几点：
1. 确认 yt-dlp 已正确安装
2. 检查 B站 凭据是否有效
3. 查看日志文件中的错误信息
4. 可能触发了 B站 风控，等待一段时间后重试

### Q: 如何从 SQLite 迁移到 PostgreSQL？

A: 请参考 [数据库迁移指南](./docs/migration.md)。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

TODO: 添加许可证

## 致谢

- 原项目：[bili-sync](https://github.com/amtoaer/bili-sync)
- 下载工具：[yt-dlp](https://github.com/yt-dlp/yt-dlp)
- B站 API 文档：[bilibili-API-collect](https://github.com/SocialSisterYi/bilibili-API-collect)

## 免责声明

本项目仅供学习交流使用，请勿用于商业用途。下载的视频内容版权归原作者所有，请尊重原作者的劳动成果。
