#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

: "${CIDS:?CIDS is required}"
: "${OIDC_CLIENT_ID:?OIDC_CLIENT_ID is required}"
SERVER_ADDR="${SERVER_ADDR:-}"
AUTH_TOKEN="${AUTH_TOKEN:-}"
MAX_RETRIES="${MAX_RETRIES:-3}"
CLEANUP_ON_FAILURE="${CLEANUP_ON_FAILURE:-false}"
RETRY_DELAY_SECONDS="${RETRY_DELAY_SECONDS:-30}"

if [ -n "${DIRCTL_PATH:-}" ]; then
  DIRCTL_BIN="${DIRCTL_PATH}"
elif command -v dirctl >/dev/null 2>&1; then
  DIRCTL_BIN="$(command -v dirctl)"
else
  echo "::error::dirctl not found. Set DIRCTL_PATH or add dirctl to PATH." >&2
  exit 1
fi

if [ ! -x "${DIRCTL_BIN}" ]; then
  echo "::error::dirctl is not executable at ${DIRCTL_BIN}" >&2
  exit 1
fi

if ! [[ "$MAX_RETRIES" =~ ^[1-9][0-9]*$ ]]; then
  echo "::error::MAX_RETRIES must be a positive integer (got: ${MAX_RETRIES})" >&2
  exit 1
fi

SIGNED=0
FAILED=0
CLEANED=0
ALL_SUCCESS=true
FAILED_CIDS_FILE="$(mktemp)"
CLEANED_CIDS_FILE="$(mktemp)"
trap 'rm -f "$FAILED_CIDS_FILE" "$CLEANED_CIDS_FILE"' EXIT

echo "=== Sign Records (OIDC) ==="
echo "dirctl: ${DIRCTL_BIN}"
echo "Server address: ${SERVER_ADDR:-"(dirctl default)"}"
echo "Max retries per CID: ${MAX_RETRIES}"
echo "Cleanup on failure: ${CLEANUP_ON_FAILURE}"
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

fetch_sigstore_oidc_token() {
  local token
  token=$(curl -sSf -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
    "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=sigstore" | jq -r '.value')
  if [ -z "$token" ] || [ "$token" = "null" ]; then
    return 1
  fi
  printf '%s' "$token"
}

append_dirctl_auth_args() {
  local -n cmd_ref=$1
  [ -n "$SERVER_ADDR" ] && cmd_ref+=(--server-addr "$SERVER_ADDR")
  [ -n "$AUTH_TOKEN" ] && cmd_ref+=(--auth-mode=oidc "--auth-token=$AUTH_TOKEN")
}

sign_cid_once() {
  local cid=$1
  local oidc_token=$2
  local -a sign_cmd=(
    "${DIRCTL_BIN}" sign "$cid"
    "--oidc-token=${oidc_token}"
    --oidc-provider-url=https://token.actions.githubusercontent.com
    "--oidc-client-id=${OIDC_CLIENT_ID}"
  )
  append_dirctl_auth_args sign_cmd
  "${sign_cmd[@]}"
}

delete_cid() {
  local cid=$1
  local -a delete_cmd=("${DIRCTL_BIN}" delete "$cid")
  append_dirctl_auth_args delete_cmd
  "${delete_cmd[@]}"
}

retry_delay() {
  local attempt=$1
  local delay=$((RETRY_DELAY_SECONDS * attempt))
  echo "Waiting ${delay}s before retry..."
  sleep "$delay"
}

echo "Signing ${#CID_ARRAY[@]} record(s)"
echo ""

for CID in "${CID_ARRAY[@]}"; do
  echo "----------------------------------------"
  echo "Processing: $CID"
  echo "----------------------------------------"

  signed=false
  last_sign_output=""

  for ((attempt = 1; attempt <= MAX_RETRIES; attempt++)); do
    echo "Sign attempt ${attempt}/${MAX_RETRIES}"

    set +e
    OIDC_TOKEN=$(fetch_sigstore_oidc_token)
    token_exit=$?
    set -e
    if [ $token_exit -ne 0 ] || [ -z "$OIDC_TOKEN" ]; then
      last_sign_output="Failed to obtain Sigstore OIDC token"
      echo "::warning::${last_sign_output} (attempt ${attempt}/${MAX_RETRIES})"
    else
      set +e
      last_sign_output=$(sign_cid_once "$CID" "$OIDC_TOKEN" 2>&1)
      sign_exit=$?
      set -e
      if [ $sign_exit -eq 0 ]; then
        echo "Successfully signed"
        echo "::notice::Signed CID: $CID"
        SIGNED=$((SIGNED + 1))
        signed=true
        break
      fi
      echo "::warning::Failed to sign $CID (attempt ${attempt}/${MAX_RETRIES}): ${last_sign_output}"
    fi

    if [ "$attempt" -lt "$MAX_RETRIES" ]; then
      retry_delay "$attempt"
    fi
  done

  if [ "$signed" = true ]; then
    echo ""
    continue
  fi

  echo "::error::All sign attempts failed for $CID"

  if [ "$CLEANUP_ON_FAILURE" = "true" ]; then
    echo "Attempting cleanup: deleting unsigned record $CID"
    set +e
    delete_output=$(delete_cid "$CID" 2>&1)
    delete_exit=$?
    set -e
    if [ $delete_exit -eq 0 ]; then
      echo "Deleted unsigned record $CID so a future import can retry push"
      echo "::notice::Cleaned up unsigned CID: $CID"
      echo "$CID" >> "$CLEANED_CIDS_FILE"
      CLEANED=$((CLEANED + 1))
    else
      echo "::error::Failed to delete $CID after sign failure: ${delete_output}"
      echo "$CID" >> "$FAILED_CIDS_FILE"
      FAILED=$((FAILED + 1))
      ALL_SUCCESS=false
    fi
  else
    echo "$CID" >> "$FAILED_CIDS_FILE"
    FAILED=$((FAILED + 1))
    ALL_SUCCESS=false
  fi

  echo ""
done

echo "=== Sign Summary ==="
echo "Signed: $SIGNED"
echo "Cleaned up (deleted after sign failure): $CLEANED"
echo "Failed (unsigned and not cleaned): $FAILED"
echo ""

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "success=$ALL_SUCCESS" >> "$GITHUB_OUTPUT"
  echo "signed=$SIGNED" >> "$GITHUB_OUTPUT"
  echo "cleaned=$CLEANED" >> "$GITHUB_OUTPUT"
  echo "failed=$FAILED" >> "$GITHUB_OUTPUT"

  if [ -s "$FAILED_CIDS_FILE" ]; then
    {
      echo 'failed_cids<<EOF'
      cat "$FAILED_CIDS_FILE"
      echo 'EOF'
    } >> "$GITHUB_OUTPUT"
  fi

  if [ -s "$CLEANED_CIDS_FILE" ]; then
    {
      echo 'cleaned_cids<<EOF'
      cat "$CLEANED_CIDS_FILE"
      echo 'EOF'
    } >> "$GITHUB_OUTPUT"
  fi
fi

if [ "$ALL_SUCCESS" = false ]; then
  echo "::error::One or more records could not be signed (and cleanup did not succeed for all failures)" >&2
  exit 1
fi

echo "All records signed successfully."
