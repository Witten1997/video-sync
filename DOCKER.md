# Docker 部署指南

本文档详细说明如何使用 Docker 部署 video-sync 项目。

## 项目架构

本项目采用**单容器架构**，一个 Docker 镜像包含以下所有组件：

- **前端服务**：Vue.js 应用，通过 Nginx 提供静态文件服务（端口 80）
- **后端服务**：Go 应用，提供 API 服务（内部端口 8080，不对外开放）
- **PostgreSQL 数据库**：存储应用数据（内部端口 5432，不对外开放）
- **yt-dlp**：视频下载工具
- **FFmpeg**：视频处理工具

## 快速开始

### 1. 准备配置文件

首先，复制配置示例文件并根据需要修改：

```bash
# 确保 configs 目录存在配置文件
cp configs/config.example.yaml configs/config.yaml
```

编辑 `configs/config.yaml`，配置 B站 认证信息等：

```yaml
bilibili:
  credential:
    sessdata: "your_sessdata"
    bili_jct: "your_bili_jct"
    buvid3: "your_buvid3"
    dedeuserid: "your_dedeuserid"
    ac_time_value: "your_ac_time_value"
```

### 2. 修改环境变量（可选）

编辑 `docker-compose.yml` 文件，修改数据库密码等敏感信息：

```yaml
environment:
  - POSTGRES_USER=bili_sync
  - POSTGRES_PASSWORD=your_secure_password_here  # 修改为强密码
  - POSTGRES_DB=bili_sync
```

### 3. 构建并启动服务

```bash
# 构建镜像并启动容器
docker-compose up -d --build

# 查看日志
docker-compose logs -f

# 查看特定服务的日志
docker logs video-sync
```

### 4. 访问应用

打开浏览器访问：http://localhost

## 目录结构

```
video-sync/
├── Dockerfile              # Docker 镜像构建文件
├── docker-compose.yml      # Docker Compose 配置文件
├── .dockerignore          # Docker 构建忽略文件
├── configs/               # 配置文件目录
│   └── config.yaml        # 应用配置文件（需手动创建）
├── downloads/             # 视频下载目录（自动创建）
├── metadata/              # 元数据目录（自动创建）
└── logs/                  # 日志目录（自动创建）
```

## 数据持久化

以下目录通过数据卷进行持久化：

| 容器路径 | 宿主机路径 | 说明 |
|---------|-----------|------|
| `/var/lib/postgresql/data` | `postgres_data` 卷 | PostgreSQL 数据 |
| `/downloads` | `./downloads` | 下载的视频文件 |
| `/metadata` | `./metadata` | 视频元数据 |
| `/var/log/bili-sync` | `./logs` | 应用日志 |
| `/app/configs` | `./configs` | 配置文件 |

## 常用命令

### 启动服务

```bash
# 启动服务
docker-compose up -d

# 查看运行状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 停止服务

```bash
# 停止服务
docker-compose stop

# 停止并删除容器（数据卷不会删除）
docker-compose down

# 停止并删除容器和数据卷（谨慎使用！）
docker-compose down -v
```

### 重启服务

```bash
# 重启服务
docker-compose restart

# 重新构建并重启
docker-compose up -d --build
```

### 查看日志

```bash
# 查看所有日志
docker-compose logs

# 实时查看日志
docker-compose logs -f

# 查看最近 100 行日志
docker-compose logs --tail=100

# 进入容器查看详细日志
docker exec -it video-sync bash
tail -f /var/log/supervisor/backend.log
tail -f /var/log/supervisor/postgresql.log
tail -f /var/log/supervisor/nginx.log
```

### 进入容器

```bash
# 进入容器 Shell
docker exec -it video-sync bash

# 检查服务状态
docker exec -it video-sync supervisorctl status

# 重启特定服务
docker exec -it video-sync supervisorctl restart backend
docker exec -it video-sync supervisorctl restart nginx
```

### 备份与恢复

#### 备份数据库

```bash
# 备份数据库
docker exec -it video-sync su - postgres -c "pg_dump bili_sync > /tmp/backup.sql"
docker cp video-sync:/tmp/backup.sql ./backup_$(date +%Y%m%d_%H%M%S).sql
```

#### 恢复数据库

```bash
# 恢复数据库
docker cp backup.sql video-sync:/tmp/
docker exec -it video-sync su - postgres -c "psql bili_sync < /tmp/backup.sql"
```

#### 备份下载文件

```bash
# 备份下载目录
tar -czf downloads_backup_$(date +%Y%m%d_%H%M%S).tar.gz downloads/
```

## 端口说明

| 端口 | 服务 | 访问权限 |
|-----|------|---------|
| 80 | Nginx (前端) | 对外开放 |
| 8080 | Go Backend | 仅容器内部 |
| 5432 | PostgreSQL | 仅容器内部 |

**安全说明**：后端服务和数据库端口不对外开放，只能在容器内部访问，所有外部请求通过 Nginx 代理转发。

## 更新应用

### 更新代码

```bash
# 拉取最新代码
git pull

