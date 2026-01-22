"""
etcd-based storage for discovery server (read-only).

Key structure:
  /discovery/workloads/{id}          → Workload JSON
  /discovery/metadata/{id}/{processor} → Processor metadata JSON

Uses native etcd3 library for proper gRPC watch support.
Indices are built in-memory and updated via etcd watch.
"""

import json
import threading
from collections import defaultdict
from typing import Optional

import etcd3
from etcd3.events import PutEvent, DeleteEvent

from models import Workload, ReachabilityResult
from config import EtcdConfig, logger
from storage.interface import StorageInterface


class EtcdStorage(StorageInterface):
    """
    etcd storage with in-memory indices for fast queries.
    
    Uses native etcd3 watch for real-time updates.
    Watches two separate prefixes:
    - workloads_prefix: /discovery/workloads/ (workload data from watcher)
    - metadata_prefix: /discovery/metadata/ (metadata from inspector)
    """
    
    def __init__(self, config: EtcdConfig):
        self.host = config.host
        self.port = config.port
        self.workloads_prefix = config.workloads_prefix
        self.metadata_prefix = config.metadata_prefix
        self.username = config.username
        self.password = config.password
        
        self._client: Optional[etcd3.Etcd3Client] = None
        self._connected = False
        
        # In-memory indices (protected by lock)
        self._lock = threading.RLock()
        self._workloads: dict[str, Workload] = {}          # id → Workload
        self._metadata: dict[str, dict] = {}               # id → {processor: data}
        self._by_hostname: dict[str, str] = {}             # hostname → id
        self._by_name: dict[str, str] = {}                 # "namespace/name" or "name" → id
        self._by_group: dict[str, set[str]] = defaultdict(set)  # group → {ids}
        
        # Watch threads
        self._workload_watch_thread: Optional[threading.Thread] = None
        self._metadata_watch_thread: Optional[threading.Thread] = None
        self._stop_watch = threading.Event()
    
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
        """Connect to etcd and start watching."""
        try:
            self.client.status()
            self._connected = True
            logger.info("Connected to etcd at %s:%d", self.host, self.port)
            
            # Initial load
            self._load_workloads()
            self._load_metadata()
            
            # Start watch threads
            self._start_watches()
            
            return True
        except Exception as e:
            logger.error("Failed to connect to etcd: %s", e)
            return False

    def close(self):
        """Stop watches and close connection."""
        self._stop_watch.set()
        
        if self._workload_watch_thread:
            self._workload_watch_thread.join(timeout=5)
        if self._metadata_watch_thread:
            self._metadata_watch_thread.join(timeout=5)
        
        if self._client:
            self._client.close()
            self._client = None
        
        self._connected = False
        logger.info("Storage closed")
    
    # ==================== Read Operations ====================
    
    def get(self, workload_id: str) -> Optional[Workload]:
        """Get workload by ID from index."""
        with self._lock:
            workload = self._workloads.get(workload_id)
            if workload:
                return self._workload_with_metadata(workload)
        return None
    
    def get_by_hostname(self, hostname: str) -> Optional[Workload]:
        """Get workload by hostname."""
        with self._lock:
            workload_id = self._by_hostname.get(hostname)
            if workload_id:
                workload = self._workloads.get(workload_id)
                if workload:
                    return self._workload_with_metadata(workload)
        return None
    
    def get_by_name(self, name: str, namespace: str = None) -> Optional[Workload]:
        """Get workload by name (and namespace)."""
        with self._lock:
            key = f"{namespace}/{name}" if namespace else name
            workload_id = self._by_name.get(key)
            if not workload_id:
                # Try without namespace
                workload_id = self._by_name.get(name)
            
            if workload_id:
                workload = self._workloads.get(workload_id)
                if workload:
                    return self._workload_with_metadata(workload)
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
                
                results.append(self._workload_with_metadata(workload))
            
            return results
    
    def _workload_with_metadata(self, workload: Workload) -> Workload:
        """Return a copy of workload with merged metadata."""
        metadata = self._metadata.get(workload.id, {})
        if metadata:
            # Merge stored metadata with any existing workload metadata
            workload_meta = workload.metadata or {}
            merged = {**workload_meta, **metadata}
            return Workload(
                id=workload.id,
                name=workload.name,
                hostname=workload.hostname,
                runtime=workload.runtime,
                workload_type=workload.workload_type,
                node=workload.node,
                namespace=workload.namespace,
                addresses=workload.addresses,
                isolation_groups=workload.isolation_groups,
                ports=workload.ports,
                labels=workload.labels,
                annotations=workload.annotations,
                metadata=merged,
                registrar=workload.registrar,
            )
        return workload
    
    # ==================== Reachability Queries ====================
    
    def find_reachable(self, caller_identity: str) -> ReachabilityResult:
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
                filtered_addresses = self._filter_addresses(workload.addresses, shared_groups)
                
                # Create a copy with filtered addresses and metadata
                metadata = self._metadata.get(wid, {})
                workload_meta = workload.metadata or {}
                filtered_workload = Workload(
                    id=workload.id,
                    name=workload.name,
                    hostname=workload.hostname,
                    runtime=workload.runtime,
                    workload_type=workload.workload_type,
                    node=workload.node,
                    namespace=workload.namespace,
                    addresses=filtered_addresses,
                    isolation_groups=list(shared_groups),
                    ports=workload.ports,
                    labels=workload.labels,
                    annotations=workload.annotations,
                    metadata={**workload_meta, **metadata},
                    registrar=workload.registrar,
                )
                reachable.append(filtered_workload)
            
            # Sort by name
            reachable.sort(key=lambda w: w.name)
            
            return ReachabilityResult(caller=caller, reachable=reachable)
    
    def _filter_addresses(self, addresses: list, shared_groups: set) -> list:
        """Filter addresses to only those in shared isolation groups."""
        filtered = []
        for addr in addresses:
            parts = addr.split(".")
            if len(parts) >= 3 and parts[-1] in ("pod", "svc"):
                # Kubernetes format: extract namespace
                namespace = parts[-2]
                if namespace in shared_groups:
                    filtered.append(addr)
            elif len(parts) == 2:
                # Docker format: {name}.{network}
                network = parts[1]
                if network in shared_groups:
                    filtered.append(addr)
            else:
                # Keep addresses that don't follow expected formats
                filtered.append(addr)
        return filtered
    
    # ==================== Internal Methods ====================
    
    def _identify_workload(self, identity: str) -> Optional[Workload]:
        """Find workload by hostname, name, or ID."""
        with self._lock:
            # Try hostname first (most common for $HOSTNAME)
            if identity in self._by_hostname:
                wid = self._by_hostname[identity]
                return self._workload_with_metadata(self._workloads[wid])
            
            # Try name
            if identity in self._by_name:
                wid = self._by_name[identity]
                return self._workload_with_metadata(self._workloads[wid])
            
            # Try ID directly
            if identity in self._workloads:
                return self._workload_with_metadata(self._workloads[identity])
            
            # Try ID prefix
            for wid, workload in self._workloads.items():
                if wid.startswith(identity):
                    return self._workload_with_metadata(workload)
        
        return None
    
    def _load_workloads(self):
        """Load all workloads from etcd."""
        logger.info("Loading workloads from %s", self.workloads_prefix)
        
        try:
            for value, meta in self.client.get_prefix(self.workloads_prefix):
                if not value:
                    continue
                
                key = meta.key.decode() if isinstance(meta.key, bytes) else meta.key
                workload_id = key.replace(self.workloads_prefix, "")
                
                try:
                    data = json.loads(value.decode() if isinstance(value, bytes) else value)
                    workload = Workload.from_dict(data)
                    self._update_workload_index(workload_id, workload)
                except (json.JSONDecodeError, KeyError) as e:
                    logger.warning("Failed to parse workload %s: %s", workload_id, e)
            
            logger.info("Loaded %d workloads", len(self._workloads))
        except Exception as e:
            logger.error("Failed to load workloads: %s", e)
    
    def _load_metadata(self):
        """Load all metadata from etcd."""
        logger.info("Loading metadata from %s", self.metadata_prefix)
        
        try:
            for value, meta in self.client.get_prefix(self.metadata_prefix):
                if not value:
                    continue
                
                key = meta.key.decode() if isinstance(meta.key, bytes) else meta.key
                relative_key = key.replace(self.metadata_prefix, "")
                parts = relative_key.split("/")
                
                if len(parts) >= 2:
                    workload_id = parts[0]
                    processor = parts[1]
                    
                    try:
                        data = json.loads(value.decode() if isinstance(value, bytes) else value)
                        with self._lock:
                            if workload_id not in self._metadata:
                                self._metadata[workload_id] = {}
                            self._metadata[workload_id][processor] = data
                    except json.JSONDecodeError as e:
                        logger.warning("Failed to parse metadata %s/%s: %s", workload_id, processor, e)
            
            logger.info("Loaded metadata for %d workloads", len(self._metadata))
        except Exception as e:
            logger.error("Failed to load metadata: %s", e)
    
    def _start_watches(self):
        """Start watch threads for workloads and metadata."""
        self._stop_watch.clear()
        
        self._workload_watch_thread = threading.Thread(
            target=self._watch_workloads,
            daemon=True,
            name="workload-watcher"
        )
        self._workload_watch_thread.start()
        
        self._metadata_watch_thread = threading.Thread(
            target=self._watch_metadata,
            daemon=True,
            name="metadata-watcher"
        )
        self._metadata_watch_thread.start()
        
        logger.info("Started watch threads")
    
    def _watch_workloads(self):
        """Watch for workload changes."""
        while not self._stop_watch.is_set():
            try:
                events_iterator, cancel = self.client.watch_prefix(self.workloads_prefix)
                logger.info("Watching workloads at %s", self.workloads_prefix)
                
                for event in events_iterator:
                    if self._stop_watch.is_set():
                        break
                    
                    key = event.key.decode() if isinstance(event.key, bytes) else event.key
                    workload_id = key.replace(self.workloads_prefix, "")
                    
                    if isinstance(event, PutEvent):
                        try:
                            value = event.value.decode() if isinstance(event.value, bytes) else event.value
                            data = json.loads(value)
                            workload = Workload.from_dict(data)
                            self._update_workload_index(workload_id, workload)
                            logger.info("Watch: updated workload %s", workload.name)
                        except (json.JSONDecodeError, KeyError) as e:
                            logger.warning("Watch: failed to parse workload %s: %s", workload_id, e)
                    
                    elif isinstance(event, DeleteEvent):
                        self._remove_workload_index(workload_id)
                        # Also remove associated metadata
                        with self._lock:
                            self._metadata.pop(workload_id, None)
                        logger.info("Watch: removed workload %s", workload_id[:12])
                
                cancel()
            except Exception as e:
                if not self._stop_watch.is_set():
                    logger.warning("Workload watch error: %s, reconnecting...", e)
                    import time
                    time.sleep(1)
    
    def _watch_metadata(self):
        """Watch for metadata changes."""
        while not self._stop_watch.is_set():
            try:
                events_iterator, cancel = self.client.watch_prefix(self.metadata_prefix)
                logger.info("Watching metadata at %s", self.metadata_prefix)
                
                for event in events_iterator:
                    if self._stop_watch.is_set():
                        break
                    
                    key = event.key.decode() if isinstance(event.key, bytes) else event.key
                    relative_key = key.replace(self.metadata_prefix, "")
                    parts = relative_key.split("/")
                    
                    if len(parts) < 2:
                        continue
                    
                    workload_id = parts[0]
                    processor = parts[1]
                    
                    if isinstance(event, PutEvent):
                        try:
                            value = event.value.decode() if isinstance(event.value, bytes) else event.value
                            data = json.loads(value)
                            with self._lock:
                                if workload_id not in self._metadata:
                                    self._metadata[workload_id] = {}
                                self._metadata[workload_id][processor] = data
                            logger.debug("Watch: updated metadata %s for %s", processor, workload_id[:12])
                        except json.JSONDecodeError as e:
                            logger.warning("Watch: failed to parse metadata: %s", e)
                    
                    elif isinstance(event, DeleteEvent):
                        with self._lock:
                            if workload_id in self._metadata:
                                self._metadata[workload_id].pop(processor, None)
                        logger.debug("Watch: removed metadata %s for %s", processor, workload_id[:12])
                
                cancel()
            except Exception as e:
                if not self._stop_watch.is_set():
                    logger.warning("Metadata watch error: %s, reconnecting...", e)
                    import time
                    time.sleep(1)
    
    def _update_workload_index(self, workload_id: str, workload: Workload):
        """Update in-memory indices for a workload."""
        with self._lock:
            # Remove old entries if exists
            self._remove_workload_index(workload_id)
            
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
    
    def _remove_workload_index(self, workload_id: str):
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
