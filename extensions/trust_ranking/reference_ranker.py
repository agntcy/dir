"""
Reference trust ranker (stub).

This implementation does NOT perform real trust evaluation.
It exists only to demonstrate the interface.
"""

from typing import List, Dict, Any
from .interface import rank_agents as _interface  # for signature reference


def rank_agents(
    agents: List[Dict[str, Any]],
    query: Dict[str, Any] | None = None,
    context: Dict[str, Any] | None = None,
) -> List[Dict[str, Any]]:
    """
    Stub implementation.

    Returns agents in original order with a neutral trust annotation.
    """
    ranked = []

    for agent in agents:
        agent_copy = dict(agent)
        agent_copy["trust"] = {
            "score": 50,
            "band": "yellow",
            "reasons": ["reference stub"],
        }
        ranked.append(agent_copy)

    return ranked
