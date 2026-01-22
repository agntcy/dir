"""
Configuration for the metadata inspector service.
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

logger = logging.getLogger("discovery.inspector")


@dataclass
class EtcdConfig:
    """etcd storage configuration."""
    host: str = field(default_factory=lambda: os.getenv("ETCD_HOST", "localhost"))
    port: int = field(default_factory=lambda: int(os.getenv("ETCD_PORT", "2379")))
    workloads_prefix: str = field(default_factory=lambda: os.getenv("ETCD_WORKLOADS_PREFIX", "/discovery/workloads/"))
    metadata_prefix: str = field(default_factory=lambda: os.getenv("ETCD_METADATA_PREFIX", "/discovery/metadata/"))
    username: Optional[str] = field(default_factory=lambda: os.getenv("ETCD_USERNAME"))
    password: Optional[str] = field(default_factory=lambda: os.getenv("ETCD_PASSWORD"))
    
    @property
    def url(self) -> str:
        return f"http://{self.host}:{self.port}"


@dataclass
class ProcessorConfig:
    """Processor configuration."""
    # Health check settings
    health_enabled: bool = field(default_factory=lambda: os.getenv("HEALTH_ENABLED", "true").lower() == "true")
    health_timeout: int = field(default_factory=lambda: int(os.getenv("HEALTH_TIMEOUT", "5")))
    health_paths: list = field(default_factory=lambda: os.getenv("HEALTH_PATHS", "/").split(","))
    
    # OpenAPI settings
    openapi_enabled: bool = field(default_factory=lambda: os.getenv("OPENAPI_ENABLED", "true").lower() == "true")
    openapi_timeout: int = field(default_factory=lambda: int(os.getenv("OPENAPI_TIMEOUT", "10")))
    openapi_paths: list = field(default_factory=lambda: os.getenv("OPENAPI_PATHS", "/openapi.json,/swagger.json,/api-docs").split(","))
    
    # Processing settings
    retry_count: int = field(default_factory=lambda: int(os.getenv("PROCESSOR_RETRY_COUNT", "3")))
    retry_delay: int = field(default_factory=lambda: int(os.getenv("PROCESSOR_RETRY_DELAY", "5")))
    worker_count: int = field(default_factory=lambda: int(os.getenv("PROCESSOR_WORKERS", "4")))


@dataclass
class Config:
    """Main configuration for the inspector."""
    etcd: EtcdConfig = field(default_factory=EtcdConfig)
    processor: ProcessorConfig = field(default_factory=ProcessorConfig)
    
    @classmethod
    def from_env(cls) -> "Config":
        """Create configuration from environment variables."""
        return cls()


def load_config() -> Config:
    """Load configuration from environment."""
    return Config.from_env()
