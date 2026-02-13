#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

: "${CIDS:?CIDS is required}"
SERVER_ADDR="${SERVER_ADDR:-}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"

PUBLISHED=0
FAILED=0
ALL_SUCCESS=true

echo "=== Publish Records to DHT ==="
echo "Server address: ${SERVER_ADDR:-(dirctl default)}"
echo ""

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
  echo "::error::No CIDs to publish" >&2
  exit 1
fi

echo "CIDs to publish: ${#CID_ARRAY[@]}"
echo ""

for CID in "${CID_ARRAY[@]}"; do
  echo "----------------------------------------"
  echo "Publishing: $CID"
  echo "----------------------------------------"
  set +e
  PUBLISH_CMD="dirctl routing publish \"$CID\""
  [ -n "$SERVER_ADDR" ] && PUBLISH_CMD="$PUBLISH_CMD --server-addr=\"$SERVER_ADDR\""
  [ -n "$GITHUB_TOKEN" ] && PUBLISH_CMD="$PUBLISH_CMD --github-token=\"$GITHUB_TOKEN\""
  OUTPUT=$(eval "$PUBLISH_CMD" 2>&1)
  EXIT=$?
  set -e
  if [ $EXIT -eq 0 ]; then
    echo "Successfully published"
    echo "::notice::Published CID: $CID"
    PUBLISHED=$((PUBLISHED + 1))
  else
    echo "::error::Failed to publish $CID: $OUTPUT"
    FAILED=$((FAILED + 1))
    ALL_SUCCESS=false
  fi
  echo ""
done

echo "=== Publish Summary ==="
echo "Published: $PUBLISHED"
echo "Failed: $FAILED"
echo ""

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "success=$ALL_SUCCESS" >> "$GITHUB_OUTPUT"
  echo "published=$PUBLISHED" >> "$GITHUB_OUTPUT"
  echo "failed=$FAILED" >> "$GITHUB_OUTPUT"
fi

if [ "$ALL_SUCCESS" = false ]; then
  echo "::error::One or more records failed to publish" >&2
  exit 1
fi

echo "All records published successfully."
