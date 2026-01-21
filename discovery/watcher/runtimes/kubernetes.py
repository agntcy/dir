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
    
    def list_workloads(self) -> list:
        """List all discoverable Pods with their service addresses."""
        if not self._v1:
            return []
        
        workloads = []
        
        # Build label selector string
        selector_parts = [f"{self.label_key}={self.label_value}"]
        selector_str = ",".join(selector_parts)
        
        # Cache all services for lookups (services selecting discoverable pods)
        services_by_namespace = self._get_services_by_namespace()
        
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
                    # Find services that select this pod
                    matching_services = self._find_services_for_pod(
                        pod, services_by_namespace.get(pod.metadata.namespace, [])
                    )
                    workload = self._pod_to_workload(pod, matching_services)
                    if workload:
                        workloads.append(workload)
        except Exception as e:
            print(f"[kubernetes] Failed to list pods: {e}")
        
        return workloads
    
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
    
    def _pod_to_workload(self, pod, services: list = None) -> Optional[Workload]:
        """Convert Kubernetes Pod to Workload, including service addresses."""
        try:
            labels = pod.metadata.labels or {}
            namespace = pod.metadata.namespace
            
            # Build addresses list
            addresses = []
            
            # 1. Pod DNS: {pod-ip-dashed}.{namespace}.pod
            if pod.status.pod_ip:
                ip_dashed = pod.status.pod_ip.replace(".", "-")
                addresses.append(f"{ip_dashed}.{namespace}.pod")
            
            # 2. Service DNS: {service-name}.{namespace}.svc for each service selecting this pod
            if services:
                for svc in services:
                    addresses.append(f"{svc.metadata.name}.{namespace}.svc")
            
            # Extract ports from all containers
            ports = []
            for container in pod.spec.containers:
                for port in (container.ports or []):
                    ports.append(str(port.container_port))
            
            # Isolation groups: namespace
            isolation_groups = [namespace]
            
            # Check for network policies affecting this pod
            policy_info = self._get_pod_network_policies(pod)
            
            # Build service names annotation
            service_names = [svc.metadata.name for svc in (services or [])]
            
            return Workload(
                id=pod.metadata.uid,
                name=pod.metadata.name,
                hostname=pod.spec.hostname or pod.metadata.name,
                runtime=Runtime.KUBERNETES.value,
                workload_type=WorkloadType.POD.value,
                node=pod.spec.node_name,
                namespace=namespace,
                addresses=addresses,
                isolation_groups=isolation_groups,
                ports=ports,
                labels=labels,
                annotations={
                    **(pod.metadata.annotations or {}),
                    "network_policies": policy_info,
                    "services": ",".join(service_names) if service_names else "",
                },
            )
        except Exception as e:
            print(f"[kubernetes] Failed to convert pod: {e}")
            return None
    
    def _get_services_by_namespace(self) -> dict:
        """Get all services grouped by namespace."""
        services_by_ns = {}
        try:
            if self.namespace:
                services = self._v1.list_namespaced_service(namespace=self.namespace)
            else:
                services = self._v1.list_service_for_all_namespaces()
            
            for svc in services.items:
                ns = svc.metadata.namespace
                if ns not in services_by_ns:
                    services_by_ns[ns] = []
                services_by_ns[ns].append(svc)
        except Exception as e:
            print(f"[kubernetes] Failed to list services: {e}")
        
        return services_by_ns
    
    def _find_services_for_pod(self, pod, services: list) -> list:
        """Find services that select this pod."""
        matching = []
        pod_labels = pod.metadata.labels or {}
        
        for svc in services:
            selector = svc.spec.selector
            if not selector:
                continue
            
            # Check if all selector labels match pod labels
            if all(pod_labels.get(k) == v for k, v in selector.items()):
                matching.append(svc)
        
        return matching
    
    def _service_to_workload(self, svc) -> Optional[Workload]:
        """Convert Kubernetes Service to Workload (kept for compatibility)."""
        """Convert Kubernetes Service to Workload."""
        try:
            labels = svc.metadata.labels or {}
            
            # Build address in format {service_name}.{namespace}.svc
            addresses = [f"{svc.metadata.name}.{svc.metadata.namespace}.svc"]
            
            # Extract ports from service spec
            ports = [str(port.port) for port in (svc.spec.ports or [])]
            
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
                ports=ports,
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
    
    def _watch_pods(self, callback: Callable[[str, Workload], None]):
        """Watch Pod events."""
        w = watch.Watch()
        selector = f"{self.label_key}={self.label_value}"
        
        while not self._stop_event.is_set():
            try:
                # Cache services for this watch cycle
                services_by_ns = self._get_services_by_namespace()
                
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
                    
                    # Find services that select this pod
                    matching_services = self._find_services_for_pod(
                        pod, services_by_ns.get(pod.metadata.namespace, [])
                    )
                    
                    workload = self._pod_to_workload(pod, matching_services)
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
        """Watch Service events and update affected pods."""
        w = watch.Watch()
        resource_version = None
        selector = f"{self.label_key}={self.label_value}"
        
        while not self._stop_event.is_set():
            try:
                # Watch ALL services (not just labeled ones) since they may select discoverable pods
                if self.namespace:
                    stream = w.stream(
                        self._v1.list_namespaced_service,
                        namespace=self.namespace,
                        resource_version=resource_version,
                        timeout_seconds=300,
                    )
                else:
                    stream = w.stream(
                        self._v1.list_service_for_all_namespaces,
                        resource_version=resource_version,
                        timeout_seconds=300,
                    )
                
                for event in stream:
                    if self._stop_event.is_set():
                        break
                    
                    event_type = event['type']
                    svc = event['object']
                    resource_version = svc.metadata.resource_version
                    
                    # When a service changes, refresh all discoverable pods it might select
                    if event_type in ("ADDED", "MODIFIED", "DELETED"):
                        self._refresh_pods_for_service(svc, callback, selector)
            
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
    
    def _refresh_pods_for_service(self, svc, callback, pod_selector: str):
        """Refresh pods that a service selects."""
        if not svc.spec.selector:
            return
        
        try:
            # Get all services in this namespace (for full address list)
            services_by_ns = self._get_services_by_namespace()
            namespace_services = services_by_ns.get(svc.metadata.namespace, [])
            
            # Find discoverable pods in the same namespace
            pods = self._v1.list_namespaced_pod(
                namespace=svc.metadata.namespace,
                label_selector=pod_selector
            )
            
            for pod in pods.items:
                if pod.status.phase != "Running":
                    continue
                
                # Check if this service selects this pod
                pod_labels = pod.metadata.labels or {}
                if all(pod_labels.get(k) == v for k, v in svc.spec.selector.items()):
                    # Refresh this pod with updated service list
                    matching_services = self._find_services_for_pod(pod, namespace_services)
                    workload = self._pod_to_workload(pod, matching_services)
                    if workload:
                        callback(EventType.MODIFIED, workload)
        except Exception as e:
            print(f"[kubernetes] Failed to refresh pods for service {svc.metadata.name}: {e}")
