# syntax=docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.10.0@sha256:40fe21f44bb8cddf1eb1226db190d0a984118f62ee02a91b77d99448d28264a6 AS xx

FROM --platform=$BUILDPLATFORM golang:1.27.2-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61 AS builder

COPY --link --from=xx / /

ARG TARGETPLATFORM

RUN --mount=type=cache,id=${TARGETPLATFORM}-apt,target=/var/cache/apt,sharing=locked \
    apt-get update \
    && xx-apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev

WORKDIR /build/cli

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,source=.,target=/build,ro \
    xx-go mod download -x

ARG BUILD_OPTS
ARG BUILD_GCFLAGS
ARG BUILD_LDFLAGS

# TODO(adamtagscherer): Currently we don't need C libraries but in the future we may need to turn this on once we add
# security libraries, etc.
ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,source=.,target=/build,ro \
    xx-go build ${BUILD_OPTS} -gcflags="${BUILD_GCFLAGS}" -ldflags="${BUILD_LDFLAGS}" \
    -o /bin/dirctl ./cli.go

RUN xx-verify /bin/dirctl

# Runtime stage - Alpine rather than distroless so that documented workflows composing
# multiple dirctl calls (pipes, `--output json` into jq) are runnable with this image.
FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

RUN apk upgrade --no-cache zlib \
    && apk add --no-cache ca-certificates jq \
    && addgroup -g 65532 -S nonroot \
    && adduser -u 65532 -S nonroot -G nonroot -h /home/nonroot \
    && mkdir -p /home/nonroot && chown 65532:65532 /home/nonroot

WORKDIR /

COPY --from=builder /bin/dirctl /usr/local/bin/dirctl

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/dirctl"]
