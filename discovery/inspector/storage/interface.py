"""
Storage interface for metadata inspector.

Key structure:
  /discovery/workloads/{id}            → Workload JSON (watched by inspector)
  /discovery/metadata/{id}/{processor} → Processor metadata JSON (written by inspector)

The inspector watches workloads and writes metadata to a separate prefix.
"""

from abc import ABC, abstractmethod
from typing import Optional, Generator, Tuple

from models import Workload


class StorageInterface(ABC):
    """
    Abstract interface for inspector storage.
    
    The inspector:
    - Watches workloads from /discovery/workloads/{id}
    - Writes metadata to /discovery/metadata/{id}/{processor}
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
    
    # ==================== Workload Reading ====================
    
    @abstractmethod
    def list_workloads(self) -> list:
        """List all workloads (ignores metadata keys)."""
        pass
    
    @abstractmethod
    def get_workload(self, workload_id: str) -> Optional[Workload]:
        """Get a specific workload by ID."""
        pass
    
    @abstractmethod
    def watch_workloads(self) -> Generator[Tuple[str, Optional[Workload]], None, None]:
        """
        Watch for workload changes (ignores metadata changes).
        
        Yields:
            Tuple of (event_type, workload) where event_type is 'PUT' or 'DELETE'
        """
        pass
    
    # ==================== Metadata Writing ====================
    
    @abstractmethod
    def set_metadata(self, workload_id: str, processor_key: str, data: dict) -> bool:
        """
        Write metadata for a workload.
        
        Args:
            workload_id: The workload ID
            processor_key: Processor name (e.g., 'health', 'openapi')
            data: Metadata dict to store
        
        Writes to: /discovery/workloads/{workload_id}/metadata/{processor_key}
        
        Returns:
            True on success
        """
        pass
    
    @abstractmethod
    def delete_metadata(self, workload_id: str, processor_key: Optional[str] = None) -> bool:
        """
        Delete metadata for a workload.
        
        Args:
            workload_id: The workload ID
            processor_key: Optional specific processor to delete, or all metadata if None
            
        Returns:
            True on success
        """
        pass
