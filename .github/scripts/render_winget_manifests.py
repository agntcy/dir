#!/usr/bin/env python3
# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

"""Render the WinGet manifests for a dirctl release.

The manifests describe the Windows zip the release workflow uploads
(dirctl-<tag>-windows-amd64.zip, holding dirctl.exe) as a zip installer with a
nested portable executable. The installer hash is computed from the published
asset, so run this after the release is public.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
import urllib.request
from pathlib import Path
from urllib.error import HTTPError

MANIFEST_VERSION = "1.12.0"
PACKAGE_IDENTIFIER = "AGNTCY.dirctl"
PACKAGE_LOCALE = "en-US"
PUBLISHER = "AGNTCY"
REPOSITORY = "agntcy/dir"
DOCS_URL = "https://docs.agntcy.org/dir/getting-started/"

# Stable releases only: WinGet versions are what users install by default.
TAG_PATTERN = re.compile(r"^v(?P<version>\d+\.\d+\.\d+)$")
DATE_PATTERN = re.compile(r"^\d{4}-\d{2}-\d{2}$")
SHA256_PATTERN = re.compile(r"^[0-9A-F]{64}$")
URL_PATTERN = re.compile(r"^https://github\.com/[\w.-]+/[\w.-]+/releases/download/[^\s]+\.zip$")


def installer_name(version: str) -> str:
    return f"dirctl-v{version}-windows-amd64.zip"


def github_headers() -> dict[str, str]:
    headers = {"Accept": "application/vnd.github+json", "User-Agent": "dirctl-winget-manifests"}
    token = os.environ.get("GITHUB_TOKEN") or os.environ.get("GH_TOKEN")
    if token:
        headers["Authorization"] = f"Bearer {token}"
    return headers


def open_url(url: str):
    request = urllib.request.Request(url, headers=github_headers())
    try:
        return urllib.request.urlopen(request, timeout=120)  # noqa: S310 - https GitHub URLs only
    except HTTPError as error:
        raise SystemExit(f"failed to fetch {url}: HTTP {error.code}") from error


def sha256_of_stream(stream) -> str:
    digest = hashlib.sha256()
    while chunk := stream.read(1024 * 1024):
        digest.update(chunk)
    return digest.hexdigest().upper()


def fetch_release(tag: str) -> dict:
    with open_url(f"https://api.github.com/repos/{REPOSITORY}/releases/tags/{tag}") as response:
        return json.loads(response.read())


def render(version: str, release_date: str, installer_url: str, installer_sha256: str, tag: str) -> dict[str, str]:
    """Return the file name and content of each manifest."""
    release_url = f"https://github.com/{REPOSITORY}/releases/tag/{tag}"

    version_manifest = f"""# yaml-language-server: $schema=https://aka.ms/winget-manifest.version.{MANIFEST_VERSION}.schema.json

PackageIdentifier: {PACKAGE_IDENTIFIER}
PackageVersion: {version}
DefaultLocale: {PACKAGE_LOCALE}
ManifestType: version
ManifestVersion: {MANIFEST_VERSION}
"""

    installer_manifest = f"""# yaml-language-server: $schema=https://aka.ms/winget-manifest.installer.{MANIFEST_VERSION}.schema.json

PackageIdentifier: {PACKAGE_IDENTIFIER}
PackageVersion: {version}
InstallerType: zip
NestedInstallerType: portable
Commands:
- dirctl
ReleaseDate: {release_date}
Installers:
- Architecture: x64
  InstallerUrl: {installer_url}
  InstallerSha256: {installer_sha256}
  NestedInstallerFiles:
  - RelativeFilePath: dirctl.exe
    PortableCommandAlias: dirctl
ManifestType: installer
ManifestVersion: {MANIFEST_VERSION}
"""

    locale_manifest = f"""# yaml-language-server: $schema=https://aka.ms/winget-manifest.defaultLocale.{MANIFEST_VERSION}.schema.json

