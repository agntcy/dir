#!/bin/bash

# This script deploys the Sigstore components to a Kubernetes cluster using Kind.
# Requirements:
#   - kind
#   - helm
#   - kubectl
#   - cosign

## KIND: Deploy cluster
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF

# Configure ingress
kubectl apply -f https://kind.sigs.k8s.io/examples/ingress/deploy-ingress-nginx.yaml
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s

## Set up Sigstore
helm repo add sigstore https://sigstore.github.io/helm-charts
helm upgrade \
 -i scaffold \
 sigstore/scaffold \
 -n sigstore \
 --create-namespace \
 --values ./scaffold.values.yaml

# Wait for sigstore to be ready
kubectl rollout status deployment scaffold-tuf-tuf -n tuf-system --timeout=300s

### Create TLS certificates for sigstore components
### TODO: generate certificates, add to trust store, configure explicit trust

# for service_name in rekor fulcio tuf; do
#     kubectl create secret tls $service_name-tls \
#         --namespace=$service_name-system \
#         --cert=./install/$service_name.signed.cert.pem \
#         --key=./install/$service_name.private.pem
# done
