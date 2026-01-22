"""
Storage module for inspector.
"""

from storage.interface import StorageInterface
from storage.etcd import EtcdStorage
from config import logger, Config


def create_storage(config: Config) -> StorageInterface:
    """
    Create storage backend from config.
    
    Currently only supports etcd.
    """
    logger.info("Connecting to etcd at %s:%d", config.etcd.host, config.etcd.port)
    
    storage = EtcdStorage(config=config.etcd)
    storage.connect()

    logger.info("Storage initialized")

    return storage
