"""
etcd-based storage implementation with in-memory indexing.

Key structure:
  /discovery/workloads/{id}/data     → Workload JSON
  /discovery/workloads/{id}/metadata → Scraped metadata JSON

Indices are built in-memory and updated via etcd watch.
"""

import json
import threading
import time
from collections import defaultdict
from typing import Optional

import etcd3gw

from models import Workload, ReachabilityResult, WorkloadType
from config import EtcdConfig
from storage.interface import StorageInterface


class EtcdStorage(StorageInterface):
    """
    etcd storage with in-memory indices for fast queries.
    
    Uses etcd watch to keep indices up-to-date in real-time.
    """
    
    PREFIX = "/discovery/workloads/"
    
    def __init__(self, config: EtcdConfig,):
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
        self._metadata: dict[str, dict] = {}               # id → metadata
        
        # Watch thread
        self._watch_thread: Optional[threading.Thread] = None
        self._stop_watch = threading.Event()
    
    @property
    def client(self) -> etcd3gw.Etcd3Client:
        if self._client is None:
            self._client = etcd3gw.Etcd3Client(host=self.host, port=self.port)
        return self._client
    
    def connect(self) -> bool:
        """Connect to etcd with retry."""
        self.client.status()
        self._connected = True
        self.rebuild_indices()
        self._start_watch()

    def close(self):
        """Stop watch and close connection."""
        self._stop_watch.set()
        if self._watch_thread:
            self._watch_thread.join(timeout=5)
        self._connected = False
    
    # ==================== Write Operations ====================
    
    def register(self, workload: Workload) -> bool:
        """Store workload and update indices."""
        try:
            key = f"{self.PREFIX}{workload.id}/data"
            self.client.put(key, workload.to_json())
            
            # Update local index immediately (watch will also trigger)
            self._update_index(workload.id, workload)
            return True
        except Exception as e:
            print(f"[storage] Failed to register {workload.name}: {e}")
            return False
    
    def deregister(self, workload_id: str) -> bool:
        """Remove workload and update indices."""
        try:
            prefix = f"{self.PREFIX}{workload_id}/"
            self.client.delete_prefix(prefix)
            
            # Update local index immediately
            self._remove_from_index(workload_id)
            return True
        except Exception as e:
            print(f"[storage] Failed to deregister {workload_id}: {e}")
            return False
    
    def set_metadata(self, workload_id: str, metadata: dict) -> bool:
        """Store scraped metadata."""
        try:
            key = f"{self.PREFIX}{workload_id}/metadata"
            self.client.put(key, json.dumps(metadata))
            
            with self._lock:
                self._metadata[workload_id] = metadata
            return True
        except Exception as e:
            print(f"[storage] Failed to set metadata for {workload_id}: {e}")
            return False
    
    # ==================== Read Operations ====================
    
    def get(self, workload_id: str) -> Optional[Workload]:
        """Get workload by ID from index."""
        with self._lock:
            workload = self._workloads.get(workload_id)
            if workload:
                # Attach metadata if available
                workload.metadata = self._metadata.get(workload_id)
            return workload
    
    def get_by_hostname(self, hostname: str) -> Optional[Workload]:
        """Get workload by hostname."""
        with self._lock:
            workload_id = self._by_hostname.get(hostname)
            if workload_id:
                return self.get(workload_id)
        return None
    
    def get_by_name(self, name: str, namespace: str = None) -> Optional[Workload]:
        """Get workload by name (and namespace)."""
        with self._lock:
            key = f"{namespace}/{name}" if namespace else name
            workload_id = self._by_name.get(key)
            if workload_id:
                return self.get(workload_id)
            # Also try without namespace
            workload_id = self._by_name.get(name)
            if workload_id:
                return self.get(workload_id)
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
                
                # Attach metadata
                w = Workload.from_dict(workload.to_dict())
                w.metadata = self._metadata.get(workload.id)
                results.append(w)
            
            return results
    
    def list_by_isolation_group(self, group: str) -> list:
        """List all workloads in an isolation group."""
        with self._lock:
            workload_ids = self._by_group.get(group, set())
            return [self.get(wid) for wid in workload_ids if wid in self._workloads]
    
    # ==================== Reachability Queries ====================
    
    def find_reachable(
        self,
        caller_identity: str,
        include_services: bool = True,
    ) -> ReachabilityResult:
        """
        Find all workloads reachable from caller.
        
        Reachability is based on shared isolation groups (networks/namespaces).
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
            
            # Build result list
            reachable = []
            for wid in reachable_ids:
                workload = self._workloads.get(wid)
                if not workload:
                    continue
                
                # Filter out services if requested
                if not include_services and workload.workload_type == WorkloadType.SERVICE.value:
                    continue
                
                # Attach metadata
                w = Workload.from_dict(workload.to_dict())
                w.metadata = self._metadata.get(wid)
                reachable.append(w)
            
            # Sort by name
            reachable.sort(key=lambda w: w.name)
            
            return ReachabilityResult(caller=caller, reachable=reachable)
    
    def can_reach(self, from_id: str, to_id: str) -> tuple:
        """Check if from can reach to based on shared groups."""
        with self._lock:
            from_workload = self._workloads.get(from_id)
            to_workload = self._workloads.get(to_id)
            
            if not from_workload:
                return False, f"source not found: {from_id}"
            if not to_workload:
                return False, f"target not found: {to_id}"
            
            from_groups = set(from_workload.isolation_groups)
            to_groups = set(to_workload.isolation_groups)
            
            shared = from_groups & to_groups
            if shared:
                return True, f"shared groups: {shared}"
            
            return False, "no shared isolation groups"
    
    # ==================== Index Management ====================
    
    def rebuild_indices(self):
        """Rebuild all indices from etcd."""
        print("[storage] Rebuilding indices from etcd...")
        
        with self._lock:
            self._workloads.clear()
            self._by_hostname.clear()
            self._by_name.clear()
            self._by_group.clear()
            self._metadata.clear()
        
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
                
                # Parse key: /discovery/workloads/{id}/data or /metadata
                parts = key.replace(self.PREFIX, "").split("/")
                if len(parts) < 2:
                    continue
                
                workload_id = parts[0]
                key_type = parts[1]
                
                if key_type == "data":
                    workload = Workload.from_json(value)
                    self._update_index(workload_id, workload)
                elif key_type == "metadata":
                    with self._lock:
                        self._metadata[workload_id] = json.loads(value)
            
            print(f"[storage] Loaded {len(self._workloads)} workloads")
        except Exception as e:
            print(f"[storage] Failed to rebuild indices: {e}")
    
    def get_stats(self) -> dict:
        """Get storage statistics."""
        with self._lock:
            return {
                "workloads": len(self._workloads),
                "by_hostname": len(self._by_hostname),
                "by_name": len(self._by_name),
                "isolation_groups": len(self._by_group),
                "with_metadata": len(self._metadata),
                "connected": self._connected,
            }
    
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
            self._metadata.pop(workload_id, None)
    
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
                    
                    if event_type == 'DELETE':
                        if key_type == "data":
                            self._remove_from_index(workload_id)
                            print(f"[storage] Watch: removed {workload_id[:12]}")
                        elif key_type == "metadata":
                            with self._lock:
                                self._metadata.pop(workload_id, None)
                    else:
                        value = kv.get('value', b'')
                        if isinstance(value, bytes):
                            value = value.decode()
                        
                        if key_type == "data" and value:
                            workload = Workload.from_json(value)
                            self._update_index(workload_id, workload)
                            print(f"[storage] Watch: updated {workload.name}")
                        elif key_type == "metadata" and value:
                            with self._lock:
                                self._metadata[workload_id] = json.loads(value)
                
                cancel()
            except Exception as e:
                if not self._stop_watch.is_set():
                    print(f"[storage] Watch error: {e}, restarting...")
                    time.sleep(1)
