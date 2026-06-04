"""Print mike version: local from mike_versions.ini; release from VERSION env (CI)."""

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

    if args.mode == "local":
        ini = Path(__file__).resolve().parent / "mike_versions.ini"
        cfg = configparser.ConfigParser()
        if not cfg.read(ini, encoding="utf-8"):
            raise SystemExit(f"could not read {ini}")
        raw = cfg.get("versions", "local", fallback="dev")
        out = strip_leading_v(raw) or "dev"
        print(out)
        return

    env = os.environ.get("VERSION", "").strip()
    if not env:
        raise SystemExit(
            "VERSION is required (docs-deploy workflow sets it from the tag or release)"
        )
    print(strip_leading_v(env))


if __name__ == "__main__":
    main()
