# ============================================
# Stage 1: build frontend
# ============================================
FROM node:20-alpine AS frontend-builder

RUN npm config set registry https://registry.npmmirror.com

WORKDIR /app/web

COPY web/package*.json ./
RUN npm ci

COPY web/ ./
RUN npx vite build

# ============================================
# Stage 2: build backend
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
# Stage 3: final image
# ============================================
FROM video-sync-alpine-base:v0.0.3

LABEL maintainer="video-sync"
LABEL description="video-sync frontend embedded into the binary"

RUN mkdir -p \
    /app/configs \
    /downloads/bilibili \
    /metadata/people \
    /var/log/video-sync

RUN python3 -m pip install --no-cache-dir --break-system-packages --upgrade yt-dlp && \
    echo "yt-dlp runtime: $(yt-dlp --version)"

COPY --from=backend-builder /app/video-sync /app/
COPY configs/config.example.yaml /app/configs/config.yaml
COPY bili-sync-schema.sql /app/
COPY scripts/docker/entrypoint.sh /app/entrypoint.sh

RUN chmod +x /app/entrypoint.sh && \
    sed -i 's/\r$//' /app/entrypoint.sh

EXPOSE 8080

WORKDIR /app

VOLUME ["/downloads", "/metadata", "/var/log/video-sync", "/app/configs"]

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/entrypoint.sh"]
