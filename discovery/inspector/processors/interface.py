"""
Processor interface.
"""

from abc import ABC, abstractmethod
from typing import Optional

from models import Workload


class ProcessorInterface(ABC):
    """
    Abstract interface for metadata processors.
    
    Each processor extracts specific metadata from workloads.
    """
    
    @property
    @abstractmethod
    def name(self) -> str:
        """Processor name (used as metadata key)."""
        pass
    
    @property
    def enabled(self) -> bool:
        """Whether this processor is enabled."""
        return True
    
    @abstractmethod
    def process(self, workload: Workload) -> Optional[dict]:
        """
        Process a workload and extract metadata.
        
        Args:
            workload: The workload to process
            
        Returns:
            dict of metadata to store, or None if processing failed/skipped
        """
        pass
    
    @abstractmethod
    def should_process(self, workload: Workload) -> bool:
        """
        Check if this processor should handle the workload.
        
        Override to add filtering logic (e.g., by labels, runtime type).
        
        Args:
            workload: The workload to check
            
        Returns:
            True if processor should handle this workload
        """
        pass
