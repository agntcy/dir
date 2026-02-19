#!/usr/bin/env bash
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

VERSION="${VERSION:-latest}"
OS="${OS:-linux}"
ARCH="${ARCH:-amd64}"
REPO="agntcy/dir"
INSTALL_DIR="${INSTALL_DIR:-$PWD/bin}"

# Resolve latest to actual dirctl release
if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -sSLf "https://api.github.com/repos/${REPO}/releases?per_page=100" \
    | jq -r '.[].tag_name' \
    | grep -E '^v[0-9]' \
    | sort -V \
    | tail -1)
  if [ -z "$VERSION" ]; then
    echo "::error::Could not resolve latest dirctl release (no v* tags found excluding gui/)" >&2
    exit 1
  fi
  echo "Resolved latest to $VERSION"
fi

BINARY="dirctl-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}"

echo "=== Setup dirctl ==="
echo "Version: $VERSION"
echo "OS: $OS"
echo "Arch: $ARCH"
echo ""

# Download to a temp file then move (avoids partial writes)
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT
curl -sSLf "$URL" -o "$TMP"
chmod +x "$TMP"

mkdir -p "$INSTALL_DIR"
mv "$TMP" "$INSTALL_DIR/dirctl"

if [ -n "${GITHUB_PATH:-}" ]; then
  echo "$INSTALL_DIR" >> "$GITHUB_PATH"
fi

echo "Installed dirctl to $INSTALL_DIR"
"$INSTALL_DIR/dirctl" version
