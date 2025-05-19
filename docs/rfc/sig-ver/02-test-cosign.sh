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

## 0. FIX MODEL
cat agent.json | jq . > agent.json.tmp
rm -rf agent.json
mv agent.json.tmp agent.json

## 1. Sign agent
echo -e "\n\nSigning agent locally..."
cosign sign-blob \
 --fulcio-url=$FULCIO_URL \
 --rekor-url=$REKOR_URL \
 --yes \
 --b64=false \
 --bundle='agent.sig' \
 ./agent.json

read -p "Press enter to continue"

# Append signature to agent model
echo -e "\n\nGenerating signed agent locally..."
cat agent.json | jq ".signature += $(cat agent.sig | jq .)" > pushed.agent.json


read -p "Press enter to continue"

## 2. Push signed agent
echo -e "\n\nPushing signed agent..."
# dirctl push pushed.agent.json

## 3. Pull signed agent
echo -e "\n\nPulling signed agent..."
# dirctl pull $DIGEST

## 4. Extract signature
cat pushed.agent.json | jq '.signature' > pulled.agent.sig
cat pushed.agent.json | jq 'del(.signature)' > pulled.agent.json

read -p "Press enter to continue"

## 5. Verify agent
# NOTE: the implementation can happen in the following steps:
# 1. Upload the signature to Dir storage
# 2. Append the signature link to the agent model
# 3. Pull agent model
# 4. Set the signature field to nil
# 5. Verify the signature
echo -e "\n\nVerifying blob signature..."
cosign verify-blob \
 --rekor-url=$REKOR_URL \
 --bundle 'pulled.agent.sig' \
 --certificate-identity=rpolic@cisco.com \
 --certificate-oidc-issuer=https://github.com/login/oauth \
 ./pulled.agent.json
