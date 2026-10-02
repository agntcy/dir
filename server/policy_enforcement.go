// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"

	policyv1 "github.com/agntcy/dir/api/policy/v1"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/controller"
	"github.com/agntcy/dir/server/database"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/metrics"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/policy/enforcement"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/grpc"
)

// openDatabase returns the database the server was given, or else the
// configured one, the view of it the server's APIs read through, and what
// that view enforces; see servedDatabase.
func openDatabase(cfg *config.Config, given types.DatabaseAPI, metricsServer *metrics.Server) (types.DatabaseAPI, types.DatabaseAPI, func() types.PolicyEnforcement, error) {
	db := given
	if db == nil {
		var err error

		db, err = database.New(cfg.Database)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create database API: %w", err)
		}
	}

	served, current, err := servedDatabase(db, cfg.Policy.Enforcement, cfg.Authz.Enabled, metricsServer)
	if err != nil {
		return nil, nil, nil, err
	}

	return db, served, current, nil
}

// registerPolicyAudit offers the policy audit service, which returns the
// records the content policies exclude, when policies are checked and
// authorization is enabled: the service is protected by its own permission
// alone, so without authorization it would hand excluded records to anyone.
func registerPolicyAudit(registrar grpc.ServiceRegistrar, authzEnabled bool, db types.DatabaseAPI, store types.StoreAPI, current func() types.PolicyEnforcement) {
	if current == nil {
		return
	}

	if !authzEnabled {
		logger.Warn("Policy audit service not offered: it requires authorization to be enabled")

		return
	}

	auditDB, ok := db.(controller.PolicyAuditDatabase)
	if !ok {
		logger.Warn("Policy audit service not offered: the database cannot list excluded records", "database", fmt.Sprintf("%T", db))

		return
	}

	policyv1.RegisterPolicyAuditServiceServer(registrar, controller.NewPolicyAuditController(auditDB, store, current))
	logger.Info("Policy audit service offered to the SPIFFE IDs the authorization policies name for it")
}

// servedDatabase returns the view of db the server's APIs read through, which
// applies the enforced content policies, and what it enforces, nil when no
// read checks them. db itself applies none: ingestion and the reconciler read
// through it and must see every record. With a metrics server, the gate
// reports what it excludes, or would.
func servedDatabase(db types.DatabaseAPI, cfg policyconfig.EnforcementConfig, authzEnabled bool, metricsServer *metrics.Server) (types.DatabaseAPI, func() types.PolicyEnforcement, error) {
	if err := cfg.Validate(); err != nil {
		return nil, nil, fmt.Errorf("invalid policy enforcement config: %w", err)
	}

	if !cfg.Active() {
		return db, nil, nil
	}

	gated, ok := db.(*gormdb.DB)
	if !ok {
		return nil, nil, fmt.Errorf("database %T cannot enforce content policies", db)
	}

	resolver, err := enforcement.NewResolver(cfg, gated)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve enforced policy versions: %w", err)
	}

	current := resolver.Current

	var observer types.PolicyGateObserver

	if metricsServer != nil {
		gate := metrics.NewPolicyGate(current, gated)
		if err := metricsServer.Registry().Register(gate); err != nil {
			return nil, nil, fmt.Errorf("register policy gate metrics: %w", err)
		}

		observer = gate
	}

	resolved := current()
	logger.Info("Content policy enforcement", "search", cfg.Search, "fetch", cfg.Fetch,
		"policies", resolved.Policies, "pending", resolved.Pending)

	if !authzEnabled {
		logger.Warn("Content policies are checked but authorization is disabled: any caller can obtain registry credentials and read excluded records straight from the registry")
	}

	return gated.Served(current, observer), current, nil
}
