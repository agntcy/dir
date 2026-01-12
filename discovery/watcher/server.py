"""
HTTP API server for multi-runtime service discovery.
Provides endpoints to query reachability across Docker, containerd, and Kubernetes.
"""

import logging
from typing import Optional, List

from flask import Flask, jsonify, request

from models import Runtime
from storage.interface import StorageInterface

logger = logging.getLogger("discovery.server")


def create_app(storage: StorageInterface) -> Flask:
    """Create and configure the Flask application."""
    app = Flask(__name__)
    app.config['storage'] = storage
    
    @app.route('/discover', methods=['GET'])
    @app.route('/reachable', methods=['GET'])
    def discover():
        """
        Query reachability from a workload.
        
        Query parameters:
        - from: Workload ID or hostname (required)
        - runtime: Filter by runtime (docker, containerd, kubernetes)
        - type: Filter by workload type (container, pod, service)
        """
        from_identity = request.args.get('from')
        if not from_identity:
            return jsonify({"error": "Missing 'from' parameter (workload ID or hostname)"}), 400
        
        runtime_filter = request.args.get('runtime')
        type_filter = request.args.get('type')
        
        # Find source workload by ID, hostname, or name
        source = storage.get(from_identity)
        if not source:
            source = storage.get_by_hostname(from_identity)
        if not source:
            source = storage.get_by_name(from_identity)
        
        if not source:
            return jsonify({"error": f"Unknown workload: {from_identity}"}), 404
        
        # Find reachable workloads
        result = storage.find_reachable(source.id)
        reachable = result.reachable
        
        # Apply filters
        if runtime_filter:
            try:
                runtime = Runtime(runtime_filter.lower())
                reachable = [w for w in reachable if w.runtime == runtime.value]
            except ValueError:
                return jsonify({"error": f"Invalid runtime: {runtime_filter}"}), 400
        
        if type_filter:
            reachable = [w for w in reachable if w.workload_type == type_filter.lower()]
        
        return jsonify({
            "source": {
                "id": source.id,
                "name": source.name,
                "hostname": source.hostname,
                "runtime": source.runtime,
                "type": source.workload_type,
                "isolation_groups": list(source.isolation_groups),
                "addresses": source.addresses,
            },
            "reachable": [
                {
                    "id": w.id,
                    "name": w.name,
                    "hostname": w.hostname,
                    "runtime": w.runtime,
                    "type": w.workload_type,
                    "isolation_groups": list(w.isolation_groups),
                    "addresses": w.addresses,
                    "labels": w.labels,
                    "shared_groups": list(set(source.isolation_groups) & set(w.isolation_groups)),
                }
                for w in reachable
            ],
            "count": len(reachable),
            "query": {
                "from": from_identity,
                "runtime_filter": runtime_filter,
                "type_filter": type_filter,
            }
        })
    
    @app.route('/workloads', methods=['GET'])
    def list_workloads():
        """
        List all registered workloads.
        
        Query parameters:
        - runtime: Filter by runtime
        - group: Filter by isolation group
        """
        runtime_filter = request.args.get('runtime')
        group_filter = request.args.get('group')
        
        workloads = storage.list_all()
        
        if runtime_filter:
            try:
                runtime = Runtime(runtime_filter.lower())
                workloads = [w for w in workloads if w.runtime == runtime.value]
            except ValueError:
                return jsonify({"error": f"Invalid runtime: {runtime_filter}"}), 400
        
        if group_filter:
            workloads = [w for w in workloads if group_filter in w.isolation_groups]
        
        return jsonify({
            "workloads": [
                {
                    "id": w.id,
                    "name": w.name,
                    "hostname": w.hostname,
                    "runtime": w.runtime,
                    "type": w.workload_type,
                    "isolation_groups": list(w.isolation_groups),
                    "addresses": w.addresses,
                }
                for w in workloads
            ],
            "count": len(workloads),
        })
    
    @app.route('/workload/<workload_id>', methods=['GET'])
    def get_workload(workload_id: str):
        """Get a specific workload by ID."""
        workload = storage.get(workload_id)
        if not workload:
            return jsonify({"error": f"Workload not found: {workload_id}"}), 404
        
        return jsonify({
            "id": workload.id,
            "name": workload.name,
            "hostname": workload.hostname,
            "runtime": workload.runtime,
            "type": workload.workload_type,
            "isolation_groups": list(workload.isolation_groups),
            "addresses": workload.addresses,
            "labels": workload.labels,
        })
    
    @app.route('/health', methods=['GET'])
    @app.route('/healthz', methods=['GET'])
    def health():
        """Health check endpoint."""
        return jsonify({"status": "healthy"})
    
    @app.route('/ready', methods=['GET'])
    @app.route('/readyz', methods=['GET'])
    def ready():
        """Readiness check endpoint."""
        try:
            storage.list_all()
            return jsonify({"status": "ready"})
        except Exception as e:
            return jsonify({"error": f"Not ready: {e}"}), 503
    
    @app.route('/stats', methods=['GET'])
    def stats():
        """Get discovery statistics."""
        workloads = storage.list_all()
        
        by_runtime: dict = {}
        by_type: dict = {}
        all_groups: set = set()
        
        for w in workloads:
            by_runtime[w.runtime] = by_runtime.get(w.runtime, 0) + 1
            by_type[w.workload_type] = by_type.get(w.workload_type, 0) + 1
            all_groups.update(w.isolation_groups)
        
        return jsonify({
            "total_workloads": len(workloads),
            "by_runtime": by_runtime,
            "by_type": by_type,
            "isolation_groups": len(all_groups),
        })
    
    return app


class DiscoveryServer:
    """HTTP server for service discovery API."""
    
    def __init__(
        self,
        storage: StorageInterface,
        host: str = "0.0.0.0",
        port: int = 8080,
    ):
        self.storage = storage
        self.host = host
        self.port = port
        self.app = create_app(storage)
    
    def serve(self) -> None:
        """Run server in foreground (blocking)."""
        logger.info("HTTP server listening on %s:%d", self.host, self.port)
        from waitress import serve
        serve(self.app, host=self.host, port=self.port, _quiet=True)
