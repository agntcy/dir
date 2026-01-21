"""
etcd-based storage for discovery server (read-only).

Key structure:
  /discovery/workloads/{id}/data → Workload JSON

Indices are built in-memory and updated via etcd watch.
"""

import json
import threading
import time
from collections import defaultdict
from typing import Optional

import etcd3gw

from models import Workload, ReachabilityResult
from config import EtcdConfig
from storage.interface import StorageInterface


class EtcdStorage(StorageInterface):
    """
    etcd storage with in-memory indices for fast queries.
    
    Uses etcd watch to keep indices up-to-date in real-time.
    This is a read-only view - writes come from the watcher service.
    """
    
    PREFIX = "/discovery/workloads/"
    
    def __init__(self, config: EtcdConfig):
        self.host = config.host
        self.port = config.port
        
        self._client: Optional[etcd3gw.Etcd3Client] = None
        self._connected = False
        
        # In-memory indices (protected by lock)
        self._lock = threading.RLock()
        self._workloads: dict[str, Workload] = {}          # id → Workload
        self._by_hostname: dict[str, str] = {}             # hostname → id
        self._by_name: dict[str, str] = {}                 # "namespace/name" or "name" → id
        self._by_group: dict[str, set[str]] = defaultdict(set)  # group → {ids}
        
        # Watch thread
        self._watch_thread: Optional[threading.Thread] = None
        self._stop_watch = threading.Event()
    
    @property
    def client(self) -> etcd3gw.Etcd3Client:
        if self._client is None:
            self._client = etcd3gw.Etcd3Client(host=self.host, port=self.port)
        return self._client
    
    def connect(self) -> bool:
        """Connect to etcd."""
        self.client.status()
        self._connected = True
        self._rebuild_indices()
        self._start_watch()
        return True

    def close(self):
        """Stop watch and close connection."""
        self._stop_watch.set()
        if self._watch_thread:
            self._watch_thread.join(timeout=5)
        self._connected = False
    
    # ==================== Read Operations ====================
    
    def get(self, workload_id: str) -> Optional[Workload]:
        """Get workload by ID from index."""
        with self._lock:
            return self._workloads.get(workload_id)
    
    def get_by_hostname(self, hostname: str) -> Optional[Workload]:
        """Get workload by hostname."""
        with self._lock:
            workload_id = self._by_hostname.get(hostname)
            if workload_id:
                return self._workloads.get(workload_id)
        return None
    
    def get_by_name(self, name: str, namespace: str = None) -> Optional[Workload]:
        """Get workload by name (and namespace)."""
        with self._lock:
            key = f"{namespace}/{name}" if namespace else name
            workload_id = self._by_name.get(key)
            if workload_id:
                return self._workloads.get(workload_id)
            # Also try without namespace
            workload_id = self._by_name.get(name)
            if workload_id:
                return self._workloads.get(workload_id)
        return None
    
    def list_all(self, runtime: str = None, label_filter: dict = None) -> list:
        """List all workloads with optional filters."""
        with self._lock:
            results = []
            for workload in self._workloads.values():
                # Filter by runtime
                if runtime and workload.runtime != runtime:
                    continue
                
                # Filter by labels
                if label_filter:
                    match = all(
                        workload.labels.get(k) == v 
                        for k, v in label_filter.items()
                    )
                    if not match:
                        continue
                
                results.append(workload)
            
            return results
    
    # ==================== Reachability Queries ====================
    
    def find_reachable(
        self,
        caller_identity: str,
    ) -> ReachabilityResult:
        """
        Find all workloads reachable from caller.
        
        Reachability is based on shared isolation groups (networks/namespaces).
        Addresses are filtered to only include those in shared isolation groups.
        """
        # Find caller workload
        caller = self._identify_workload(caller_identity)
        if not caller:
            raise ValueError(f"Caller not found: {caller_identity}")
        
        with self._lock:
            # Get caller's effective isolation groups
            caller_groups = set(caller.isolation_groups)
            
            if not caller_groups:
                return ReachabilityResult(caller=caller, reachable=[])
            
            # Find all workloads sharing at least one group
            reachable_ids = set()
            for group in caller_groups:
                reachable_ids.update(self._by_group.get(group, set()))
            
            # Remove caller
            reachable_ids.discard(caller.id)
            
            # Build result list with filtered addresses
            reachable = []
            for wid in reachable_ids:
                workload = self._workloads.get(wid)
                if not workload:
                    continue
                
                # Find shared groups between caller and this workload
                workload_groups = set(workload.isolation_groups)
                shared_groups = caller_groups & workload_groups
                
                # Filter addresses to only include those in shared groups
                # Address format is {name}.{network}, so check if network is in shared groups
                filtered_addresses = []
                for addr in workload.addresses:
                    # Extract network name from address (format: name.network)
                    parts = addr.rsplit(".", 1)
                    if len(parts) == 2:
                        network = parts[1]
                        if network in shared_groups:
                            filtered_addresses.append(addr)
                    else:
                        # Keep addresses that don't follow the format
                        filtered_addresses.append(addr)
                
                # Create a copy with filtered addresses
                filtered_workload = Workload(
                    id=workload.id,
                    name=workload.name,
                    hostname=workload.hostname,
                    runtime=workload.runtime,
                    workload_type=workload.workload_type,
                    node=workload.node,
                    namespace=workload.namespace,
                    addresses=filtered_addresses,
                    isolation_groups=list(shared_groups),  # Only show shared groups
                    ports=workload.ports,
                    labels=workload.labels,
                    annotations=workload.annotations,
                    metadata=workload.metadata,
                    registrar=workload.registrar,
                )
                reachable.append(filtered_workload)
            
            # Sort by name
            reachable.sort(key=lambda w: w.name)
            
            return ReachabilityResult(caller=caller, reachable=reachable)
    
    # ==================== Internal Methods ====================
    
    def _identify_workload(self, identity: str) -> Optional[Workload]:
        """Find workload by hostname, name, or ID."""
        with self._lock:
            # Try hostname first (most common for $HOSTNAME)
            if identity in self._by_hostname:
                return self._workloads.get(self._by_hostname[identity])
            
            # Try name
            if identity in self._by_name:
                return self._workloads.get(self._by_name[identity])
            
            # Try ID directly
            if identity in self._workloads:
                return self._workloads.get(identity)
            
            # Try ID prefix
            for wid, workload in self._workloads.items():
                if wid.startswith(identity):
                    return workload
        
        return None
    
    def _rebuild_indices(self):
        """Rebuild all indices from etcd."""
        print("[storage] Rebuilding indices from etcd...")
        
        with self._lock:
            self._workloads.clear()
            self._by_hostname.clear()
            self._by_name.clear()
            self._by_group.clear()
        
        try:
            results = self.client.get_prefix(self.PREFIX)
            for item in results:
                if not item or len(item) < 2:
                    continue
                
                value = item[0]
                meta = item[1] if len(item) > 1 else {}
                key = meta.get('key', b'')
                if isinstance(key, bytes):
                    key = key.decode()
                if isinstance(value, bytes):
                    value = value.decode()
                
                if not key or not value:
                    continue
                
                # Parse key: /discovery/workloads/{id}/data
                parts = key.replace(self.PREFIX, "").split("/")
                if len(parts) < 2:
                    continue
                
                workload_id = parts[0]
                key_type = parts[1]
                
                if key_type == "data":
                    workload = Workload.from_json(value)
                    self._update_index(workload_id, workload)
            
            print(f"[storage] Loaded {len(self._workloads)} workloads")
        except Exception as e:
            print(f"[storage] Failed to rebuild indices: {e}")
    
    def _update_index(self, workload_id: str, workload: Workload):
        """Update in-memory indices for a workload."""
        with self._lock:
            # Remove old entries if exists
            self._remove_from_index(workload_id)
            
            # Add new entries
            self._workloads[workload_id] = workload
            
            if workload.hostname:
                self._by_hostname[workload.hostname] = workload_id
            
            # Index by name (with namespace if present)
            if workload.namespace:
                self._by_name[f"{workload.namespace}/{workload.name}"] = workload_id
            self._by_name[workload.name] = workload_id
            
            # Index by isolation groups
            for group in workload.isolation_groups:
                self._by_group[group].add(workload_id)
    
    def _remove_from_index(self, workload_id: str):
        """Remove workload from all indices."""
        with self._lock:
            workload = self._workloads.get(workload_id)
            if not workload:
                return
            
            # Remove from hostname index
            if workload.hostname and self._by_hostname.get(workload.hostname) == workload_id:
                del self._by_hostname[workload.hostname]
            
            # Remove from name index
            name_key = f"{workload.namespace}/{workload.name}" if workload.namespace else workload.name
            if self._by_name.get(name_key) == workload_id:
                del self._by_name[name_key]
            if self._by_name.get(workload.name) == workload_id:
                del self._by_name[workload.name]
            
            # Remove from group indices
            for group in workload.isolation_groups:
                self._by_group[group].discard(workload_id)
                if not self._by_group[group]:
                    del self._by_group[group]
            
            # Remove from main store
            del self._workloads[workload_id]
    
    def _start_watch(self):
        """Start background thread watching etcd for changes."""
        self._stop_watch.clear()
        self._watch_thread = threading.Thread(target=self._watch_loop, daemon=True)
        self._watch_thread.start()
    
    def _watch_loop(self):
        """Watch etcd for changes and update indices."""
        while not self._stop_watch.is_set():
            try:
                events, cancel = self.client.watch_prefix(self.PREFIX)
                
                for event in events:
                    if self._stop_watch.is_set():
                        break
                    
                    kv = event.get('kv', {})
                    key = kv.get('key', b'')
                    if isinstance(key, bytes):
                        key = key.decode()
                    
                    if not key:
                        continue
                    
                    # Parse key
                    parts = key.replace(self.PREFIX, "").split("/")
                    if len(parts) < 2:
                        continue
                    
                    workload_id = parts[0]
                    key_type = parts[1]
                    event_type = event.get('type', 'PUT')
                    
                    if key_type != "data":
                        continue
                    
                    if event_type == 'DELETE':
                        self._remove_from_index(workload_id)
                        print(f"[storage] Watch: removed {workload_id[:12]}")
                    else:
                        value = kv.get('value', b'')
                        if isinstance(value, bytes):
                            value = value.decode()
                        
                        if value:
                            workload = Workload.from_json(value)
                            self._update_index(workload_id, workload)
                            print(f"[storage] Watch: updated {workload.name}")
                
                cancel()
            except Exception as e:
                if not self._stop_watch.is_set():
                    print(f"[storage] Watch error: {e}, restarting...")
                    time.sleep(1)
