# stage 1: 前端静态导出
FROM node:22-alpine AS web
WORKDIR /web
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# stage 2: Go 二进制
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/cli ./cmd/cli

# stage 3: 运行时（含与 PostgreSQL 16 匹配的 pg_dump/pg_restore）
FROM postgres:16-bookworm
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl tzdata && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/server /out/cli ./
COPY --from=web /web/out ./web
ENV APP_HOST=0.0.0.0 APP_PORT=17860 DATA_DIR=/data GIN_MODE=release
EXPOSE 17860
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD curl -fsS http://127.0.0.1:17860/api/v1/health || exit 1
ENTRYPOINT ["/app/server"]
