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

	"github.com/agntcy/dir/server/policy"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("policy/enforcement")

// VersionStore is what the resolver reads and writes to follow the versions
// of the policies, and to tell when a policy's verdicts first cover the
// records.
type VersionStore interface {
	GetCurrentPolicyVersion(policyID string) (string, bool, error)
	PolicyBackfilled(policyID string) (bool, error)
	CountRecordsNeedingPolicyEvaluation(policyID, policyVersion string) (int64, error)
	MarkPolicyBackfilled(policyID string) error
}

// Resolver keeps the enforcement the server's reads apply. Nobody states a
// policy's version: the reconciler registers the version of each policy it
// runs, derived from the policy's content, and the resolver follows it.
//
// The current version is enforced strictly, as soon as the resolver sees it:
// a verdict under any other version does not count, so after a policy changes
// no record the new rule rejects is served while it is being re-evaluated. The
// price is that records are hidden until then.
//
// A policy is first enforced only once its verdicts cover every indexed
// record, so turning one on does not empty search while the records are
// evaluated; before that, reads are as they were without the policy. From then
// on it stays enforced: records indexed later have no verdict until evaluated,
// and are hidden until they do.
type Resolver struct {
	search      []policy.SearchPolicy
	evaluations []string
	store       VersionStore
	interval    time.Duration
	now         func() time.Time

	current     atomic.Pointer[types.PolicyEnforcement]
	refreshedAt atomic.Int64
	refreshing  atomic.Bool

	// mu serializes refreshes and guards warned and versions.
	mu sync.Mutex

	// warned holds the policies already reported as registered by no
	// evaluator.
	warned map[string]bool

	// versions holds the last version seen for each registered policy, so a
	// change can be reported when it happens. It is only updated by a refresh
	// that succeeds.
	versions map[string]string
}

