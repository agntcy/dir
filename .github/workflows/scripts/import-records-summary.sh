#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

: "${GITHUB_STEP_SUMMARY:?GITHUB_STEP_SUMMARY is required}"
: "${GITHUB_RUN_ID:?GITHUB_RUN_ID is required}"
: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"

RUN_URL="https://github.com/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}"
JOBS_JSON=$(gh run view "$GITHUB_RUN_ID" --repo "$GITHUB_REPOSITORY" --json jobs)

append_overview() {
  {
    echo "## Import Records Run Summary"
    echo ""
    echo "| Setting | Value |"
    echo "|---------|-------|"
    echo "| Server | \`${SERVER_ADDRESS}\` |"
    echo "| Dry run | ${DRY_RUN} |"
    echo "| Sign | ${SIGN} |"
    echo "| MCP import enabled | ${RUN_MCP_IMPORT} |"
    echo "| Agent skills import enabled | ${RUN_AGENT_SKILLS_IMPORT} |"
    echo ""
    echo "| Job | Result |"
    echo "|-----|--------|"
    echo "| setup-matrix | ${SETUP_MATRIX_RESULT} |"
    echo "| build-dirctl | ${BUILD_DIRCTL_RESULT} |"
    echo "| import-mcp | ${IMPORT_MCP_RESULT} |"
    echo "| import-agent-skills | ${IMPORT_AGENT_SKILLS_RESULT} |"
    echo ""
    echo "[View workflow run](${RUN_URL})"
    echo ""
  } >> "$GITHUB_STEP_SUMMARY"
}

append_matrix_table() {
  local title=$1
  local filter=$2
  local conclusion=${3:-}

  local rows
  if [ -n "$conclusion" ]; then
    rows=$(echo "$JOBS_JSON" | jq -r --arg filter "$filter" --arg conclusion "$conclusion" '
      [.jobs[]
        | select(.name | startswith($filter))
        | select((.conclusion // .status // "unknown") == $conclusion)
        | {
            name: .name,
            conclusion: (.conclusion // .status // "unknown"),
            url: .html_url
          }
      ]
      | sort_by(.name)
      | .[]
      | "| [\(.name)](\(.url)) | \(.conclusion) |"
    ')
  else
    rows=$(echo "$JOBS_JSON" | jq -r --arg filter "$filter" '
      [.jobs[]
        | select(.name | startswith($filter))
        | {
            name: .name,
            conclusion: (.conclusion // .status // "unknown"),
            url: .html_url
          }
      ]
      | sort_by(.name)
      | .[]
      | "| [\(.name)](\(.url)) | \(.conclusion) |"
    ')
  fi

  if [ -z "$rows" ]; then
    return 0
  fi

  {
    echo "### ${title}"
    echo ""
    echo "| Matrix job | Result |"
    echo "|------------|--------|"
    echo "$rows"
    echo ""
  } >> "$GITHUB_STEP_SUMMARY"
}

append_counts() {
  local counts
  counts=$(echo "$JOBS_JSON" | jq -r '
    [.jobs[] | select(.name | startswith("import-mcp (") or startswith("import-agent-skills ("))]
    | group_by(.conclusion // .status // "unknown")
    | map({key: (.[0].conclusion // .[0].status // "unknown"), count: length})
    | sort_by(.key)
    | .[]
    | "\(.key): \(.count)"
  ')

  if [ -z "$counts" ]; then
    return 0
  fi

  {
    echo "### Matrix job counts"
    echo ""
    while IFS= read -r line; do
      echo "- ${line}"
    done <<< "$counts"
    echo ""
  } >> "$GITHUB_STEP_SUMMARY"
}

append_overview
append_counts
append_matrix_table "Failed matrix jobs" "import-" "failure"
append_matrix_table "Cancelled matrix jobs" "import-" "cancelled"
append_matrix_table "Timed out matrix jobs" "import-" "timed_out"

FAILED=$(echo "$JOBS_JSON" | jq '[.jobs[] | select(.name | startswith("import-mcp (") or startswith("import-agent-skills (")) | select((.conclusion // "") | IN("failure", "cancelled", "timed_out"))] | length')
if [ "$FAILED" -gt 0 ]; then
  {
    echo "> **${FAILED} import matrix job(s) failed, were cancelled, or timed out.**"
    echo ""
  } >> "$GITHUB_STEP_SUMMARY"
  exit 1
fi

SUCCESS=$(echo "$JOBS_JSON" | jq '[.jobs[] | select(.name | startswith("import-mcp (") or startswith("import-agent-skills (")) | select((.conclusion // "") == "success")] | length')
if [ "$SUCCESS" -gt 0 ]; then
  append_matrix_table "Successful matrix jobs" "import-" "success"
fi

{
  echo "All completed import matrix jobs succeeded (${SUCCESS})."
  echo ""
} >> "$GITHUB_STEP_SUMMARY"
