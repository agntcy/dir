// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/server/database"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Discovery implements the DiscoveryServiceServer interface.
type Discovery struct {
	runtimev1.UnimplementedDiscoveryServiceServer
	db *database.Database
}

// NewDiscovery creates a new gRPC server instance.
func NewDiscovery(db *database.Database) *Discovery {
	return &Discovery{db: db}
}

// GetWorkload retrieves a workload by its identifier.
func (s *Discovery) GetWorkload(ctx context.Context, req *runtimev1.GetWorkloadRequest) (*runtimev1.Workload, error) {
	if req.GetId() == "" {
		//nolint:wrapcheck
		return nil, status.Error(codes.InvalidArgument, "workload ID is required")
	}

	workload, err := s.db.Get(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workload not found: %v", err)
	}

	return workload, nil
}

// ListWorkloads streams workloads based on optional filters.
func (s *Discovery) ListWorkloads(req *runtimev1.ListWorkloadsRequest, stream runtimev1.DiscoveryService_ListWorkloadsServer) error {
	ctx := stream.Context()

	// Get all workloads from database
	workloads, err := s.db.List(ctx, req.GetLabels())
	if err != nil {
		return status.Errorf(codes.Internal, "failed to list workloads: %v", err)
	}

	// Apply label filters if provided
	for _, workload := range workloads {
		if err := stream.Send(workload); err != nil {
			return status.Errorf(codes.Internal, "failed to send workload: %v", err)
		}
	}

	return nil
}
