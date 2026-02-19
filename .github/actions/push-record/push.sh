#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

: "${RECORD_PATHS:?RECORD_PATHS is required}"
SERVER_ADDR="${SERVER_ADDR:-}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"

CIDS="{}"
FAILED_FILES="[]"
ALL_SUCCESS=true

echo "=== Push OASF Records ==="
echo "Server address: ${SERVER_ADDR:-(dirctl default)}"
echo ""

# Collect all files from all path patterns
ALL_FILES=()
shopt -s nullglob

while IFS= read -r PATTERN || [[ -n "$PATTERN" ]]; do
  [[ -z "$PATTERN" ]] && continue
  PATTERN="${PATTERN#"${PATTERN%%[![:space:]]*}"}"
  PATTERN="${PATTERN%"${PATTERN##*[![:space:]]}"}"
  [[ -z "$PATTERN" ]] && continue

  echo "Pattern: $PATTERN"
  # shellcheck disable=SC2206
  MATCHED_FILES=($PATTERN)
  EXISTING_FILES=()
  for f in "${MATCHED_FILES[@]}"; do
    [ -f "$f" ] && EXISTING_FILES+=("$f")
  done

  if [ ${#EXISTING_FILES[@]} -eq 0 ]; then
    echo "  Warning: No files found matching pattern: $PATTERN"
  else
    echo "  Matched ${#EXISTING_FILES[@]} file(s)"
    ALL_FILES+=("${EXISTING_FILES[@]}")
  fi
done <<< "$RECORD_PATHS"

shopt -u nullglob
echo ""

if [ ${#ALL_FILES[@]} -eq 0 ]; then
  echo "::error::No files found matching any of the specified patterns" >&2
  exit 1
fi

echo "Total files to push: ${#ALL_FILES[@]}"
echo ""

for FILE in "${ALL_FILES[@]}"; do
  echo "----------------------------------------"
  echo "Pushing: $FILE"
  echo "----------------------------------------"

  set +e
  PUSH_CMD="dirctl push \"$FILE\" --output raw"
  [ -n "$SERVER_ADDR" ] && PUSH_CMD="$PUSH_CMD --server-addr=\"$SERVER_ADDR\""
  [ -n "$GITHUB_TOKEN" ] && PUSH_CMD="$PUSH_CMD --github-token=\"$GITHUB_TOKEN\""
  CID=$(eval "$PUSH_CMD" 2>&1)
  EXIT_CODE=$?
  set -e

  if [ $EXIT_CODE -eq 0 ] && [[ "$CID" =~ ^bae ]]; then
    echo "Successfully pushed: $CID"
    echo "::notice file=$FILE::Pushed with CID: $CID"
    CIDS=$(echo "$CIDS" | jq -c ". + {\"$FILE\": \"$CID\"}")
  else
    echo "Failed to push: $CID"
    ERROR_MSG=$(echo "$CID" | grep -iE "error|failed|invalid" | head -1)
    [ -z "$ERROR_MSG" ] && ERROR_MSG="$CID"
    echo "::error file=$FILE::Failed to push: $ERROR_MSG"
    FAILED_FILES=$(echo "$FAILED_FILES" | jq -c ". + [\"$FILE\"]")
    ALL_SUCCESS=false
  fi
  echo ""
done

echo "=== Push Summary ==="
echo "Total files: ${#ALL_FILES[@]}"
echo "Pushed: $(echo "$CIDS" | jq 'keys | length')"
echo "Failed: $(echo "$FAILED_FILES" | jq 'length')"
echo ""

if [ "$(echo "$CIDS" | jq 'keys | length')" -gt 0 ]; then
  echo "Pushed CIDs:"
  echo "$CIDS" | jq -r 'to_entries[] | "  \(.key): \(.value)"'
fi

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "success=$ALL_SUCCESS" >> "$GITHUB_OUTPUT"
  echo "cids=$CIDS" >> "$GITHUB_OUTPUT"
  echo "failed_files=$FAILED_FILES" >> "$GITHUB_OUTPUT"
fi

if [ "$ALL_SUCCESS" = false ]; then
  echo "::error::One or more records failed to push" >&2
  exit 1
fi

echo ""
echo "All records pushed successfully."
