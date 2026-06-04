"""Print mike version from mike_versions.ini; optional VERSION env overrides release."""

from __future__ import annotations

import argparse
import configparser
import os
from pathlib import Path


def strip_leading_v(value: str) -> str:
    value = value.strip()
    return value[1:] if value.startswith("v") else value


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("local", "release"))
    args = parser.parse_args()

    ini = Path(__file__).resolve().parent / "mike_versions.ini"
    cfg = configparser.ConfigParser()
    if not cfg.read(ini, encoding="utf-8"):
        raise SystemExit(f"could not read {ini}")

    if args.mode == "local":
        raw = cfg.get("versions", "local", fallback="dev")
        out = strip_leading_v(raw) or "dev"
        print(out)
        return

    env = os.environ.get("VERSION", "").strip()
    if env:
        print(strip_leading_v(env))
        return

    raw = cfg.get("versions", "release", fallback="").strip()
    if not raw:
        raise SystemExit(
            "Set [versions] release in docs/mike_versions.ini or export VERSION=..."
        )
    print(strip_leading_v(raw))


if __name__ == "__main__":
    main()
