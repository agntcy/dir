# syntax=docker/dockerfile:1@sha256:4a43a54dd1fedceb30ba47e76cfcf2b47304f4161c0caeac2db1c61804ea3c91

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
FROM gcr.io/distroless/static:nonroot@sha256:e3f945647ffb95b5839c07038d64f9811adf17308b9121d8a2b87b6a22a80a39 AS production

WORKDIR /

COPY --from=builder /bin/apiserver ./apiserver
COPY --from=ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.48 /ko-app/grpc-health-probe /bin/grpc-health-probe

USER 65532:65532

ENTRYPOINT ["./apiserver", "run"]

# Coverage image - includes tar for kubectl cp to work
FROM alpine:3.23@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659 AS coverage

RUN apk add --no-cache tar

WORKDIR /

COPY --from=builder /bin/apiserver ./apiserver
COPY --from=ghcr.io/grpc-ecosystem/grpc-health-probe:v0.4.48 /ko-app/grpc-health-probe /bin/grpc-health-probe

# Create a non-root user for coverage
RUN addgroup -g 65532 -S nonroot && adduser -u 65532 -S nonroot -G nonroot

# Create coverage directory with proper permissions
RUN mkdir -p /tmp/coverage && chown -R 65532:65532 /tmp/coverage

USER 65532:65532

ENTRYPOINT ["./apiserver", "run"]
