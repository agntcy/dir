"""
Configuration for the discovery API server.
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

logger = logging.getLogger("discovery.server")


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
class ServerConfig:
    """HTTP server configuration."""
    host: str = field(default_factory=lambda: os.getenv("SERVER_HOST", "0.0.0.0"))
    port: int = field(default_factory=lambda: int(os.getenv("SERVER_PORT", "8080")))
    debug: bool = field(default_factory=lambda: os.getenv("DEBUG", "false").lower() == "true")


@dataclass
class Config:
    """Main configuration for the discovery server."""
    etcd: EtcdConfig = field(default_factory=EtcdConfig)
    server: ServerConfig = field(default_factory=ServerConfig)
    
    @classmethod
    def from_env(cls) -> "Config":
        """Create configuration from environment variables."""
        return cls()


def load_config() -> Config:
    """Load configuration from environment."""
    return Config.from_env()
