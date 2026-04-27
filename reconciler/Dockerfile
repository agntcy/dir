# syntax=docker/dockerfile:1@sha256:2780b5c3bab67f1f76c781860de469442999ed1a0d7992a5efdf2cffc0e3d769

# Regsync version
ARG REGSYNC_VERSION=v0.11.1

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.9.0@sha256:c64defb9ed5a91eacb37f96ccc3d4cd72521c4bd18d5442905b95e2226b0e707 AS xx

FROM --platform=$BUILDPLATFORM golang:1.26.2-bookworm@sha256:47ce5636e9936b2c5cbf708925578ef386b4f8872aec74a67bd13a627d242b19 AS builder

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

# Build regsync from source using the same Go version.
# This ensures we have a statically linked binary compatible with distroless
# and avoids security issues with different Go versions in prebuilt binaries.
ARG REGSYNC_VERSION

RUN git clone --depth 1 --branch ${REGSYNC_VERSION} \
    https://github.com/regclient/regclient.git /tmp/regclient

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,source=.,target=/build,ro \
    xx-go -C /tmp/regclient build -o /bin/regsync ./cmd/regsync

RUN xx-verify /bin/regsync

# Debug image - includes Delve for remote debugging. For local development only
FROM builder AS debug

WORKDIR /

COPY --from=builder /bin/reconciler ./reconciler
COPY --from=builder /bin/regsync /usr/local/bin/regsync

RUN xx-go install github.com/go-delve/delve/cmd/dlv@v1.26.2

ENTRYPOINT ["/go/bin/dlv", "--listen=0.0.0.0:2345", "--headless=true", "--accept-multiclient=true", "--api-version=2", "exec", "--continue", "./reconciler"]

# Runtime stage - using Busybox because regsync may need shell/tools
# and alpine has more vulnerabilities than busybox.
FROM busybox:1.37.0-musl@sha256:19b646668802469d968a05342a601e78da4322a414a7c09b1c9ee25165042138

# Create non-root user matching distroless nonroot
RUN addgroup -g 55532 -S nonroot && adduser -u 55532 -S nonroot -G nonroot

WORKDIR /

COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /usr/share/ca-certificates /usr/share/ca-certificates

COPY --from=builder /bin/reconciler ./reconciler
COPY --from=builder /bin/regsync /usr/local/bin/regsync

USER 55532:55532

ENTRYPOINT ["./reconciler"]

# Coverage image - includes tar for kubectl cp to work
FROM busybox:1.37.0-musl@sha256:19b646668802469d968a05342a601e78da4322a414a7c09b1c9ee25165042138 AS coverage

WORKDIR /

COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /usr/share/ca-certificates /usr/share/ca-certificates

COPY --from=builder /bin/reconciler ./reconciler
COPY --from=builder /bin/regsync /usr/local/bin/regsync

# Create a non-root user for coverage
RUN addgroup -g 55532 -S nonroot && adduser -u 55532 -S nonroot -G nonroot

# Create coverage directory with proper permissions
RUN mkdir -p /tmp/coverage && chown -R 55532:55532 /tmp/coverage

USER 55532:55532

ENTRYPOINT ["./reconciler"]
