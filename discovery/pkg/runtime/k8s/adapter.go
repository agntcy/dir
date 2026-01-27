package k8s

import (
	"context"
	"log"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/agntcy/dir/discovery/pkg/types"
)

const RuntimeType types.RuntimeType = "kubernetes"

// adapter implements the Adapter interface for Kubernetes.
type adapter struct {
	clientset     *kubernetes.Clientset
	namespace     string
	labelKey      string
	labelValue    string
	watchServices bool
}

// NewAdapter creates a new Kubernetes adapter.
func NewAdapter(cfg Config) (types.RuntimeAdapter, error) {
	var kubeConfig *rest.Config
	var err error

	if cfg.InCluster {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		log.Printf("[kubernetes] Using in-cluster config")
	} else if cfg.Kubeconfig != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
		if err != nil {
			return nil, err
		}
		log.Printf("[kubernetes] Using kubeconfig: %s", cfg.Kubeconfig)
	} else {
		// Try in-cluster first, then default kubeconfig
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			kubeConfigPath := os.Getenv("HOME") + "/.kube/config"
			kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
			if err != nil {
				return nil, err
			}
			log.Printf("[kubernetes] Using default kubeconfig")
		} else {
			log.Printf("[kubernetes] Using in-cluster config")
		}
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	return &adapter{
		clientset:     clientset,
		namespace:     cfg.Namespace,
		labelKey:      cfg.LabelKey,
		labelValue:    cfg.LabelValue,
		watchServices: cfg.WatchServices,
	}, nil
}

// RuntimeType returns the Kubernetes runtime type.
func (k *adapter) Type() types.RuntimeType {
	return RuntimeType
}

// Connect verifies the Kubernetes connection.
func (k *adapter) Connect(ctx context.Context) error {
	_, err := k.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return err
	}
	log.Printf("[kubernetes] Connected to Kubernetes API")
	return nil
}

// Close closes the Kubernetes client (no-op, client doesn't need explicit close).
func (k *adapter) Close() error {
	return nil
}

// ListWorkloads returns all discoverable pods.
func (k *adapter) ListWorkloads(ctx context.Context) ([]*types.Workload, error) {
	labelSelector := k.labelKey + "=" + k.labelValue

	var pods *corev1.PodList
	var err error

	if k.namespace != "" {
		pods, err = k.clientset.CoreV1().Pods(k.namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
	} else {
		pods, err = k.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
	}
	if err != nil {
		return nil, err
	}

	// Get services for address building
	servicesByNamespace := k.getServicesByNamespace(ctx)

	var workloads []*types.Workload
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		matchingServices := k.findServicesForPod(&pod, servicesByNamespace[pod.Namespace])
		workload := k.podToWorkload(&pod, matchingServices)
		if workload != nil {
			workloads = append(workloads, workload)
		}
	}

	return workloads, nil
}

// WatchEvents watches Kubernetes pod events and sends workload events to the channel.
func (k *adapter) WatchEvents(ctx context.Context, eventChan chan<- *types.RuntimeEvent) error {
	labelSelector := k.labelKey + "=" + k.labelValue

	// Watch pods
	go k.watchPods(ctx, labelSelector, eventChan)

	// Watch services if enabled (for updating pod addresses)
	if k.watchServices {
		go k.watchServices_(ctx, eventChan)
	}

	<-ctx.Done()
	return ctx.Err()
}

// watchPods watches pod events.
func (k *adapter) watchPods(ctx context.Context, labelSelector string, eventChan chan<- *types.RuntimeEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var watcher watch.Interface
		var err error

		if k.namespace != "" {
			watcher, err = k.clientset.CoreV1().Pods(k.namespace).Watch(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})
		} else {
			watcher, err = k.clientset.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{
				LabelSelector: labelSelector,
			})
		}

		if err != nil {
			log.Printf("[kubernetes] Pod watch error: %v", err)
			continue
		}

		servicesByNamespace := k.getServicesByNamespace(ctx)

		for event := range watcher.ResultChan() {
			if ctx.Err() != nil {
				watcher.Stop()
				return
			}

			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				continue
			}

			matchingServices := k.findServicesForPod(pod, servicesByNamespace[pod.Namespace])
			workload := k.podToWorkload(pod, matchingServices)
			if workload == nil {
				continue
			}

			switch event.Type {
			case watch.Added:
				if pod.Status.Phase == corev1.PodRunning {
					eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeAdded, Workload: workload}
				}

			case watch.Modified:
				if pod.Status.Phase == corev1.PodRunning {
					eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeModified, Workload: workload}
				} else if pod.Status.Phase == corev1.PodSucceeded ||
					pod.Status.Phase == corev1.PodFailed ||
					pod.Status.Phase == corev1.PodPending {
					eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeDeleted, Workload: workload}
				}

			case watch.Deleted:
				eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeDeleted, Workload: workload}
			}
		}

		watcher.Stop()
		log.Printf("[kubernetes] Pod watch ended, restarting...")
	}
}

