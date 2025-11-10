# 前端构建阶段
FROM node:20-alpine AS web-builder

WORKDIR /web
COPY web/package*.json ./
RUN npm ci --only=production
COPY web/ ./
RUN npm run build

# 后端构建阶段
FROM golang:1.23.3-alpine AS backend-builder

RUN apk add --no-cache git make
WORKDIR /build

# 优化构建缓存：先复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 再复制源码
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# 构建二进制文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo \
    -ldflags '-w -s -extldflags "-static"' \
    -o video-sync ./cmd/server

# 最终运行阶段
FROM alpine:latest

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    python3 \
    py3-pip \
    ffmpeg \
    wget \
    su-exec && \
    # 安装 yt-dlp
    pip3 install --no-cache-dir yt-dlp && \
    # 清理缓存
    rm -rf /var/cache/apk/* /root/.cache

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非特权用户
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制文件
COPY --from=backend-builder /build/video-sync .
COPY --from=web-builder /web/dist ./web/dist

# 复制启动脚本
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# 创建必要的目录并设置权限
RUN mkdir -p \
    /app/configs \
    /downloads \
    /metadata \
    /var/log/video-sync && \
    chown -R appuser:appgroup \
    /app \
    /downloads \
    /metadata \
    /var/log/video-sync

# 声明数据卷
VOLUME ["/downloads", "/metadata", "/app/configs"]

# 切换到非特权用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 使用启动脚本
ENTRYPOINT ["docker-entrypoint.sh"]
