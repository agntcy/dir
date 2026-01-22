"""
Data models for the inspector service.
"""

import json
from dataclasses import dataclass, field
from typing import Optional


@dataclass
class Workload:
    """Workload data from runtime storage."""
    id: str
    name: str
    hostname: str
    runtime: str
    workload_type: str
    addresses: list = field(default_factory=list)
    isolation_groups: list = field(default_factory=list)
    ports: list = field(default_factory=list)
    labels: dict = field(default_factory=dict)
    annotations: dict = field(default_factory=dict)
    node: Optional[str] = None
    namespace: Optional[str] = None
    metadata: dict = field(default_factory=dict)
    registrar: Optional[str] = None
    
    @classmethod
    def from_dict(cls, data: dict) -> "Workload":
        return cls(
            id=data.get("id", ""),
            name=data.get("name", ""),
            hostname=data.get("hostname", ""),
            runtime=data.get("runtime", ""),
            workload_type=data.get("workload_type", data.get("type", "")),
            addresses=data.get("addresses", []),
            isolation_groups=data.get("isolation_groups", []),
            ports=data.get("ports", []),
            labels=data.get("labels", {}),
            annotations=data.get("annotations", {}),
            node=data.get("node"),
            namespace=data.get("namespace"),
            metadata=data.get("metadata", {}),
            registrar=data.get("registrar"),
        )
    
    @classmethod
    def from_json(cls, data: str) -> "Workload":
        return cls.from_dict(json.loads(data))

