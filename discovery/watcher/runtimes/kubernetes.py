"""
Kubernetes runtime adapter.

Watches Kubernetes API for Pod and Service events.
Evaluates NetworkPolicies for reachability (stored as metadata).
"""

import threading
from typing import Optional, Callable

from kubernetes import client, config, watch
from kubernetes.client.rest import ApiException

from models import Workload, Runtime, WorkloadType, EventType
from config import KubernetesConfig
from runtimes.interface import RuntimeAdapter


class KubernetesAdapter(RuntimeAdapter):
    """
    Kubernetes runtime adapter.
    
    Watches Pods and Services via Kubernetes API.
    Supports both in-cluster and kubeconfig authentication.
    Evaluates NetworkPolicies and stores as workload annotations.
    """
    
    def __init__(self, config: KubernetesConfig):
        """
        Initialize Kubernetes adapter.
        
        Args:
            config: KubernetesConfig instance
        """
        self.namespace = config.namespace  # None = all namespaces
        self.label_key = config.label_key
        self.label_value = config.label_value
        self.include_services = config.watch_services
        self.in_cluster = config.in_cluster
        self.kubeconfig = config.kubeconfig
        
        self._v1: Optional[client.CoreV1Api] = None
        self._networking: Optional[client.NetworkingV1Api] = None
        self._stop_event = threading.Event()
        self._resource_version = None
    
    @property
    def runtime_type(self) -> Runtime:
        return Runtime.KUBERNETES
    
    def connect(self) -> bool:
        """Connect to Kubernetes API."""
        try:
            # Load config based on settings
            if self.in_cluster:
                config.load_incluster_config()
                print("[kubernetes] Using in-cluster config")
            elif self.kubeconfig:
                config.load_kube_config(config_file=self.kubeconfig)
                print(f"[kubernetes] Using kubeconfig: {self.kubeconfig}")
            else:
                # Try in-cluster first, then kubeconfig
                try:
                    config.load_incluster_config()
                    print("[kubernetes] Using in-cluster config")
                except config.ConfigException:
                    config.load_kube_config()
                    print("[kubernetes] Using default kubeconfig")
            
            self._v1 = client.CoreV1Api()
            self._networking = client.NetworkingV1Api()
            
            # Verify connection
            self._v1.list_namespace(limit=1)
            return True
        
        except Exception as e:
            print(f"[kubernetes] Failed to connect: {e}")
            return False
    
    def close(self):
        """Close Kubernetes connection."""
        self._stop_event.set()
        self._v1 = None
        self._networking = None
    
    # ==================== Discovery ====================
    
    def list_workloads(self, label_selector: dict = None) -> list:
        """List all discoverable Pods and Services."""
        if not self._v1:
            return []
        
        workloads = []
        
        # Build label selector string
        selector_parts = [f"{self.label_key}={self.label_value}"]
        if label_selector:
            selector_parts.extend(f"{k}={v}" for k, v in label_selector.items())
        selector_str = ",".join(selector_parts)
        
        # List Pods
        try:
            if self.namespace:
                pods = self._v1.list_namespaced_pod(
                    namespace=self.namespace,
                    label_selector=selector_str
                )
            else:
                pods = self._v1.list_pod_for_all_namespaces(
                    label_selector=selector_str
                )
            
            for pod in pods.items:
                if pod.status.phase == "Running":
                    workload = self._pod_to_workload(pod)
                    if workload:
                        workloads.append(workload)
        except Exception as e:
            print(f"[kubernetes] Failed to list pods: {e}")
        
        # List Services
        if self.include_services:
            try:
                if self.namespace:
                    services = self._v1.list_namespaced_service(
                        namespace=self.namespace,
                        label_selector=selector_str
                    )
                else:
                    services = self._v1.list_service_for_all_namespaces(
                        label_selector=selector_str
                    )
                
                for svc in services.items:
                    workload = self._service_to_workload(svc)
                    if workload:
                        workloads.append(workload)
            except Exception as e:
                print(f"[kubernetes] Failed to list services: {e}")
        
        return workloads
    
    def get_workload(self, identity: str) -> Optional[Workload]:
        """Get Pod or Service by identity."""
        if not self._v1:
            return None
        
        # Try to parse namespace/name format
        namespace = self.namespace
        name = identity
        if "/" in identity:
            namespace, name = identity.split("/", 1)
        
        # Try as Pod first
        try:
            if namespace:
                pod = self._v1.read_namespaced_pod(name=name, namespace=namespace)
            else:
                # Search all namespaces
                pods = self._v1.list_pod_for_all_namespaces(
                    field_selector=f"metadata.name={name}"
                )
                pod = pods.items[0] if pods.items else None
            
            if pod:
                return self._pod_to_workload(pod)
        except ApiException:
            pass
        
        # Try as Service
        if self.include_services:
            try:
                if namespace:
                    svc = self._v1.read_namespaced_service(name=name, namespace=namespace)
                else:
                    services = self._v1.list_service_for_all_namespaces(
                        field_selector=f"metadata.name={name}"
                    )
                    svc = services.items[0] if services.items else None
                
                if svc:
                    return self._service_to_workload(svc)
            except ApiException:
                pass
        
        return None
    
    def _pod_to_workload(self, pod) -> Optional[Workload]:
        """Convert Kubernetes Pod to Workload."""
        try:
            labels = pod.metadata.labels or {}
            
            # Build addresses
            addresses = []
            if pod.status.pod_ip:
                for container in pod.spec.containers:
                    for port in (container.ports or []):
                        addresses.append(f"{pod.status.pod_ip}:{port.container_port}")
                
                # If no ports defined, use pod IP
                if not addresses:
                    addresses.append(pod.status.pod_ip)
            
            # Isolation groups: namespace + any network policy annotations
            isolation_groups = [pod.metadata.namespace]
            
            # Check for network policies affecting this pod
            policy_info = self._get_pod_network_policies(pod)
            
            return Workload(
                id=pod.metadata.uid,
                name=pod.metadata.name,
                hostname=pod.spec.hostname or pod.metadata.name,
                runtime=Runtime.KUBERNETES.value,
                workload_type=WorkloadType.POD.value,
                node=pod.spec.node_name,
                namespace=pod.metadata.namespace,
                addresses=addresses,
                isolation_groups=isolation_groups,
                labels=labels,
                annotations={
                    **(pod.metadata.annotations or {}),
                    "network_policies": policy_info,
                },
            )
        except Exception as e:
            print(f"[kubernetes] Failed to convert pod: {e}")
            return None
    
    def _service_to_workload(self, svc) -> Optional[Workload]:
        """Convert Kubernetes Service to Workload."""
        try:
            labels = svc.metadata.labels or {}
            
            # Build addresses
            addresses = []
            for port in (svc.spec.ports or []):
                # ClusterIP:port
                if svc.spec.cluster_ip and svc.spec.cluster_ip != "None":
                    addresses.append(f"{svc.spec.cluster_ip}:{port.port}")
                
                # DNS name
                dns_name = f"{svc.metadata.name}.{svc.metadata.namespace}.svc.cluster.local"
                addresses.append(f"{dns_name}:{port.port}")
            
            return Workload(
                id=svc.metadata.uid,
                name=svc.metadata.name,
                hostname=svc.metadata.name,
                runtime=Runtime.KUBERNETES.value,
                workload_type=WorkloadType.SERVICE.value,
                node=None,  # Services are cluster-wide
                namespace=svc.metadata.namespace,
                addresses=addresses,
                isolation_groups=[svc.metadata.namespace],
                labels=labels,
                annotations=svc.metadata.annotations or {},
            )
        except Exception as e:
            print(f"[kubernetes] Failed to convert service: {e}")
            return None
    
    def _get_pod_network_policies(self, pod) -> str:
        """Get NetworkPolicies affecting a pod."""
        if not self._networking:
            return "unknown"
        
        try:
            policies = self._networking.list_namespaced_network_policy(
                namespace=pod.metadata.namespace
            )
            
            if not policies.items:
                return "none (default allow)"
            
            # Find policies that apply to this pod
            affecting = []
            for policy in policies.items:
                selector = policy.spec.pod_selector
                if self._selector_matches(selector, pod.metadata.labels or {}):
                    affecting.append(policy.metadata.name)
            
            if affecting:
                return f"restricted by: {', '.join(affecting)}"
            return "not targeted by any policy"
        
        except Exception as e:
            return f"error: {e}"
    
    def _selector_matches(self, selector, labels: dict) -> bool:
        """Check if label selector matches labels."""
        if not selector or not selector.match_labels:
            return True  # Empty selector matches all
        
        for key, value in selector.match_labels.items():
            if labels.get(key) != value:
                return False
        return True
    
    # ==================== Events ====================
    
    def watch_events(self, callback: Callable[[str, Workload], None]) -> None:
        """Watch Kubernetes Pod and Service events."""
        if not self._v1:
            return
        
        self._stop_event.clear()
        
        # Watch Pods
        pod_thread = threading.Thread(
            target=self._watch_pods,
            args=(callback,),
            daemon=True
        )
        pod_thread.start()
        
        # Watch Services (if enabled)
        if self.include_services:
            svc_thread = threading.Thread(
                target=self._watch_services,
                args=(callback,),
                daemon=True
            )
            svc_thread.start()
        
        # Keep main thread alive
        while not self._stop_event.is_set():
            self._stop_event.wait(1)
    
    def _watch_pods(self, callback: Callable[[str, Workload], None]):
        """Watch Pod events."""
        w = watch.Watch()
        selector = f"{self.label_key}={self.label_value}"
        
        while not self._stop_event.is_set():
            try:
                if self.namespace:
                    stream = w.stream(
                        self._v1.list_namespaced_pod,
                        namespace=self.namespace,
                        label_selector=selector,
                        resource_version=self._resource_version,
                        timeout_seconds=300,
                    )
                else:
                    stream = w.stream(
                        self._v1.list_pod_for_all_namespaces,
                        label_selector=selector,
                        resource_version=self._resource_version,
                        timeout_seconds=300,
                    )
                
                for event in stream:
                    if self._stop_event.is_set():
                        break
                    
                    event_type = event['type']
                    pod = event['object']
                    
                    # Update resource version for resume
                    self._resource_version = pod.metadata.resource_version
                    
                    workload = self._pod_to_workload(pod)
                    if not workload:
                        continue
                    
                    if event_type == "ADDED":
                        if pod.status.phase == "Running":
                            callback(EventType.ADDED, workload)
                    elif event_type == "MODIFIED":
                        if pod.status.phase == "Running":
                            callback(EventType.MODIFIED, workload)
                        elif pod.status.phase in ("Succeeded", "Failed"):
                            callback(EventType.DELETED, workload)
                    elif event_type == "DELETED":
                        callback(EventType.DELETED, workload)
            
            except ApiException as e:
                if e.status == 410:  # Gone - resource version too old
                    print("[kubernetes] Watch expired, restarting...")
                    self._resource_version = None
                else:
                    print(f"[kubernetes] Pod watch error: {e}")
                    if not self._stop_event.is_set():
                        self._stop_event.wait(5)
            except Exception as e:
                print(f"[kubernetes] Pod watch error: {e}")
                if not self._stop_event.is_set():
                    self._stop_event.wait(5)
    
    def _watch_services(self, callback: Callable[[str, Workload], None]):
        """Watch Service events."""
        w = watch.Watch()
        selector = f"{self.label_key}={self.label_value}"
        resource_version = None
        
        while not self._stop_event.is_set():
            try:
                if self.namespace:
                    stream = w.stream(
                        self._v1.list_namespaced_service,
                        namespace=self.namespace,
                        label_selector=selector,
                        resource_version=resource_version,
                        timeout_seconds=300,
                    )
                else:
                    stream = w.stream(
                        self._v1.list_service_for_all_namespaces,
                        label_selector=selector,
                        resource_version=resource_version,
                        timeout_seconds=300,
                    )
                
                for event in stream:
                    if self._stop_event.is_set():
                        break
                    
                    event_type = event['type']
                    svc = event['object']
                    
                    resource_version = svc.metadata.resource_version
                    
                    workload = self._service_to_workload(svc)
                    if not workload:
                        continue
                    
                    if event_type == "ADDED":
                        callback(EventType.ADDED, workload)
                    elif event_type == "MODIFIED":
                        callback(EventType.MODIFIED, workload)
                    elif event_type == "DELETED":
                        callback(EventType.DELETED, workload)
            
            except ApiException as e:
                if e.status == 410:
                    resource_version = None
                else:
                    print(f"[kubernetes] Service watch error: {e}")
                    if not self._stop_event.is_set():
                        self._stop_event.wait(5)
            except Exception as e:
                print(f"[kubernetes] Service watch error: {e}")
                if not self._stop_event.is_set():
                    self._stop_event.wait(5)
    
    def stop_watch(self):
        """Stop the event watch loops."""
        self._stop_event.set()
