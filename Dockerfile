# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app
COPY backend/go.* ./backend/
WORKDIR /app/backend
RUN go mod download

# Copy built frontend into embedded static path
COPY backend/ /app/backend/
COPY --from=frontend-builder /app/frontend/dist /app/backend/web/dist

# Build statically linked standalone binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /clickhouse-manager ./cmd/server

# Stage 3: Minimal Production Runner
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    procps \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/clickhouse-manager
COPY --from=backend-builder /clickhouse-manager /opt/clickhouse-manager/clickhouse-manager

ENV PORT=8080
EXPOSE 8080

VOLUME ["/opt/clickhouse-manager/data"]

ENTRYPOINT ["/opt/clickhouse-manager/clickhouse-manager"]
CMD ["-port", "8080", "-db", "/opt/clickhouse-manager/data/clickhouse-manager.db"]