PackageIdentifier: {PACKAGE_IDENTIFIER}
PackageVersion: {version}
PackageLocale: {PACKAGE_LOCALE}
Publisher: {PUBLISHER}
PublisherUrl: https://github.com/agntcy
PublisherSupportUrl: https://github.com/{REPOSITORY}/issues
Author: AGNTCY Contributors
PackageName: dirctl
PackageUrl: https://github.com/{REPOSITORY}
License: Apache-2.0
LicenseUrl: https://github.com/{REPOSITORY}/blob/{tag}/LICENSE.md
ShortDescription: Command-line client for AGNTCY Directory, to publish, sign, search and verify agent records.
Moniker: dirctl
Tags:
- a2a
- agent
- agntcy
- cli
- mcp
- oasf
Documentations:
- DocumentLabel: Documentation
  DocumentUrl: {DOCS_URL}
ReleaseNotesUrl: {release_url}
ManifestType: defaultLocale
ManifestVersion: {MANIFEST_VERSION}
"""

    return {
        f"{PACKAGE_IDENTIFIER}.yaml": version_manifest,
        f"{PACKAGE_IDENTIFIER}.installer.yaml": installer_manifest,
        f"{PACKAGE_IDENTIFIER}.locale.{PACKAGE_LOCALE}.yaml": locale_manifest,
    }


def write_github_output(path: Path, values: dict[str, str]) -> None:
    with path.open("a", encoding="utf-8") as handle:
        for key, value in values.items():
            handle.write(f"{key}={value}\n")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Render WinGet manifests for a dirctl release.")
    parser.add_argument("--tag", required=True, help="Release tag, for example v1.8.0.")
    parser.add_argument("--output-dir", type=Path, default=Path("dist/winget"), help="Where to write the manifests.")
    parser.add_argument("--github-output", type=Path, help="GitHub Actions output file to append the manifest metadata to.")
    parser.add_argument("--installer-zip", type=Path, help="Use this local zip for the hash instead of downloading the release asset (testing).")
    parser.add_argument("--release-date", help="YYYY-MM-DD, instead of the release's publication date (testing).")
    return parser.parse_args()


def main() -> int:
    args = parse_args()

    match = TAG_PATTERN.match(args.tag)
    if not match:
        raise SystemExit(f"expected a stable release tag like v1.8.0, got: {args.tag}")
    version = match.group("version")

    name = installer_name(version)
    installer_url = f"https://github.com/{REPOSITORY}/releases/download/{args.tag}/{name}"

    if args.installer_zip:
        with args.installer_zip.open("rb") as handle:
            installer_sha256 = sha256_of_stream(handle)
        release_date = args.release_date
        if not release_date:
            raise SystemExit("--release-date is required with --installer-zip")
    else:
        release = fetch_release(args.tag)
        assets = {asset["name"]: asset["browser_download_url"] for asset in release.get("assets", [])}
        if name not in assets:
            raise SystemExit(f"release {args.tag} has no asset {name}; it must be built by the release workflow")
        if release.get("draft"):
            raise SystemExit(f"release {args.tag} is still a draft; publish it first so the installer URL is public")
        if release.get("prerelease"):
            raise SystemExit(f"release {args.tag} is a prerelease; only stable releases are published to WinGet")
        with open_url(assets[name]) as response:
            installer_sha256 = sha256_of_stream(response)
        release_date = release["published_at"][:10]

    if not DATE_PATTERN.match(release_date):
        raise SystemExit(f"bad release date: {release_date}")
    if not SHA256_PATTERN.match(installer_sha256):
        raise SystemExit(f"bad installer hash: {installer_sha256}")
    if not URL_PATTERN.match(installer_url):
        raise SystemExit(f"bad installer URL: {installer_url}")

    package_dir = "/".join(["manifests", PACKAGE_IDENTIFIER[0].lower(), *PACKAGE_IDENTIFIER.split("."), version])
    target = args.output_dir / package_dir
    target.mkdir(parents=True, exist_ok=True)

    for file_name, content in render(version, release_date, installer_url, installer_sha256, args.tag).items():
        (target / file_name).write_text(content, encoding="utf-8", newline="\n")
        print(f"wrote {target / file_name}")

    if args.github_output:
        write_github_output(
            args.github_output,
            {
                "manifest_rel_dir": package_dir,
                "package_identifier": PACKAGE_IDENTIFIER,
                "package_version": version,
                "release_url": f"https://github.com/{REPOSITORY}/releases/tag/{args.tag}",
                "installer_sha256": installer_sha256,
            },
        )

    return 0


if __name__ == "__main__":
    sys.exit(main())
