# syntax=docker/dockerfile:1@sha256:2780b5c3bab67f1f76c781860de469442999ed1a0d7992a5efdf2cffc0e3d769

# Regsync version
ARG REGSYNC_VERSION=v0.11.1

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.9.0@sha256:c64defb9ed5a91eacb37f96ccc3d4cd72521c4bd18d5442905b95e2226b0e707 AS xx

FROM --platform=$BUILDPLATFORM golang:1.26.2-bookworm@sha256:4f4ab2c90005e7e63cb631f0b4427f05422f241622ee3ec4727cc5febbf83e34 AS builder

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

# Runtime stage - using Alpine because regsync may need shell/tools
# https://github.com/docker-library/repo-info/blob/master/repos/alpine/tag-details.md
FROM alpine:3.23@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user matching distroless nonroot
RUN addgroup -g 65532 -S nonroot && adduser -u 65532 -S nonroot -G nonroot

WORKDIR /

COPY --from=builder /bin/reconciler ./reconciler
COPY --from=builder /bin/regsync /usr/local/bin/regsync

USER 65532:65532

ENTRYPOINT ["./reconciler"]

# Coverage image - includes tar for kubectl cp to work
FROM alpine:3.23@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659 AS coverage

RUN apk add --no-cache tar

WORKDIR /

COPY --from=builder /bin/reconciler ./reconciler
COPY --from=builder /bin/regsync /usr/local/bin/regsync

# Create a non-root user for coverage
RUN addgroup -g 65532 -S nonroot && adduser -u 65532 -S nonroot -G nonroot

# Create coverage directory with proper permissions
RUN mkdir -p /tmp/coverage && chown -R 65532:65532 /tmp/coverage

USER 65532:65532

ENTRYPOINT ["./reconciler"]
