# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

"""MkDocs hooks.

- Trust store for urllib (HTTPS includes in mkdocs-include-markdown-plugin).
- Inject the Directory release version into pages so docs never drift from the
  single source of truth (``versions.yaml`` at the repo root). Use the
  ``{{ dir_version }}`` placeholder in any Markdown page and it is replaced at
  build time with the current release tag (e.g. ``v1.4.0``).
"""
import re
import ssl
from pathlib import Path

import certifi

# Repo root is two levels up from docs/mkdocs/hooks.py.
_VERSIONS_FILE = Path(__file__).resolve().parents[2] / "versions.yaml"
# Match the same first `version:` entry that Taskfile.vars.yml reads.
_VERSION_RE = re.compile(r"^\s*version:\s*(\S+)\s*$", re.MULTILINE)
_PLACEHOLDER_RE = re.compile(r"\{\{\s*dir_version\s*\}\}")

_dir_version: str | None = None


def _load_dir_version() -> str:
    """Read the Directory release version from versions.yaml."""
    text = _VERSIONS_FILE.read_text(encoding="utf-8")
    match = _VERSION_RE.search(text)
    if not match:
        raise RuntimeError(
            f"Could not find a 'version:' entry in {_VERSIONS_FILE}"
        )
    return match.group(1)


def on_startup(**kwargs):
    """Configure SSL context to use certifi certificates and cache the version."""
    import urllib.request

    ssl_context = ssl.create_default_context(cafile=certifi.where())
    https_handler = urllib.request.HTTPSHandler(context=ssl_context)
    opener = urllib.request.build_opener(https_handler)
    urllib.request.install_opener(opener)

    global _dir_version
    _dir_version = _load_dir_version()


def on_page_markdown(markdown: str, **kwargs) -> str:
    """Replace the ``{{ dir_version }}`` placeholder with the release version."""
    version = _dir_version or _load_dir_version()
    return _PLACEHOLDER_RE.sub(version, markdown)
