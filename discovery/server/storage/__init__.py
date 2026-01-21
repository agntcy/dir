"""Storage module for the discovery API server."""

from storage.interface import StorageInterface
from storage.etcd import EtcdStorage
from config import logger, EtcdConfig


def create_storage(config: EtcdConfig) -> StorageInterface:
    """
    Create storage backend from config.
    
    Currently only supports etcd.
    """
    logger.info("Connecting to etcd at %s:%d", config.host, config.port)
    
    storage = EtcdStorage(config=config)
    storage.connect()

    logger.info("Storage initialized")

    return storage
