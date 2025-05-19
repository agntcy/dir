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

## Sign blob
echo -e "\n\nSigning blob..."
cosign sign-blob \
 --fulcio-url=$FULCIO_URL \
 --rekor-url=$REKOR_URL \
 --yes \
 --b64=false \
 --bundle='agent.sig' \
 ./agent.json

## Verify blob
# NOTE: the implementation can happen in the following steps:
# 1. Upload the signature to Dir storage
# 2. Append the signature link to the agent model
# 3. Pull agent model
# 4. Set the signature field to nil
# 5. Verify the signature
echo -e "\n\nVerifying blob signature..."
cosign verify-blob \
 --rekor-url=$REKOR_URL \
 --bundle 'agent.sig' \
 --certificate-identity=ramiz.polic@hotmail.com \
 --certificate-oidc-issuer=https://github.com/login/oauth \
 ./agent.json
