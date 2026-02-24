# syntax=docker/dockerfile:1@sha256:fe40cf4e92cd0c467be2cfc30657a680ae2398318afd50b0c80585784c604f28

# Regsync version
ARG REGSYNC_VERSION=v0.11.1

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.9.0@sha256:c64defb9ed5a91eacb37f96ccc3d4cd72521c4bd18d5442905b95e2226b0e707 AS xx

FROM --platform=$BUILDPLATFORM golang:1.26.0-bookworm@sha256:eae3cdfa040d0786510a5959d36a836978724d03b34a166ba2e0e198baac9196 AS builder

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

# Runtime stage - using Alpine because regsync may need shell/tools
# https://github.com/docker-library/repo-info/blob/master/repos/alpine/tag-details.md
FROM alpine:3.21@sha256:c3f8e73fdb79deaebaa2037150150191b9dcbfba68b4a46d70103204c53f4709

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user matching distroless nonroot
RUN addgroup -g 65532 -S nonroot && adduser -u 65532 -S nonroot -G nonroot

WORKDIR /

COPY --from=builder /bin/reconciler ./reconciler
COPY --from=builder /bin/regsync /usr/local/bin/regsync

USER 65532:65532

ENTRYPOINT ["./reconciler"]

# Coverage image - includes tar for kubectl cp to work
FROM alpine:3.21@sha256:21dc6063fd678b478f57c0e13f47560d0ea4eeba26dfc947b2a4f81f686b9f45 AS coverage

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
