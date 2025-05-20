#!/bin/bash

# This script configures Cosign to be used for signing and verification.
# Requirements:
#   - cosign

## Initialize cosign (required to setup trust chain)
# TODO: you need to add "ca.cert.pem" to your OS trust store.
# cosign initialize \
#  --root https://tuf.sigstore.local/root.json \
#  --mirror https://tuf.sigstore.local

## Prepare the environment
REKOR_URL=https://rekor.sigstore.dev
FULCIO_URL=https://fulcio.sigstore.dev
export COSIGN_EXPERIMENTAL=1

## Fix model by stripping the signature and applying proper JSON formatting
cat agent.json | jq . > agent.json.tmp
cat agent.json.tmp | jq 'del(.signature)' > agent.json
rm -rf agent.json.tmp

## 1. Sign agent
cosign sign-blob \
 --fulcio-url=$FULCIO_URL \
 --rekor-url=$REKOR_URL \
 --yes \
 --b64=false \
 --bundle='agent.sig' \
 ./agent.json

# Append signature to agent model
cat agent.json | jq ".signature += $(cat agent.sig | jq .)" > pushed.agent.json

## 2. Push signed agent
# DIGEST=$(dirctl push pushed.agent.json)

## 3. Pull signed agent
# dirctl pull $DIGEST

## 4. Extract signature
cat pushed.agent.json | jq '.signature'      > pulled.agent.sig.json
cat pushed.agent.json | jq 'del(.signature)' > pulled.agent.json

## 5. Verify agent
echo -e "\n\nVerifying blob signature..."
cosign verify-blob \
 --rekor-url=$REKOR_URL \
 --bundle 'pulled.agent.sig.json' \
 --certificate-identity=".*" \
 --certificate-oidc-issuer=https://github.com/login/oauth \
 ./pulled.agent.json

## 6. CLEANUP
rm -rf pulled.agent.*
