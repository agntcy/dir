ARG IMAGE_TAG=latest

FROM ghcr.io/agntcy/dir-ctl:${IMAGE_TAG} AS dirctl-bin
FROM ghcr.io/sigstore/cosign/cosign:v2.4.1 AS cosign-bin

FROM ghcr.io/astral-sh/uv:python3.13-alpine

COPY --from=dirctl-bin /dirctl /bin/dirctl
COPY --from=cosign-bin /ko-app/cosign /bin/cosign

ENV DIRCTL_PATH="/bin/dirctl"
ENV DIRECTORY_CLIENT_SERVER_ADDRESS="dir-apiserver.dir-server.svc.cluster.local:8888"
ENV DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH=""

WORKDIR /tmp/
COPY ./sdk /tmp/

WORKDIR /tmp/dir-py

RUN uv sync

RUN apk add --update --no-cache nodejs npm

WORKDIR /tmp/dir-js
RUN npm install

WORKDIR /tmp

RUN printf "#!/bin/sh\n\n cd ./dir-py && uv run pytest\n cd .. \n cd ./dir-js/ && npm run test" >> entrypoint.sh && chmod +x entrypoint.sh

ENTRYPOINT [ "/tmp/entrypoint.sh" ]
