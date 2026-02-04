// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/discovery/types"
	"github.com/agntcy/dir/runtime/utils"
)

const ResolverType types.WorkloadResolverType = "a2a"

var logger = utils.NewLogger("resolver", "a2a")

// resolver resolves A2A services on workloads.
type resolver struct {
	timeout    time.Duration
	paths      []string
	client     *http.Client
	labelKey   string
	labelValue string
}

// NewResolver creates a new A2A resolver.
func NewResolver(cfg Config) types.WorkloadResolver {
	return &resolver{
		timeout: cfg.Timeout,
		paths:   cfg.Paths,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		labelKey:   cfg.LabelKey,
		labelValue: cfg.LabelValue,
	}
}

// Name returns the resolver name.
func (r *resolver) Name() types.WorkloadResolverType {
	return ResolverType
}

// CanResolve returns whether this resolver can resolve the workload.
func (r *resolver) CanResolve(workload *runtimev1.Workload) bool {
	// If workload does not have a label key with expected value, skip it
	labelValue, hasLabel := workload.GetLabels()[r.labelKey]
	if !hasLabel || !strings.EqualFold(labelValue, r.labelValue) {
		return false
	}

	// Only resolve workloads with addresses and ports
	return len(workload.GetAddresses()) > 0 && len(workload.GetPorts()) > 0
}

// Resolve probes A2A endpoints on the workload.
//
//nolint:nosprintfhostport
func (r *resolver) Resolve(ctx context.Context, workload *runtimev1.Workload) (any, error) {
	// Build list of URLs to try
	var urls []string

	for _, addr := range workload.GetAddresses() {
		for _, port := range workload.GetPorts() {
			for _, path := range r.paths {
				urls = append(urls, fmt.Sprintf("http://%s:%s%s", addr, port, path))
			}
		}
	}

	logger.Info("probing URLs", "workload", workload.GetId(), "urls", strings.Join(urls, ","))

	// Try each URL
	for _, url := range urls {
		result := r.probeURL(ctx, url)
		if result != nil {
			logger.Info("scraped successfully", "workload", workload.GetId())

			return result, nil
		}
	}

	// All failed
	return nil, fmt.Errorf("no reachable A2A endpoints found for workload %s", workload.GetId())
}

// Apply implements types.WorkloadResolver.
func (r *resolver) Apply(ctx context.Context, workload *runtimev1.Workload, result any) error {
	// Convert result to map
	data, err := utils.InterfaceToStruct(result)
	if err != nil {
		return fmt.Errorf("failed to convert result to struct: %w", err)
	}

	// Update A2A field on workload
	if workload.GetServices() == nil {
		workload.Services = &runtimev1.WorkloadServices{}
	}

	workload.Services.A2A = data

	return nil
}

// probeURL probes a single URL.
func (r *resolver) probeURL(ctx context.Context, url string) map[string]any {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	// Expect HTTP 200 OK
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	// Read returned body as metadata
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result) == 0 {
		return nil
	}

	return result
}
