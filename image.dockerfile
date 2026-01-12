FROM alpine

RUN apk add --no-cache \
    curl \
    bash

# Install nerdctl
RUN curl -LO https://github.com/containerd/nerdctl/releases/download/v1.7.0/nerdctl-1.7.0-linux-amd64.tar.gz && \
    tar -xzf nerdctl-1.7.0-linux-amd64.tar.gz -C /usr/local/bin && \
    rm nerdctl-1.7.0-linux-amd64.tar.gz

ENV CONTAINERD_NAMESPACE=moby

CMD ["nsenter", "-t", "1", "-m", "-u", "-n", "-i", "sh"]
