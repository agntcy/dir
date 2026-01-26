# Watcher Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go.mod first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the watcher binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /watcher ./cmd/watcher

# Final image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /watcher /usr/local/bin/watcher

# Run as root to access Docker socket
# The socket is typically owned by root:docker with 660 permissions

ENTRYPOINT ["/usr/local/bin/watcher"]
