#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

: "${CIDS:?CIDS is required}"
: "${OIDC_CLIENT_ID:?OIDC_CLIENT_ID is required}"
SERVER_ADDR="${SERVER_ADDR:-}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"

SIGNED=0
FAILED=0
ALL_SUCCESS=true

echo "=== Sign Records (OIDC) ==="
echo "Server address: ${SERVER_ADDR:-(dirctl default)}"
echo ""

if [ -z "${OIDC_CLIENT_ID:-}" ] || [ -z "${ACTIONS_ID_TOKEN_REQUEST_TOKEN:-}" ] || [ -z "${ACTIONS_ID_TOKEN_REQUEST_URL:-}" ]; then
  echo "::error::Signing requires: (1) permissions: id-token: write (2) oidc_client_id: https://github.com/\${{ github.repository }}/.github/workflows/YOUR_WORKFLOW.yaml@\${{ github.ref }}"
  exit 1
fi

# Normalize to one CID per line (accept JSON object from push-record output)
if [[ "$CIDS" == \{* ]]; then
  CID_LIST=$(echo "$CIDS" | jq -r '.[]')
else
  CID_LIST="$CIDS"
fi

CID_ARRAY=()
while IFS= read -r line || [[ -n "$line" ]]; do
  line="${line#"${line%%[![:space:]]*}"}"
  line="${line%"${line##*[![:space:]]}"}"
  [[ -z "$line" ]] && continue
  CID_ARRAY+=("$line")
done <<< "$CID_LIST"

if [ ${#CID_ARRAY[@]} -eq 0 ]; then
  echo "::error::No CIDs to sign" >&2
  exit 1
fi

echo "Obtaining OIDC token..."
OIDC_TOKEN=$(curl -sSf -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
  "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=sigstore" | jq -r '.value')
if [ -z "$OIDC_TOKEN" ] || [ "$OIDC_TOKEN" = "null" ]; then
  echo "::error::Failed to obtain OIDC token. Ensure the job has permissions: id-token: write"
  exit 1
fi

echo "Signing ${#CID_ARRAY[@]} record(s)"
echo ""

for CID in "${CID_ARRAY[@]}"; do
  echo "----------------------------------------"
  echo "Signing: $CID"
  echo "----------------------------------------"
  set +e
  SIGN_CMD="dirctl sign $CID --oidc-token=$OIDC_TOKEN --oidc-provider-url=https://token.actions.githubusercontent.com --oidc-client-id=$OIDC_CLIENT_ID"
  [ -n "$SERVER_ADDR" ] && SIGN_CMD="$SIGN_CMD --server-addr=$SERVER_ADDR"
  [ -n "$GITHUB_TOKEN" ] && SIGN_CMD="$SIGN_CMD --github-token=$GITHUB_TOKEN"
  OUTPUT=$(eval "$SIGN_CMD" 2>&1)
  EXIT=$?
  set -e
  if [ $EXIT -eq 0 ]; then
    echo "Successfully signed"
    echo "::notice::Signed CID: $CID"
    SIGNED=$((SIGNED + 1))
  else
    echo "::error::Failed to sign $CID: $OUTPUT"
    FAILED=$((FAILED + 1))
    ALL_SUCCESS=false
  fi
  echo ""
done

echo "=== Sign Summary ==="
echo "Signed: $SIGNED"
echo "Failed: $FAILED"
echo ""

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "success=$ALL_SUCCESS" >> "$GITHUB_OUTPUT"
  echo "signed=$SIGNED" >> "$GITHUB_OUTPUT"
  echo "failed=$FAILED" >> "$GITHUB_OUTPUT"
fi

if [ "$ALL_SUCCESS" = false ]; then
  echo "::error::One or more records failed to sign" >&2
  exit 1
fi

echo "All records signed successfully."
