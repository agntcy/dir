"""
Metadata Inspector entry point.

Watches for workloads in etcd and runs processors to extract metadata.
"""

import signal
import sys
import threading
import queue
from typing import Optional

from models import Workload
from config import load_config, Config, logger
from storage import create_storage, StorageInterface
from processors import create_processors, ProcessorInterface


class MetadataInspector:
    """
    Metadata inspector coordinator.
    
    Watches etcd for workload changes and runs processors to extract metadata.
    """
    
    def __init__(self, config: Config):
        self.config = config
        self.storage: Optional[StorageInterface] = None
        self.processors: list[ProcessorInterface] = []
        self._running = False
        self._stop_event = threading.Event()
        self._work_queue: queue.Queue = queue.Queue()
        self._workers: list[threading.Thread] = []
    
    def _process_workload(self, workload: Workload) -> None:
        """Run all processors on a workload."""
        for processor in self.processors:
            if not processor.should_process(workload):
                continue
            
            try:
                result = processor.process(workload)
                if result:
                    self.storage.set_metadata(workload.id, processor.name, result)
            except Exception as e:
                logger.error(
                    "Processor %s failed for %s: %s",
                    processor.name,
                    workload.name,
                    e
                )
    
    def _worker(self, worker_id: int) -> None:
        """Worker thread that processes workloads from the queue."""
        logger.info("Worker %d started", worker_id)
        
        while not self._stop_event.is_set():
            try:
                # Get workload from queue with timeout
                try:
                    event_type, workload = self._work_queue.get(timeout=1)
                except queue.Empty:
                    continue
                
                if event_type == "PUT" and workload:
                    logger.debug(
                        "Worker %d processing %s (%s)",
                        worker_id,
                        workload.name,
                        workload.id[:12]
                    )
                    self._process_workload(workload)
                
                elif event_type == "DELETE" and workload:
                    # Clean up metadata when workload is deleted
                    self.storage.delete_metadata(workload.id)
                
                self._work_queue.task_done()
            
            except Exception as e:
                logger.error("Worker %d error: %s", worker_id, e)
        
        logger.info("Worker %d stopped", worker_id)
    
    def _watcher(self) -> None:
        """Watch etcd for workload changes and queue them."""
        logger.info("Watcher started")
        
        try:
            for event_type, workload in self.storage.watch_workloads():
                if self._stop_event.is_set():
                    break
                
                if workload:
                    logger.info(
                        "[%s] Queued %s (%s)",
                        event_type,
                        workload.name,
                        workload.id[:12]
                    )
                    self._work_queue.put((event_type, workload))
        
        except Exception as e:
            logger.error("Watcher error: %s", e)
        
        logger.info("Watcher stopped")
    
    def start(self) -> None:
        """Start the metadata inspector."""
        self._running = True
        
        # Initialize storage
        self.storage = create_storage(self.config)
        
        # Initialize processors
        self.processors = create_processors(self.config)
        if not self.processors:
            logger.error("Failed to initialize processors!")
            sys.exit(1)
        
        # Start worker threads
        worker_count = self.config.processor.worker_count
        for i in range(worker_count):
            worker = threading.Thread(
                target=self._worker,
                args=(i,),
                daemon=True,
                name=f"worker-{i}"
            )
            worker.start()
            self._workers.append(worker)
        
        logger.info("Started %d worker threads", worker_count)
        
        # Start watcher in main thread
        logger.info("Inspector running, press Ctrl+C to stop...")
        self._watcher()
    
    def stop(self) -> None:
        """Stop the metadata inspector."""
        logger.info("Stopping inspector...")
        self._running = False
        self._stop_event.set()
        
        # Wait for workers to finish
        for worker in self._workers:
            worker.join(timeout=5)
        
        # Close storage
        if self.storage:
            self.storage.close()
        
        logger.info("Inspector stopped")


def main():
    """Main entry point."""
    config = load_config()
    
    # Log configuration
    logger.info("=" * 60)
    logger.info("Metadata Inspector")
    logger.info("=" * 60)
    logger.info("etcd: %s:%d", config.etcd.host, config.etcd.port)
    logger.info("Runtime prefix: %s", config.etcd.runtime_prefix)
    logger.info("Metadata prefix: %s", config.etcd.metadata_prefix)
    logger.info("Workers: %d", config.processor.worker_count)
    logger.info("Health check: %s", "enabled" if config.processor.health_enabled else "disabled")
    logger.info("OpenAPI discovery: %s", "enabled" if config.processor.openapi_enabled else "disabled")
    logger.info("=" * 60)
    
    inspector = MetadataInspector(config)
    
    # Handle signals
    def signal_handler(sig, frame):
        logger.info("Received signal %s, shutting down...", sig)
        inspector.stop()
        sys.exit(0)
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # Start
    try:
        inspector.start()
    except KeyboardInterrupt:
        inspector.stop()


if __name__ == "__main__":
    main()
