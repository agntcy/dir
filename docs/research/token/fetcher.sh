#!/bin/bash

# Deploy example fetcher
kubectl apply -f fetcher.yaml

# Fetch SVID data
kubectl exec alice -c client -- cat /svids/tls.crt > svids.tls.crt
kubectl exec alice -c client -- cat /svids/tls.key > svids.tls.key
kubectl exec alice -c client -- cat /svids/svid_bundle.pem > svids.bundle.pem

# Set ENV variables for the client
export DIRECTORY_CLIENT_SERVER_ADDRESS=127.0.0.1:8888
export DIRECTORY_CLIENT_AUTH_MODE=tls
export DIRECTORY_CLIENT_TLS_SKIP_VERIFY=true
export DIRECTORY_CLIENT_TLS_CERT_FILE=$(pwd)/svids.tls.crt
export DIRECTORY_CLIENT_TLS_CA_FILE=$(pwd)/svids.bundle.pem
export DIRECTORY_CLIENT_TLS_KEY_FILE=$(pwd)/svids.tls.key
