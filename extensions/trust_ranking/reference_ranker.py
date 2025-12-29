"""
Reference trust ranker (toy scoring).

This is NOT a security system.
It is a demo of how directories could incorporate trust signals
into ranking decisions in an explainable way.
"""

from __future__ import annotations

from typing import List, Dict, Any, Tuple
from datetime import datetime, timezone


def _parse_date_yyyy_mm_dd(s: str | None) -> datetime | None:
    if not s:
        return None
    try:
        return datetime.strptime(s, "%Y-%m-%d").replace(tzinfo=timezone.utc)
    except ValueError:
        return None


def _clamp(x: float, lo: float, hi: float) -> float:
    return max(lo, min(hi, x))


def _completeness(agent: Dict[str, Any]) -> Tuple[float, List[str]]:
    """
    Completeness based on presence of common directory fields.
    Returns score 0..1 and reasons.
    """
    required = ["id", "name", "url", "capabilities", "contact", "updated_at"]
    present = 0
    missing = []

    for k in required:
        v = agent.get(k)
        ok = v is not None and v != "" and (v != [] if k == "capabilities" else True)
        if ok:
            present += 1
        else:
            missing.append(k)

    score = present / float(len(required))
    reasons = []
    if score >= 0.9:
        reasons.append("Profile is complete")
    elif score >= 0.6:
        reasons.append("Profile is somewhat complete")
    else:
        reasons.append("Profile is missing key fields")

    if missing:
        reasons.append("Missing: " + ", ".join(missing[:3]) + ("..." if len(missing) > 3 else ""))

    return score, reasons


def _freshness(agent: Dict[str, Any]) -> Tuple[float, List[str]]:
    """
    Freshness score based on updated_at. 0..1.
    Newer is better. Very old is bad.
    """
    dt = _parse_date_yyyy_mm_dd(agent.get("updated_at"))
    if not dt:
        return 0.2, ["No valid updated_at date"]

    now = datetime.now(timezone.utc)
    days = (now - dt).days

    # Simple buckets
    if days <= 30:
        return 1.0, ["Recently updated"]
    if days <= 120:
        return 0.7, ["Updated this quarter"]
    if days <= 365:
        return 0.4, ["Updated within a year"]

    return 0.1, ["Stale profile"]


def _verification(agent: Dict[str, Any]) -> Tuple[float, List[str]]:
    """
    Verification score based on flags. 0..1.
    """
    domain_verified = bool(agent.get("domain_verified"))
    key_present = bool(agent.get("key_present"))

    score = 0.0
    reasons = []

    if domain_verified:
        score += 0.6
        reasons.append("Domain verified")
    else:
        reasons.append("Domain not verified")

    if key_present:
        score += 0.4
        reasons.append("Key present")
    else:
        reasons.append("No key")

    return score, reasons


def _behavior(agent: Dict[str, Any]) -> Tuple[float, List[str]]:
    """
    Behavior score from simulated hints. 0..1.
    Lower failures/violations/complaints is better.
    """
    fail_ratio = agent.get("handshake_fail_ratio")
    violations = agent.get("rate_limit_violations")
    complaints = agent.get("complaint_flags")

    # defaults if absent
    try:
        fail_ratio = float(fail_ratio) if fail_ratio is not None else 0.10
    except (TypeError, ValueError):
        fail_ratio = 0.10

    try:
        violations = int(violations) if violations is not None else 0
    except (TypeError, ValueError):
        violations = 0

    try:
        complaints = int(complaints) if complaints is not None else 0
    except (TypeError, ValueError):
        complaints = 0

    # Convert to penalties (toy)
    # Fail ratio: 0.0 -> 0 penalty, 0.5 -> heavy penalty
    fail_pen = _clamp(fail_ratio / 0.5, 0.0, 1.0)

    # Violations: 0 -> 0 penalty, 25+ -> heavy penalty
    viol_pen = _clamp(violations / 25.0, 0.0, 1.0)

    # Complaints: 0 -> 0 penalty, 10+ -> heavy penalty
    comp_pen = _clamp(complaints / 10.0, 0.0, 1.0)

    penalty = 0.5 * fail_pen + 0.3 * viol_pen + 0.2 * comp_pen
    score = 1.0 - _clamp(penalty, 0.0, 1.0)

    reasons = []
    if fail_ratio >= 0.30:
        reasons.append("High handshake failure rate")
    elif fail_ratio <= 0.05:
        reasons.append("Low handshake failure rate")

    if violations >= 10:
        reasons.append("Many rate limit violations")
    elif violations == 0:
        reasons.append("No rate limit violations")

    if complaints >= 3:
        reasons.append("Multiple complaint flags")
    elif complaints == 0:
        reasons.append("No complaint flags")

    return score, reasons


def _band(score_0_100: float) -> str:
    if score_0_100 >= 80:
        return "green"
    if score_0_100 >= 50:
        return "yellow"
    return "red"


def _top_reasons(reasons: List[str], limit: int = 3) -> List[str]:
    # Keep unique, preserve order
    out = []
    seen = set()
    for r in reasons:
        r = r.strip()
        if not r or r in seen:
            continue
        out.append(r)
        seen.add(r)
        if len(out) >= limit:
            break
    return out


def rank_agents(
    agents: List[Dict[str, Any]],
    query: Dict[str, Any] | None = None,
    context: Dict[str, Any] | None = None,
) -> List[Dict[str, Any]]:
    """
    Toy ranker. Produces:
      trust.score 0..100
      trust.band  green|yellow|red
      trust.reasons[] (top 3, human-readable)

    Returns ranked list (descending trust.score).
    """
    scored: List[Tuple[float, Dict[str, Any]]] = []

    for agent in agents:
        a = dict(agent)

        comp, comp_r = _completeness(a)
        fresh, fresh_r = _freshness(a)
        ver, ver_r = _verification(a)
        beh, beh_r = _behavior(a)

        # Weights (toy). Sum to 1.0.
        score_0_1 = 0.35 * comp + 0.20 * fresh + 0.25 * ver + 0.20 * beh
        score_0_100 = round(_clamp(score_0_1, 0.0, 1.0) * 100.0, 1)

        # Build an explanation that covers different categories.
        # We want: completeness, freshness, and either verification OR behavior,
        # but behavior should show up when it has something to say.
        comp_pick = comp_r[:1]
        fresh_pick = fresh_r[:1]
        ver_pick = ver_r[:1]
        beh_pick = beh_r[:1]

        # Start with completeness + freshness
        preferred = comp_pick + fresh_pick

        # Then prefer behavior if it's informative (not empty)
        if beh_pick:
            preferred += beh_pick
        else:
            preferred += ver_pick

        # Fill remaining slots from everything else, preserving uniqueness
        reasons_all = comp_r + fresh_r + ver_r + beh_r
        reasons = _top_reasons(preferred + reasons_all, limit=3)

        a["trust"] = {
            "score": score_0_100,
            "band": _band(score_0_100),
            "reasons": reasons,
        }

        scored.append((score_0_100, a))

    # Highest score first
    scored.sort(key=lambda t: t[0], reverse=True)
    return [a for _, a in scored]
