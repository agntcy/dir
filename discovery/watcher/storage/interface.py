"""
Abstract storage interface for workload discovery.

All storage implementations must implement this interface.
The storage layer is responsible for:
- Storing workload data
- Building and maintaining indices
- Answering reachability queries
"""

from abc import ABC, abstractmethod
from typing import Optional
from models import Workload, ReachabilityResult


class StorageInterface(ABC):
    """
    Abstract interface for workload storage.
    
    Implementations handle data persistence and reachability queries.
    Runtime adapters only read/write workloads; query logic lives here.
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
    
    # ==================== Write Operations ====================
    
    @abstractmethod
    def register(self, workload: Workload) -> bool:
        """
        Register or update a workload.
        Idempotent - overwrites if exists.
        Returns True on success.
        """
        pass
    
    @abstractmethod
    def deregister(self, workload_id: str) -> bool:
        """
        Remove a workload by ID.
        Idempotent - no error if not exists.
        Returns True on success.
        """
        pass
    
    @abstractmethod
    def set_metadata(self, workload_id: str, metadata: dict) -> bool:
        """
        Set scraped metadata for a workload.
        Returns True on success.
        """
        pass
    
    # ==================== Read Operations ====================
    
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
    def list_by_isolation_group(self, group: str) -> list:
        """
        List all workloads in an isolation group (network/namespace).
        Returns List[Workload].
        """
        pass
    
    # ==================== Reachability Queries ====================
    
    @abstractmethod
    def find_reachable(
        self,
        caller_identity: str,
        include_services: bool = True,
    ) -> ReachabilityResult:
        """
        Find all workloads reachable from caller.
        
        Args:
            caller_identity: Hostname, name, or ID of caller
            include_services: Whether to include K8s Services
            
        Returns:
            ReachabilityResult with caller and reachable workloads
        """
        pass
    
    @abstractmethod
    def can_reach(self, from_id: str, to_id: str) -> tuple:
        """
        Check if from_workload can reach to_workload.
        
        Args:
            from_id: Source workload ID
            to_id: Target workload ID
            
        Returns (can_reach: bool, reason: str).
        """
        pass
    
    # ==================== Index Management ====================
    
    @abstractmethod
    def rebuild_indices(self):
        """
        Rebuild all in-memory indices from storage.
        Called on startup and after errors.
        """
        pass
    
    @abstractmethod
    def get_stats(self) -> dict:
        """
        Get storage statistics.
        Returns dict with counts, index sizes, etc.
        """
        pass
