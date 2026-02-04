// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oasf

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/client"
	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/discovery/types"
	"github.com/agntcy/dir/runtime/utils"
)

const ResolverType types.WorkloadResolverType = "oasf"

var logger = utils.NewLogger("resolver", "oasf")

// resolver resolves OASF records for workloads.
type resolver struct {
	timeout  time.Duration
	client   *client.Client
	labelKey string
}

// NewResolver creates a new OASF resolver.
func NewResolver(ctx context.Context, cfg Config) (types.WorkloadResolver, error) {
	// Create context with timeout, inheriting from parent context
	initCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	// Create Dir client
	dirClient, err := client.New(initCtx, client.WithEnvConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create Dir client: %w", err)
	}

	// Return resolver
	return &resolver{
		timeout:  cfg.Timeout,
		client:   dirClient,
		labelKey: cfg.LabelKey,
	}, nil
}

// Name returns the resolver name.
func (r *resolver) Name() types.WorkloadResolverType {
	return ResolverType
}

// CanResolve returns whether this resolver can resolve the workload.
func (r *resolver) CanResolve(workload *runtimev1.Workload) bool {
	// If workload does not have a label key, skip it
	if _, hasLabel := workload.GetLabels()[r.labelKey]; !hasLabel {
		return false
	}

	return true
}

// Resolve fetches OASF record for the workload.
func (r *resolver) Resolve(ctx context.Context, workload *runtimev1.Workload) (any, error) {
	// Get the name of the OASF record from the workload labels
	name, version := recordNameVersion(workload.GetLabels()[r.labelKey])
	nameVersion := name + ":" + version

	logger.Info("started resolving", "workload", workload.GetId(), "record", nameVersion)

	// Fetch the OASF record using the provided context
	resolve, err := r.client.Resolve(ctx, name, version)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OASF record %s: %w", nameVersion, err)
	}

	// Get the first info
	if len(resolve.GetRecords()) == 0 {
		return nil, fmt.Errorf("no OASF info found for record %s", nameVersion)
	}

	recordRef := resolve.GetRecords()[0]

	// Pull the full record data using the provided context
	record, err := r.client.Pull(ctx, &corev1.RecordRef{Cid: recordRef.GetCid()})
	if err != nil {
		return nil, fmt.Errorf("failed to pull OASF record %s: %w", nameVersion, err)
	}

	// Get the record signature verified status
	verified, err := r.client.Verify(ctx, &signv1.VerifyRequest{
		RecordRef: &corev1.RecordRef{Cid: recordRef.GetCid()},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to verify OASF record %s: %w", nameVersion, err)
	}

	logger.Info("resolved successfully", "workload", workload.GetId(), "record", nameVersion)

	// Return the record data along with its validity
	return map[string]any{
		"cid":      recordRef.GetCid(),
		"name":     nameVersion,
		"verified": verified,
		"record":   record.GetData().AsMap(),
	}, nil
}

// Apply implements types.WorkloadResolver.
func (r *resolver) Apply(ctx context.Context, workload *runtimev1.Workload, result any) error {
	// Convert result to map
	data, err := utils.InterfaceToStruct(result)
	if err != nil {
		return fmt.Errorf("failed to convert result to struct: %w", err)
	}

	// Update OASF field on workload
	if workload.GetServices() == nil {
		workload.Services = &runtimev1.WorkloadServices{}
	}

	workload.Services.Oasf = data

	return nil
}

//nolint:mnd
func recordNameVersion(recordFQDN string) (string, string) {
	// Split the record FQDN into name and version
	// Expected formats: name, name:version
	nameParts := strings.SplitN(recordFQDN, ":", 2)
	if len(nameParts) == 2 {
		return nameParts[0], nameParts[1]
	}

	if len(nameParts) == 1 {
		return nameParts[0], ""
	}

	return "", ""
}
