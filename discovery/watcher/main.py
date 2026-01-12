"""
Service discovery entry point.
Supports Docker, containerd, or Kubernetes workload discovery with etcd storage.
"""

import signal
import sys
import threading
from typing import Optional

from config import load_config, Config, logger
from models import Workload, EventType
from storage import create_storage, StorageInterface
from runtimes import create_runtime, RuntimeAdapter
from server import DiscoveryServer


class ServiceDiscovery:
    """
    Service discovery coordinator.
    
    Manages a runtime adapter and routes workload events to storage.
    """
    
    def __init__(self, config: Config):
        self.config = config
        self.storage: Optional[StorageInterface] = None
        self.server: Optional[DiscoveryServer] = None
        self.runtime: Optional[RuntimeAdapter] = None
        self._running = False
    
    def _handle_event(self, event_type: EventType, workload: Workload) -> None:
        """Handle workload event from runtime."""
        if not self.storage:
            return
        
        if event_type in (EventType.ADDED, EventType.MODIFIED, EventType.NETWORK_CHANGED):
            self.storage.register(workload)
            logger.info(
                "[%s] %s workload: %s (%s) groups=%s",
                event_type.value.upper(),
                workload.runtime,
                workload.name,
                workload.id[:12],
                list(workload.isolation_groups)
            )
        elif event_type == EventType.DELETED:
            self.storage.deregister(workload.id)
            logger.info(
                "[DELETED] %s workload: %s (%s)",
                workload.runtime,
                workload.name,
                workload.id[:12]
            )
    
    def _sync_initial_state(self) -> None:
        """Sync initial workload state from runtime."""
        logger.info("Syncing initial workload state...")
        
        try:
            workloads = self.runtime.list_workloads()
            for workload in workloads:
                self.storage.register(workload)
            logger.info("Synced %d workloads from %s", len(workloads), self.runtime.runtime_type.value)
        except Exception as e:
            logger.error("Failed to sync from %s: %s", self.runtime.runtime_type.value, e)
    
    def _start_watcher(self) -> None:
        """Start event watcher for runtime in background thread."""
        try:
            thread = threading.Thread(
                target=self.runtime.watch_events,
                args=(self._handle_event,),
                daemon=True,
                name=f"watcher-{self.runtime.runtime_type.value}"
            )
            thread.start()
            logger.info("Started event watcher for %s", self.runtime.runtime_type.value)
        except Exception as e:
            logger.error("Failed to start watcher for %s: %s", self.runtime.runtime_type.value, e)
    
    def start(self) -> None:
        """Start the service discovery system."""
        self._running = True
        
        # Initialize storage
        self.storage = create_storage(self.config)
        
        # Initialize runtime
        self.runtime = create_runtime(self.config)
        if not self.runtime:
            logger.error("Failed to initialize runtime!")
            sys.exit(1)
        
        # Sync and start watcher
        self._sync_initial_state()
        self._start_watcher()
        
        # Initialize and start HTTP server (blocking)
        self.server = DiscoveryServer(
            storage=self.storage,
            host=self.config.server.host,
            port=self.config.server.port,
        )
        self.server.serve()
    
    def stop(self) -> None:
        """Stop the service discovery system."""
        self._running = False
        
        # Stop watcher
        if self.runtime:
            try:
                self.runtime.stop_watch()
            except Exception as e:
                logger.warning("Error stopping %s watcher: %s", self.runtime.runtime_type.value, e)
        
        # Stop server
        if self.server:
            self.server.stop()
        
        # Stop storage
        if self.storage:
            self.storage.close()
        
        logger.info("Service discovery stopped")


def main():
    """Main entry point."""
    config = load_config()
    
    # Log configuration
    logger.info("=" * 60)
    logger.info("Service Discovery")
    logger.info("=" * 60)
    logger.info("Runtime: %s", config.runtime)
    logger.info("Storage: etcd @ %s:%d (prefix=%s)", config.etcd.host, config.etcd.port, config.etcd.prefix)
    logger.info("HTTP server: %s:%d", config.server.host, config.server.port)
    logger.info("=" * 60)
    
    discovery = ServiceDiscovery(config)
    
    # Handle signals
    def signal_handler(sig, frame):
        logger.info("Received signal %s, shutting down...", sig)
        discovery.stop()
        sys.exit(0)
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # Start
    try:
        discovery.start()
    except KeyboardInterrupt:
        discovery.stop()


if __name__ == "__main__":
    main()
