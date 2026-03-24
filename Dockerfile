# ============================================
# 阶段 1: 构建前端
# ============================================
FROM node:20-alpine AS frontend-builder

RUN npm config set registry https://registry.npmmirror.com

WORKDIR /app/web

COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npx vite build

# ============================================
# 阶段 2: 构建后端（前端通过 embed 嵌入二进制）
# ============================================
FROM golang:1.24-alpine AS backend-builder

RUN apk add --no-cache tzdata
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY web/frontend.go ./web/
COPY --from=frontend-builder /app/web/dist ./web/dist/

ARG VERSION=1.0.0
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s -X 'bili-download/internal/version.Version=${VERSION}' -X 'bili-download/internal/version.GitTag=${VERSION}' -X 'bili-download/internal/version.BuildTime=$(TZ=Asia/Shanghai date +%Y-%m-%d\ %H:%M:%S)'" \
    -o /app/video-sync ./cmd/server

# ============================================
# 阶段 3: 最终镜像
# ============================================
FROM video-sync-alpine-base:v0.0.2

LABEL maintainer="video-sync"
LABEL description="video-sync - 前端已嵌入二进制，无需Nginx"

RUN mkdir -p \
    /app/configs \
    /downloads/bilibili \
    /metadata/people \
    /var/log/video-sync

COPY --from=backend-builder /app/video-sync /app/
COPY configs/config.example.yaml /app/configs/config.yaml
COPY bili-sync-schema.sql /app/
COPY migrations/ /app/migrations/

# 启动脚本
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

# 执行数据库迁移
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

chmod 755 /downloads /metadata /var/log/video-sync

echo "启动服务..."
exec /app/video-sync -config /app/configs/config.yaml
EOF

RUN chmod +x /app/entrypoint.sh && \
    sed -i 's/\r$//' /app/entrypoint.sh

EXPOSE 8080

WORKDIR /app

VOLUME ["/downloads", "/metadata", "/var/log/video-sync", "/app/configs"]

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/entrypoint.sh"]
