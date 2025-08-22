#!/bin/bash

export CREATE_CLUSTER=${CREATE_CLUSTER:-}

export DIR_TRUST_DOMAIN="dir.example"
export DIRCTL_TRUST_DOMAIN="dirctl.example"

##################################################################
###################### SERVER CLUSTER
##################################################################

export SPIRE_TRUST_DOMAIN=$DIR_TRUST_DOMAIN
export KIND_CLUSTER_NAME=$DIR_TRUST_DOMAIN
export KIND_CREATE_OPTS="--config /tmp/kind-config-server.yaml"
export HELM_EXTRA_ARGS="-f /tmp/server-federation.yaml"

###
# 1. CLUSTER DEPLOYMENT
###
if [[ "$CREATE_CLUSTER" == "true" ]]; then

# Define cluster config
cat <<EOF > /tmp/kind-config-server.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
EOF

# Deploy cluster with SPIRE
task deploy:k8s:spire:cleanup
task deploy:k8s:spire:cluster
task deploy:k8s:spire:chart

# Patch spire service to expose federation nodeport federation (ID 1)
# kubectl patch svc -n spire spire-server -p '{"spec":{"ports":[{"name":"federation","port":8443,"targetPort":"federation","nodePort":30950,"protocol":"TCP"}]}}'

# Export federation bundle
kubectl get configmap -n spire spire-bundle -o 'go-template={{index .data "bundle.spiffe"}}' > /tmp/$DIR_TRUST_DOMAIN.spiffe

fi

###
# 2. CHART DEPLOYMENT
###
if [[ "$CREATE_CLUSTER" == "false" ]]; then

# Define federation config
cat <<EOF > /tmp/server-federation.yaml
apiserver:
  service:
    type: LoadBalancer

  spire:
    enabled: true
    trustDomain: $DIR_TRUST_DOMAIN
    federation:
      - trustDomain: $DIRCTL_TRUST_DOMAIN
        bundleEndpointURL: https://172.18.0.4:8443
        bundleEndpointProfile:
          type: https_spiffe
          endpointSPIFFEID: spiffe://$DIRCTL_TRUST_DOMAIN/spire/server
        trustDomainBundle: |-
          $(cat /tmp/$DIRCTL_TRUST_DOMAIN.spiffe)
EOF

# Deploy server
task deploy:k8s:context
task deploy:k8s:spire:dir

fi

##################################################################
###################### CLIENT CLUSTER
##################################################################

export SPIRE_TRUST_DOMAIN=$DIRCTL_TRUST_DOMAIN
export KIND_CLUSTER_NAME=$DIRCTL_TRUST_DOMAIN
export KIND_CREATE_OPTS="--config /tmp/kind-config-client.yaml"
export HELM_EXTRA_ARGS="-f /tmp/client-federation.yaml"

###
# 1. CLUSTER DEPLOYMENT
###
if [[ "$CREATE_CLUSTER" == "true" ]]; then

# Define cluster config
cat <<EOF > /tmp/kind-config-client.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
EOF

# Create cluster with SPIRE
task deploy:k8s:spire:cleanup
task deploy:k8s:spire:cluster
task deploy:k8s:spire:chart

# Patch spire service to expose federation nodeport federation (ID 1)
# kubectl patch svc -n spire spire-server -p '{"spec":{"ports":[{"name":"federation","port":8443,"targetPort":"federation","nodePort":30950,"protocol":"TCP"}]}}'

# Explort federation bundle
kubectl get configmap -n spire spire-bundle -o 'go-template={{index .data "bundle.spiffe"}}' > /tmp/$DIRCTL_TRUST_DOMAIN.spiffe

fi

###
# 2. CHART DEPLOYMENT
###
if [[ "$CREATE_CLUSTER" == "false" ]]; then

# Define federation config
cat <<EOF > /tmp/client-federation.yaml
env:
  - name: DIRECTORY_CLIENT_SERVER_ADDRESS
    value: 172.18.0.6:8888

spire:
  enabled: true
  trustDomain: $DIRCTL_TRUST_DOMAIN
  federation:
    - trustDomain: $DIR_TRUST_DOMAIN
      bundleEndpointURL: https://172.18.0.5:8443
      bundleEndpointProfile:
        type: https_spiffe
        endpointSPIFFEID: spiffe://$DIR_TRUST_DOMAIN/spire/server
      trustDomainBundle: |-
        $(cat /tmp/$DIR_TRUST_DOMAIN.spiffe)
EOF

# Deploy client
task deploy:k8s:context
task deploy:k8s:spire:dirctl

fi