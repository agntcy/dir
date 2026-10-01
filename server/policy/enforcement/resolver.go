// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package enforcement decides which version of each enforced content policy
// the server's reads apply.
package enforcement

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("policy/enforcement")

// VersionStore is what the resolver reads to follow the versions of the
// policies.
type VersionStore interface {
	GetCurrentPolicyVersion(policyID string) (string, bool, error)
}

// Resolver keeps the enforcement the server's reads apply. Nobody states a
// policy's version: the reconciler registers the version of each policy it
// runs, derived from the policy's content, and the resolver follows it.
//
// The current version is enforced strictly, as soon as the resolver sees it:
// a verdict under any other version does not count, so after a policy changes
// no record the new rule rejects is served while it is being re-evaluated. The
// price is that records are hidden until then.
type Resolver struct {
	cfg      policyconfig.EnforcementConfig
	store    VersionStore
	interval time.Duration
	now      func() time.Time

	current     atomic.Pointer[types.PolicyEnforcement]
	refreshedAt atomic.Int64
	refreshing  atomic.Bool

	// mu serializes refreshes and guards warned.
	mu sync.Mutex

	// warned holds the policies already reported as registered by no
	// evaluator.
	warned map[string]bool
}

// NewResolver resolves the enforcement once, so the server does not start
// without knowing what it enforces.
func NewResolver(cfg policyconfig.EnforcementConfig, store VersionStore) (*Resolver, error) {
	resolver := &Resolver{
		cfg:      cfg,
		store:    store,
		interval: cfg.GetRefreshInterval(),
		now:      time.Now,
		warned:   map[string]bool{},
	}

	if err := resolver.Refresh(); err != nil {
		return nil, err
	}

	return resolver, nil
}

// Current returns the enforcement in force. Once it is older than the
// refresh interval, it is refreshed in the background, so no read waits on
// it; a failed refresh keeps the last enforcement and is retried after the
// interval.
func (r *Resolver) Current() types.PolicyEnforcement {
	stale := r.now().Sub(time.Unix(0, r.refreshedAt.Load())) >= r.interval
	if stale && r.refreshing.CompareAndSwap(false, true) {
		go func() {
			defer r.refreshing.Store(false)

			if err := r.Refresh(); err != nil {
				logger.Warn("Could not recheck which policy versions are in force; keeping the last", "error", err)
			}
		}()
	}

	return *r.current.Load()
}

// Refresh rechecks now which version of each policy is in force.
func (r *Resolver) Refresh() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refreshedAt.Store(r.now().UnixNano())

	next := types.PolicyEnforcement{Search: r.cfg.Search, Fetch: r.cfg.Fetch}

	for _, id := range r.cfg.Policies {
		version, registered, err := r.store.GetCurrentPolicyVersion(id)
		if err != nil {
			return fmt.Errorf("get current version of policy %s: %w", id, err)
		}

		if !registered {
			if !r.warned[id] {
				r.warned[id] = true

				logger.Warn("Policy is configured for enforcement but no evaluator has registered it; it is not enforced", "policy_id", id)
			}

			next.Pending = append(next.Pending, types.EnforcedPolicy{ID: id})

			continue
		}

		next.Policies = append(next.Policies, types.EnforcedPolicy{ID: id, Version: version})
	}

	r.current.Store(&next)

	return nil
}
