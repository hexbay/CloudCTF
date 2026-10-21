# Build context MUST be the repository root.

# --- Stage 1: build the unified frontend (admin + player SPA) ---
FROM node:22-alpine AS frontend
WORKDIR /web
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# --- Stage 2: build the Go binaries with the frontend embedded ---
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Place the frontend build where //go:embed all:dist expects it.
COPY --from=frontend /web/dist ./internal/web/dist
RUN go build -tags embed -o /out/cloudctf-api ./cmd/server

# --- Stage 3: minimal runtime image ---
FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/cloudctf-api /usr/local/bin/cloudctf-api
EXPOSE 8005
CMD ["cloudctf-api"]

