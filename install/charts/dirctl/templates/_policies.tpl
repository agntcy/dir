{{/*
Copyright AGNTCY Contributors (https://github.com/agntcy)
SPDX-License-Identifier: Apache-2.0
*/}}

{{/*
Single-quote a match value for the generated shell script.

A value containing a single quote would break out of the quoting, so it is
rejected at template time rather than producing a broken CronJob.

Input: dict with "name" (policy name), "key" (match key) and "value".
*/}}
{{- define "chart.policy.shellQuote" -}}
{{- $value := printf "%v" .value -}}
{{- if contains "'" $value -}}
{{- fail (printf "policy %q: match value for %q contains a single quote and cannot be safely shell-quoted: %s" .name .key $value) -}}
{{- end -}}
{{- printf "'%s'" $value -}}
{{- end -}}

{{/*
Render a policy's `match` map into a dirctl flag string.

Keys are dirctl flag names without the leading "--", so any filter the CLI grows
works without a chart change. Values render by kind:
  list   -> the flag repeated once per item
  bool   -> --flag=true / --flag=false
  scalar -> --flag 'value'

`limit`, `format` and `output` are set by the chart itself and are rejected here
so a policy cannot silently emit a duplicate flag.

Input: dict with "name" (policy name) and "match" (the map).
*/}}
{{- define "chart.policy.matchFlags" -}}
{{- $name := .name -}}
{{- $rendered := list -}}
{{- range $key, $value := .match -}}
{{-   if has $key (list "limit" "offset" "format" "output") -}}
{{-     fail (printf "policy %q: match key %q is set by the chart; use the policy's own fields instead" $name $key) -}}
{{-   end -}}
{{/*
  Keys are interpolated into the generated shell command unquoted, so they are
  constrained to the CLI's flag-name grammar. Without this a key such as
  "name; rm -rf /" would render as executable shell rather than a flag.
*/}}
{{-   if not (regexMatch "^[a-z0-9][a-z0-9-]*$" $key) -}}
{{-     fail (printf "policy %q: match key %q is not a valid dirctl flag name (expected lowercase letters, digits and hyphens)" $name $key) -}}
{{-   end -}}
{{-   if kindIs "slice" $value -}}
{{-     range $item := $value -}}
{{-       $rendered = append $rendered (printf "--%s %s" $key (include "chart.policy.shellQuote" (dict "name" $name "key" $key "value" $item))) -}}
{{-     end -}}
{{-   else if kindIs "bool" $value -}}
{{-     $rendered = append $rendered (printf "--%s=%t" $key $value) -}}
{{-   else -}}
{{-     $rendered = append $rendered (printf "--%s %s" $key (include "chart.policy.shellQuote" (dict "name" $name "key" $key "value" $value))) -}}
{{-   end -}}
{{- end -}}
{{- join " " $rendered -}}
{{- end -}}

