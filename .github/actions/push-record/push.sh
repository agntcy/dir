#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

: "${RECORD_PATHS:?RECORD_PATHS is required}"
: "${DIRECTORY_ADDRESS:?DIRECTORY_ADDRESS is required}"
SIGN_RECORDS="${SIGN_RECORDS:-false}"
PUBLISH_RECORDS="${PUBLISH_RECORDS:-false}"

# Initialize result tracking
CIDS="{}"
FAILED_FILES="[]"
ALL_SUCCESS=true

echo "=== Push OASF Records ==="
echo "Directory: $DIRECTORY_ADDRESS"
echo "Sign records: $SIGN_RECORDS"
echo "Publish to DHT: $PUBLISH_RECORDS"
echo ""

# Collect all files from all path patterns
ALL_FILES=()
shopt -s nullglob

# Read paths line by line
while IFS= read -r PATTERN || [[ -n "$PATTERN" ]]; do
  # Skip empty lines
  [[ -z "$PATTERN" ]] && continue
  
  # Trim whitespace
  PATTERN=$(echo "$PATTERN" | xargs)
  [[ -z "$PATTERN" ]] && continue
  
  echo "Pattern: $PATTERN"
  
  # Expand glob pattern
  # shellcheck disable=SC2206
  MATCHED_FILES=($PATTERN)
  
  # Filter to only existing files (nullglob doesn't work for non-glob patterns)
  EXISTING_FILES=()
  for f in "${MATCHED_FILES[@]}"; do
    if [ -f "$f" ]; then
      EXISTING_FILES+=("$f")
    fi
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
  
  # Build push command
  PUSH_CMD="dirctl push \"$FILE\" --server-addr=\"$DIRECTORY_ADDRESS\" --github-token=\"$DIRECTORY_CLIENT_GITHUB_TOKEN\" --output raw"
  if [ "$SIGN_RECORDS" = "true" ]; then
    PUSH_CMD="$PUSH_CMD --sign"
  fi
  
  # Run push and capture CID
  set +e
  CID=$(eval "$PUSH_CMD" 2>&1)
  EXIT_CODE=$?
  set -e
  
  if [ $EXIT_CODE -eq 0 ] && [[ "$CID" =~ ^bae ]]; then
    echo "Successfully pushed: $CID"
    echo "::notice file=$FILE::Pushed with CID: $CID"
    CIDS=$(echo "$CIDS" | jq -c ". + {\"$FILE\": \"$CID\"}")
    
    # Publish to DHT if enabled
    if [ "$PUBLISH_RECORDS" = "true" ]; then
      echo "Publishing to DHT..."
      set +e
      PUBLISH_OUTPUT=$(dirctl routing publish "$CID" --server-addr="$DIRECTORY_ADDRESS" --github-token="$DIRECTORY_CLIENT_GITHUB_TOKEN" 2>&1)
      PUBLISH_EXIT=$?
      set -e
      
      if [ $PUBLISH_EXIT -eq 0 ]; then
        echo "Successfully published to DHT"
        echo "::notice file=$FILE::Published to DHT"
      else
        echo "Warning: Failed to publish to DHT: $PUBLISH_OUTPUT"
        echo "::warning file=$FILE::Push succeeded but DHT publish failed: $PUBLISH_OUTPUT"
      fi
    fi
  else
    echo "Failed to push: $CID"
    # Extract meaningful error message
    ERROR_MSG=$(echo "$CID" | grep -iE "error|failed|invalid" | head -1)
    if [ -z "$ERROR_MSG" ]; then
      ERROR_MSG="$CID"
    fi
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

# Print CID mapping
if [ "$(echo "$CIDS" | jq 'keys | length')" -gt 0 ]; then
  echo "Pushed CIDs:"
  echo "$CIDS" | jq -r 'to_entries[] | "  \(.key): \(.value)"'
fi

# Set outputs if running in GitHub Actions
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "success=$ALL_SUCCESS" >> "$GITHUB_OUTPUT"
  echo "cids=$CIDS" >> "$GITHUB_OUTPUT"
  echo "failed_files=$FAILED_FILES" >> "$GITHUB_OUTPUT"
fi

# Exit with error if any push failed
if [ "$ALL_SUCCESS" = false ]; then
  echo ""
  echo "::error::One or more records failed to push" >&2
  exit 1
fi

echo ""
echo "All records pushed successfully!"
