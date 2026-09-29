#!/usr/bin/env bash
# ==============================================================================
# ClickHouse Manager - Full Build Script (Frontend + Backend Embed)
# ==============================================================================
set -e

echo "[1/4] Building Frontend..."
cd frontend
npm install
npm run build
cd ..

echo "[2/4] Syncing frontend dist to backend/web/dist..."
rm -rf backend/web/dist
mkdir -p backend/web/dist
cp -r frontend/dist/* backend/web/dist/

echo "[3/4] Building Go Backend with embedded assets..."
cd backend
go mod tidy

# Native build
go build -ldflags="-s -w" -o clickhouse-manager ./cmd/server

# Linux amd64 cross-compile (if needed)
if [ "$1" == "release" ]; then
    echo "Cross-compiling for Linux amd64..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o clickhouse-manager-linux-amd64 ./cmd/server
    echo "Cross-compiling for Linux arm64..."
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o clickhouse-manager-linux-arm64 ./cmd/server
fi

cd ..

echo "[4/4] Build complete! Single executable is located at backend/clickhouse-manager"
