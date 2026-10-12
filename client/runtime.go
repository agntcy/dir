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
	return c.ListWorkloadsStreamWithRequest(ctx, &runtimev1.ListWorkloadsRequest{
		Labels: labels,
	})
}

// ListWorkloadsStreamWithRequest streams the workloads matching every filter set in req
// (labels, skills, domains and locator protocols).
func (c *Client) ListWorkloadsStreamWithRequest(ctx context.Context, req *runtimev1.ListWorkloadsRequest) (streaming.StreamResult[runtimev1.Workload], error) {
	stream, err := c.DiscoveryServiceClient.ListWorkloads(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create list workload stream: %w", err)
	}

	//nolint:wrapcheck
	return streaming.ProcessServerStream(ctx, stream)
}

func (c *Client) ListWorkloads(ctx context.Context, labels map[string]string) ([]*runtimev1.Workload, error) {
	return c.ListWorkloadsWithRequest(ctx, &runtimev1.ListWorkloadsRequest{
		Labels: labels,
	})
}

// ListWorkloadsWithRequest returns the workloads matching every filter set in req
// (labels, skills, domains and locator protocols).
func (c *Client) ListWorkloadsWithRequest(ctx context.Context, req *runtimev1.ListWorkloadsRequest) ([]*runtimev1.Workload, error) {
	// Use channel to communicate error safely (no race condition)
	result, err := c.ListWorkloadsStreamWithRequest(ctx, req)
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
