# syntax=docker/dockerfile:1@sha256:fe40cf4e92cd0c467be2cfc30657a680ae2398318afd50b0c80585784c604f28

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.4.0@sha256:0cd3f05c72d6c9b038eb135f91376ee1169ef3a330d34e418e65e2a5c2e9c0d4 AS xx

FROM --platform=$BUILDPLATFORM golang:1.25.2-bookworm@sha256:42d8e9dea06f23d0bfc908826455213ee7f3ed48c43e287a422064220c501be9 AS builder

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

# Production image - minimal distroless
FROM gcr.io/distroless/static:nonroot@sha256:c0f429e16b13e583da7e5a6ec20dd656d325d88e6819cafe0adb0828976529dc AS production

WORKDIR /

COPY --from=builder /bin/apiserver ./apiserver

USER 65532:65532

ENTRYPOINT ["./apiserver", "run"]

# Coverage image - includes tar for kubectl cp to work
FROM alpine:3.21@sha256:21dc6063fd678b478f57c0e13f47560d0ea4eeba26dfc947b2a4f81f686b9f45 AS coverage

RUN apk add --no-cache tar

WORKDIR /

COPY --from=builder /bin/apiserver ./apiserver

# Create a non-root user for coverage
RUN addgroup -g 65532 -S nonroot && adduser -u 65532 -S nonroot -G nonroot

# Create coverage directory with proper permissions
RUN mkdir -p /tmp/coverage && chown -R 65532:65532 /tmp/coverage

USER 65532:65532

ENTRYPOINT ["./apiserver", "run"]
