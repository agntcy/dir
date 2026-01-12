"""
Unified data models for multi-runtime service discovery.
"""

from dataclasses import dataclass, field, asdict
from enum import Enum
from typing import Optional
import json


class Runtime(Enum):
    DOCKER = "docker"
    CONTAINERD = "containerd"
    KUBERNETES = "kubernetes"


class WorkloadType(Enum):
    CONTAINER = "container"  # Docker/containerd container
    POD = "pod"              # K8s Pod
    SERVICE = "service"      # K8s Service (virtual endpoint)


class EventType(Enum):
    """Types of workload events emitted by runtime adapters."""
    ADDED = "added"
    MODIFIED = "modified"
    DELETED = "deleted"
    NETWORK_CHANGED = "network_changed"


@dataclass
class Workload:
    """
    Unified workload representation across all runtimes.
    
    This is the core data structure stored in etcd and returned by queries.
    """
    
    # Identity
    id: str                              # Unique ID (container ID, pod UID, service UID)
    name: str                            # Human-readable name
    hostname: str                        # What $HOSTNAME returns inside workload
    
    # Runtime info
    runtime: str                         # Runtime enum value as string
    workload_type: str                   # WorkloadType enum value as string
    
    # Location
    node: Optional[str] = None           # Node/host where running
    namespace: Optional[str] = None      # K8s namespace (None for Docker/containerd)
    
    # Network
    addresses: list = field(default_factory=list)         # ["ip:port", "hostname:port"]
    isolation_groups: list = field(default_factory=list)  # Networks (Docker/containerd) or namespaces (K8s)
    
    # Discovery metadata
    labels: dict = field(default_factory=dict)
    annotations: dict = field(default_factory=dict)
    
    # Scraped metadata (populated async)
    metadata: Optional[dict] = None
    
    # Internal tracking
    registrar: Optional[str] = None      # Which watcher instance registered this
    
    def to_dict(self) -> dict:
        """Convert to dictionary for JSON serialization."""
        return {k: v for k, v in asdict(self).items() if v is not None}
    
    def to_json(self) -> str:
        """Convert to JSON string."""
        return json.dumps(self.to_dict())
    
    @classmethod
    def from_dict(cls, data: dict) -> "Workload":
        """Create Workload from dictionary."""
        return cls(
            id=data.get("id", ""),
            name=data.get("name", ""),
            hostname=data.get("hostname", ""),
            runtime=data.get("runtime", Runtime.DOCKER.value),
            workload_type=data.get("workload_type", WorkloadType.CONTAINER.value),
            node=data.get("node"),
            namespace=data.get("namespace"),
            addresses=data.get("addresses", []),
            isolation_groups=data.get("isolation_groups", []),
            labels=data.get("labels", {}),
            annotations=data.get("annotations", {}),
            metadata=data.get("metadata"),
            registrar=data.get("registrar"),
        )
    
    @classmethod
    def from_json(cls, json_str: str) -> "Workload":
        """Create Workload from JSON string."""
        return cls.from_dict(json.loads(json_str))


@dataclass
class ReachabilityResult:
    """Result of a reachability query."""
    
    caller: Workload
    reachable: list  # List[Workload]
    count: int = 0
    
    def __post_init__(self):
        self.count = len(self.reachable)
    
    def to_dict(self) -> dict:
        return {
            "caller": self.caller.to_dict(),
            "reachable": [w.to_dict() for w in self.reachable],
            "count": self.count,
        }
