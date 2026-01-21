"""
etcd-based storage for workload watcher (write-only).

Key structure:
  /discovery/workloads/{id}/data → Workload JSON
"""

from typing import Optional

import etcd3gw

from models import Workload
from config import EtcdConfig
from storage.interface import StorageInterface


class EtcdStorage(StorageInterface):
    """
    etcd storage for registering/deregistering workloads.
    
    This is write-only - the server handles reads.
    """
    
    PREFIX = "/discovery/workloads/"
    
    def __init__(self, config: EtcdConfig):
        self.host = config.host
        self.port = config.port
        self._client: Optional[etcd3gw.Etcd3Client] = None
        self._connected = False
    
    @property
    def client(self) -> etcd3gw.Etcd3Client:
        if self._client is None:
            self._client = etcd3gw.Etcd3Client(host=self.host, port=self.port)
        return self._client
    
    def connect(self) -> bool:
        """Connect to etcd."""
        self.client.status()
        self._connected = True
        return True

    def close(self):
        """Close connection."""
        self._connected = False
    
    def register(self, workload: Workload) -> bool:
        """Store workload in etcd."""
        try:
            key = f"{self.PREFIX}{workload.id}/data"
            self.client.put(key, workload.to_json())
            return True
        except Exception as e:
            print(f"[storage] Failed to register {workload.name}: {e}")
            return False
    
    def deregister(self, workload_id: str) -> bool:
        """Remove workload from etcd."""
        try:
            prefix = f"{self.PREFIX}{workload_id}/"
            self.client.delete_prefix(prefix)
            return True
        except Exception as e:
            print(f"[storage] Failed to deregister {workload_id}: {e}")
            return False
