// Package runtime provides the Docker runtime adapter.
package runtime

import (
	"context"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"github.com/agntcy/dir/discovery/pkg/config"
	"github.com/agntcy/dir/discovery/pkg/models"
)

// DockerAdapter implements the Adapter interface for Docker.
type DockerAdapter struct {
	client     *client.Client
	labelKey   string
	labelValue string
}

// NewDockerAdapter creates a new Docker adapter.
func NewDockerAdapter(cfg *config.DockerConfig) (*DockerAdapter, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(cfg.Host),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	return &DockerAdapter{
		client:     cli,
		labelKey:   cfg.LabelKey(),
		labelValue: cfg.LabelValue(),
	}, nil
}

// RuntimeType returns the Docker runtime type.
func (d *DockerAdapter) RuntimeType() models.Runtime {
	return models.RuntimeDocker
}

// Connect verifies the Docker connection.
func (d *DockerAdapter) Connect(ctx context.Context) error {
	_, err := d.client.Ping(ctx)
	if err != nil {
		return err
	}
	log.Printf("[docker] Connected to Docker daemon")
	return nil
}

// Close closes the Docker client.
func (d *DockerAdapter) Close() error {
	return d.client.Close()
}

// ListWorkloads returns all discoverable containers.
func (d *DockerAdapter) ListWorkloads(ctx context.Context) ([]*models.Workload, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", d.labelKey+"="+d.labelValue)
	filterArgs.Add("status", "running")

	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return nil, err
	}

	var workloads []*models.Workload
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
func (d *DockerAdapter) WatchEvents(ctx context.Context, eventChan chan<- *models.WorkloadEvent) error {
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
func (d *DockerAdapter) handleEvent(ctx context.Context, msg events.Message, eventChan chan<- *models.WorkloadEvent) {
	containerID := msg.Actor.ID
	if containerID == "" {
		return
	}

	switch msg.Action {
	case "start":
		workload := d.getContainerWorkload(ctx, containerID)
		if workload != nil {
			eventChan <- &models.WorkloadEvent{Type: models.EventTypeAdded, Workload: workload}
		}

	case "stop", "die", "kill":
		workload := &models.Workload{
			ID:           containerID,
			Name:         msg.Actor.Attributes["name"],
			Hostname:     containerID[:12],
			Runtime:      models.RuntimeDocker,
			WorkloadType: models.WorkloadTypeContainer,
		}
		if workload.Name == "" {
			workload.Name = containerID[:12]
		}
		eventChan <- &models.WorkloadEvent{Type: models.EventTypeDeleted, Workload: workload}

	case "pause":
		workload := &models.Workload{
			ID:           containerID,
			Name:         msg.Actor.Attributes["name"],
			Hostname:     containerID[:12],
			Runtime:      models.RuntimeDocker,
			WorkloadType: models.WorkloadTypeContainer,
		}
		if workload.Name == "" {
			workload.Name = containerID[:12]
		}
		eventChan <- &models.WorkloadEvent{Type: models.EventTypePaused, Workload: workload}

	case "unpause":
		workload := d.getContainerWorkload(ctx, containerID)
		if workload != nil {
			eventChan <- &models.WorkloadEvent{Type: models.EventTypeAdded, Workload: workload}
		}

	case "connect", "disconnect":
		workload := d.getContainerWorkload(ctx, containerID)
		if workload != nil {
			eventChan <- &models.WorkloadEvent{Type: models.EventTypeNetworkChanged, Workload: workload}
		}
	}
}

// getContainerWorkload gets a workload for a container ID.
func (d *DockerAdapter) getContainerWorkload(ctx context.Context, containerID string) *models.Workload {
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
func (d *DockerAdapter) containerToWorkload(ctx context.Context, c types.Container) *models.Workload {
	// Get full inspection for network details
	inspect, err := d.client.ContainerInspect(ctx, c.ID)
	if err != nil {
		log.Printf("[docker] Failed to inspect container %s: %v", c.ID[:12], err)
		return nil
	}

	return d.inspectToWorkload(inspect)
}

// inspectToWorkload converts container inspection to workload.
func (d *DockerAdapter) inspectToWorkload(inspect types.ContainerJSON) *models.Workload {
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

	return &models.Workload{
		ID:              inspect.ID,
		Name:            name,
		Hostname:        hostname,
		Runtime:         models.RuntimeDocker,
		WorkloadType:    models.WorkloadTypeContainer,
		Addresses:       addresses,
		IsolationGroups: isolationGroups,
		Ports:           ports,
		Labels:          labels,
	}
}
