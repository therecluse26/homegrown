# ── Stage 1: Go binary build ─────────────────────────────────────────────────
FROM golang:1.25-alpine AS go-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/server \
    ./cmd/server/


# ── Stage 2: Frontend build ──────────────────────────────────────────────────
FROM node:22-alpine AS fe-builder

WORKDIR /src

# Install dependencies first for better layer caching
COPY frontend/package.json frontend/package-lock.json* frontend/pnpm-lock.yaml* ./
RUN npm install

COPY frontend/ .
RUN npm run build


# ── Stage 3: Runtime image ───────────────────────────────────────────────────
FROM nginx:alpine

# tini for proper signal handling / reaping of zombie child processes
RUN apk add --no-cache tini

WORKDIR /app

# Go binary and migrations (goose reads "migrations" relative to CWD at startup)
COPY --from=go-builder /out/server        ./server
COPY --from=go-builder /src/migrations    ./migrations

# Built SPA assets served by Nginx
COPY --from=fe-builder /src/dist          /usr/share/nginx/html

# Nginx vhost: serve SPA + proxy API routes to the Go process
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf

# Startup script that launches both nginx and the Go API
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 80

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/entrypoint.sh"]
