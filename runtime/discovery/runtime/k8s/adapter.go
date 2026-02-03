// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/discovery/types"
	"github.com/agntcy/dir/runtime/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const RuntimeType types.RuntimeType = "kubernetes"

var logger = utils.NewLogger("runtime", "kubernetes")

// adapter implements the RuntimeAdapter interface for Kubernetes.
type adapter struct {
	clientset  *kubernetes.Clientset
	namespace  string
	labelKey   string
	labelValue string
}

// NewAdapter creates a new Kubernetes adapter.
func NewAdapter(cfg Config) (types.RuntimeAdapter, error) {
	var (
		kubeConfig *rest.Config
		err        error
	)

	if cfg.Kubeconfig == "" {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build in-cluster config: %w", err)
		}

		logger.Info("using in-cluster config")
	} else {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig from flags: %w", err)
		}

		logger.Info("using kubeconfig", "path", cfg.Kubeconfig)
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	namespace := cfg.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	return &adapter{
		clientset:  clientset,
		namespace:  namespace,
		labelKey:   cfg.LabelKey,
		labelValue: cfg.LabelValue,
	}, nil
}

// Type returns the runtime type.
func (a *adapter) Type() types.RuntimeType {
	return RuntimeType
}

// Close closes the adapter.
func (a *adapter) Close() error {
	return nil
}

// ListWorkloads returns all discoverable pods.
func (k *adapter) ListWorkloads(ctx context.Context) ([]*runtimev1.Workload, error) {
	pods, err := k.clientset.CoreV1().Pods(k.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", k.labelKey, k.labelValue),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Get services for address building
	servicesByNamespace := k.getServicesByNamespace(ctx)

	// Convert pods to workloads
	var workloads []*runtimev1.Workload

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		matchingServices := k.findServicesForPod(&pod, servicesByNamespace[pod.Namespace])

		workload := k.podToWorkload(&pod, matchingServices)
		if workload == nil {
			continue
		}

		workloads = append(workloads, workload)
	}

	return workloads, nil
}

// WatchEvents watches Kubernetes pod events and sends workload events to the channel.
//
//nolint:wrapcheck
func (k *adapter) WatchEvents(ctx context.Context, eventChan chan<- *types.RuntimeEvent) error {
	labelSelector := k.labelKey + "=" + k.labelValue

	// Watch pods
	go k.watchPods(ctx, labelSelector, eventChan)

	// Watch services  (for updating pod addresses)
	go k.watchServices(ctx, eventChan)

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

		// Start pod watcher
		watcher, err := k.clientset.CoreV1().Pods(k.namespace).Watch(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			logger.Error("pod watch error", "error", err)

			continue
		}

		for event := range watcher.ResultChan() {
			if ctx.Err() != nil {
				watcher.Stop()

				return
			}

			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				continue
			}

			// Refresh services on each event to ensure we have current service state
			servicesByNamespace := k.getServicesByNamespace(ctx)
			matchingServices := k.findServicesForPod(pod, servicesByNamespace[pod.Namespace])

			workload := k.podToWorkload(pod, matchingServices)
			if workload == nil {
				continue
			}

			// nolint:exhaustive
			switch event.Type {
			case watch.Added:
				if pod.Status.Phase == corev1.PodRunning {
					eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeAdded, Workload: workload}
				}

			case watch.Modified:
				switch pod.Status.Phase {
				case corev1.PodRunning:
					eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeModified, Workload: workload}
				case corev1.PodSucceeded, corev1.PodFailed, corev1.PodPending:
					eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeDeleted, Workload: workload}
				}

			case watch.Deleted:
				eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeDeleted, Workload: workload}
			}
		}

		watcher.Stop()
		logger.Info("pod watch ended, restarting")
	}
}

// watchServices watches service events to update pod addresses.
func (k *adapter) watchServices(ctx context.Context, eventChan chan<- *types.RuntimeEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Start service watcher
		watcher, err := k.clientset.CoreV1().Services(k.namespace).Watch(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Error("service watch error", "error", err)

			continue
		}

		// Watch for service events
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
		logger.Info("service watch ended, restarting")
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
		logger.Error("failed to list pods for service", "service", svc.Name, "error", err)

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

	// List all services
	services, err := k.clientset.CoreV1().Services(k.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error("failed to list services", "error", err)

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
func (k *adapter) podToWorkload(pod *corev1.Pod, services []*corev1.Service) *runtimev1.Workload {
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
			ports = append(ports, strconv.Itoa(int(port.ContainerPort)))
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

	return &runtimev1.Workload{
		Id:              string(pod.UID),
		Name:            pod.Name,
		Hostname:        hostname,
		Runtime:         runtimev1.RuntimeType_RUNTIME_TYPE_KUBERNETES.GetName(),
		Type:            runtimev1.WorkloadType_WORKLOAD_TYPE_POD.GetName(),
		Addresses:       addresses,
		IsolationGroups: []string{namespace},
		Ports:           ports,
		Labels:          labels,
		Annotations:     annotations,
	}
}
