package docker

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
)

var logger = logging.Logger("runtime")

type discoverer struct {
	store  types.StoreAPI
	client *containerd.Client
}

func New(store types.StoreAPI) (types.DiscoveryAPI, error) {
	client, err := containerd.New("/var/run/containerd/containerd.sock")
	if err != nil {
		return nil, fmt.Errorf("failed to create containerd client: %w", err)
	}

	return &discoverer{
		store:  store,
		client: client,
	}, nil
}

func (d *discoverer) Type() string { return "containerd" }

func (d *discoverer) IsReady(ctx context.Context) bool {
	_, err := d.client.Version(ctx)
	if err != nil {
		return false
	}

	return true
}

func (d *discoverer) Stop() error {
	if err := d.client.Close(); err != nil {
		return fmt.Errorf("failed to close client: %w", err)
	}

	return nil
}

func (d *discoverer) ListProcesses(ctx context.Context, req *runtimev1.ListProcessesRequest, handlerFn func(*runtimev1.Process) error) error {
	// use "moby" namespace for containers
	ctx = namespaces.WithNamespace(ctx, "moby")

	// list containers
	containers, err := d.client.ContainerService().List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// process containers
	for _, container := range containers {
		// // serialize to json
		// data, err := json.Marshal(container)
		// if err != nil {
		// 	return fmt.Errorf("failed to marshal container data: %w", err)
		// }

		// // store to labels
		// labels := container.Labels
		// if labels == nil {
		// 	labels = make(map[string]string)
		// }
		// labels["containerd/container"] = string(data)

		// construct process
		process := &runtimev1.Process{
			Pid:         container.ID,
			Runtime:     d.Type(),
			Annotations: container.Labels,
			CreatedAt:   container.CreatedAt.String(),
			Record:      &corev1.RecordMeta{},
		}

		// call handler function
		if err := handlerFn(process); err != nil {
			return fmt.Errorf("handler function error: %w", err)
		}
	}

	return nil
}
