"""
etcd storage for inspector - watches workloads, writes metadata.

Key structure:
  /discovery/workloads/{id}            → Workload JSON (watched by inspector)
  /discovery/metadata/{id}/{processor} → Metadata JSON (written by inspector)

Uses native etcd3 library for proper gRPC watch support.
"""

import json
from typing import Optional, Generator, Tuple

import etcd3
from etcd3.events import PutEvent, DeleteEvent

from config import EtcdConfig, logger
from models import Workload
from storage.interface import StorageInterface


class EtcdStorage(StorageInterface):
    """
    etcd storage for watching workload entries and writing metadata.
    
    Uses native etcd3 watch for real-time updates.
    """
    
    def __init__(self, config: EtcdConfig):
        self.host = config.host
        self.port = config.port
        self.workloads_prefix = config.workloads_prefix
        self.metadata_prefix = config.metadata_prefix
        self.username = config.username
        self.password = config.password
        self._client: Optional[etcd3.Etcd3Client] = None
    
    @property
    def client(self) -> etcd3.Etcd3Client:
        if self._client is None:
            self._client = etcd3.client(
                host=self.host,
                port=self.port,
                user=self.username,
                password=self.password,
            )
        return self._client
    
    def connect(self) -> bool:
        """Connect to etcd."""
        try:
            self.client.status()
            logger.info("Connected to etcd at %s:%d", self.host, self.port)
            return True
        except Exception as e:
            logger.error("Failed to connect to etcd: %s", e)
            return False
    
    def close(self):
        """Close connection."""
        if self._client:
            self._client.close()
            self._client = None
    
    # ==================== Workload Reading ====================
    
    def list_workloads(self) -> list:
        """List all workloads from storage."""
        workloads = []
        try:
            for value, meta in self.client.get_prefix(self.workloads_prefix):
                if not value:
                    continue
                
                key = meta.key.decode() if isinstance(meta.key, bytes) else meta.key
                workload_id = key.replace(self.workloads_prefix, "")
                
                try:
                    data = json.loads(value.decode() if isinstance(value, bytes) else value)
                    workloads.append(Workload.from_dict(data))
                except (json.JSONDecodeError, KeyError) as e:
                    logger.warning("Failed to parse workload %s: %s", workload_id, e)
        except Exception as e:
            logger.error("Failed to list workloads: %s", e)
        return workloads
    
    def get_workload(self, workload_id: str) -> Optional[Workload]:
        """Get a specific workload by ID."""
        try:
            key = f"{self.workloads_prefix}{workload_id}"
            value, _ = self.client.get(key)
            if value:
                data = json.loads(value.decode() if isinstance(value, bytes) else value)
                return Workload.from_dict(data)
        except Exception as e:
            logger.warning("Failed to get workload %s: %s", workload_id, e)
        return None
    
    def watch_workloads(self) -> Generator[Tuple[str, Optional[Workload]], None, None]:
        """
        Watch for workload changes using native etcd3 watch.
        
        Yields:
            Tuple of (event_type, workload) where event_type is 'PUT' or 'DELETE'
        """
        # First, yield all existing workloads
        logger.info("Initial sync: loading existing workloads...")
        for workload in self.list_workloads():
            yield ("PUT", workload)
        
        logger.info("Starting watch on %s", self.workloads_prefix)
        
        # Watch for changes
        while True:
            try:
                events_iterator, cancel = self.client.watch_prefix(self.workloads_prefix)
                
                for event in events_iterator:
                    key = event.key.decode() if isinstance(event.key, bytes) else event.key
                    workload_id = key.replace(self.workloads_prefix, "")
                    
                    if isinstance(event, PutEvent):
                        try:
                            value = event.value.decode() if isinstance(event.value, bytes) else event.value
                            data = json.loads(value)
                            workload = Workload.from_dict(data)
                            logger.info("Watch: workload updated: %s", workload.name)
                            yield ("PUT", workload)
                        except (json.JSONDecodeError, KeyError) as e:
                            logger.warning("Watch: failed to parse workload %s: %s", workload_id, e)
                    
                    elif isinstance(event, DeleteEvent):
                        # Create a minimal workload object for delete event
                        logger.info("Watch: workload removed: %s", workload_id[:12])
                        deleted_workload = Workload(
                            id=workload_id,
                            name=workload_id[:12],
                            hostname=workload_id[:12],
                            runtime="unknown",
                            workload_type="unknown",
                            addresses=[],
                            ports=[],
                        )
                        yield ("DELETE", deleted_workload)
                
                cancel()
            except Exception as e:
                logger.warning("Watch error: %s, reconnecting...", e)
                import time
                time.sleep(1)
    
    # ==================== Metadata Writing ====================
    
    def set_metadata(self, workload_id: str, processor_key: str, data: dict) -> bool:
        """
        Write metadata for a workload.
        
        Args:
            workload_id: The workload ID
            processor_key: Processor name (e.g., 'health', 'openapi')
            data: Metadata dict to store
        
        Stores at: /discovery/metadata/{workload_id}/{processor_key}
        """
        try:
            key = f"{self.metadata_prefix}{workload_id}/{processor_key}"
            self.client.put(key, json.dumps(data))
            logger.debug("Set metadata %s for %s", processor_key, workload_id[:12])
            return True
        except Exception as e:
            logger.error("Failed to set metadata %s for %s: %s", processor_key, workload_id[:12], e)
            return False
    
    def delete_metadata(self, workload_id: str, processor_key: Optional[str] = None) -> bool:
        """
        Delete metadata for a workload.
        
        Args:
            workload_id: The workload ID
            processor_key: Optional specific processor to delete, or all metadata if None
        """
        try:
            if processor_key:
                key = f"{self.metadata_prefix}{workload_id}/{processor_key}"
                self.client.delete(key)
            else:
                prefix = f"{self.metadata_prefix}{workload_id}/"
                self.client.delete_prefix(prefix)
            logger.debug("Deleted metadata for %s", workload_id[:12])
            return True
        except Exception as e:
            logger.error("Failed to delete metadata for %s: %s", workload_id[:12], e)
            return False
