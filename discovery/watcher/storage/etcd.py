"""
etcd-based storage for workload watcher (write-only).

Key structure:
  /discovery/workloads/{id} → Workload JSON

Uses native etcd3 library for proper gRPC communication.
"""

from typing import Optional

import etcd3

from models import Workload
from config import EtcdConfig, logger
from storage.interface import StorageInterface


class EtcdStorage(StorageInterface):
    """
    etcd storage for registering/deregistering workloads.
    
    This is write-only - the server handles reads.
    Uses etcd3 library for native gRPC support.
    """
    
    def __init__(self, config: EtcdConfig):
        self.host = config.host
        self.port = config.port
        self.workloads_prefix = config.workloads_prefix
        self.username = config.username
        self.password = config.password
        self._client: Optional[etcd3.Etcd3Client] = None
        self._connected = False
    
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
            # Test connection by getting cluster status
            self.client.status()
            self._connected = True
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
        self._connected = False
    
    def register(self, workload: Workload) -> bool:
        """Store workload in etcd."""
        try:
            key = f"{self.workloads_prefix}{workload.id}"
            self.client.put(key, workload.to_json())
            logger.debug("Registered workload: %s", workload.name)
            return True
        except Exception as e:
            logger.error("Failed to register %s: %s", workload.name, e)
            return False
    
    def deregister(self, workload_id: str) -> bool:
        """Remove workload from etcd."""
        try:
            key = f"{self.workloads_prefix}{workload_id}"
            self.client.delete(key)
            logger.debug("Deregistered workload: %s", workload_id[:12])
            return True
        except Exception as e:
            logger.error("Failed to deregister %s: %s", workload_id[:12], e)
            return False
