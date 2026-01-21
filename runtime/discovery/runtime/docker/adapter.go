// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/discovery/types"
	"github.com/agntcy/dir/runtime/utils"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

const RuntimeType types.RuntimeType = "docker"

var logger = utils.NewLogger("runtime", "docker")

// adapter implements the RuntimeAdapter interface for Docker.
type adapter struct {
	client     *client.Client
	labelKey   string
	labelValue string
}

// NewAdapter creates a new Docker adapter.
func NewAdapter(cfg Config) (types.RuntimeAdapter, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
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
		return fmt.Errorf("failed to ping Docker daemon: %w", err)
	}

	logger.Info("connected to Docker daemon")

	return nil
}

// Close closes the Docker client.
func (d *adapter) Close() error {
	if err := d.client.Close(); err != nil {
		return fmt.Errorf("failed to close Docker client: %w", err)
	}

	return nil
}

// ListWorkloads returns all running containers with the discover label.
func (d *adapter) ListWorkloads(ctx context.Context) ([]*runtimev1.Workload, error) {
	// List containers with the discover label
	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", fmt.Sprintf("%s=%s", d.labelKey, d.labelValue)),
			filters.Arg("status", "running"),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var workloads []*runtimev1.Workload

	for _, c := range containers {
		workload := d.containerToWorkload(c)
		if workload != nil {
			workloads = append(workloads, workload)
		}
	}

	return workloads, nil
}

// WatchEvents watches Docker events and sends workload events to the channel.
//
//nolint:wrapcheck
func (d *adapter) WatchEvents(ctx context.Context, eventChan chan<- *types.RuntimeEvent) error {
	// Subscribe to Docker events with filters
	msgChan, errChan := d.client.Events(ctx, events.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("type", "container"),
			filters.Arg("label", fmt.Sprintf("%s=%s", d.labelKey, d.labelValue)),
		),
	})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errChan:
			return fmt.Errorf("error watching Docker events: %w", err)
		case msg := <-msgChan:
			d.handleEvent(ctx, msg, eventChan)
		}
	}
}

// handleEvent processes a Docker event and sends a workload event.
func (d *adapter) handleEvent(ctx context.Context, msg events.Message, eventChan chan<- *types.RuntimeEvent) {
	workload, err := d.getContainerWorkload(ctx, msg.Actor.ID)
	if err != nil {
		logger.Error("failed to get workload", "container_id", msg.Actor.ID, "error", err)

		return
	}

	if workload == nil {
		return
	}

	var eventType types.RuntimeEventType

	// nolint:exhaustive
	switch msg.Action {
	case "start", "unpause", "connect":
		eventType = types.RuntimeEventTypeAdded
	case "stop", "die", "pause", "disconnect":
		eventType = types.RuntimeEventTypeDeleted
	default:
		return
	}

	eventChan <- &types.RuntimeEvent{
		Type:     eventType,
		Workload: workload,
	}
}

// getContainerWorkload retrieves a container and converts it to a workload.
func (d *adapter) getContainerWorkload(ctx context.Context, containerID string) (*runtimev1.Workload, error) {
	inspect, err := d.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	return d.inspectToWorkload(inspect), nil
}

// containerToWorkload converts a Docker container summary to a workload.
func (d *adapter) containerToWorkload(c container.Summary) *runtimev1.Workload {
	// Extract container name (remove leading /)
	name := ""
	if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}

	// Extract networks
	var (
		addresses       []string
		isolationGroups []string
	)

	if c.NetworkSettings != nil {
		for netName := range c.NetworkSettings.Networks {
			addresses = append(addresses, name+"."+netName)
			isolationGroups = append(isolationGroups, netName)
		}
	}

	// Extract ports
	var ports []string
	for _, p := range c.Ports {
		ports = append(ports, strconv.Itoa(int(p.PrivatePort)))
	}

	return &runtimev1.Workload{
		Id:              c.ID,
		Name:            name,
		Hostname:        name,
		Runtime:         string(RuntimeType),
		WorkloadType:    "container",
		Addresses:       addresses,
		IsolationGroups: isolationGroups,
		Ports:           ports,
		Labels:          c.Labels,
		Annotations:     make(map[string]string),
	}
}

// inspectToWorkload converts a Docker container inspect result to a workload.
func (d *adapter) inspectToWorkload(inspect container.InspectResponse) *runtimev1.Workload {
	// Extract container name (remove leading /)
	name := strings.TrimPrefix(inspect.Name, "/")

	// Extract networks
	var (
		addresses       []string
		isolationGroups []string
		ports           []string
	)

	if inspect.NetworkSettings != nil {
		for netName := range inspect.NetworkSettings.Networks {
			addresses = append(addresses, name+"."+netName)
			isolationGroups = append(isolationGroups, netName)
		}

		for port := range inspect.NetworkSettings.Ports {
			ports = append(ports, port.Port())
		}
	}

	// Extract labels
	labels := make(map[string]string)
	if inspect.Config != nil && inspect.Config.Labels != nil {
		labels = inspect.Config.Labels
	}

	// Hostname
	hostname := name
	if inspect.Config != nil && inspect.Config.Hostname != "" {
		hostname = inspect.Config.Hostname
	}

	return &runtimev1.Workload{
		Id:              inspect.ID,
		Name:            name,
		Hostname:        hostname,
		Runtime:         string(RuntimeType),
		WorkloadType:    "container",
		Addresses:       addresses,
		IsolationGroups: isolationGroups,
		Ports:           ports,
		Labels:          labels,
		Annotations:     make(map[string]string),
	}
}
