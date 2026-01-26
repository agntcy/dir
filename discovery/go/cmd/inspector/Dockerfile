# Inspector Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go.mod first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the inspector binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /inspector ./cmd/inspector

# Final image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /inspector /usr/local/bin/inspector

# Non-root user for security
RUN adduser -D -g '' appuser
USER appuser

ENTRYPOINT ["/usr/local/bin/inspector"]
