#!/usr/bin/env sh
# Build both container images. Usage: scripts/build-images.sh [tag]   (default: dev)
set -eu

TAG="${1:-dev}"
cd "$(dirname "$0")/.."

docker build -f deploy/docker/backend.Dockerfile  -t "connect4-backend:${TAG}"  .
docker build -f deploy/docker/frontend.Dockerfile -t "connect4-frontend:${TAG}" .

echo "Built connect4-backend:${TAG} and connect4-frontend:${TAG}"
