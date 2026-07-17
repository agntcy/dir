# syntax=docker/dockerfile:1@sha256:2780b5c3bab67f1f76c781860de469442999ed1a0d7992a5efdf2cffc0e3d769

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.9.0@sha256:c64defb9ed5a91eacb37f96ccc3d4cd72521c4bd18d5442905b95e2226b0e707 AS xx

FROM --platform=$BUILDPLATFORM golang:1.26.5-bookworm@sha256:e60d708a92ad26a6d61901334510d3debd23ddcba125663ecd6008d42e8ec669 AS builder

COPY --link --from=xx / /

ARG TARGETPLATFORM

RUN --mount=type=cache,id=${TARGETPLATFORM}-apt,target=/var/cache/apt,sharing=locked \
    apt-get update \
    && xx-apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    ca-certificates

WORKDIR /build/reconciler

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,source=.,target=/build,ro \
    xx-go mod download -x

ARG BUILD_OPTS
ARG BUILD_GCFLAGS
ARG BUILD_LDFLAGS

ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,source=.,target=/build,ro \
    xx-go build ${BUILD_OPTS} -gcflags="${BUILD_GCFLAGS}" -ldflags="${BUILD_LDFLAGS}" \
    -o /bin/reconciler .

RUN xx-verify /bin/reconciler

# Debug image - includes Delve for remote debugging. For local development only
FROM builder AS debug

WORKDIR /

COPY --from=builder /bin/reconciler ./reconciler

RUN xx-go install github.com/go-delve/delve/cmd/dlv@v1.26.2

ENTRYPOINT ["/go/bin/dlv", "--listen=0.0.0.0:2345", "--headless=true", "--accept-multiclient=true", "--api-version=2", "exec", "--continue", "./reconciler"]

# Runtime stage - using Busybox because regsync may need shell/tools
# and alpine has more vulnerabilities than busybox.
FROM busybox:1.38.0-musl@sha256:8635836765b0c4c43970660219739baa58b0883c2e429e4b8918f7dd1519455c

# Create non-root user matching distroless nonroot
RUN addgroup -g 55532 -S nonroot && adduser -u 55532 -S nonroot -G nonroot \
    && mkdir -p /home/nonroot && chown 55532:55532 /home/nonroot

WORKDIR /

COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /usr/share/ca-certificates /usr/share/ca-certificates

COPY --from=builder /bin/reconciler ./reconciler

USER 55532:55532

ENTRYPOINT ["./reconciler"]

# Coverage image - includes tar for kubectl cp to work
FROM busybox:1.38.0-musl@sha256:8635836765b0c4c43970660219739baa58b0883c2e429e4b8918f7dd1519455c AS coverage

WORKDIR /

COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /usr/share/ca-certificates /usr/share/ca-certificates

COPY --from=builder /bin/reconciler ./reconciler

# Create a non-root user for coverage
RUN addgroup -g 55532 -S nonroot && adduser -u 55532 -S nonroot -G nonroot \
    && mkdir -p /home/nonroot && chown 55532:55532 /home/nonroot

# Create coverage directory with proper permissions
RUN mkdir -p /tmp/coverage && chown -R 55532:55532 /tmp/coverage

USER 55532:55532

ENTRYPOINT ["./reconciler"]

FROM python:3.13-slim AS scanner

RUN groupadd -g 55532 -r nonroot && useradd -u 55532 -r -g nonroot nonroot \
    && mkdir -p /home/nonroot && chown 55532:55532 /home/nonroot

WORKDIR /

COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /usr/share/ca-certificates /usr/share/ca-certificates
COPY --from=builder /bin/reconciler ./reconciler

RUN apt-get update \
    && apt-get install -y --no-install-recommends git \
    && rm -rf /var/lib/apt/lists/*

ARG MCP_SCANNER_VERSION
ARG SKILL_SCANNER_VERSION
ARG A2A_SCANNER_VERSION
RUN pip install --no-cache-dir \
    cisco-ai-mcp-scanner==${MCP_SCANNER_VERSION} \
    cisco-ai-skill-scanner==${SKILL_SCANNER_VERSION} \
    cisco-ai-a2a-scanner==${A2A_SCANNER_VERSION}

USER 55532:55532

ENTRYPOINT ["./reconciler"]