{{/*
Render the shell script that implements a policy.

Every action is the same shape: search, then act on the matches. The match count
is taken before the sink runs because `dirctl delete`, `dirctl routing publish`
and `dirctl routing unpublish` reject empty stdin ("at least one CID is
required"), so a policy that matches nothing would otherwise fail every run.

An empty search result is checked separately from a zero count. A search can exit
0 while printing nothing (`dirctl routing search` does exactly that when given no
criteria, reporting to stderr), which leaves `count` empty; `[ "" -eq 0 ]` then
errors, and because that error is an `if` condition `set -e` does not stop the
script, so it would fall through to the sink with empty input.

Pagination differs by action, because only some of them change whether a record
still matches:
  prune            deleting a record stops it matching, so the set shrinks and
                   offset 0 always sees the remaining work.
  publish
  unpublish        the action does not change the record, so the same first page
                   would be reprocessed forever. These loop over --offset.
  sync             `dirctl routing search` has no --offset, so it cannot page.
                   `limit` must exceed the expected match count.

There is no `set -o pipefail`: the container shell is busybox ash. Capturing the
search output in a variable under `set -eu` surfaces a failing search just as
well, and yields the count and the dry-run dump for free.

Input: dict with "name" (policy name) and "policy" (the policy map).
*/}}
{{- define "chart.policy.script" -}}
{{- $name := .name -}}
{{- $policy := .policy -}}
{{- $action := $policy.action | default "" -}}
{{/*
  Not `$policy.limit | default 100`: sprig's default treats 0 as empty, which
  would silently turn an explicit `limit: 0` into 100 instead of rejecting it.
*/}}
{{- $limit := 100 -}}
{{- if hasKey $policy "limit" -}}
{{-   $limit = $policy.limit -}}
{{- end -}}
{{- $match := $policy.match | default dict -}}
{{/*
  `limit` is interpolated into the shell command unquoted, so it is constrained to
  a positive decimal rather than trusted verbatim.
*/}}
{{- if not (regexMatch "^[1-9][0-9]*$" (printf "%v" $limit)) -}}
{{-   fail (printf "policy %q: limit must be a positive integer, got %v" $name $limit) -}}
{{- end -}}
{{- $flags := include "chart.policy.matchFlags" (dict "name" $name "match" $match) -}}
{{- $source := "" -}}
{{- $sink := "" -}}
{{- $paginate := false -}}
{{- if eq $action "sync" -}}
{{/*
    `dirctl routing search` needs at least one routing criterion; with none it
    exits 0 having printed nothing, which is not a failure the script can report
    usefully.
  */}}
{{-   if not (or (hasKey $match "skill") (hasKey $match "domain") (hasKey $match "locator") (hasKey $match "module")) -}}
{{-     fail (printf "policy %q: a sync policy needs at least one of match.skill, match.domain, match.locator or match.module (dirctl routing search requires a criterion)" $name) -}}
{{-   end -}}
{{-   $source = printf "dirctl routing search --output json --limit %v %s" $limit $flags | trim -}}
{{-   $sink = "dirctl sync create --stdin" -}}
{{- else if eq $action "prune" -}}
{{-   $source = printf "dirctl search --format cid --output json --limit %v %s" $limit $flags | trim -}}
{{-   $sink = "dirctl delete --stdin" -}}
{{- else if eq $action "publish" -}}
{{-   $source = printf "dirctl search --format cid --output json --limit %v %s" $limit $flags | trim -}}
{{-   $sink = "dirctl routing publish --stdin" -}}
{{-   $paginate = true -}}
{{- else if eq $action "unpublish" -}}
{{-   $source = printf "dirctl search --format cid --output json --limit %v %s" $limit $flags | trim -}}
{{-   $sink = "dirctl routing unpublish --stdin" -}}
{{-   $paginate = true -}}
{{- else -}}
{{-   fail (printf "policy %q: unknown action %q (must be one of: sync, prune, publish, unpublish)" $name $action) -}}
{{- end -}}
{{- if $paginate -}}
set -eu
offset=0
total=0
while :; do
  out="$({{ $source }} --offset "$offset")"
  if [ -z "$out" ]; then
    echo "[{{ $name }}] search returned no output" >&2
    exit 1
  fi
  count="$(printf '%s' "$out" | jq 'length')"
  if [ "$count" -eq 0 ]; then
    break
  fi
  echo "[{{ $name }}] matched ${count} record(s) at offset ${offset}"
  printf '%s' "$out" | jq .
  {{ if $policy.dryRun }}echo "[{{ $name }}] dry run: not applying action '{{ $action }}'"{{ else }}printf '%s' "$out" | {{ $sink }}{{ end }}
  total=$((total + count))
  if [ "$count" -lt {{ $limit }} ]; then
    break
  fi
  offset=$((offset + {{ $limit }}))
done
echo "[{{ $name }}] processed ${total} record(s)"
{{- else -}}
set -eu
out="$({{ $source }})"
if [ -z "$out" ]; then
  echo "[{{ $name }}] search returned no output" >&2
  exit 1
fi
count="$(printf '%s' "$out" | jq 'length')"
echo "[{{ $name }}] matched ${count} record(s)"
if [ "$count" -eq 0 ]; then
  exit 0
fi
printf '%s' "$out" | jq .
{{ if $policy.dryRun }}echo "[{{ $name }}] dry run: not applying action '{{ $action }}'"{{ else }}printf '%s' "$out" | {{ $sink }}{{ end }}
{{- end -}}
{{- end -}}
