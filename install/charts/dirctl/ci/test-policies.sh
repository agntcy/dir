#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0
#
# Asserts on the policy CronJobs the dirctl chart renders. `helm lint` only
# checks that the chart renders at all, so none of this is covered by it.

set -uo pipefail

HELM="${HELM:-helm}"
CHART="$(cd "$(dirname "$0")/.." && pwd)"
FAILURES=0

pass() { echo "ok   - $1"; }
fail() {
  echo "FAIL - $1"
  shift
  for detail in "$@"; do echo "       $detail"; done
  FAILURES=$((FAILURES + 1))
}

assert_contains() {
  local haystack="$1" needle="$2" what="$3"
  if printf '%s' "$haystack" | grep -qF -- "$needle"; then
    pass "$what"
  else
    fail "$what" "expected to find: $needle"
  fi
}

assert_not_contains() {
  local haystack="$1" needle="$2" what="$3"
  if printf '%s' "$haystack" | grep -qF -- "$needle"; then
    fail "$what" "did not expect to find: $needle"
  else
    pass "$what"
  fi
}

# assert_render_fails <expected message fragment> <description> <helm args...>
assert_render_fails() {
  local needle="$1" what="$2"
  shift 2
  local output
  if output="$("$HELM" template policy-test "$CHART" "$@" 2>&1)"; then
    fail "$what" "render unexpectedly succeeded"
  elif printf '%s' "$output" | grep -qF -- "$needle"; then
    pass "$what"
  else
    fail "$what" "expected error to mention: $needle" "got: $output"
  fi
}

RENDERED="$("$HELM" template policy-test "$CHART" --values "$CHART/ci/policies-values.yaml")" || {
  echo "FAIL - chart failed to render with ci/policies-values.yaml"
  exit 1
}

# policy_doc <policy name> prints just that policy's CronJob manifest, so a
# per-policy assertion cannot accidentally match another policy's script.
policy_doc() {
  printf '%s' "$RENDERED" | awk -v want="  name: policy-test-dirctl-policy-$1" '
    $0 == want { f = 1 }
    f && /^---$/ { exit }
    f { print }
  '
}

assert_contains "$RENDERED" "name: policy-test-dirctl-policy-sync-netsec" \
  "sync policy renders a CronJob"
assert_contains "$RENDERED" "dirctl routing search --output json --limit 100 --domain 'network_security'" \
  "sync searches the routing index with the match flags"
assert_contains "$RENDERED" "dirctl sync create --stdin" \
  "sync pipes matches into sync create"

assert_contains "$RENDERED" "dirctl search --format cid --output json --limit 100 --scan-severity 'MEDIUM' --trusted=false" \
  "prune renders a negated bool and a scalar match"
assert_contains "$RENDERED" "dirctl delete --stdin" \
  "prune pipes matches into delete"

assert_contains "$RENDERED" "dirctl search --format cid --output json --limit 25 --name 'cisco.com/*' --name 'agntcy.org/*' --trusted=true --offset \"\$offset\"" \
  "publish repeats a list match and honours the policy limit"
assert_contains "$RENDERED" "dirctl routing publish --stdin" \
  "publish pipes matches into routing publish"

# Pagination. publish/unpublish do not change whether a record matches, so they
# must page; prune shrinks its own result set and sync cannot page at all
# (dirctl routing search has no --offset).
assert_contains "$RENDERED" 'offset=$((offset + 25))' \
  "publish pages through offsets"
assert_contains "$RENDERED" "processed \${total} record(s)" \
  "publish reports a total across pages"
assert_not_contains "$(policy_doc prune-untrusted)" "--offset" \
  "prune does not paginate"
assert_not_contains "$(policy_doc sync-netsec)" "--offset" \
  "sync does not paginate"

assert_contains "$RENDERED" 'if [ -z "$out" ]; then' \
  "scripts fail loudly when the search prints nothing"

assert_contains "$RENDERED" "dry run: not applying action 'unpublish'" \
  "dryRun replaces the sink with a log line"
assert_not_contains "$RENDERED" "dirctl routing unpublish --stdin" \
  "dryRun does not render the sink"

assert_contains "$RENDERED" 'if [ "$count" -eq 0 ]; then' \
  "generated scripts guard against an empty match set"
assert_not_contains "$RENDERED" "pipefail" \
  "generated scripts avoid pipefail (busybox ash)"

assert_not_contains "$RENDERED" "policy-never-enabled" \
  "a disabled policy renders nothing"

DEFAULTS="$("$HELM" template policy-test "$CHART")" || {
  echo "FAIL - chart failed to render with default values"
  exit 1
}
assert_not_contains "$DEFAULTS" "-policy-" \
  "the shipped example policies are disabled by default"

assert_render_fails 'unknown action "nope"' "an unknown action fails the render" \
  --set policies.bad.enabled=true --set policies.bad.schedule='@daily' --set policies.bad.action=nope

assert_render_fails "is set by the chart" "a reserved match key fails the render" \
  --set policies.bad.enabled=true --set policies.bad.schedule='@daily' \
  --set policies.bad.action=prune --set policies.bad.match.limit=5

assert_render_fails "cannot be safely shell-quoted" "a quote in a match value fails the render" \
  --set policies.bad.enabled=true --set policies.bad.schedule='@daily' \
  --set policies.bad.action=prune --set "policies.bad.match.name=o'brien"

# Keys and limit are interpolated into the shell command unquoted, so both are
# constrained at template time rather than trusted.
assert_render_fails "is not a valid dirctl flag name" "a shell metacharacter in a match key fails the render" \
  --set policies.bad.enabled=true --set policies.bad.schedule='@daily' \
  --set policies.bad.action=prune --set 'policies.bad.match.name; echo pwned #=v'

assert_render_fails "limit must be a positive integer" "a non-numeric limit fails the render" \
  --set policies.bad.enabled=true --set policies.bad.schedule='@daily' \
  --set policies.bad.action=prune --set-string 'policies.bad.limit=1; echo pwned'

assert_render_fails "limit must be a positive integer" "a zero limit fails the render" \
  --set policies.bad.enabled=true --set policies.bad.schedule='@daily' \
  --set policies.bad.action=prune --set policies.bad.limit=0

assert_render_fails "a sync policy needs at least one of" "a sync policy with no routing criterion fails the render" \
  --set policies.bad.enabled=true --set policies.bad.schedule='@daily' \
  --set policies.bad.action=sync --set policies.bad.match.trusted=false

if [ "$FAILURES" -ne 0 ]; then
  echo
  echo "$FAILURES assertion(s) failed"
  exit 1
fi

echo
echo "all policy template assertions passed"
