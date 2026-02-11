#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

: "${RECORD_PATHS:?RECORD_PATHS is required}"
: "${SCHEMA_URL:?SCHEMA_URL is required}"
FAIL_ON_WARNING="${FAIL_ON_WARNING:-false}"

# Initialize result tracking
VALIDATED_FILES="[]"
FAILED_FILES="[]"
ALL_VALID=true
HAS_ERRORS=false

echo "=== OASF Record Validation ==="
echo "Schema URL: $SCHEMA_URL"
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

echo "Total files to validate: ${#ALL_FILES[@]}"
echo ""

for FILE in "${ALL_FILES[@]}"; do
  echo "----------------------------------------"
  echo "Validating: $FILE"
  echo "----------------------------------------"
  
  # Run validation and capture output
  set +e
  OUTPUT=$(dirctl validate "$FILE" --url="$SCHEMA_URL" 2>&1)
  EXIT_CODE=$?
  set -e
  
  echo "$OUTPUT"
  echo ""
  
  if [ $EXIT_CODE -eq 0 ]; then
    # Check for warnings if fail_on_warning is enabled
    if [[ "$FAIL_ON_WARNING" == "true" && "$OUTPUT" == *"warning"* ]]; then
      echo "::warning file=$FILE::Validation passed with warnings"
      FAILED_FILES=$(echo "$FAILED_FILES" | jq -c ". + [\"$FILE\"]")
      ALL_VALID=false
    else
      echo "::notice file=$FILE::Validation passed"
      VALIDATED_FILES=$(echo "$VALIDATED_FILES" | jq -c ". + [\"$FILE\"]")
    fi
  else
    # Extract error message for GitHub annotation
    ERROR_MSG=$(echo "$OUTPUT" | grep -E "^\s+[0-9]+\." | head -5 | tr '\n' ' ')
    echo "::error file=$FILE::Validation failed: $ERROR_MSG"
    FAILED_FILES=$(echo "$FAILED_FILES" | jq -c ". + [\"$FILE\"]")
    ALL_VALID=false
    HAS_ERRORS=true
  fi
done

echo ""
echo "=== Validation Summary ==="
echo "Total files: ${#ALL_FILES[@]}"
echo "Passed: $(echo "$VALIDATED_FILES" | jq 'length')"
echo "Failed: $(echo "$FAILED_FILES" | jq 'length')"
echo ""

# Set outputs if running in GitHub Actions
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "valid=$ALL_VALID" >> "$GITHUB_OUTPUT"
  echo "validated_files=$VALIDATED_FILES" >> "$GITHUB_OUTPUT"
  echo "failed_files=$FAILED_FILES" >> "$GITHUB_OUTPUT"
fi

# Exit with error if any validation failed
if [ "$HAS_ERRORS" = true ]; then
  echo "::error::One or more records failed validation" >&2
  exit 1
fi

if [ "$ALL_VALID" = false ]; then
  echo "::error::One or more records have validation issues" >&2
  exit 1
fi

echo "All records passed validation!"
