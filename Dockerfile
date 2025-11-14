# ============================================
# 基于自定义 Alpine 基础镜像的应用容器
# ============================================
# 使用预先构建好的 video-sync-alpine-base 基础镜像
# 这样可以避免每次构建时的网络问题
# ============================================

# ============================================
# 阶段 1: 构建前端
# ============================================
FROM node:20-alpine AS frontend-builder

# 使用国内 npm 镜像
RUN npm config set registry https://registry.npmmirror.com

WORKDIR /app/web

COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npx vite build

# ============================================
# 阶段 2: 构建后端
# ============================================
FROM golang:1.23-alpine AS backend-builder

# 设置 Go 代理
ENV GOPROXY=https://goproxy.io,direct

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY migrations/ ./migrations/

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /app/bili-sync ./cmd/server

# ============================================
# 阶段 3: 最终镜像（基于自定义基础镜像）
# ============================================
FROM video-sync-alpine-base:latest

LABEL maintainer="video-sync"
LABEL description="video-sync application based on Alpine"

# 创建应用目录
RUN mkdir -p \
    /app/backend \
    /app/frontend \
    /app/configs \
    /downloads/bilibili \
    /metadata/people \
    /var/log/bili-sync \
    /var/log/supervisor

# 从构建阶段复制文件
COPY --from=backend-builder /app/bili-sync /app/backend/
COPY --from=frontend-builder /app/web/dist /app/frontend/

# 复制配置文件
COPY configs/config.example.yaml /app/configs/config.yaml
COPY bili-sync-schema.sql /app/
COPY migrations/ /app/migrations/

# 配置 Nginx
RUN rm -f /etc/nginx/http.d/default.conf
COPY <<'EOF' /etc/nginx/http.d/video-sync.conf
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

# 配置 Supervisor
COPY <<'EOF' /etc/supervisor.d/supervisord.ini
[supervisord]
nodaemon=true
user=root
logfile=/var/log/supervisor/supervisord.log
pidfile=/run/supervisord.pid

[program:backend]
command=/app/backend/bili-sync -config /app/configs/config.yaml
directory=/app/backend
autostart=true
autorestart=true
priority=1
stdout_logfile=/var/log/supervisor/backend.log
stderr_logfile=/var/log/supervisor/backend_err.log
environment=TZ="Asia/Shanghai"

[program:nginx]
command=/usr/sbin/nginx -g "daemon off;"
autostart=true
autorestart=true
priority=2
stdout_logfile=/var/log/supervisor/nginx.log
stderr_logfile=/var/log/supervisor/nginx_err.log
EOF

# 创建启动脚本
COPY <<'EOF' /app/entrypoint.sh
#!/bin/bash
set -e

# 等待 PostgreSQL 准备就绪
echo "等待 PostgreSQL 启动..."
until PGPASSWORD=${POSTGRES_PASSWORD:-video_sync} psql -h ${DB_HOST:-postgres} -p ${DB_PORT:-5432} -U ${POSTGRES_USER:-video_sync} -d ${POSTGRES_DB:-video_sync} -c '\q' 2>/dev/null; do
    echo "PostgreSQL 未就绪，等待..."
    sleep 2
done
echo "PostgreSQL 已就绪"

# 初始化数据库（如果需要）
echo "检查数据库是否需要初始化..."
TABLE_COUNT=$(PGPASSWORD=${POSTGRES_PASSWORD:-video_sync} psql -h ${DB_HOST:-postgres} -p ${DB_PORT:-5432} -U ${POSTGRES_USER:-video_sync} -d ${POSTGRES_DB:-video_sync} -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';")

if [ "$TABLE_COUNT" -eq "0" ]; then
    echo "初始化数据库表..."
    PGPASSWORD=${POSTGRES_PASSWORD:-video_sync} psql -h ${DB_HOST:-postgres} -p ${DB_PORT:-5432} -U ${POSTGRES_USER:-video_sync} -d ${POSTGRES_DB:-video_sync} -f /app/bili-sync-schema.sql || true
    echo "数据库初始化完成"
else
    echo "数据库已存在表，跳过初始化"
fi

# 执行数据库迁移（总是执行，使用 IF NOT EXISTS 避免重复）
echo "执行数据库迁移..."
if [ -d "/app/migrations" ]; then
    for migration in /app/migrations/*.sql; do
        if [ -f "$migration" ]; then
            echo "执行迁移: $(basename $migration)"
            PGPASSWORD=${POSTGRES_PASSWORD:-video_sync} psql -h ${DB_HOST:-postgres} -p ${DB_PORT:-5432} -U ${POSTGRES_USER:-video_sync} -d ${POSTGRES_DB:-video_sync} -f "$migration" 2>&1 | grep -v "already exists" || true
        fi
    done
    echo "数据库迁移完成"
fi

# 更新配置文件中的数据库连接信息
sed -i "s/host: .*/host: ${DB_HOST:-postgres}/g" /app/configs/config.yaml
sed -i "s/user: .*/user: ${POSTGRES_USER:-video_sync}/g" /app/configs/config.yaml
sed -i "s/password: .*/password: ${POSTGRES_PASSWORD:-video_sync}/g" /app/configs/config.yaml
sed -i "s/dbname: .*/dbname: ${POSTGRES_DB:-video_sync}/g" /app/configs/config.yaml
sed -i "s/port: .*/port: ${DB_PORT:-5432}/g" /app/configs/config.yaml

# 确保目录权限正确
chmod -R 755 /downloads /metadata /var/log/bili-sync

# 启动 supervisor
echo "启动所有服务..."
exec /usr/bin/supervisord -c /etc/supervisord.conf
EOF

# 修复启动脚本
RUN chmod +x /app/entrypoint.sh && \
    sed -i 's/\r$//' /app/entrypoint.sh

# 暴露前端端口
EXPOSE 80

# 设置工作目录
WORKDIR /app

# 创建数据卷
VOLUME ["/downloads", "/metadata", "/var/log/bili-sync", "/app/configs"]

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost/api/health || exit 1

# 启动脚本
ENTRYPOINT ["/app/entrypoint.sh"]
