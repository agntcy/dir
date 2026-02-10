ARG IMAGE_TAG=latest
ARG IMAGE_REPO=ghcr.io/agntcy

FROM ${IMAGE_REPO}/dir-ctl:${IMAGE_TAG} AS dirctl-bin
FROM ghcr.io/sigstore/cosign/cosign:v3.0.4 AS cosign-bin

FROM astral/uv:0.9.26-python3.12-alpine

COPY --from=dirctl-bin /dirctl /bin/dirctl
COPY --from=cosign-bin /ko-app/cosign /bin/cosign

ENV DIRCTL_PATH="/bin/dirctl"
ENV DIRECTORY_CLIENT_SERVER_ADDRESS="dir-apiserver.dir-server.svc.cluster.local:8888"
ENV DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH=""
ENV DIRECTORY_CLIENT_AUTH_MODE="x509"
ENV DIRECTORY_CLIENT_JWT_AUDIENCE="spiffe://dir.example/spire/server"
ENV DIRECTORY_LOGGER_LOG_FILE="/dev/stderr"

WORKDIR /tmp/
COPY ./sdk /tmp/

WORKDIR /tmp/dir-py

RUN uv sync --all-packages

RUN apk add --update --no-cache nodejs npm

WORKDIR /tmp/dir-js
RUN npm install

WORKDIR /tmp

RUN printf "#!/bin/sh\n\ncd ./dir-py && uv run pytest\npy_status=\$?\ncd ..\ncd ./dir-js && npm run test\njs_status=\$?\n\nif [ \$py_status -ne 0 ] || [ \$js_status -ne 0 ]; then\n  exit 1\nfi" >> entrypoint.sh && chmod +x entrypoint.sh


ENTRYPOINT [ "/tmp/entrypoint.sh" ]
