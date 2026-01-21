// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package crd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	crdv1 "github.com/agntcy/dir/runtime/api/crd/v1"
	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/store/types"
	"github.com/agntcy/dir/runtime/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const StoreType types.StoreType = "crd"

var logger = utils.NewLogger("store", "crd")

// store provides storage operations for CRDs.
type store struct {
	client    dynamic.Interface
	gvr       schema.GroupVersionResource
	namespace string
}

// New creates a new CRD store.
func New(cfg Config) (types.Store, error) {
	// Build Kubernetes config
	var (
		restConfig *rest.Config
		err        error
	)

	if cfg.Kubeconfig == "" {
		restConfig, err = rest.InClusterConfig()
	} else {
		restConfig, err = clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build k8s config: %w", err)
	}

	// Create dynamic client
	client, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    crdv1.GroupVersion.Group,
		Version:  crdv1.GroupVersion.Version,
		Resource: "discoveredworkloads",
	}

	logger.Info("writer initialized",
		"group", gvr.Group,
		"version", gvr.Version,
		"resource", gvr.Resource,
		"namespace", cfg.Namespace,
	)

	return &store{
		client:    client,
		gvr:       gvr,
		namespace: cfg.Namespace,
	}, nil
}

// Close closes the store (no-op for CRD client).
func (s *store) Close() error {
	return nil
}

// RegisterWorkload creates or updates a DiscoveredWorkload CR.
func (s *store) RegisterWorkload(ctx context.Context, workload *runtimev1.Workload) error {
	obj, err := s.workloadToCR(workload)
	if err != nil {
		return fmt.Errorf("failed to convert workload to CR: %w", err)
	}

	_, err = s.client.Resource(s.gvr).Namespace(s.namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		// If already exists, update it
		if strings.Contains(err.Error(), "already exists") {
			//nolint:wrapcheck
			return s.PatchWorkload(ctx, workload)
		}

		return fmt.Errorf("failed to create CR: %w", err)
	}

	logger.Info("registered workload", "workload", workload.GetId())

	return nil
}

// DeregisterWorkload deletes a DiscoveredWorkload CR.
func (s *store) DeregisterWorkload(ctx context.Context, workloadID string) error {
	err := s.client.Resource(s.gvr).Namespace(s.namespace).Delete(ctx, crName(workloadID), metav1.DeleteOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}

		return fmt.Errorf("failed to delete CR: %w", err)
	}

	logger.Info("deregistered workload", "workload", workloadID)

	return nil
}

// PatchWorkload performs a full update of an existing workload CR.
func (s *store) PatchWorkload(ctx context.Context, workload *runtimev1.Workload) error {
	// Get existing CR
	existing, err := s.client.Resource(s.gvr).Namespace(s.namespace).Get(ctx, crName(workload.GetId()), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get CR for patch: %w", err)
	}

	// Convert object to unstructured
	obj, err := s.workloadToCR(workload)
	if err != nil {
		return fmt.Errorf("failed to convert workload to CR: %w", err)
	}

	// Set resource version to match existing
	obj.SetResourceVersion(existing.GetResourceVersion())

	// Update the CR in Kubernetes
	_, err = s.client.Resource(s.gvr).Namespace(s.namespace).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update CR: %w", err)
	}

	logger.Info("patched workload", "workload", workload.GetId())

	return nil
}

// ListWorkloadIDs returns all workload IDs from CRDs.
func (s *store) ListWorkloadIDs(ctx context.Context) (map[string]struct{}, error) {
	list, err := s.client.Resource(s.gvr).Namespace(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list CRs: %w", err)
	}

	ids := make(map[string]struct{})

	for _, item := range list.Items {
		id, found, _ := unstructured.NestedString(item.Object, "spec", "id")
		if found && id != "" {
			ids[id] = struct{}{}
		}
	}

	return ids, nil
}

// ListWorkloads implements types.Store.
func (s *store) ListWorkloads(ctx context.Context) ([]*runtimev1.Workload, error) {
	list, err := s.client.Resource(s.gvr).Namespace(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list CRs: %w", err)
	}

	var workloads []*runtimev1.Workload

	for _, item := range list.Items {
		workload, err := s.crToWorkload(&item)
		if err != nil {
			logger.Error("failed to convert CR to workload", "error", err)

			continue
		}

		workloads = append(workloads, workload)
	}

	return workloads, nil
}

// WatchWorkloads watches for workload changes.
func (s *store) WatchWorkloads(ctx context.Context, handler func(workload *runtimev1.Workload, deleted bool)) error {
	watcher, err := s.client.Resource(s.gvr).Namespace(s.namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to watch CRs: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			//nolint:wrapcheck
			return ctx.Err()
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}

			obj, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				continue
			}

			workload, err := s.crToWorkload(obj)
			if err != nil {
				logger.Error("failed to convert CR to workload", "error", err)

				continue
			}

			handler(workload, event.Type == watch.Deleted)
		}
	}
}

// workloadToCR converts a Workload to an unstructured CR object.
func (s *store) workloadToCR(workload *runtimev1.Workload) (*unstructured.Unstructured, error) {
	// Generate JSON representation
	data, err := json.Marshal(&crdv1.DiscoveredWorkload{
		TypeMeta: metav1.TypeMeta{
			APIVersion: s.gvr.GroupVersion().String(),
			Kind:       "DiscoveredWorkload",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      crName(workload.GetId()),
			Namespace: s.namespace,
			Labels: map[string]string{
				"discovery.agntcy.io/workload-id":   workload.GetId(),
				"discovery.agntcy.io/runtime":       workload.GetRuntime(),
				"discovery.agntcy.io/workload-type": workload.GetWorkloadType(),
			},
		},
		Spec: workload,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workload to CR: %w", err)
	}

	// Convert to unstructured
	var unstructuredObj unstructured.Unstructured

	err = json.Unmarshal(data, &unstructuredObj.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal workload JSON to unstructured: %w", err)
	}

	return &unstructuredObj, nil
}

// crToWorkload converts an unstructured CR to a Workload.
func (s *store) crToWorkload(obj *unstructured.Unstructured) (*runtimev1.Workload, error) {
	// Get spec field
	spec, found, _ := unstructured.NestedMap(obj.Object, "spec")
	if !found {
		return nil, fmt.Errorf("spec not found in unstructured object")
	}

	// Convert spec to workload
	specData, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %w", err)
	}

	var workload runtimev1.Workload

	err = json.Unmarshal(specData, &workload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec to workload: %w", err)
	}

	return &workload, nil
}

// crName converts a workload ID to a valid Kubernetes resource name.
//
//nolint:mnd
func crName(workloadID string) string {
	name := strings.ToLower(workloadID)
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = "dw-" + name

	if len(name) > 63 {
		name = name[:63]
	}

	name = strings.TrimRight(name, "-")

	return name
}
