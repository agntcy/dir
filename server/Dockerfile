# syntax=docker/dockerfile:1@sha256:fe40cf4e92cd0c467be2cfc30657a680ae2398318afd50b0c80585784c604f28

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.4.0@sha256:0cd3f05c72d6c9b038eb135f91376ee1169ef3a330d34e418e65e2a5c2e9c0d4 AS xx

FROM --platform=$BUILDPLATFORM golang:1.24.5-bullseye@sha256:254c0d1f13aad57bb210caa9e049deaee17ab7b8a976dba755cba1adf3fbe291 AS builder

COPY --link --from=xx / /

ARG TARGETPLATFORM

RUN --mount=type=cache,id=${TARGETPLATFORM}-apt,target=/var/cache/apt,sharing=locked \
    apt-get update \
    && xx-apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev

WORKDIR /build/server

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,source=.,target=/build,ro \
    xx-go mod download -x

ARG BUILD_OPTS
ARG EXTRA_LDFLAGS

# TODO(adamtagscherer): Currently we don't need C libraries but in the future we may need to turn this on once we add
# security libraries, etc.
ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,source=.,target=/build,ro \
    xx-go build ${BUILD_OPTS} -ldflags="-s -w -extldflags -static ${EXTRA_LDFLAGS}" \
    -o /bin/apiserver ./cmd/main.go

RUN xx-verify /bin/apiserver

FROM alpine:3.20@sha256:beefdbd8a1da6d2915566fde36db9db0b524eb737fc57cd1367effd16dc0d06d AS runtime

ARG COSIGN_VERSION=2.5.3
ARG TARGETPLATFORM

# Install cosign - using fixed version for now, will enhance later
RUN apk add --no-cache ca-certificates curl && \
    ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then COSIGN_ARCH="amd64"; elif [ "$ARCH" = "aarch64" ]; then COSIGN_ARCH="arm64"; else echo "Unsupported arch: $ARCH" && exit 1; fi && \
    COSIGN_URL="https://github.com/sigstore/cosign/releases/download/v${COSIGN_VERSION:-2.5.3}/cosign-linux-${COSIGN_ARCH}" && \
    echo "Downloading cosign from: $COSIGN_URL" && \
    curl -fsSL "$COSIGN_URL" -o /usr/local/bin/cosign && \
    chmod +x /usr/local/bin/cosign && \
    /usr/local/bin/cosign version || echo "Cosign installed but version check failed"

WORKDIR /

COPY --from=builder /bin/apiserver ./apiserver

# Create non-root user
RUN adduser -D -s /bin/sh -u 65532 nonroot
USER 65532:65532

ENTRYPOINT ["./apiserver", "run"]
