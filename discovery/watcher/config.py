"""
Configuration for multi-runtime service discovery.
Supports Docker, containerd, and Kubernetes runtimes with etcd storage.
"""

import logging
import os
import sys
from dataclasses import dataclass, field
from typing import Optional

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - [%(name)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    stream=sys.stdout,
    force=True
)

logger = logging.getLogger("discovery")


@dataclass
class EtcdConfig:
    """etcd storage configuration."""
    host: str = field(default_factory=lambda: os.getenv("ETCD_HOST", "localhost"))
    port: int = field(default_factory=lambda: int(os.getenv("ETCD_PORT", "2379")))
    prefix: str = field(default_factory=lambda: os.getenv("ETCD_PREFIX", "/discovery"))
    username: Optional[str] = field(default_factory=lambda: os.getenv("ETCD_USERNAME"))
    password: Optional[str] = field(default_factory=lambda: os.getenv("ETCD_PASSWORD"))
    
    @property
    def url(self) -> str:
        return f"http://{self.host}:{self.port}"


@dataclass
class DockerConfig:
    """Docker runtime configuration."""
    socket: str = field(default_factory=lambda: os.getenv("DOCKER_SOCKET", "unix:///var/run/docker.sock"))
    label_key: str = field(default_factory=lambda: os.getenv("DOCKER_LABEL_KEY", "discover"))
    label_value: str = field(default_factory=lambda: os.getenv("DOCKER_LABEL_VALUE", "true"))


@dataclass  
class ContainerdConfig:
    """containerd runtime configuration."""
    socket: str = field(default_factory=lambda: os.getenv("CONTAINERD_SOCKET", "/run/containerd/containerd.sock"))
    namespace: str = field(default_factory=lambda: os.getenv("CONTAINERD_NAMESPACE", "default"))
    cni_state_dir: str = field(default_factory=lambda: os.getenv("CONTAINERD_CNI_STATE_DIR", "/var/lib/cni/results"))
    label_key: str = field(default_factory=lambda: os.getenv("CONTAINERD_LABEL_KEY", "discover"))
    label_value: str = field(default_factory=lambda: os.getenv("CONTAINERD_LABEL_VALUE", "true"))


@dataclass
class KubernetesConfig:
    """Kubernetes runtime configuration."""
    kubeconfig: Optional[str] = field(default_factory=lambda: os.getenv("KUBECONFIG"))
    namespace: Optional[str] = field(default_factory=lambda: os.getenv("KUBERNETES_NAMESPACE"))  # None = all namespaces
    in_cluster: bool = field(default_factory=lambda: os.getenv("KUBERNETES_IN_CLUSTER", "false").lower() == "true")
    label_key: str = field(default_factory=lambda: os.getenv("KUBERNETES_LABEL_KEY", "discover"))
    label_value: str = field(default_factory=lambda: os.getenv("KUBERNETES_LABEL_VALUE", "true"))
    watch_services: bool = field(default_factory=lambda: os.getenv("KUBERNETES_WATCH_SERVICES", "true").lower() == "true")


@dataclass
class ServerConfig:
    """HTTP server configuration."""
    host: str = field(default_factory=lambda: os.getenv("SERVER_HOST", "0.0.0.0"))
    port: int = field(default_factory=lambda: int(os.getenv("SERVER_PORT", "8080")))
    debug: bool = field(default_factory=lambda: os.getenv("DEBUG", "false").lower() == "true")


@dataclass
class Config:
    """Main configuration container for service discovery."""
    runtime: str = field(default_factory=lambda: os.getenv("RUNTIME", "docker"))
    etcd: EtcdConfig = field(default_factory=EtcdConfig)
    docker: DockerConfig = field(default_factory=DockerConfig)
    containerd: ContainerdConfig = field(default_factory=ContainerdConfig)
    kubernetes: KubernetesConfig = field(default_factory=KubernetesConfig)
    server: ServerConfig = field(default_factory=ServerConfig)
    
    @classmethod
    def from_env(cls) -> "Config":
        """Create configuration from environment variables."""
        return cls()


def load_config() -> Config:
    """Load configuration from environment."""
    return Config.from_env()
