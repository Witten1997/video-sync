# Docker 部署

## 使用 Docker Compose（推荐）

### 1. 下载项目

```bash
git clone https://github.com/yourusername/video-sync.git
cd video-sync
```

### 2. 检查配置

编辑 `docker-compose.yml`（如需修改端口）：

```yaml
services:
  video-sync:
    ports:
      - "21001:80"  # 修改宿主机端口
```

### 3. 启动服务

```bash
docker-compose up -d
```

### 4. 查看日志

```bash
docker-compose logs -f
```

### 5. 停止服务

```bash
docker-compose down
```

## 目录说明

```
.
├── downloads/        # 视频下载目录
├── metadata/         # UP主元数据
├── configs/          # 配置文件
├── logs/             # 日志文件
└── docker-compose.yml
```

## 镜像说明

**当前版本**：`v0.0.0.4`

**包含组件**：
- Alpine Linux 基础镜像
- Nginx（反向代理）
- PostgreSQL 16
- yt-dlp + FFmpeg
- Go 后端服务
- Vue3 前端

## 数据持久化

### 数据卷挂载

```yaml
volumes:
  - ./downloads:/downloads          # 视频文件
  - ./metadata:/metadata            # 元数据
  - ./configs:/app/configs          # 配置文件
  - ./logs:/var/log/bili-sync       # 日志
  - postgres_data:/var/lib/postgresql/data  # 数据库
```

### 备份数据

```bash
# 备份数据库
docker exec postgres pg_dump -U bili_sync bili_sync > backup.sql

# 备份配置
cp -r configs configs.backup
```

## 环境变量

```yaml
environment:
  - DB_HOST=postgres
  - DB_PORT=5432
  - POSTGRES_USER=bili_sync
  - POSTGRES_PASSWORD=bili_sync_password
  - POSTGRES_DB=bili_sync
  - APP_PORT=8080
```

## 网络配置

### 自定义网络

```yaml
networks:
  video-sync-network:
    driver: bridge
```

### 连接其他服务

```yaml
# 同一网络下的 Emby 容器可直接访问
networks:
  - video-sync-network
```

## 更新镜像

```bash
docker-compose pull
docker-compose up -d
```

## 故障排查

### 查看容器状态

```bash
docker-compose ps
```

### 进入容器

```bash
docker-compose exec video-sync sh
```

### 查看数据库

```bash
docker-compose exec postgres psql -U bili_sync
```

### 重启服务

```bash
docker-compose restart
```
