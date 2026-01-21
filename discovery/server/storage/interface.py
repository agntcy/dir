"""
Storage interface for discovery server (read-only).

The server only needs to query workloads from etcd.
"""

from abc import ABC, abstractmethod
from typing import Optional
from models import Workload, ReachabilityResult


class StorageInterface(ABC):
    """
    Abstract interface for workload storage (read operations only).
    
    The server queries workloads and computes reachability.
    """
    
    @abstractmethod
    def connect(self) -> bool:
        """
        Connect to storage backend.
        Returns True if connected successfully.
        """
        pass
    
    @abstractmethod
    def close(self):
        """Close connection to storage backend."""
        pass
    
    @abstractmethod
    def get(self, workload_id: str) -> Optional[Workload]:
        """Get a workload by ID."""
        pass
    
    @abstractmethod
    def get_by_hostname(self, hostname: str) -> Optional[Workload]:
        """Get a workload by hostname."""
        pass
    
    @abstractmethod
    def get_by_name(self, name: str, namespace: str = None) -> Optional[Workload]:
        """Get a workload by name (and namespace for K8s)."""
        pass
    
    @abstractmethod
    def list_all(self, runtime: str = None, label_filter: dict = None) -> list:
        """
        List all workloads, optionally filtered by runtime or labels.
        Returns List[Workload].
        """
        pass
    
    @abstractmethod
    def find_reachable(
        self,
        caller_identity: str,
    ) -> ReachabilityResult:
        """
        Find all workloads reachable from caller.
        
        Args:
            caller_identity: Hostname, name, or ID of caller
            
        Returns:
            ReachabilityResult with caller and reachable workloads
        """
        pass
