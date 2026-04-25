#!/bin/bash
set -e

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT_DIR=$(cd -- "$SCRIPT_DIR/../.." && pwd)
cd "$ROOT_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  Alpine Docker build${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""

echo -e "${YELLOW}[1/2] Build base image...${NC}"
echo ""

BASE_IMAGE_NAME="video-sync-alpine-base"
BASE_IMAGE_TAG="${BASE_IMAGE_TAG:-latest}"
BASE_IMAGE="${BASE_IMAGE_NAME}:${BASE_IMAGE_TAG}"

if docker images | grep -q "video-sync-alpine-base"; then
    echo -e "${YELLOW}Base image already exists, rebuild? (y/n)${NC}"
    read -r response
    if [[ "$response" != "y" ]]; then
        echo "Skip base image build"
    else
        echo "Rebuilding base image..."
        docker build -f docker/Dockerfile.alpine-base -t "$BASE_IMAGE" . || {
            echo -e "${RED}Base image build failed${NC}"
            exit 1
        }
        echo -e "${GREEN}Base image build complete${NC}"
    fi
else
    echo "Building base image..."
    docker build -f docker/Dockerfile.alpine-base -t "$BASE_IMAGE" . || {
        echo -e "${RED}Base image build failed${NC}"
        exit 1
    }
    echo -e "${GREEN}Base image build complete${NC}"
fi

echo ""
echo -e "${YELLOW}[2/2] Build app image...${NC}"
echo ""

APP_IMAGE="video-sync:v0.0.1"

docker build \
    --build-arg BASE_IMAGE_NAME="$BASE_IMAGE_NAME" \
    --build-arg BASE_IMAGE_TAG="$BASE_IMAGE_TAG" \
    -f docker/Dockerfile \
    -t "$APP_IMAGE" . || {
    echo -e "${RED}App image build failed${NC}"
    exit 1
}

echo -e "${GREEN}App image build complete${NC}"
echo ""
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  Build complete${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""
docker images | grep -E "video-sync-alpine-base|video-sync" | head -5
