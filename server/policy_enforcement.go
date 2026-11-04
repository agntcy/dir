// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"

	policyv1 "github.com/agntcy/dir/api/policy/v1"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/controller"
	"github.com/agntcy/dir/server/database"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/metrics"
	"github.com/agntcy/dir/server/policy"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/policy/enforcement"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/grpc"
)

// wiredPolicies is the policy machinery the server reads through, tied
// together by wirePolicies.
type wiredPolicies struct {
	// Served is the view of the database the APIs read: the search policies
	// applied by the query builder itself.
	Served types.DatabaseAPI

	// Registry is every configured policy, compiled.
	Registry *policy.Registry

	// Gate is what every read of one record by CID passes through.
	Gate *policy.RecordGate

	// Enforcement is what is in force now, or nil when nothing is enforced
	// and so there is nothing to audit.
	Enforcement func() types.PolicyEnforcement
}

// openDatabase returns the database the server was given, or else the
// configured one.
func openDatabase(cfg *config.Config, given types.DatabaseAPI) (types.DatabaseAPI, error) {
	if given != nil {
		return given, nil
	}

	db, err := database.New(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create database API: %w", err)
	}

	return db, nil
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

// wirePolicies builds the three things that depend on one another: the
// registry of configured policies, the resolver that follows which version of
// each is in force, and the view of db the server's APIs read through. db
// itself applies no policy: ingestion and the reconciler read through it and
// must see every record.
//
// The registry and the resolver each need the other. A policy at pull or
// search is acted on only once the policies it reads are enforced, which the
// resolver says; and the resolver has to follow the versions the search
// policies read, which only the compiled registry knows. Neither reads the
// other while it is being built, so the knot is tied here, once, instead of
// being spread through the server as a setter called at the right moment.
//
// The returned Enforcement is nil when nothing is configured to be enforced,
// which is what decides whether there is anything to audit.
func wirePolicies(ctx context.Context, cfg *config.Config, db types.DatabaseAPI, metricsServer *metrics.Server,
) (wiredPolicies, error) {
	gated, ok := db.(*gormdb.DB)
	if !ok {
		// A database no query can be built on carries no gate. Nothing is
		// quietly unenforced by that: a policy reading a verdict is refused
		// by the registry below, which is given no verdict source, and a
		// policy reading none needs no gate.
		//
		// Policies that read no verdict still load, so another database is
		// not stopped from validating records at push.
		registry, err := policy.NewRegistry(ctx, cfg.Policy.Evaluations, cfg.Policy.Operations, cfg.Policy.Dir)
		if err != nil {
			return wiredPolicies{}, fmt.Errorf("failed to create policy registry: %w", err)
		}

		return wiredPolicies{Served: db, Registry: registry, Gate: policy.NewRecordGate(registry, nil)}, nil
	}

	var resolver *enforcement.Resolver

	current := func() types.PolicyEnforcement { return resolver.Current() }

	registry, err := policy.NewRegistry(ctx, cfg.Policy.Evaluations, cfg.Policy.Operations, cfg.Policy.Dir,
		policy.WithVerdicts(db, current))
	if err != nil {
		return wiredPolicies{}, fmt.Errorf("failed to create policy registry: %w", err)
	}

	searchPolicies := registry.SearchPolicies()

	resolver, err = enforcement.NewResolver(
		cfg.Policy.GetRefreshInterval(), gated, searchPolicies, registry.EvaluationIDs())
	if err != nil {
		return wiredPolicies{}, fmt.Errorf("resolve enforced policy versions: %w", err)
	}

	if len(searchPolicies) == 0 && len(registry.OperationIDs(policyconfig.OpPull)) == 0 {
		// Nothing is enforced, so there is nothing to audit. The view is
		// still the gated one: it applies no condition when there is none,
		// and keeping one path means a policy added later needs no restart
		// of anything but the process that read the config.
		return wiredPolicies{
			Served:   gated.Served(current),
			Registry: registry,
			Gate:     policy.NewRecordGate(registry, nil),
		}, nil
	}

	observer, err := gateObserver(current, gated, metricsServer)
	if err != nil {
		return wiredPolicies{}, err
	}

	resolved := current()
	logger.Info("Content policy enforcement",
		"evaluations", len(resolved.Evaluations),
		"search_policies", len(searchPolicies), "search_policies_applied", len(resolved.SearchPolicies),
		"pull_policies", len(registry.OperationIDs(policyconfig.OpPull)))

	if !cfg.Authz.Enabled {
		logger.Warn("Content policies are checked but authorization is disabled: any caller can obtain registry credentials and read excluded records straight from the registry")
	}

	return wiredPolicies{
		Served:      gated.Served(current),
		Registry:    registry,
		Gate:        policy.NewRecordGate(registry, observer),
		Enforcement: current,
	}, nil
}

// gateObserver reports what the gate excludes, or would, when there is a
// metrics server to report it to.
func gateObserver(current func() types.PolicyEnforcement, gated *gormdb.DB, metricsServer *metrics.Server,
) (types.PolicyGateObserver, error) {
	if metricsServer == nil {
		return nil, nil //nolint:nilnil // no observer is not an error.
	}

	gate := metrics.NewPolicyGate(current, gated)
	if err := metricsServer.Registry().Register(gate); err != nil {
		return nil, fmt.Errorf("register policy gate metrics: %w", err)
	}

	return gate, nil
}