// watchServices_ watches service events to update pod addresses.
func (k *adapter) watchServices_(ctx context.Context, eventChan chan<- *types.RuntimeEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var watcher watch.Interface
		var err error

		if k.namespace != "" {
			watcher, err = k.clientset.CoreV1().Services(k.namespace).Watch(ctx, metav1.ListOptions{})
		} else {
			watcher, err = k.clientset.CoreV1().Services("").Watch(ctx, metav1.ListOptions{})
		}

		if err != nil {
			log.Printf("[kubernetes] Service watch error: %v", err)
			continue
		}

		for event := range watcher.ResultChan() {
			if ctx.Err() != nil {
				watcher.Stop()
				return
			}

			svc, ok := event.Object.(*corev1.Service)
			if !ok {
				continue
			}

			// When a service changes, re-emit events for affected pods
			if event.Type == watch.Added || event.Type == watch.Modified || event.Type == watch.Deleted {
				k.updatePodsForService(ctx, svc, eventChan)
			}
		}

		watcher.Stop()
		log.Printf("[kubernetes] Service watch ended, restarting...")
	}
}

// updatePodsForService re-emits events for pods affected by a service change.
func (k *adapter) updatePodsForService(ctx context.Context, svc *corev1.Service, eventChan chan<- *types.RuntimeEvent) {
	if svc.Spec.Selector == nil {
		return
	}

	// Find pods matching this service's selector
	var selectorParts []string
	for k, v := range svc.Spec.Selector {
		selectorParts = append(selectorParts, k+"="+v)
	}
	labelSelector := strings.Join(selectorParts, ",")

	pods, err := k.clientset.CoreV1().Pods(svc.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		log.Printf("[kubernetes] Failed to list pods for service %s: %v", svc.Name, err)
		return
	}

	servicesByNamespace := k.getServicesByNamespace(ctx)

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		// Check if pod has discover label
		if pod.Labels[k.labelKey] != k.labelValue {
			continue
		}

		matchingServices := k.findServicesForPod(&pod, servicesByNamespace[pod.Namespace])
		workload := k.podToWorkload(&pod, matchingServices)
		if workload != nil {
			eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeModified, Workload: workload}
		}
	}
}

// getServicesByNamespace returns all services grouped by namespace.
func (k *adapter) getServicesByNamespace(ctx context.Context) map[string][]*corev1.Service {
	result := make(map[string][]*corev1.Service)

	var services *corev1.ServiceList
	var err error

	if k.namespace != "" {
		services, err = k.clientset.CoreV1().Services(k.namespace).List(ctx, metav1.ListOptions{})
	} else {
		services, err = k.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		log.Printf("[kubernetes] Failed to list services: %v", err)
		return result
	}

	for i := range services.Items {
		svc := &services.Items[i]
		result[svc.Namespace] = append(result[svc.Namespace], svc)
	}

	return result
}

// findServicesForPod finds services that select the given pod.
func (k *adapter) findServicesForPod(pod *corev1.Pod, services []*corev1.Service) []*corev1.Service {
	var matching []*corev1.Service
	podLabels := pod.Labels

	for _, svc := range services {
		if svc.Spec.Selector == nil {
			continue
		}

		// Check if all selector labels match pod labels
		match := true
		for key, value := range svc.Spec.Selector {
			if podLabels[key] != value {
				match = false
				break
			}
		}

		if match {
			matching = append(matching, svc)
		}
	}

	return matching
}

// podToWorkload converts a Kubernetes pod to a workload.
func (k *adapter) podToWorkload(pod *corev1.Pod, services []*corev1.Service) *types.Workload {
	labels := pod.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	namespace := pod.Namespace

	// Build addresses
	var addresses []string

	// Pod DNS: {pod-ip-dashed}.{namespace}.pod
	if pod.Status.PodIP != "" {
		ipDashed := strings.ReplaceAll(pod.Status.PodIP, ".", "-")
		addresses = append(addresses, ipDashed+"."+namespace+".pod")
	}

	// Service DNS: {service-name}.{namespace}.svc
	for _, svc := range services {
		addresses = append(addresses, svc.Name+"."+namespace+".svc")
	}

	// Extract ports from all containers
	var ports []string
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			ports = append(ports, string(port.ContainerPort))
		}
	}

	// Hostname
	hostname := pod.Spec.Hostname
	if hostname == "" {
		hostname = pod.Name
	}

	// Annotations
	annotations := pod.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Add service names to annotations
	var serviceNames []string
	for _, svc := range services {
		serviceNames = append(serviceNames, svc.Name)
	}
	if len(serviceNames) > 0 {
		annotations["services"] = strings.Join(serviceNames, ",")
	}

	return &types.Workload{
		ID:           string(pod.UID),
		Name:         pod.Name,
		Hostname:     hostname,
		Runtime:      RuntimeType,
		WorkloadType: types.WorkloadTypePod,
		// Node:            pod.Spec.NodeName,
		// Namespace:       namespace,
		Addresses:       addresses,
		IsolationGroups: []string{namespace},
		Ports:           ports,
		Labels:          labels,
		Annotations:     annotations,
	}
}
