"""Runtime adapters module for service discovery."""

import logging

from config import Config
from runtimes.interface import RuntimeAdapter

logger = logging.getLogger("discovery")


def create_runtime(config: Config) -> RuntimeAdapter:
    """
    Create the configured runtime adapter.
    
    Returns connected adapter or None if connection fails.
    """
    runtime_type = config.runtime
    
    # Initialize appropriate runtime adapter
    if runtime_type == "docker":
        from runtimes.docker import DockerAdapter
        adapter = DockerAdapter(config.docker)
    elif runtime_type == "containerd":
        from runtimes.containerd import ContainerdAdapter
        adapter = ContainerdAdapter(config.containerd)
    elif runtime_type == "kubernetes":
        from runtimes.kubernetes import KubernetesAdapter
        adapter = KubernetesAdapter(config.kubernetes)
    else:
        logger.error("Unknown runtime: %s (use: docker, containerd, kubernetes)", runtime_type)
        return None

    # Connect to runtime    
    adapter.connect()
    logger.info("%s runtime connected", runtime_type)
    return adapter
