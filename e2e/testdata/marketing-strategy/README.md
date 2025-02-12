# Multi-Agent Marketing Strategy Example

Example with multiple agent running in different processes and communicating via IPC.

## Prerequisites

### Ollama

Run Ollama locally and make it available on all the interfaces (not only to localhost)

```bash
# Make sure to install the formula and not the cask.
brew install ollama
brew services start ollama
launchctl setenv OLLAMA_HOST "0.0.0.0"
brew services restart ollama
```

### Kind

[Install kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installing-from-release-binaries) and [run a local cluster with a local registry](https://kind.sigs.k8s.io/docs/user/local-registry/) (assuming you have docker desktop available).

```bash
#!/bin/sh
set -o errexit

brew install kind

# 1. Create registry container unless it already exists
reg_name='kind-registry'
reg_port='5001'
if [ "$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)" != 'true' ]; then
  docker run \
    -d --restart=always -p "127.0.0.1:${reg_port}:5000" --network bridge --name "${reg_name}" \
    registry:2
fi

# 2. Create kind cluster with containerd registry config dir enabled
# TODO: kind will eventually enable this by default and this patch will
# be unnecessary.
#
# See:
# https://github.com/kubernetes-sigs/kind/issues/2875
# https://github.com/containerd/containerd/blob/main/docs/cri/config.md#registry-configuration
# See: https://github.com/containerd/containerd/blob/main/docs/hosts.md
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry]
    config_path = "/etc/containerd/certs.d"
EOF

# 3. Add the registry config to the nodes
#
# This is necessary because localhost resolves to loopback addresses that are
# network-namespace local.
# In other words: localhost in the container is not localhost on the host.
#
# We want a consistent name that works from both ends, so we tell containerd to
# alias localhost:${reg_port} to the registry container when pulling images
REGISTRY_DIR="/etc/containerd/certs.d/localhost:${reg_port}"
for node in $(kind get nodes); do
  docker exec "${node}" mkdir -p "${REGISTRY_DIR}"
  cat <<EOF | docker exec -i "${node}" cp /dev/stdin "${REGISTRY_DIR}/hosts.toml"
[host."http://${reg_name}:5000"]
EOF
done

# 4. Connect the registry to the cluster network if not already connected
# This allows kind to bootstrap the network but ensures they're on the same network
if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${reg_name}")" = 'null' ]; then
  docker network connect "kind" "${reg_name}"
fi

# 5. Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

```

### Argo workflow

Once kind is installed, install argo workflow as workflow scheduler for out multiagent application:

```bash
ARGO_WORKFLOWS_VERSION="v3.5.10"

kubectl create namespace argo
kubectl apply -n argo -f "https://github.com/argoproj/argo-workflows/releases/download/${ARGO_WORKFLOWS_VERSION}/quick-start-minimal.yaml"


# We now give to the devault role in the argo namespace the admin privileges to perform operations with the kube api server.
# Never do this in production cluster!!!
kubectl create rolebinding default-admin --clusterrole=admin --serviceaccount=argo:default -n argo

# Also install the argo cli (it is nice)
brew install argo
```

You are all set!

## Build the docker containers and push it to local registry

```bash
docker build -t localhost:5001/msm:1 -f Dockerfile .
docker push localhost:5001/msm:1
```

## Set telemetry

To enable telemetry from the app you need to set the right OTel endpoint.
Check the TELEMETRY_ENDPOINT in the argoworkflow.yaml file. If this env variable
is not set, the telemetry is disabled.

## Run
Now we have one container with the 3 agents inside. Now we can run the argo workflow:

```bash
LOCAL_ADDRESS=$(ipconfig getifaddr en0)
sed -E -i "s,http://[0-9]+.[0-9]+.[0-9]+.[0-9]+:11434/v1,http://${LOCAL_ADDRESS}:11434/v1,g" workflow.yaml
# For MacOS using the built-in sed, an empty backup file path is also needed
# sed -E -i '' "s,http://192.168.1.111:11434/v1,http://${LOCAL_ADDRESS}:11434/v1,g" workflow.yaml

argo submit workflow.yaml -n argo --wait
```

Example of output:

```
Name:                marketing-strategy-7cknw
Namespace:           argo
ServiceAccount:      unset (will run with the default ServiceAccount)
Status:              Pending
Created:             Tue Sep 10 20:19:11 +0200 (now)
Progress:
marketing-strategy-7cknw Succeeded at 2024-09-10 20:35:59 +0200 CEST
❯ git status
On branch tiger-team
Your branch is up to date with 'origin/tiger-team'.
```
