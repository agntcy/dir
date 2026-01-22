"""
etcd storage for inspector - watches runtime, writes metadata.
"""

import json
from typing import Optional, Generator, Tuple

import etcd3gw

from config import EtcdConfig, logger
from models import Workload


class EtcdStorage:
    """
    etcd storage for watching runtime entries and writing metadata.
    """
    
    def __init__(self, config: EtcdConfig):
        self.host = config.host
        self.port = config.port
        self.runtime_prefix = config.runtime_prefix
        self.metadata_prefix = config.metadata_prefix
        self._client: Optional[etcd3gw.Etcd3Client] = None
    
    @property
    def client(self) -> etcd3gw.Etcd3Client:
        if self._client is None:
            self._client = etcd3gw.Etcd3Client(host=self.host, port=self.port)
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
        self._client = None
    
    # ==================== Runtime Reading ====================
    
    def list_workloads(self) -> list:
        """List all workloads from runtime storage."""
        workloads = []
        try:
            results = self.client.get_prefix(f"{self.runtime_prefix}/")
            for value, meta in results:
                if value and meta.get("key", b"").decode().endswith("/data"):
                    try:
                        data = json.loads(value.decode())
                        workloads.append(Workload.from_dict(data))
                    except (json.JSONDecodeError, KeyError) as e:
                        logger.warning("Failed to parse workload: %s", e)
        except Exception as e:
            logger.error("Failed to list workloads: %s", e)
        return workloads
    
    def get_workload(self, workload_id: str) -> Optional[Workload]:
        """Get a specific workload by ID."""
        try:
            key = f"{self.runtime_prefix}/{workload_id}/data"
            results = self.client.get(key)
            if results and results[0]:
                data = json.loads(results[0].decode())
                return Workload.from_dict(data)
        except Exception as e:
            logger.warning("Failed to get workload %s: %s", workload_id, e)
        return None
    
    def watch_workloads(self) -> Generator[Tuple[str, Optional[Workload]], None, None]:
        """
        Watch for workload changes.
        
        Yields:
            Tuple of (event_type, workload) where event_type is 'PUT' or 'DELETE'
        """
        # Note: etcd3gw doesn't support watch natively, so we poll
        # In production, use etcd3 library with proper watch support

        known_ids = set()
        
        # Initial sync
        for workload in self.list_workloads():
            known_ids.add(workload.id)
            yield ("PUT", workload)
        
        # Poll for changes
        while True:
            time.sleep(5)  # Poll interval
            
            current_workloads = {w.id: w for w in self.list_workloads()}
            current_ids = set(current_workloads.keys())
            
            # New workloads
            for wid in current_ids - known_ids:
                yield ("PUT", current_workloads[wid])
            
            # Deleted workloads
            for wid in known_ids - current_ids:
                yield ("DELETE", None)
            
            # Modified workloads (simplified - just re-process all)
            for wid in current_ids & known_ids:
                yield ("PUT", current_workloads[wid])
            
            known_ids = current_ids
    
    # ==================== Metadata Writing ====================
    
    def set_metadata(self, workload_id: str, key: str, data: dict) -> bool:
        """
        Write metadata for a workload.
        
        Args:
            workload_id: The workload ID
            key: Metadata key (e.g., 'health', 'openapi')
            data: Metadata dict to store
        
        Stores at: /discovery/metadata/{workload_id}/{key}
        """
        try:
            etcd_key = f"{self.metadata_prefix}/{workload_id}/{key}"
            self.client.put(etcd_key, json.dumps(data))
            logger.debug("Set metadata %s for %s", key, workload_id[:12])
            return True
        except Exception as e:
            logger.error("Failed to set metadata %s for %s: %s", key, workload_id[:12], e)
            return False
    
    def delete_metadata(self, workload_id: str, key: Optional[str] = None) -> bool:
        """
        Delete metadata for a workload.
        
        Args:
            workload_id: The workload ID
            key: Optional specific key to delete, or all if None
        """
        try:
            if key:
                etcd_key = f"{self.metadata_prefix}/{workload_id}/{key}"
                self.client.delete(etcd_key)
            else:
                prefix = f"{self.metadata_prefix}/{workload_id}/"
                self.client.delete_prefix(prefix)
            return True
        except Exception as e:
            logger.error("Failed to delete metadata for %s: %s", workload_id[:12], e)
            return False
