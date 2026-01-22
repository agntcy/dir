"""
Storage interface for metadata inspector.

The inspector reads workloads and writes metadata.
"""

from abc import ABC, abstractmethod
from typing import Optional, Generator, Tuple

from models import Workload


class StorageInterface(ABC):
    """
    Abstract interface for inspector storage.
    
    The inspector:
    - Reads/watches workloads from runtime storage
    - Writes metadata results to metadata storage
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
    
    # ==================== Runtime Reading ====================
    
    @abstractmethod
    def list_workloads(self) -> list:
        """List all workloads from runtime storage."""
        pass
    
    @abstractmethod
    def get_workload(self, workload_id: str) -> Optional[Workload]:
        """Get a specific workload by ID."""
        pass
    
    @abstractmethod
    def watch_workloads(self) -> Generator[Tuple[str, Optional[Workload]], None, None]:
        """
        Watch for workload changes.
        
        Yields:
            Tuple of (event_type, workload) where event_type is 'PUT' or 'DELETE'
        """
        pass
    
    # ==================== Metadata Writing ====================
    
    @abstractmethod
    def set_metadata(self, workload_id: str, key: str, data: dict) -> bool:
        """
        Write metadata for a workload.
        
        Args:
            workload_id: The workload ID
            key: Metadata key (e.g., 'health', 'openapi')
            data: Metadata dict to store
        
        Returns:
            True on success
        """
        pass
    
    @abstractmethod
    def delete_metadata(self, workload_id: str, key: Optional[str] = None) -> bool:
        """
        Delete metadata for a workload.
        
        Args:
            workload_id: The workload ID
            key: Optional specific key to delete, or all if None
            
        Returns:
            True on success
        """
        pass
