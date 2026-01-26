# 源码部署

## 环境要求

- **Go**：1.23.3+
- **Node.js**：20+
- **PostgreSQL**：16+
- **FFmpeg**：最新版
- **yt-dlp**：最新版

## 安装依赖

### 1. PostgreSQL

```bash
# Ubuntu/Debian
apt install postgresql-16

# 创建数据库
sudo -u postgres psql
CREATE DATABASE bili_sync;
CREATE USER bili_sync WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE bili_sync TO bili_sync;
```

### 2. FFmpeg 和 yt-dlp

```bash
# Ubuntu/Debian
apt install ffmpeg
pip install yt-dlp

# macOS
brew install ffmpeg yt-dlp
```

## 构建步骤

### 1. 克隆仓库

```bash
git clone https://github.com/yourusername/video-sync.git
cd video-sync
```

### 2. 构建前端

```bash
cd web
npm install
npm run build
cd ..
```

### 3. 构建后端

```bash
go mod download
go build -o video-sync cmd/server/main.go
```

## 配置

### 1. 创建配置文件

```bash
mkdir -p configs downloads metadata logs
```

### 2. 编辑配置

编辑 `configs/config.yaml`：

```yaml
server:
  bind_address: "0.0.0.0:8080"

database:
  host: "localhost"
  port: 5432
  user: "bili_sync"
  password: "your_password"
  dbname: "bili_sync"

paths:
  download_base: "./downloads/bilibili"
  upper_path: "./metadata/people"
```

## 运行

### 开发模式

```bash
# 后端
go run cmd/server/main.go

# 前端（另一个终端）
cd web
npm run dev
```

### 生产模式

```bash
./video-sync
```

访问 `http://localhost:8080`

## 系统服务

### Systemd 配置

创建 `/etc/systemd/system/video-sync.service`：

```ini
[Unit]
Description=Video-Sync Service
After=network.target postgresql.service

[Service]
Type=simple
User=yourusername
WorkingDirectory=/path/to/video-sync
ExecStart=/path/to/video-sync
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable video-sync
sudo systemctl start video-sync
```

## Nginx 反向代理（可选）

```nginx
server {
    listen 80;
    server_name yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }

    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```
