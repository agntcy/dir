"""
containerd runtime adapter.

Watches containerd for task events and reads CNI state for network info.
Uses the pycontainerd gRPC bindings (not a high-level client).
"""

import json
import os
import threading
from pathlib import Path
from typing import Optional, Callable

import grpc
from containerd.services.containers.v1 import containers_pb2_grpc, containers_pb2
from containerd.services.tasks.v1 import tasks_pb2_grpc, tasks_pb2
from containerd.services.events.v1 import events_pb2_grpc, events_pb2, unwrap
from containerd.types.task import task_pb2
from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler

from models import Workload, Runtime, WorkloadType, EventType
from config import ContainerdConfig
from runtimes.interface import RuntimeAdapter


class ContainerdAdapter(RuntimeAdapter):
    """
    containerd runtime adapter.
    
    Uses containerd gRPC API for container events.
    Reads CNI state files for network information.
    Optionally watches CNI state directory for network changes.
    """
    
    SOCKET_PATH = "/run/containerd/containerd.sock"
    CNI_STATE_PATH = "/var/lib/cni/results"
    
    def __init__(self, config: ContainerdConfig):
        """
        Initialize containerd adapter.
        
        Args:
            config: ContainerdConfig instance
        """
        self.socket_path = config.socket or self.SOCKET_PATH
        self.namespace = config.namespace
        self.cni_state_path = Path(config.cni_state_dir or self.CNI_STATE_PATH)
        self.label_key = config.label_key
        self.label_value = config.label_value
        
        self._channel = None
        self._stop_event = threading.Event()
        self._cni_observer = None
        self._event_callback = None
    
    def _get_metadata(self):
        """Get gRPC metadata with namespace."""
        return (('containerd-namespace', self.namespace),)

    @property
    def runtime_type(self) -> Runtime:
        return Runtime.CONTAINERD
    
    def connect(self) -> bool:
        """Verify containerd connection."""
        try:
            channel = grpc.insecure_channel(f'unix://{self.socket_path}')
            containers_stub = containers_pb2_grpc.ContainersStub(channel)
            # Test connection by listing containers
            containers_stub.List(
                containers_pb2.ListContainersRequest(),
                metadata=self._get_metadata()
            )
            channel.close()
            return True
        except Exception as e:
            print(f"[containerd] Failed to connect: {e}")
            return False

    def close(self):
        """Close containerd connection and stop watchers."""
        self._stop_event.set()
        if self._channel:
            self._channel.close()
            self._channel = None
        if self._cni_observer:
            self._cni_observer.stop()
            self._cni_observer = None

    def list_workloads(self) -> list:
        """List all discoverable containers."""
        workloads = []
        
        try:
            channel = grpc.insecure_channel(f'unix://{self.socket_path}')
            containers_stub = containers_pb2_grpc.ContainersStub(channel)
            tasks_stub = tasks_pb2_grpc.TasksStub(channel)
            
            # List all containers
            response = containers_stub.List(
                containers_pb2.ListContainersRequest(),
                metadata=self._get_metadata()
            )
            
            for container in response.containers:
                # Convert protobuf labels to dict
                labels = dict(container.labels) if container.labels else {}
                
                # Check discover label
                if labels.get(self.label_key) != self.label_value:
                    continue
                
                # Check if running by getting task status
                try:
                    task_response = tasks_stub.Get(
                        tasks_pb2.GetRequest(container_id=container.id),
                        metadata=self._get_metadata()
                    )
                    # Status enum: UNKNOWN=0, CREATED=1, RUNNING=2, STOPPED=3, PAUSED=4, PAUSING=5
                    if task_response.process.status != task_pb2.RUNNING:
                        continue
                except grpc.RpcError:
                    # No task means not running
                    continue
                
                workload = self._container_to_workload(container, labels)
                if workload:
                    workloads.append(workload)
            
            channel.close()
        
        except Exception as e:
            print(f"[containerd] Failed to list containers: {e}")
        
        return workloads

    def watch_events(self, callback: Callable[[str, Workload], None]) -> None:
        """
        Watch containerd events and CNI changes.
        
        containerd events cover: task start/exit, container create/delete
        CNI watcher covers: network connect/disconnect (via file changes)
        """
        self._stop_event.clear()
        self._event_callback = callback
        
        # Start CNI file watcher in background
        if self.cni_state_path.exists():
            self._start_cni_watcher(callback)
        
        # Watch containerd events via gRPC
        try:
            channel = grpc.insecure_channel(f'unix://{self.socket_path}')
            events_stub = events_pb2_grpc.EventsStub(channel)
            
            # Subscribe to all events (filter by namespace via metadata)
            for envelope in events_stub.Subscribe(
                events_pb2.SubscribeRequest(),
                metadata=self._get_metadata()
            ):
                if self._stop_event.is_set():
                    break
                
                # Unwrap the event to get the actual event object
                try:
                    event = unwrap(envelope)
                    topic = envelope.topic
                    
                    # Extract container ID from the event
                    container_id = getattr(event, 'container_id', None) or getattr(event, 'id', None)
                    
                    if not container_id:
                        continue
                    
                    if topic == "/tasks/start":
                        workload = self._get_workload(container_id)
                        if workload:
                            callback(EventType.ADDED, workload)
                    
                    elif topic in ("/tasks/exit", "/tasks/delete", "/containers/delete"):
                        # Create minimal workload for deletion
                        workload = Workload(
                            id=container_id,
                            name=container_id[:12],
                            hostname=container_id[:12],
                            runtime=Runtime.CONTAINERD.value,
                            workload_type=WorkloadType.CONTAINER.value,
                        )
                        callback(EventType.DELETED, workload)
                
                except Exception as e:
                    print(f"[containerd] Failed to process event: {e}")
                    continue
            
            channel.close()
        
        except Exception as e:
            if not self._stop_event.is_set():
                print(f"[containerd] Event watch error: {e}")

    def _get_workload(self, identity: str) -> Optional[Workload]:
        """Get container by hostname, name, or ID."""
        try:
            channel = grpc.insecure_channel(f'unix://{self.socket_path}')
            containers_stub = containers_pb2_grpc.ContainersStub(channel)
            
            # List all and filter (containerd doesn't have get-by-name)
            response = containers_stub.List(
                containers_pb2.ListContainersRequest(),
                metadata=self._get_metadata()
            )
            
            for container in response.containers:
                labels = dict(container.labels) if container.labels else {}
                
                # Match by ID or ID prefix
                if container.id == identity or container.id.startswith(identity):
                    channel.close()
                    return self._container_to_workload(container, labels)
                
                # Match by hostname (short ID)
                if container.id[:12] == identity:
                    channel.close()
                    return self._container_to_workload(container, labels)
                
                # Match by name label (nerdctl style)
                if labels.get("nerdctl/name") == identity:
                    channel.close()
                    return self._container_to_workload(container, labels)
            
            channel.close()
            return None
        except Exception as e:
            print(f"[containerd] Failed to get container {identity}: {e}")
            return None

    def _container_to_workload(self, container, labels: dict) -> Optional[Workload]:
        """Convert containerd container to Workload."""
        try:
            # Get network info from CNI state
            networks, ips = self._get_cni_networks(container.id)
            
            # Container name (from nerdctl label or short ID)
            name = labels.get("nerdctl/name", container.id[:12])
            hostname = container.id[:12]
            
            # Get exposed ports from labels (nerdctl stores them as labels)
            ports = []
            for key, value in labels.items():
                if key.startswith("nerdctl/ports/"):
                    # Format: nerdctl/ports/tcp/80 = 0.0.0.0:8080
                    parts = key.split("/")
                    if len(parts) >= 4:
                        ports.append(parts[3])
            
            # Build addresses in format {container_name}.{network_name}
            # This is how containers are reachable in containerd/nerdctl
            addresses = []
            for network in networks:
                addresses.append(f"{name}.{network}")
            
            return Workload(
                id=container.id,
                name=name,
                hostname=hostname,
                runtime=Runtime.CONTAINERD.value,
                workload_type=WorkloadType.CONTAINER.value,
                node=None,
                namespace=None,
                addresses=addresses,
                isolation_groups=networks,
                ports=ports,
                labels=labels,
            )
        except Exception as e:
            print(f"[containerd] Failed to convert container: {e}")
            return None

    def _start_cni_watcher(self, callback: Callable[[str, Workload], None]):
        """Start watching CNI state directory for network changes."""
        class CNIEventHandler(FileSystemEventHandler):
            def __init__(handler_self, adapter):
                handler_self.adapter = adapter
            
            def on_created(handler_self, event):
                if event.is_directory:
                    return
                handler_self._handle_cni_change(event.src_path)
            
            def on_deleted(handler_self, event):
                if event.is_directory:
                    return
                handler_self._handle_cni_change(event.src_path)
            
            def _handle_cni_change(handler_self, filepath):
                # Extract container ID from filename
                filename = os.path.basename(filepath)
                parts = filename.split("-")
                if len(parts) >= 2:
                    # Try to find the container ID part (usually second)
                    for part in parts[1:]:
                        if len(part) >= 12:
                            workload = handler_self.adapter._get_workload(part[:12])
                            if workload:
                                callback(EventType.NETWORK_CHANGED, workload)
                                break
        
        self._cni_observer = Observer()
        self._cni_observer.schedule(
            CNIEventHandler(self),
            str(self.cni_state_path),
            recursive=False
        )
        self._cni_observer.start()
        print(f"[containerd] Started CNI state watcher on {self.cni_state_path}")

    def _get_cni_networks(self, container_id: str) -> tuple:
        """
        Get networks and IPs from CNI state files.
        
        CNI stores results in files like:
          {network}-{namespace}-{container_id}-{interface}
        
        Network names may contain hyphens (e.g., "discovery_team-a").
        Container IDs are 64-char hex strings, so we match on the ID pattern.
        The namespace suffix (e.g., "-default", "-moby") is stripped.
        
        Returns (networks: list[str], ips: list[str])
        """
        networks = []
        ips = []
        
        if not self.cni_state_path.exists():
            return networks, ips
        
        # CNI files contain container ID (full or prefix)
        short_id = container_id[:12]
        
        for result_file in self.cni_state_path.glob(f"*{short_id}*"):
            try:
                with open(result_file) as f:
                    result = json.load(f)
                
                # Parse network name from filename
                # Format: {network}-{namespace}-{container_id}-{interface}
                # Container ID is 64 hex chars, interface is like "eth0"
                # Split on container ID to get network name
                filename = result_file.name
                
                # Find container ID position in filename
                id_pos = filename.find(container_id[:12])
                if id_pos > 0:
                    # Network name is everything before the container ID minus trailing dash
                    network_name = filename[:id_pos].rstrip("-")
                    
                    # Strip namespace suffix (e.g., "-default", "-moby")
                    namespace_suffix = f"-{self.namespace}"
                    if network_name.endswith(namespace_suffix):
                        network_name = network_name[:-len(namespace_suffix)]
                    
                    if network_name:
                        networks.append(network_name)
                
                # Get IPs from CNI result
                for ip_config in result.get("ips", []):
                    addr = ip_config.get("address", "").split("/")[0]
                    if addr:
                        ips.append(addr)
            
            except (json.JSONDecodeError, KeyError, IOError):
                continue
        
        return list(set(networks)), list(set(ips))
