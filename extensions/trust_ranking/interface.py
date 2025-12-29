"""
Trust ranking interface (reference only).

This defines the minimal contract a trust ranker must implement.
It is intentionally simple and non-prescriptive.
"""

from typing import List, Dict, Any


def rank_agents(
    agents: List[Dict[str, Any]],
    query: Dict[str, Any] | None = None,
    context: Dict[str, Any] | None = None,
) -> List[Dict[str, Any]]:
    """
    Rank a list of agent directory entries.

    Parameters:
        agents: list of agent-like dicts from a directory
        query: optional user or agent query context
        context: optional execution or environment context

    Returns:
        The same agents, ordered by preference.
        Each agent MAY include a 'trust' field with scoring metadata.
    """
    raise NotImplementedError("Trust ranker not implemented")
