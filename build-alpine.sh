#!/bin/bash
# ============================================
# Alpine 镜像两阶段构建脚本
# ============================================
# 用途：解决 Alpine 软件源网络不稳定问题
# 策略：先构建基础镜像（包含所有依赖），再构建应用镜像
# ============================================

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  Alpine 镜像两阶段构建脚本${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""

# ============================================
# 阶段 1: 构建基础镜像
# ============================================
echo -e "${YELLOW}[阶段 1/2] 构建 Alpine 基础镜像...${NC}"
echo "这个镜像包含所有依赖（FFmpeg, Python, Nginx 等）"
echo "只需要构建一次，后续可以复用"
echo ""

BASE_IMAGE="video-sync-alpine-base:latest"

# 检查基础镜像是否已存在
if docker images | grep -q "video-sync-alpine-base"; then
    echo -e "${YELLOW}基础镜像已存在，是否重新构建？ (y/n)${NC}"
    read -r response
    if [[ "$response" != "y" ]]; then
        echo "跳过基础镜像构建"
    else
        echo "重新构建基础镜像..."
        docker build -f Dockerfile.alpine-base -t $BASE_IMAGE . || {
            echo -e "${RED}❌ 基础镜像构建失败！${NC}"
            echo "可能的原因："
            echo "1. 网络连接问题（Alpine 软件源不可达）"
            echo "2. 依赖包下载失败"
            echo ""
            echo "解决方案："
            echo "1. 检查网络连接"
            echo "2. 尝试使用 VPN 或更换网络"
            echo "3. 稍后重试（Alpine CDN 可能临时不可用）"
            exit 1
        }
        echo -e "${GREEN}✅ 基础镜像构建成功！${NC}"
    fi
else
    echo "开始构建基础镜像..."
    docker build -f Dockerfile.alpine-base -t $BASE_IMAGE . || {
        echo -e "${RED}❌ 基础镜像构建失败！${NC}"
        echo "请检查网络连接后重试"
        exit 1
    }
    echo -e "${GREEN}✅ 基础镜像构建成功！${NC}"
fi

echo ""

# ============================================
# 阶段 2: 构建应用镜像
# ============================================
echo -e "${YELLOW}[阶段 2/2] 构建应用镜像...${NC}"
echo "这个镜像基于基础镜像，添加应用代码"
echo ""

APP_IMAGE="video-sync:v0.0.1"

docker build -f Dockerfile.alpine-app -t $APP_IMAGE . || {
    echo -e "${RED}❌ 应用镜像构建失败！${NC}"
    exit 1
}

echo -e "${GREEN}✅ 应用镜像构建成功！${NC}"
echo ""

# ============================================
# 构建完成，显示信息
# ============================================
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  ✅ 构建完成！${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""
echo "已创建镜像："
docker images | grep -E "video-sync-alpine-base|video-sync.*alpine" | head -5
echo ""

# 显示镜像大小对比
echo -e "${YELLOW}镜像大小对比：${NC}"
BASE_SIZE=$(docker images video-sync-alpine-base:latest --format "{{.Size}}")
APP_SIZE=$(docker images video-sync:alpine --format "{{.Size}}")
echo "  基础镜像: $BASE_SIZE"
echo "  应用镜像: $APP_SIZE"
echo ""

echo -e "${GREEN}下一步：${NC}"
echo "1. 启动服务："
echo "   docker-compose up -d"
echo ""
echo "2. 查看日志："
echo "   docker-compose logs -f"
echo ""
echo "3. 访问应用："
echo "   http://localhost:8080"
echo ""
