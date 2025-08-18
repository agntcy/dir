#!/bin/bash

# Deploy cluster
task deploy:k8s:spire:cleanup
task deploy:k8s:spire:cluster

# Deploy server with SPIRE
export SPIRE_NAMESPACE="server-spire"
export SPIRE_CLUSTER_NAME="server-cluster"
export SPIRE_TRUST_DOMAIN="server-cluster.com"
export SPIRE_CLASS_NAME="server-spire"
export SPIRE_CSI_DRIVER="server.spiffe.io"
task deploy:k8s:spire:chart
task deploy:k8s:spire:dir

# Deploy client with SPIRE
export SPIRE_NAMESPACE="client-spire"
export SPIRE_CLUSTER_NAME="client-cluster"
export SPIRE_TRUST_DOMAIN="client-cluster.com"
export SPIRE_CLASS_NAME="client-spire"
export SPIRE_CSI_DRIVER="client.spiffe.io"
task deploy:k8s:spire:chart
task deploy:k8s:spire:dirctl

# Check logs
task logs:k8s:spire:dir
task logs:k8s:spire:dirctl
