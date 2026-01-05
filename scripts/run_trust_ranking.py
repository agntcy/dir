#!/usr/bin/env python3
"""
Run the reference trust ranking demo.

Example:
  python scripts/run_trust_ranking.py --top 10
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

# Allow running from repo root without installing as a package
REPO_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO_ROOT))

from extensions.trust_ranking.reference_ranker import rank_agents  # noqa: E402


def _load_agents(path: Path) -> list[dict]:
    data = json.loads(path.read_text(encoding="utf-8"))
    agents = data.get("agents")
    if not isinstance(agents, list):
        raise ValueError("Input JSON must contain an 'agents' list")
    return agents


def main() -> int:
    parser = argparse.ArgumentParser(description="Trust ranking PoC runner (reference only)")
    parser.add_argument(
        "--input",
        default="examples/directory_sample.json",
        help="Path to JSON file containing {'agents': [...]}",
    )
    parser.add_argument("--top", type=int, default=10, help="How many results to print")
    parser.add_argument("--json", action="store_true", help="Output full ranked list as JSON")
    args = parser.parse_args()

    input_path = (REPO_ROOT / args.input).resolve()
    if not input_path.exists():
        print(f"ERROR: input file not found: {input_path}", file=sys.stderr)
        return 2

    try:
        agents = _load_agents(input_path)
        ranked = rank_agents(agents)
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        return 2

    if args.json:
        print(json.dumps({"agents": ranked}, indent=2, ensure_ascii=False))
        return 0

    top_n = max(0, min(args.top, len(ranked)))

    print("Trust Ranking PoC (reference only)")
    print(f"Input: {args.input}")
    print(f"Results: top {top_n} of {len(ranked)}")
    print("")

    for i, a in enumerate(ranked[:top_n], start=1):
        trust = a.get("trust") or {}
        score = trust.get("score", "n/a")
        band = trust.get("band", "n/a")
        reasons = trust.get("reasons", [])
        name = a.get("name") or a.get("id") or "(unnamed)"
        url = a.get("url") or ""

        reasons_str = "; ".join(reasons) if isinstance(reasons, list) else str(reasons)

        print(f"{i:>2}. {name}")
        print(f"    id: {a.get('id')}")
        print(f"   url: {url}")
        print(f" trust: {score} ({band})")
        print(f"reason: {reasons_str}")
        print("")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
