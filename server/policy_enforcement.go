// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"

	gormdb "github.com/agntcy/dir/server/database/gorm"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
)

// servedDatabase returns the view of db the server's APIs read through, which
// applies the enforced content policies. db itself applies none: ingestion
// and the reconciler read through it and must see every record.
func servedDatabase(db types.DatabaseAPI, cfg policyconfig.EnforcementConfig, authzEnabled bool) (types.DatabaseAPI, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy enforcement config: %w", err)
	}

	if !cfg.Active() {
		return db, nil
	}

	gated, ok := db.(*gormdb.DB)
	if !ok {
		return nil, fmt.Errorf("database %T cannot enforce content policies", db)
	}

	enforcement := policyEnforcement(cfg)
	current := func() types.PolicyEnforcement { return enforcement }

	logger.Info("Content policy enforcement", "search", cfg.Search, "fetch", cfg.Fetch, "policies", enforcement.Policies)

	if !authzEnabled {
		logger.Warn("Content policies are checked but authorization is disabled: any caller can obtain registry credentials and read excluded records straight from the registry")
	}

	return gated.Served(current, nil), nil
}

func policyEnforcement(cfg policyconfig.EnforcementConfig) types.PolicyEnforcement {
	policies := make([]types.EnforcedPolicy, 0, len(cfg.Policies))
	for _, policy := range cfg.Policies {
		policies = append(policies, types.EnforcedPolicy{ID: policy.ID, Version: policy.Version})
	}

	return types.PolicyEnforcement{Policies: policies, Search: cfg.Search, Fetch: cfg.Fetch}
}
