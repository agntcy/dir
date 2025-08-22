#!/bin/bash

## Configuration
export DIR_TRUST_DOMAIN="dir.example"
export DIRCTL_TRUST_DOMAIN="dirctl.example"

## Kill background processes on exit.
trap 'kill $(jobs -p)' EXIT

## Deploy SPIRE cluster
deploy_spire_cluster() {
  # Function params
  local KIND_CLUSTER_NAME=$1
  local SPIRE_TRUST_DOMAIN=$2
  local SPIRE_BUNDLE_PATH=$3

  # Start kind cloud provider in the background.
  # This exposes LoadBalancer services to the host.
  # Sudo is required.
  go run sigs.k8s.io/cloud-provider-kind@latest > /dev/null 2>&1 &

  # Deploy cluster
  task deploy:kubernetes:cleanup
  task deploy:kubernetes:setup-cluster
  task deploy:kubernetes:spire

  # Export SPIRE bundle
  kubectl get configmap -n spire spire-bundle -o 'go-template={{index .data "bundle.spiffe"}}' > $SPIRE_BUNDLE_PATH
}

## Deploy DIR server
deploy_dir() {
  # Function params
  local KIND_CLUSTER_NAME=$1
  local DIRCTL_BUNDLE_ADDRESS=$2
  local DIRCTL_BUNDLE_PATH=$3

  # Create server federation config
  cat <<EOF > /tmp/server-federation.yaml
apiserver:
  service:
    type: LoadBalancer

  spire:
    enabled: true
    trustDomain: $DIR_TRUST_DOMAIN
    federation:
      - trustDomain: $DIRCTL_TRUST_DOMAIN
        bundleEndpointURL: https://$DIRCTL_BUNDLE_ADDRESS
        bundleEndpointProfile:
          type: https_spiffe
          endpointSPIFFEID: spiffe://$DIRCTL_TRUST_DOMAIN/spire/server
        trustDomainBundle: |-
          $(cat $DIRCTL_BUNDLE_PATH)
EOF

  # Deploy server
  local HELM_EXTRA_ARGS="-f /tmp/server-federation.yaml"
  task deploy:kubernetes:context
  task deploy:kubernetes:dir
}

## Deploy DIRCTL server
deploy_dirctl() {
  # Function params
  local KIND_CLUSTER_NAME=$1
  local DIR_SERVER_ADDRESS=$2
  local DIR_BUNDLE_ADDRESS=$3
  local DIR_BUNDLE_PATH=$4

  # Create server federation config
  cat <<EOF > /tmp/client-federation.yaml
env:
  - name: DIRECTORY_CLIENT_SERVER_ADDRESS
    value: $DIR_SERVER_ADDRESS

spire:
  enabled: true
  trustDomain: $DIRCTL_TRUST_DOMAIN
  federation:
    - trustDomain: $DIR_TRUST_DOMAIN
      bundleEndpointURL: https://$DIR_BUNDLE_ADDRESS
      bundleEndpointProfile:
        type: https_spiffe
        endpointSPIFFEID: spiffe://$DIR_TRUST_DOMAIN/spire/server
      trustDomainBundle: |-
        $(cat $DIR_BUNDLE_PATH)
EOF

  # Deploy client
  local HELM_EXTRA_ARGS="-f /tmp/client-federation.yaml"
  task deploy:kubernetes:context
  task deploy:kubernetes:dirctl
}

## Get Dir server IP Address
get_dir_server_address() {
  local KIND_CLUSTER_NAME=$1
  task deploy:kubernetes:context
  ip=$(kubectl get service -n dir-server dir-apiserver -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
  echo $ip:8888
}

## Get Dir bundle IP Address
get_spire_bundle_address() {
  local KIND_CLUSTER_NAME=$1
  task deploy:kubernetes:context
  ip=$(kubectl get service -n spire spire-server -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
  echo $ip:8443
}

########### WORKFLOW

# Deploy SPIRE clusters
deploy_spire_cluster "$DIR_TRUST_DOMAIN" "$DIR_TRUST_DOMAIN" "/tmp/$DIR_TRUST_DOMAIN.spiffe"
deploy_spire_cluster "$DIRCTL_TRUST_DOMAIN" "$DIR_TRUST_DOMAIN" "/tmp/$DIRCTL_TRUST_DOMAIN.spiffe"

# Get addresses
DIR_SERVER_ADDRESS=$(get_dir_server_address "$DIR_TRUST_DOMAIN")
DIR_BUNDLE_ADDRESS=$(get_spire_bundle_address "$DIR_TRUST_DOMAIN")
DIRCTL_BUNDLE_ADDRESS=$(get_spire_bundle_address "$DIRCTL_TRUST_DOMAIN")

# Deploy servers
deploy_dir "$DIR_TRUST_DOMAIN" "$DIRCTL_BUNDLE_ADDRESS" "/tmp/$DIRCTL_TRUST_DOMAIN.spiffe"
deploy_dirctl "$DIRCTL_TRUST_DOMAIN" "$DIR_SERVER_ADDRESS" "$DIR_BUNDLE_ADDRESS" "/tmp/$DIR_TRUST_DOMAIN.spiffe"
