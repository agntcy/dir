// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"

	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/database"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/metrics"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
)

// openDatabase returns the database the server was given, or else the
// configured one, and the view of it the server's APIs read through; see
// servedDatabase.
func openDatabase(cfg *config.Config, given types.DatabaseAPI, metricsServer *metrics.Server) (types.DatabaseAPI, types.DatabaseAPI, error) {
	db := given
	if db == nil {
		var err error

		db, err = database.New(cfg.Database)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create database API: %w", err)
		}
	}

	served, err := servedDatabase(db, cfg.Policy.Enforcement, cfg.Authz.Enabled, metricsServer)
	if err != nil {
		return nil, nil, err
	}

	return db, served, nil
}

// servedDatabase returns the view of db the server's APIs read through, which
// applies the enforced content policies. db itself applies none: ingestion
// and the reconciler read through it and must see every record. With a
// metrics server, the gate reports what it excludes, or would.
func servedDatabase(db types.DatabaseAPI, cfg policyconfig.EnforcementConfig, authzEnabled bool, metricsServer *metrics.Server) (types.DatabaseAPI, error) {
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

	var observer types.PolicyGateObserver

	if metricsServer != nil {
		gate := metrics.NewPolicyGate(current, gated)
		if err := metricsServer.Registry().Register(gate); err != nil {
			return nil, fmt.Errorf("register policy gate metrics: %w", err)
		}

		observer = gate
	}

	logger.Info("Content policy enforcement", "search", cfg.Search, "fetch", cfg.Fetch, "policies", enforcement.Policies)

	if !authzEnabled {
		logger.Warn("Content policies are checked but authorization is disabled: any caller can obtain registry credentials and read excluded records straight from the registry")
	}

	return gated.Served(current, observer), nil
}

func policyEnforcement(cfg policyconfig.EnforcementConfig) types.PolicyEnforcement {
	policies := make([]types.EnforcedPolicy, 0, len(cfg.Policies))
	for _, policy := range cfg.Policies {
		policies = append(policies, types.EnforcedPolicy{ID: policy.ID, Version: policy.Version})
	}

	return types.PolicyEnforcement{Policies: policies, Search: cfg.Search, Fetch: cfg.Fetch}
}