// NewResolver resolves the enforcement once, so the server does not start
// without knowing what it enforces.
//
// search are the policies configured at op search, which the same refresh
// follows: they depend on the versions of the policies they read, so there is
// nothing to be gained by following them on a different clock.
//
// evaluations are every configured evaluation policy. All of them are
// resolved, whether or not anything reads them: what reads a verdict is an
// operation policy, which the resolver does not see and does not need to.
func NewResolver(
	interval time.Duration, store VersionStore, search []policy.SearchPolicy, evaluations []string,
) (*Resolver, error) {
	resolver := &Resolver{
		search:      search,
		evaluations: evaluations,
		store:       store,
		interval:    interval,
		now:         time.Now,
		warned:      map[string]bool{},
		versions:    map[string]string{},
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

	next := types.PolicyEnforcement{
		Evaluations: make(map[string]types.ResolvedPolicy, len(r.evaluations)),
	}

	// One pass asks about each policy once, however many times it is named.
	// A policy reached twice — two search policies reading the same one is
	// the ordinary case — is resolved once, and resolving one still in
	// backfill counts the records it has yet to cover, which is the most
	// expensive query the server makes on a timer.
	pass := make(map[string]resolution, len(r.evaluations))

	// First, so that the configured set warms the pass and the search
	// policies are answered from it rather than the other way round.
	for _, id := range r.evaluations {
		resolved, err := r.resolveOnce(pass, id)
		if err != nil {
			return err
		}

		next.Evaluations[id] = types.ResolvedPolicy{
			VersionedPolicy: resolved.policy,
			Covered:         resolved.enforced,
		}
	}

	for _, p := range r.search {
		resolved, ready, err := r.resolveSearch(pass, p)
		if err != nil {
			return err
		}

		if ready {
			next.SearchPolicies = append(next.SearchPolicies, resolved)
		}
	}

	r.recordVersions(pass)

	r.current.Store(&next)

	return nil
}

// recordVersions remembers the version each registered policy was resolved to
// in a refresh that succeeded, and reports a change of version under an
// enforcement that is in force. A change while the policy is not enforced hides
// no record, so it is not reported; the first version seen is not a change.
func (r *Resolver) recordVersions(pass map[string]resolution) {
	for id, resolved := range pass {
		if !resolved.registered {
			continue
		}

		if previous, seen := r.versions[id]; seen && previous != resolved.policy.Version && resolved.enforced {
			logger.Info("Policy version changed; records are re-evaluated under the new version before they are served again",
				"policy_id", id, "previous_version", previous, "version", resolved.policy.Version)
		}

		r.versions[id] = resolved.policy.Version
	}
}

// resolveSearch joins a search policy to the versions of the policies it
// reads, and reports whether it is ready to be checked at all.
//
// A policy is ready only once every policy it reads is enforced, which is the
// backfill latch. Unlike the flat list, an uncovered policy cannot simply be
// left out: dropping a term from a conjunction serves records the policy
// excludes, and dropping one from a disjunction hides records it allows, so
// there is no safe direction and the whole policy waits.
//
// The latch is about the verdicts, not about acting on them, so it holds a
// shadow policy back too: counting exclusions from verdicts that do not yet
// cover the records would report a blackout as a finding.
func (r *Resolver) resolveSearch(pass map[string]resolution, p policy.SearchPolicy) (types.SearchPolicy, bool, error) {
	reads := make([]types.VersionedPolicy, 0, len(p.Reads))

	for _, id := range p.Reads {
		read, err := r.resolveOnce(pass, id)
		if err != nil {
			return types.SearchPolicy{}, false, fmt.Errorf("search policy %s: %w", p.ID, err)
		}

		if !read.enforced {
			return types.SearchPolicy{}, false, nil
		}

		reads = append(reads, read.policy)
	}

	return types.SearchPolicy{ID: p.ID, Mode: p.Mode, Predicate: p.Predicate, Reads: reads}, true, nil
}

// resolution is what one pass concluded about one policy.
type resolution struct {
	policy     types.VersionedPolicy
	enforced   bool
	registered bool
}

// resolveOnce is resolve, answered from pass when the policy has already been
// asked about during it.
//
// Within a pass the answer cannot change: the versions and the backfill marks
// are read from the same database moments apart, and a policy that reads as
// covered twice in one pass would only be marked as covered twice.
func (r *Resolver) resolveOnce(pass map[string]resolution, id string) (resolution, error) {
	if cached, asked := pass[id]; asked {
		return cached, nil
	}

	resolved, err := r.resolve(id)
	if err != nil {
		return resolution{}, err
	}

	pass[id] = resolved

	return resolved, nil
}

// resolve works out what to enforce for one policy, and whether anything is.
func (r *Resolver) resolve(id string) (resolution, error) {
	version, registered, err := r.store.GetCurrentPolicyVersion(id)
	if err != nil {
		return resolution{}, fmt.Errorf("get current version of policy %s: %w", id, err)
	}

	if !registered {
		if !r.warned[id] {
			r.warned[id] = true

			logger.Warn("Policy is configured but no evaluator has registered it; it is not enforced", "policy_id", id)
		}

		return resolution{policy: types.VersionedPolicy{ID: id}}, nil
	}

	policy := types.VersionedPolicy{ID: id, Version: version}

	covered, err := r.covered(id, version)
	if err != nil {
		return resolution{}, err
	}

	return resolution{policy: policy, enforced: covered, registered: true}, nil
}

// covered reports whether the policy has been enforced before, or its
// verdicts under version cover every indexed record now, in which case it is
// marked as covered.
func (r *Resolver) covered(id, version string) (bool, error) {
	marked, err := r.store.PolicyBackfilled(id)
	if err != nil {
		return false, fmt.Errorf("check backfill of policy %s: %w", id, err)
	}

	if marked {
		return true, nil
	}

	remaining, err := r.store.CountRecordsNeedingPolicyEvaluation(id, version)
	if err != nil {
		return false, fmt.Errorf("count records still to evaluate under policy %s version %s: %w", id, version, err)
	}

	if remaining > 0 {
		return false, nil
	}

	if err := r.store.MarkPolicyBackfilled(id); err != nil {
		return false, fmt.Errorf("mark backfill of policy %s: %w", id, err)
	}

	logger.Info("Policy verdicts cover every indexed record; enforcing it", "policy_id", id, "version", version)

	return true, nil
}
