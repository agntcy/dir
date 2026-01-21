"""
Docker runtime adapter.

Watches Docker daemon for container events and converts to unified Workload model.
"""

import threading
from typing import Optional, Callable

import docker
import docker.errors

from models import Workload, Runtime, WorkloadType, EventType
from config import DockerConfig
from runtimes.interface import RuntimeAdapter


class DockerAdapter(RuntimeAdapter):
    """
    Docker runtime adapter.
    
    Uses Docker SDK to watch for container events and list containers.
    Network changes (connect/disconnect) are fully supported.
    """
    
    SOCKET_PATH = "/var/run/docker.sock"
    
    def __init__(self, config: DockerConfig):
        """
        Initialize Docker adapter.
        
        Args:
            config: DockerConfig instance
        """
        # Extract socket path from URL if needed
        socket = config.socket
        if socket.startswith("unix://"):
            socket = socket[7:]
        
        self.socket_path = socket or self.SOCKET_PATH
        self.label_key = config.label_key
        self.label_value = config.label_value

        self._client: Optional[docker.DockerClient] = None
        self._stop_event = threading.Event()
    
    @property
    def runtime_type(self) -> Runtime:
        return Runtime.DOCKER
    
    def connect(self) -> bool:
        """Connect to Docker daemon."""
        try:
            self._client = docker.DockerClient(base_url=f"unix://{self.socket_path}")
            self._client.ping()
            return True
        except Exception as e:
            print(f"[docker] Failed to connect: {e}")
            return False
    
    def close(self):
        """Close Docker connection."""
        self._stop_event.set()
        if self._client:
            self._client.close()
            self._client = None
    
    def list_workloads(self) -> list:
        """List all discoverable containers."""
        if not self._client:
            return []
        
        # Build filter for discover label
        filters = {"label": f"{self.label_key}={self.label_value}"}
        
        workloads = []
        try:
            for container in self._client.containers.list(filters=filters):
                workload = self._container_to_workload(container)
                if workload:
                    workloads.append(workload)
        except Exception as e:
            print(f"[docker] Failed to list containers: {e}")
        
        return workloads
    
    def watch_events(self, callback: Callable[[str, Workload], None]) -> None:
        """Watch Docker events and call callback for each."""
        if not self._client:
            return
        
        self._stop_event.clear()
        
        try:
            for event in self._client.events(
                decode=True,
                filters={"type": "container"}
            ):
                if self._stop_event.is_set():
                    break
                
                action = event.get("Action")
                container_id = event.get("id")
                
                if not container_id:
                    continue
                
                if action == "start":
                    workload = self._get_container_workload(container_id)
                    if workload:
                        callback(EventType.ADDED, workload)
                
                elif action in ("stop", "die", "kill"):
                    # Create minimal workload for deletion
                    attrs = event.get("Actor", {}).get("Attributes", {})
                    workload = Workload(
                        id=container_id,
                        name=attrs.get("name", container_id[:12]),
                        hostname=container_id[:12],
                        runtime=Runtime.DOCKER.value,
                        workload_type=WorkloadType.CONTAINER.value,
                    )
                    callback(EventType.DELETED, workload)
                
                elif action in ("connect", "disconnect"):
                    # Network changed - re-fetch full workload
                    workload = self._get_container_workload(container_id)
                    if workload:
                        callback(EventType.NETWORK_CHANGED, workload)
        
        except Exception as e:
            if not self._stop_event.is_set():
                print(f"[docker] Event watch error: {e}")

    def _get_container_workload(self, container_id: str) -> Optional[Workload]:
        """Get workload for container ID, handling not found."""
        try:
            container = self._client.containers.get(container_id)
            return self._container_to_workload(container)
        except docker.errors.NotFound:
            return None
        except Exception as e:
            print(f"[docker] Failed to get container {container_id}: {e}")
            return None

    def _container_to_workload(self, container) -> Optional[Workload]:
        """Convert Docker container to Workload."""
        try:
            attrs = container.attrs
            labels = container.labels or {}
            
            # Skip if not discoverable
            if labels.get(self.label_key) != self.label_value:
                return None
            
            # Get network information
            network_settings = attrs.get("NetworkSettings", {})
            networks_info = network_settings.get("Networks", {})
            
            # Container name and hostname
            name = container.name
            hostname = container.id[:12]
            
            # Get exposed ports
            exposed_ports = attrs.get("Config", {}).get("ExposedPorts", {})
            ports = [p.split("/")[0] for p in exposed_ports.keys()] if exposed_ports else []
            
            # Build addresses in format {container_name}.{network_name}
            # This is how containers are reachable in Docker networks
            addresses = []
            for network_name in networks_info.keys():
                addresses.append(f"{name}.{network_name}")
            
            return Workload(
                id=container.id,
                name=name,
                hostname=hostname,
                runtime=Runtime.DOCKER.value,
                workload_type=WorkloadType.CONTAINER.value,
                node=None,  # Docker single-node
                namespace=None,
                addresses=addresses,
                isolation_groups=list(networks_info.keys()),
                ports=ports,
                labels=labels,
            )
        except Exception as e:
            print(f"[docker] Failed to convert container: {e}")
            return None
