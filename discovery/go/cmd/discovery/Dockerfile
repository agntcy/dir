# Discovery Service Dockerfile
# Combines watcher and inspector into a single component (writes to etcd)
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go.mod first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the discovery binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /discovery ./cmd/discovery

# Final image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /discovery /usr/local/bin/discovery

# Run as root to access Docker socket (for docker runtime)
# For kubernetes runtime, can run as non-root

ENTRYPOINT ["/usr/local/bin/discovery"]
