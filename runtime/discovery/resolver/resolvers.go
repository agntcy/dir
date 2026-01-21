// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolver

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/runtime/discovery/resolver/a2a"
	"github.com/agntcy/dir/runtime/discovery/resolver/config"
	"github.com/agntcy/dir/runtime/discovery/resolver/oasf"
	"github.com/agntcy/dir/runtime/discovery/types"
)

func NewResolvers(ctx context.Context, cfg config.Config) ([]types.WorkloadResolver, error) {
	var resolvers []types.WorkloadResolver

	// Create resolvers based on configuration
	if cfg.A2A.Enabled {
		resolvers = append(resolvers, a2a.NewResolver(cfg.A2A))
	}

	// Create OASF resolver
	if cfg.OASF.Enabled {
		oasfResolver, err := oasf.NewResolver(ctx, cfg.OASF)
		if err != nil {
			return nil, fmt.Errorf("failed to create OASF resolver: %w", err)
		}

		resolvers = append(resolvers, oasfResolver)
	}

	// Validate created resolvers
	if len(resolvers) == 0 {
		return nil, fmt.Errorf("no resolvers enabled")
	}

	return resolvers, nil
}
