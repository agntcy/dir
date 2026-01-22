"""
Storage interface for workload watcher (write-only).

The watcher only needs to register/deregister workloads to etcd.
"""

from abc import ABC, abstractmethod
from typing import Set
from models import Workload


class StorageInterface(ABC):
    """
    Abstract interface for workload storage (write operations only).
    
    The watcher registers workloads discovered from runtimes.
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
    def list_workload_ids(self) -> Set[str]:
        """
        List all registered workload IDs.
        Used for reconciliation on startup.
        Returns set of workload IDs.
        """
        pass
