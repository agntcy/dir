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
{{-   if has $key (list "limit" "format" "output") -}}
{{-     fail (printf "policy %q: match key %q is set by the chart; use the policy's own fields instead" $name $key) -}}
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

There is no `set -o pipefail`: the container shell is busybox ash. Capturing the
search output in a variable under `set -eu` surfaces a failing search just as
well, and yields the count and the dry-run dump for free.

Input: dict with "name" (policy name) and "policy" (the policy map).
*/}}
{{- define "chart.policy.script" -}}
{{- $name := .name -}}
{{- $policy := .policy -}}
{{- $action := $policy.action | default "" -}}
{{- $limit := $policy.limit | default 100 -}}
{{- $flags := include "chart.policy.matchFlags" (dict "name" $name "match" ($policy.match | default dict)) -}}
{{- $source := "" -}}
{{- $sink := "" -}}
{{- if eq $action "sync" -}}
{{-   $source = printf "dirctl routing search --output json --limit %v %s" $limit $flags | trim -}}
{{-   $sink = "dirctl sync create --stdin" -}}
{{- else if eq $action "prune" -}}
{{-   $source = printf "dirctl search --format cid --output json --limit %v %s" $limit $flags | trim -}}
{{-   $sink = "dirctl delete --stdin" -}}
{{- else if eq $action "publish" -}}
{{-   $source = printf "dirctl search --format cid --output json --limit %v %s" $limit $flags | trim -}}
{{-   $sink = "dirctl routing publish --stdin" -}}
{{- else if eq $action "unpublish" -}}
{{-   $source = printf "dirctl search --format cid --output json --limit %v %s" $limit $flags | trim -}}
{{-   $sink = "dirctl routing unpublish --stdin" -}}
{{- else -}}
{{-   fail (printf "policy %q: unknown action %q (must be one of: sync, prune, publish, unpublish)" $name $action) -}}
{{- end -}}
set -eu
out="$({{ $source }})"
count="$(printf '%s' "$out" | jq 'length')"
echo "[{{ $name }}] matched ${count} record(s)"
if [ "$count" -eq 0 ]; then
  exit 0
fi
printf '%s' "$out" | jq .
{{ if $policy.dryRun }}echo "[{{ $name }}] dry run: not applying action '{{ $action }}'"{{ else }}printf '%s' "$out" | {{ $sink }}{{ end }}
{{- end -}}
