#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"
: "${GH_TOKEN:?GH_TOKEN is required}"

# Signing is keyless, so the certificate is bound to whoever runs this. Run it
# from CI to get the repository's identity rather than a maintainer's.
if [ -n "${TAGS:-}" ]; then
  IFS=',' read -ra tags <<<"$TAGS"
else
  # Scorecard's Signed-Releases check only inspects the five newest releases.
  mapfile -t tags < <(gh release list --repo "$GITHUB_REPOSITORY" --limit 5 \
    --exclude-drafts --json tagName --jq '.[].tagName')
fi

for tag in "${tags[@]}"; do
  tag="${tag//[[:space:]]/}"
  [ -n "$tag" ] || continue

  echo "::group::$tag"
  workdir="$(mktemp -d)"

  if ! gh release download "$tag" --repo "$GITHUB_REPOSITORY" \
    --dir "$workdir" --pattern 'dirctl-*'; then
    echo "no dirctl assets on $tag, skipping"
    rm -rf "$workdir"
    echo "::endgroup::"
    continue
  fi

  for asset in "$workdir"/dirctl-*; do
    # A re-run downloads the bundles the previous run attached.
    case "$asset" in
    *.sigstore.json) continue ;;
    esac

    cosign sign-blob --yes --bundle "${asset}.sigstore.json" "$asset"
  done

  gh release upload "$tag" --repo "$GITHUB_REPOSITORY" --clobber \
    "$workdir"/*.sigstore.json

  rm -rf "$workdir"
  echo "::endgroup::"
done
