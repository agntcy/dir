package docker

import (
	"context"
	"log"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"github.com/agntcy/dir/discovery/pkg/types"
)

const RuntimeType types.RuntimeType = "docker"

// adapter implements the Adapter interface for Docker.
type adapter struct {
	client     *client.Client
	labelKey   string
	labelValue string
}

// NewAdapter creates a new Docker adapter.
func NewAdapter(cfg Config) (types.RuntimeAdapter, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(cfg.Host),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	return &adapter{
		client:     cli,
		labelKey:   cfg.LabelKey,
		labelValue: cfg.LabelValue,
	}, nil
}

// Type returns the Docker runtime type.
func (d *adapter) Type() types.RuntimeType {
	return RuntimeType
}

// Connect verifies the Docker connection.
func (d *adapter) Connect(ctx context.Context) error {
	_, err := d.client.Ping(ctx)
	if err != nil {
		return err
	}
	log.Printf("[docker] Connected to Docker daemon")
	return nil
}

// Close closes the Docker client.
func (d *adapter) Close() error {
	return d.client.Close()
}

// ListWorkloads returns all discoverable containers.
func (d *adapter) ListWorkloads(ctx context.Context) ([]*types.Workload, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", d.labelKey+"="+d.labelValue)
	filterArgs.Add("status", "running")

	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return nil, err
	}

	var workloads []*types.Workload
	for _, c := range containers {
		// Skip paused containers
		if c.State == "paused" {
			continue
		}

		workload := d.containerToWorkload(ctx, c)
		if workload != nil {
			workloads = append(workloads, workload)
		}
	}

	return workloads, nil
}

// WatchEvents watches Docker events and sends workload events to the channel.
func (d *adapter) WatchEvents(ctx context.Context, eventChan chan<- *types.RuntimeEvent) error {
	filterArgs := filters.NewArgs()
	filterArgs.Add("type", "container")

	msgChan, errChan := d.client.Events(ctx, events.ListOptions{
		Filters: filterArgs,
	})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err := <-errChan:
			if err != nil && ctx.Err() == nil {
				log.Printf("[docker] Event watch error: %v", err)
				return err
			}
			return nil

		case msg := <-msgChan:
			d.handleEvent(ctx, msg, eventChan)
		}
	}
}

// handleEvent processes a Docker event.
func (d *adapter) handleEvent(ctx context.Context, msg events.Message, eventChan chan<- *types.RuntimeEvent) {
	containerID := msg.Actor.ID
	if containerID == "" {
		return
	}

	switch msg.Action {
	case "start":
		workload := d.getContainerWorkload(ctx, containerID)
		if workload != nil {
			eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeAdded, Workload: workload}
		}

	case "stop", "die", "kill":
		workload := &types.Workload{
			ID:           containerID,
			Name:         msg.Actor.Attributes["name"],
			Hostname:     containerID[:12],
			Runtime:      RuntimeType,
			WorkloadType: types.WorkloadTypeContainer,
		}
		if workload.Name == "" {
			workload.Name = containerID[:12]
		}
		eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeDeleted, Workload: workload}

	case "pause":
		workload := &types.Workload{
			ID:           containerID,
			Name:         msg.Actor.Attributes["name"],
			Hostname:     containerID[:12],
			Runtime:      RuntimeType,
			WorkloadType: types.WorkloadTypeContainer,
		}
		if workload.Name == "" {
			workload.Name = containerID[:12]
		}
		eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypePaused, Workload: workload}

	case "unpause":
		workload := d.getContainerWorkload(ctx, containerID)
		if workload != nil {
			eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeAdded, Workload: workload}
		}

	case "connect", "disconnect":
		workload := d.getContainerWorkload(ctx, containerID)
		if workload != nil {
			eventChan <- &types.RuntimeEvent{Type: types.RuntimeEventTypeModified, Workload: workload}
		}
	}
}

// getContainerWorkload gets a workload for a container ID.
func (d *adapter) getContainerWorkload(ctx context.Context, containerID string) *types.Workload {
	inspect, err := d.client.ContainerInspect(ctx, containerID)
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil
		}
		log.Printf("[docker] Failed to inspect container %s: %v", containerID[:12], err)
		return nil
	}

	return d.inspectToWorkload(inspect)
}

// containerToWorkload converts a container list item to a workload.
func (d *adapter) containerToWorkload(ctx context.Context, c dockertypes.Container) *types.Workload {
	// Get full inspection for network details
	inspect, err := d.client.ContainerInspect(ctx, c.ID)
	if err != nil {
		log.Printf("[docker] Failed to inspect container %s: %v", c.ID[:12], err)
		return nil
	}

	return d.inspectToWorkload(inspect)
}

// inspectToWorkload converts container inspection to workload.
func (d *adapter) inspectToWorkload(inspect dockertypes.ContainerJSON) *types.Workload {
	labels := inspect.Config.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	// Check discover label
	if labels[d.labelKey] != d.labelValue {
		return nil
	}

	// Get container name (remove leading /)
	name := strings.TrimPrefix(inspect.Name, "/")
	hostname := inspect.ID[:12]

	// Get exposed ports
	var ports []string
	if inspect.Config.ExposedPorts != nil {
		for port := range inspect.Config.ExposedPorts {
			portStr := strings.Split(string(port), "/")[0]
			ports = append(ports, portStr)
		}
	}

	// Get network information
	var addresses []string
	var isolationGroups []string

	if inspect.NetworkSettings != nil && inspect.NetworkSettings.Networks != nil {
		for networkName := range inspect.NetworkSettings.Networks {
			addresses = append(addresses, name+"."+networkName)
			isolationGroups = append(isolationGroups, networkName)
		}
	}

	return &types.Workload{
		ID:              inspect.ID,
		Name:            name,
		Hostname:        hostname,
		Runtime:         RuntimeType,
		WorkloadType:    types.WorkloadTypeContainer,
		Addresses:       addresses,
		IsolationGroups: isolationGroups,
		Ports:           ports,
		Annotations:     inspect.HostConfig.Annotations,
		Labels:          labels,
	}
}
