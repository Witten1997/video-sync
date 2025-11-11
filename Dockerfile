# ============================================
# 多阶段构建 Dockerfile - video-sync 项目（国内镜像优化版）
# ============================================
# 该镜像包含：前端服务、后端服务、PostgreSQL、yt-dlp、ffmpeg
# ============================================

# ============================================
# 阶段 1: 构建前端
# ============================================
FROM node:20-alpine AS frontend-builder

# 使用阿里云 npm 镜像加速
RUN npm config set registry https://registry.npmmirror.com

WORKDIR /app/web

# 复制前端依赖文件
COPY web/package*.json ./

# 安装前端依赖（包括开发依赖，构建需要）
RUN npm ci

# 复制前端源代码
COPY web/ ./

# 构建前端（直接使用 vite build，跳过类型检查）
RUN npx vite build

# ============================================
# 阶段 2: 构建后端
# ============================================
FROM golang:1.23-alpine AS backend-builder

# 设置 Go 代理
ENV GOPROXY=https://goproxy.io,direct

WORKDIR /app

# 复制 Go 依赖文件
COPY go.mod go.sum ./

# 下载 Go 依赖
RUN go mod download

# 复制后端源代码
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY migrations/ ./migrations/

# 构建后端二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /app/bili-sync ./cmd/server

# ============================================
# 阶段 3: 最终镜像
# ============================================
FROM ubuntu:22.04

# 设置环境变量
ENV DEBIAN_FRONTEND=noninteractive \
    TZ=Asia/Shanghai \
    POSTGRES_VERSION=15 \
    PGDATA=/var/lib/postgresql/data \
    LANG=zh_CN.UTF-8 \
    LANGUAGE=zh_CN:zh \
    LC_ALL=zh_CN.UTF-8

# 使用阿里云 Ubuntu 镜像源加速
RUN sed -i 's@//.*archive.ubuntu.com@//mirrors.aliyun.com@g' /etc/apt/sources.list && \
    sed -i 's@//.*security.ubuntu.com@//mirrors.aliyun.com@g' /etc/apt/sources.list

# 添加 PostgreSQL 官方仓库（使用清华镜像）并安装所有依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    wget \
    gnupg \
    lsb-release \
    && curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /usr/share/keyrings/postgresql-keyring.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/postgresql-keyring.gpg] https://mirrors.tuna.tsinghua.edu.cn/postgresql/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
    # 基础工具
    supervisor \
    tzdata \
    locales \
    python3 \
    python3-pip \
    # FFmpeg
    ffmpeg \
    # Nginx
    nginx \
    # PostgreSQL
    postgresql-${POSTGRES_VERSION} \
    postgresql-contrib-${POSTGRES_VERSION} \
    && locale-gen zh_CN.UTF-8 \
    && update-locale LANG=zh_CN.UTF-8 \
    # 配置 pip 使用清华镜像并安装 yt-dlp
    && python3 -m pip config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple \
    && python3 -m pip install --no-cache-dir -U yt-dlp \
    # 清理缓存
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# 创建必要的目录
RUN mkdir -p \
    /app/backend \
    /app/frontend \
    /app/configs \
    /downloads/bilibili \
    /metadata/people \
    /var/log/bili-sync \
    /var/log/supervisor \
    /etc/supervisor/conf.d

# 从构建阶段复制后端二进制文件
COPY --from=backend-builder /app/bili-sync /app/backend/

# 从构建阶段复制前端构建产物
COPY --from=frontend-builder /app/web/dist /app/frontend/

# 复制配置文件
COPY configs/config.example.yaml /app/configs/config.yaml
COPY bili-sync-schema.sql /app/

# 配置 Nginx
RUN rm -f /etc/nginx/sites-enabled/default
COPY <<'EOF' /etc/nginx/sites-available/video-sync
server {
    listen 80;
    server_name _;

    # 前端静态文件
    location / {
        root /app/frontend;
        try_files $uri $uri/ /index.html;
        index index.html;
    }

    # 后端 API 代理
    location /api {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
    }

    # WebSocket 支持
    location /ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # 下载文件访问
    location /downloads {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
EOF

RUN ln -s /etc/nginx/sites-available/video-sync /etc/nginx/sites-enabled/

# 配置 Supervisor
COPY <<'EOF' /etc/supervisor/conf.d/supervisord.conf
[supervisord]
nodaemon=true
user=root
logfile=/var/log/supervisor/supervisord.log
pidfile=/var/run/supervisord.pid

[program:postgresql]
command=/usr/lib/postgresql/15/bin/postgres -D /var/lib/postgresql/data
user=postgres
autostart=true
autorestart=true
priority=1
stdout_logfile=/var/log/supervisor/postgresql.log
stderr_logfile=/var/log/supervisor/postgresql_err.log

[program:backend]
command=/app/backend/bili-sync -config /app/configs/config.yaml
directory=/app/backend
autostart=true
autorestart=true
priority=2
stdout_logfile=/var/log/supervisor/backend.log
stderr_logfile=/var/log/supervisor/backend_err.log
environment=TZ="Asia/Shanghai"

[program:nginx]
command=/usr/sbin/nginx -g "daemon off;"
autostart=true
autorestart=true
priority=3
stdout_logfile=/var/log/supervisor/nginx.log
stderr_logfile=/var/log/supervisor/nginx_err.log
EOF

# 复制启动脚本并修复换行符
COPY entrypoint.sh /app/entrypoint.sh
RUN sed -i 's/\r$//' /app/entrypoint.sh && chmod +x /app/entrypoint.sh

# 暴露前端端口
EXPOSE 80

# 设置工作目录
WORKDIR /app

# 创建数据卷
VOLUME ["/var/lib/postgresql/data", "/downloads", "/metadata", "/var/log/bili-sync", "/app/configs"]

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost/api/health || exit 1

# 启动脚本
ENTRYPOINT ["/app/entrypoint.sh"]
