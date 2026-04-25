#!/bin/bash
# bash scripts/docker/docker-build.sh 0.2.2

set -e

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT_DIR=$(cd -- "$SCRIPT_DIR/../.." && pwd)
cd "$ROOT_DIR"

VERSION=${1:?'missing version, example: 0.0.1'}
IMAGE=${2:-witten888/video-sync}
TAR_NAME="vs${VERSION//.}.tar"

echo "=== Building ${IMAGE}:${VERSION} ==="
docker build \
    --build-arg VERSION="${VERSION}" \
    -f docker/Dockerfile \
    -t "${IMAGE}:${VERSION}" \
    -t "${IMAGE}:latest" \
    .

echo "=== Saving to ${TAR_NAME} ==="
docker save -o "${TAR_NAME}" "${IMAGE}:${VERSION}"

echo "=== Done: ${TAR_NAME} ==="
