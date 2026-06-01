# syntax=docker/dockerfile:1@sha256:2780b5c3bab67f1f76c781860de469442999ed1a0d7992a5efdf2cffc0e3d769

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.9.0@sha256:c64defb9ed5a91eacb37f96ccc3d4cd72521c4bd18d5442905b95e2226b0e707 AS xx

FROM --platform=$BUILDPLATFORM golang:1.26.3-bookworm@sha256:252599aeb51ad60b83e4d8821802068127c528c707cb7dd7afd93be057c6011c AS builder

COPY --link --from=xx / /

ARG TARGETPLATFORM

RUN --mount=type=cache,id=${TARGETPLATFORM}-apt,target=/var/cache/apt,sharing=locked \
    apt-get update \
    && xx-apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    ca-certificates

WORKDIR /build/server

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
    -o /bin/apiserver ./cmd/main.go

RUN xx-verify /bin/apiserver

# Debug image - includes Delve for remote debugging. For local development only
FROM builder AS debug

WORKDIR /

COPY --from=builder /bin/apiserver ./apiserver
COPY --from=ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.48 /ko-app/grpc-health-probe /bin/grpc-health-probe

RUN xx-go install github.com/go-delve/delve/cmd/dlv@v1.26.2

ENTRYPOINT ["/go/bin/dlv", "--listen=0.0.0.0:2345", "--headless=true", "--accept-multiclient=true", "--api-version=2", "exec", "--continue", "./apiserver", "--", "run"]

# Production image - minimal distroless
FROM gcr.io/distroless/static:nonroot@sha256:963fa6c544fe5ce420f1f54fb88b6fb01479f054c8056d0f74cc2c6000df5240 AS production

WORKDIR /

COPY --from=builder /bin/apiserver ./apiserver
COPY --from=ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.48 /ko-app/grpc-health-probe /bin/grpc-health-probe

USER 65532:65532

ENTRYPOINT ["./apiserver", "run"]

# Coverage image - includes tar for kubectl cp to work
FROM busybox:1.38.0-musl@sha256:8635836765b0c4c43970660219739baa58b0883c2e429e4b8918f7dd1519455c AS coverage

WORKDIR /

COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /usr/share/ca-certificates /usr/share/ca-certificates

COPY --from=builder /bin/apiserver ./apiserver
COPY --from=ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.48 /ko-app/grpc-health-probe /bin/grpc-health-probe

# Create a non-root user for coverage
RUN addgroup -g 55532 -S nonroot && adduser -u 55532 -S nonroot -G nonroot

# Create coverage directory with proper permissions
RUN mkdir -p /tmp/coverage && chown -R 55532:55532 /tmp/coverage

USER 55532:55532

ENTRYPOINT ["./apiserver", "run"]
