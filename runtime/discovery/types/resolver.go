// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
)

type WorkloadResolverType string

// WorkloadResolver resolves services/interfaces on workloads to extract metadata.
type WorkloadResolver interface {
	// Name returns the resolver name.
	Name() WorkloadResolverType

	// CanResolve returns true if this resolver can resolve the workload.
	CanResolve(workload *runtimev1.Workload) bool

	// Resolve resolves the workload and returns metadata.
	// Resolve method is called concurrently and must be thread-safe.
	// It should not modify the workload.
	Resolve(ctx context.Context, workload *runtimev1.Workload) (any, error)

	// Apply applies fetched result to the workload.
	// Apply method is called sequentially to avoid race conditions.
	Apply(ctx context.Context, workload *runtimev1.Workload, result any) error
}
