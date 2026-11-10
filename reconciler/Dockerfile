# syntax=docker/dockerfile:1@sha256:4edf897a3ffa55b89f906fc8cc78afdb3f1834cc9c7083565e611a8a7d5fe99e

# xx is a helper for cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.10.0@sha256:40fe21f44bb8cddf1eb1226db190d0a984118f62ee02a91b77d99448d28264a6 AS xx

FROM --platform=$BUILDPLATFORM golang:1.27.3-bookworm@sha256:dc0963f32053e4bc451d24d87e3892d42d01021b61f093e98eb95c0352bb84b0 AS builder

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

# cel-helper rebuilds skill-scanner's cel-go decision helper with the pinned Go
# toolchain. The PyPI wheel bundles a helper compiled with an older Go (go1.27.1
# for 2.3.2) that carries fixed stdlib CVEs; skill-scanner honours
# SKILL_SCANNER_CEL_GO_HELPER for a trusted local build, so the scanner stage
# swaps the bundled binary for this one.
FROM --platform=$BUILDPLATFORM golang:1.27.3-bookworm@sha256:dc0963f32053e4bc451d24d87e3892d42d01021b61f093e98eb95c0352bb84b0 AS cel-helper

COPY --link --from=xx / /

ARG TARGETPLATFORM
ARG SKILL_SCANNER_VERSION

ENV CGO_ENABLED=0

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    git clone --quiet --depth 1 --branch "${SKILL_SCANNER_VERSION}" \
    https://github.com/cisco-ai-defense/skill-scanner.git /src \
    && cd /src/tools/cel_runtime \
    && xx-go build -trimpath -buildvcs=false \
    -ldflags="-X main.helperVersion=${SKILL_SCANNER_VERSION}" \
    -o /bin/skill-scanner-cel-go . \
    && xx-verify /bin/skill-scanner-cel-go

# Debug image - includes Delve for remote debugging. For local development only
FROM builder AS debug

WORKDIR /

COPY --from=builder /bin/reconciler ./reconciler

RUN xx-go install github.com/go-delve/delve/cmd/dlv@v1.26.2

ENTRYPOINT ["/go/bin/dlv", "--listen=0.0.0.0:2345", "--headless=true", "--accept-multiclient=true", "--api-version=2", "exec", "--continue", "./reconciler"]

# Runtime stage - using Busybox because regsync may need shell/tools
# and alpine has more vulnerabilities than busybox.
FROM busybox:1.38.0-musl@sha256:ea2b9914a16a4ac1981994af97b318f7c7d4db76b580c56177f08bf76f4a0be8

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
FROM busybox:1.38.0-musl@sha256:ea2b9914a16a4ac1981994af97b318f7c7d4db76b580c56177f08bf76f4a0be8 AS coverage

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

FROM python:3.13-slim@sha256:70729b46c69b4f1e97c4822c1af3df53a1476cf5ddc6c087c0c10bc3a5678c2f AS scanner

RUN groupadd -g 55532 -r nonroot && useradd -u 55532 -r -g nonroot nonroot \
    && mkdir -p /home/nonroot && chown 55532:55532 /home/nonroot

WORKDIR /

COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /usr/share/ca-certificates /usr/share/ca-certificates
COPY --from=builder /bin/reconciler ./reconciler

# The base image lags behind Debian security updates.
RUN apt-get update \
    && apt-get upgrade -y --no-install-recommends \
    && apt-get install -y --no-install-recommends git \
    && rm -rf /var/lib/apt/lists/*

ARG MCP_SCANNER_VERSION
ARG SKILL_SCANNER_VERSION
ARG A2A_SCANNER_VERSION
# mcp-scanner <= 4.8.7 pins litellm==1.93.2 (CVE-2026-114612, GHSA-g5ff-637f-6q2m).
# The pin is relaxed to a floor upstream (cisco-ai-defense/mcp-scanner#289) but
# not released yet, so install the patched litellm after the resolver has run.
# Drop this override once a mcp-scanner release ships the relaxed pin.
# renovate: datasource=pypi depName=litellm versioning=pep440
ARG LITELLM_VERSION=1.108.4
# pip vendors CVE-carrying msgpack and setuptools copies and nothing needs it
# at runtime, so drop it and its ensurepip wheel in the install layer.
# The skill-scanner wheel's prebuilt cel-go helper is removed in favour of the
# cel-helper stage build selected via SKILL_SCANNER_CEL_GO_HELPER below.
RUN pip install --no-cache-dir \
    cisco-ai-mcp-scanner==${MCP_SCANNER_VERSION} \
    cisco-ai-skill-scanner==${SKILL_SCANNER_VERSION} \
    cisco-ai-a2a-scanner==${A2A_SCANNER_VERSION} \
    && pip install --no-cache-dir litellm==${LITELLM_VERSION} \
    && python -m pip uninstall -y pip \
    && rm -rf /usr/local/lib/python3.*/ensurepip/_bundled \
    && rm -f /usr/local/lib/python3.*/site-packages/skill_scanner/core/cel/_bin/skill-scanner-cel-go-*

COPY --from=cel-helper /bin/skill-scanner-cel-go /usr/local/bin/skill-scanner-cel-go
ENV SKILL_SCANNER_CEL_GO_HELPER=/usr/local/bin/skill-scanner-cel-go

USER 55532:55532

ENTRYPOINT ["./reconciler"]
