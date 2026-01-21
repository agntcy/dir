"""
Discovery API server entry point.
Serves HTTP API for querying workload reachability from etcd storage.
"""

import signal
import sys
from typing import Optional

from config import load_config, Config, logger
from storage import create_storage, StorageInterface
from app import create_app


class DiscoveryServer:
    """HTTP server for service discovery API."""
    
    def __init__(self, config: Config):
        self.config = config
        self.storage: Optional[StorageInterface] = None
        self.app = None
    
    def start(self) -> None:
        """Start the discovery API server."""
        # Initialize storage
        self.storage = create_storage(self.config.etcd)
        
        # Create Flask app
        self.app = create_app(self.storage)
        
        # Start HTTP server (blocking)
        logger.info("HTTP server listening on %s:%d", self.config.server.host, self.config.server.port)
        from waitress import serve
        serve(
            self.app,
            host=self.config.server.host,
            port=self.config.server.port,
            _quiet=True
        )
    
    def stop(self) -> None:
        """Stop the discovery server."""
        if self.storage:
            self.storage.close()
        logger.info("Discovery server stopped")


def main():
    """Main entry point."""
    config = load_config()
    
    # Log configuration
    logger.info("=" * 60)
    logger.info("Discovery API Server")
    logger.info("=" * 60)
    logger.info("Storage: etcd @ %s:%d (prefix=%s)", config.etcd.host, config.etcd.port, config.etcd.prefix)
    logger.info("HTTP server: %s:%d", config.server.host, config.server.port)
    logger.info("=" * 60)
    
    server = DiscoveryServer(config)
    
    # Handle signals
    def signal_handler(sig, frame):
        logger.info("Received signal %s, shutting down...", sig)
        server.stop()
        sys.exit(0)
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # Start
    try:
        server.start()
    except KeyboardInterrupt:
        server.stop()


if __name__ == "__main__":
    main()
