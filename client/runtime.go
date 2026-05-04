// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"github.com/agntcy/dir/client/streaming"
)

func (c *Client) GetWorkload(ctx context.Context, workloadID string) (*runtimev1.Workload, error) {
	workload, err := c.DiscoveryServiceClient.GetWorkload(ctx, &runtimev1.GetWorkloadRequest{
		Id: workloadID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get workload %s: %w", workloadID, err)
	}

	return workload, nil
}

func (c *Client) ListWorkloadsStream(ctx context.Context, labels map[string]string) (streaming.StreamResult[runtimev1.Workload], error) {
	stream, err := c.DiscoveryServiceClient.ListWorkloads(ctx, &runtimev1.ListWorkloadsRequest{
		Labels: labels,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create list workload stream: %w", err)
	}

	//nolint:wrapcheck
	return streaming.ProcessServerStream(ctx, stream)
}

func (c *Client) ListWorkloads(ctx context.Context, labels map[string]string) ([]*runtimev1.Workload, error) {
	// Use channel to communicate error safely (no race condition)
	result, err := c.ListWorkloadsStream(ctx, labels)
	if err != nil {
		return nil, err
	}

	// Check for results
	var errs error

	var workloads []*runtimev1.Workload

	for {
		select {
		case err := <-result.ErrCh():
			errs = errors.Join(errs, err)
		case resp := <-result.ResCh():
			workloads = append(workloads, resp)
		case <-result.DoneCh():
			return workloads, errs
		}
	}
}
