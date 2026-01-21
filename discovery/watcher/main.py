"""
Workload watcher entry point.
Watches Docker, containerd, or Kubernetes for workload changes and syncs to etcd storage.
"""

import signal
import sys
import threading
import time
from typing import Optional

from config import load_config, Config, logger
from models import Workload, EventType
from storage import create_storage, StorageInterface
from runtimes import create_runtime, RuntimeAdapter


class WorkloadWatcher:
    """
    Workload watcher coordinator.
    
    Watches a runtime for workload events and syncs to etcd storage.
    """
    
    def __init__(self, config: Config):
        self.config = config
        self.storage: Optional[StorageInterface] = None
        self.runtime: Optional[RuntimeAdapter] = None
        self._running = False
        self._stop_event = threading.Event()
    
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
        """Start the workload watcher."""
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
        
        logger.info("Watcher running, press Ctrl+C to stop...")
        
        # Keep main thread alive
        while not self._stop_event.is_set():
            self._stop_event.wait(timeout=1)
    
    def stop(self) -> None:
        """Stop the workload watcher."""
        self._running = False
        self._stop_event.set()
        
        # Stop watcher
        if self.runtime:
            try:
                self.runtime.close()
            except Exception as e:
                logger.warning("Error stopping %s watcher: %s", self.runtime.runtime_type.value, e)
        
        # Stop storage
        if self.storage:
            self.storage.close()
        
        logger.info("Workload watcher stopped")


def main():
    """Main entry point."""
    config = load_config()
    
    # Log configuration
    logger.info("=" * 60)
    logger.info("Workload Watcher")
    logger.info("=" * 60)
    logger.info("Runtime: %s", config.runtime)
    logger.info("Storage: etcd @ %s:%d (prefix=%s)", config.etcd.host, config.etcd.port, config.etcd.prefix)
    logger.info("=" * 60)
    
    watcher = WorkloadWatcher(config)
    
    # Handle signals
    def signal_handler(sig, frame):
        logger.info("Received signal %s, shutting down...", sig)
        watcher.stop()
        sys.exit(0)
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # Start
    try:
        watcher.start()
    except KeyboardInterrupt:
        watcher.stop()


if __name__ == "__main__":
    main()
