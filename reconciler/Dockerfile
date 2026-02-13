# syntax=docker/dockerfile:1@sha256:fe40cf4e92cd0c467be2cfc30657a680ae2398318afd50b0c80585784c604f28

# Regsync version
ARG REGSYNC_VERSION=v0.11.1

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.9.0@sha256:c64defb9ed5a91eacb37f96ccc3d4cd72521c4bd18d5442905b95e2226b0e707 AS xx

FROM --platform=$BUILDPLATFORM golang:1.25.7-bookworm@sha256:38342f3e7a504bf1efad858c18e771f84b66dc0b363add7a57c9a0bbb6cf7b12 AS builder

COPY --link --from=xx / /

ARG TARGETPLATFORM

RUN --mount=type=cache,id=${TARGETPLATFORM}-apt,target=/var/cache/apt,sharing=locked \
    apt-get update \
    && xx-apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev

WORKDIR /build/reconciler

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,source=.,target=/build,ro \
    xx-go mod download -x

ARG BUILD_OPTS
ARG EXTRA_LDFLAGS

ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,source=.,target=/build,ro \
    xx-go build ${BUILD_OPTS} -ldflags="-s -w -extldflags -static ${EXTRA_LDFLAGS}" \
    -o /bin/reconciler .

RUN xx-verify /bin/reconciler

# Download regsync binary
FROM ghcr.io/regclient/regsync:${REGSYNC_VERSION} AS regsync

# Runtime stage - using Alpine because regsync may need shell/tools
FROM alpine:3.21@sha256:21dc6063fd678b478f57c0e13f47560d0ea4eeba26dfc947b2a4f81f686b9f45

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user matching distroless nonroot
RUN addgroup -g 65532 -S nonroot && adduser -u 65532 -S nonroot -G nonroot

WORKDIR /

COPY --from=builder /bin/reconciler ./reconciler
COPY --from=regsync /regsync /usr/local/bin/regsync

# Create config directories
RUN mkdir -p /etc/agntcy/reconciler /etc/regsync && \
    chown -R 65532:65532 /etc/agntcy /etc/regsync

USER 65532:65532

ENTRYPOINT ["./reconciler"]