# 重新构建并启动
docker-compose up -d --build
```

### 仅更新配置

```bash
# 修改 configs/config.yaml 后重启后端服务
docker exec -it video-sync supervisorctl restart backend
```

## 故障排查

### 服务无法启动

1. 查看容器日志：
```bash
docker-compose logs
```

2. 检查容器状态：
```bash
docker-compose ps
```

3. 检查容器内服务状态：
```bash
docker exec -it video-sync supervisorctl status
```

### 数据库连接失败

```bash
# 检查 PostgreSQL 是否正常运行
docker exec -it video-sync supervisorctl status postgresql

# 手动测试数据库连接
docker exec -it video-sync su - postgres -c "psql -d bili_sync -c 'SELECT 1;'"
```

### 前端无法访问

1. 检查 Nginx 状态：
```bash
docker exec -it video-sync supervisorctl status nginx
```

2. 检查 Nginx 配置：
```bash
docker exec -it video-sync nginx -t
```

3. 检查前端文件是否存在：
```bash
docker exec -it video-sync ls -la /app/frontend/
```

### 后端 API 无法访问

1. 检查后端服务状态：
```bash
docker exec -it video-sync supervisorctl status backend
```

2. 查看后端日志：
```bash
docker exec -it video-sync tail -f /var/log/supervisor/backend.log
```

3. 检查配置文件：
```bash
docker exec -it video-sync cat /app/configs/config.yaml
```

### 视频下载失败

1. 检查 yt-dlp 是否安装：
```bash
docker exec -it video-sync yt-dlp --version
```

2. 检查 FFmpeg 是否安装：
```bash
docker exec -it video-sync ffmpeg -version
```

3. 检查下载目录权限：
```bash
docker exec -it video-sync ls -la /downloads/
```

## 性能优化

### 资源限制

在 `docker-compose.yml` 中取消注释以下部分来限制资源使用：

```yaml
deploy:
  resources:
    limits:
      cpus: '2'
      memory: 4G
    reservations:
      cpus: '1'
      memory: 2G
```

### 调整并发数

编辑 `configs/config.yaml`：

```yaml
advanced:
  concurrent_limit:
    video: 5          # 增加视频并发数
    page: 3           # 增加分页并发数
```

## 安全建议

1. **修改默认密码**：务必修改 `docker-compose.yml` 中的数据库密码
2. **配置认证令牌**：在 `configs/config.yaml` 中设置 `server.auth_token`
3. **使用 HTTPS**：在生产环境中建议使用 Nginx 反向代理配置 HTTPS
4. **定期备份**：定期备份数据库和下载文件
5. **更新镜像**：定期更新基础镜像以获取安全补丁

## 常见问题

### Q: 如何修改前端端口？

A: 修改 `docker-compose.yml` 中的端口映射：
```yaml
ports:
  - "8080:80"  # 将前端服务映射到 8080 端口
```

### Q: 如何清理 Docker 资源？

A: 使用以下命令：
```bash
# 清理未使用的镜像
docker image prune -a

# 清理未使用的卷
docker volume prune

# 清理所有未使用的资源
docker system prune -a --volumes
```

### Q: 如何升级 yt-dlp 版本？

A: 进入容器手动升级：
```bash
docker exec -it video-sync python3 -m pip install -U yt-dlp
```

或重新构建镜像以获取最新版本。

### Q: 容器重启后数据会丢失吗？

A: 不会。数据通过 Docker 卷持久化，容器重启不影响数据。只有执行 `docker-compose down -v` 才会删除数据卷。

## 技术支持

如遇到问题，请：

1. 查看日志文件
2. 检查配置是否正确
3. 参考故障排查章节
4. 提交 Issue 到项目仓库

## 许可证

本项目遵循项目主仓库的许可证。
