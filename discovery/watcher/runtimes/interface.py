"""
Abstract runtime interface for container runtimes.

All runtime adapters must implement this interface.
Runtime adapters are responsible for:
- Listing workloads from their runtime
- Watching for workload events
- Converting runtime-specific data to unified Workload model

Runtime adapters do NOT handle reachability queries - that's the storage layer's job.
"""

from abc import ABC, abstractmethod
from typing import Optional, Callable

from models import Workload, Runtime, EventType


class RuntimeAdapter(ABC):
    """
    Abstract interface for container runtime adapters.
    
    Each runtime (Docker, containerd, Kubernetes) implements this interface
    to provide a unified way to discover and watch workloads.
    """
    
    @property
    @abstractmethod
    def runtime_type(self) -> Runtime:
        """Return which runtime this adapter handles."""
        pass
    
    @abstractmethod
    def connect(self) -> bool:
        """
        Connect to the runtime.
        Returns True if connected successfully.
        """
        pass
    
    @abstractmethod
    def close(self):
        """Close connection to runtime."""
        pass
    
    # ==================== Discovery ====================
    
    @abstractmethod
    def list_workloads(self) -> list:
        """
        List all discoverable workloads.
        
        Args:
            label_selector: Optional dict of labels to filter by
            
        Returns:
            List[Workload] of discoverable workloads
        """
        pass
    
    @abstractmethod
    def watch_events(self, callback: Callable[[EventType, Workload], None]) -> None:
        """
        Watch for workload events and call callback for each.
        
        This should run in a loop until stopped.
        
        Args:
            callback: Function called with (event_type, workload) for each event
                     event_type is one of the EventType enum values
        """
        pass
