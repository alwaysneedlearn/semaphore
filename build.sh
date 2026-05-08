#!/usr/bin/env bash
set -euo pipefail

BUILDER_NAME="${BUILDER_NAME:-semaphore-builder}"
CACHE_DIR="${CACHE_DIR:-.buildx-cache}"
IMAGE_TAG="${IMAGE_TAG:-semaphoreui/semaphore:local}"
DOCKERFILE_PATH="${DOCKERFILE_PATH:-deployment/docker/server/Dockerfile}"
TARGET_PLATFORM="${TARGET_PLATFORM:-linux/amd64}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker command not found in PATH" >&2
  exit 1
fi

mkdir -p "${CACHE_DIR}"

if ! docker buildx inspect "${BUILDER_NAME}" >/dev/null 2>&1; then
  docker buildx create --name "${BUILDER_NAME}" --use
else
  docker buildx use "${BUILDER_NAME}"
fi

docker buildx inspect --bootstrap >/dev/null

docker buildx build \
  --load \
  --pull=false \
  --platform "${TARGET_PLATFORM}" \
  --build-arg TARGETOS=linux \
  --build-arg TARGETARCH=amd64 \
  --build-arg INSTALL_IAC_TOOLS=false \
  --cache-from="type=local,src=${CACHE_DIR}" \
  --cache-to="type=local,dest=${CACHE_DIR},mode=max" \
  -f "${DOCKERFILE_PATH}" \
  -t "${IMAGE_TAG}" \
  .
