// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package enforcement

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agntcy/dir/server/policy"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testInterval = time.Minute

// clock is a time the tests move by hand.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func newClock() *clock { return &clock{t: time.Now()} }

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.t
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.t = c.t.Add(d)
}

// versions is a VersionStore: the version each policy's evaluator registered
// last, how many records each policy still lacks a verdict for, and which
// policies were marked as covering the records.
type versions struct {
	mu         sync.Mutex
	current    map[string]string
	remaining  int64
	counted    []string
	backfilled map[string]bool
	err        error
	errs       map[string]error // by method name
}

func newVersions() *versions {
	return &versions{current: map[string]string{}, backfilled: map[string]bool{}, errs: map[string]error{}}
}

// lack sets how many records the policies' current versions have no verdict
// for.
func (v *versions) lack(records int64) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.remaining = records
}

func (v *versions) failMethod(method string, err error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.errs[method] = err
}

func (v *versions) PolicyBackfilled(id string) (bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.backfilled[id], v.errs["PolicyBackfilled"]
}

func (v *versions) CountRecordsNeedingPolicyEvaluation(id, _ string) (int64, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.counted = append(v.counted, id)

	return v.remaining, v.errs["CountRecordsNeedingPolicyEvaluation"]
}

// countsFor is how many times the records still to evaluate under id were
// counted, which is the expensive question the resolver asks.
func (v *versions) countsFor(id string) int {
	v.mu.Lock()
	defer v.mu.Unlock()

	n := 0

	for _, counted := range v.counted {
		if counted == id {
			n++
		}
	}

	return n
}

func (v *versions) MarkPolicyBackfilled(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if err := v.errs["MarkPolicyBackfilled"]; err != nil {
		return err
	}

	v.backfilled[id] = true

	return nil
}

// register is what an evaluator does: it makes version the policy's current one.
func (v *versions) register(id, version string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.current[id] = version
}

func (v *versions) fail(err error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.err = err
}

func (v *versions) GetCurrentPolicyVersion(id string) (string, bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	version, ok := v.current[id]

	return version, ok, v.err
}

// covered is the evaluation policies the latch has released, and
// backfilling the ones it has not: between them, what Policies and Pending
// used to say. Sorted by ID, the map having no order of its own.
func covered(e types.PolicyEnforcement) []types.VersionedPolicy {
	return resolvedWhere(e, true)
}

func backfilling(e types.PolicyEnforcement) []types.VersionedPolicy {
	return resolvedWhere(e, false)
}

func resolvedWhere(e types.PolicyEnforcement, want bool) []types.VersionedPolicy {
	var policies []types.VersionedPolicy

	for _, p := range e.Evaluations {
		if p.Covered == want {
			policies = append(policies, p.VersionedPolicy)
		}
	}

	slices.SortFunc(policies, func(a, b types.VersionedPolicy) int { return strings.Compare(a.ID, b.ID) })

	return policies
}

// newResolver resolves once, then hands the resolver the test's clock.
func newResolver(t *testing.T, store *versions, c *clock, ids ...string) *Resolver {
	t.Helper()

	resolver, err := NewResolver(testInterval, store, nil, ids)
	require.NoError(t, err)

	resolver.now = c.now
	resolver.refreshedAt.Store(c.now().UnixNano())

	return resolver
}

func refresh(t *testing.T, resolver *Resolver) types.PolicyEnforcement {
	t.Helper()

	require.NoError(t, resolver.Refresh())

	return resolver.Current()
}

// A policy named more than once is asked about once per pass.
//
// Counting the records a policy has yet to cover is the most expensive query
// the resolver makes, and it runs on a timer. Two search policies reading the
// same evaluation policy is the ordinary arrangement, and each read of a
// policy still in backfill would otherwise count again.
func TestResolver_AsksAboutEachPolicyOncePerPass(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("cel:trusted", "h1")
	store.lack(3) // still in backfill, so the count is not short-circuited.

	// Named as a configured evaluation, by the enforced list, and by both
	// search policies.
	reads := []string{"cel:trusted"}
	search := []policy.SearchPolicy{
		{ID: "cel:one", Mode: policyconfig.ModeEnforce, Reads: reads},
		{ID: "cel:two", Mode: policyconfig.ModeShadow, Reads: reads},
	}

	resolver, err := NewResolver(testInterval, store, search, []string{"cel:trusted"})
	require.NoError(t, err)

	resolver.now = c.now

	assert.Equal(t, 1, store.countsFor("cel:trusted"), "three references, one question")

	require.NoError(t, resolver.Refresh())
	assert.Equal(t, 2, store.countsFor("cel:trusted"), "and once more on the next pass")
}

// The current version of each registered policy is enforced; the modes are
// the configured ones.
func TestResolver_EnforcesTheRegisteredVersion(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("opa:a", "h1")
	store.register("opa:b", "b7")

	got := newResolver(t, store, c, "opa:a", "opa:b").Current()

	assert.Equal(t, []types.VersionedPolicy{{ID: "opa:a", Version: "h1"}, {ID: "opa:b", Version: "b7"}}, covered(got))
	assert.Empty(t, backfilling(got))
}

// Every configured evaluation policy is resolved, whether or not anything
// reads it. What reads a verdict is an operation policy, which the resolver
// never sees, so it has no way to tell which of them matter and does not
// try: a policy left out here is one whose latch never releases, and a
// decision reading it would refuse nothing and report nothing, looking
// exactly like a policy that allows everything.
func TestResolver_ResolvesEveryConfiguredEvaluation(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("cel:one", "h1")
	store.register("cel:two", "b7")

	resolver, err := NewResolver(testInterval, store, nil,
		[]string{"cel:one", "cel:two", "cel:unregistered"})
	require.NoError(t, err)

	resolver.now = c.now

	assert.Equal(t, map[string]types.ResolvedPolicy{
		"cel:one": {VersionedPolicy: types.VersionedPolicy{ID: "cel:one", Version: "h1"}, Covered: true},
		"cel:two": {VersionedPolicy: types.VersionedPolicy{ID: "cel:two", Version: "b7"}, Covered: true},

		// Resolved, and resolved as not covered: a policy no evaluator has
		// registered is present with the latch shut, not absent.
		"cel:unregistered": {VersionedPolicy: types.VersionedPolicy{ID: "cel:unregistered"}},
	}, resolver.Current().Evaluations)
}

// A policy no evaluator has registered covers nothing: there is no version
// whose verdicts could be asked for.
func TestResolver_UnregisteredPolicyCoversNothing(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("opa:a", "h1")

	got := newResolver(t, store, c, "opa:a", "opa:later").Current()

	assert.Equal(t, []types.VersionedPolicy{{ID: "opa:a", Version: "h1"}}, covered(got))
	assert.Equal(t, []types.VersionedPolicy{{ID: "opa:later"}}, backfilling(got))
}

// Editing a policy registers a new version, which is enforced at once: a
// verdict under the old one no longer counts.
func TestResolver_AnEditIsEnforcedAtOnce(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("opa:a", "h1")

	resolver := newResolver(t, store, c, "opa:a")

	store.register("opa:a", "h2")

	got := refresh(t, resolver)
	assert.Equal(t, []types.VersionedPolicy{{ID: "opa:a", Version: "h2"}}, covered(got))
}

// A policy registered after the server started is picked up.
func TestResolver_ALaterRegistrationIsPickedUp(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()

	resolver := newResolver(t, store, c, "opa:a")
	require.Equal(t, []types.VersionedPolicy{{ID: "opa:a"}}, backfilling(resolver.Current()))

	store.register("opa:a", "h1")

	got := refresh(t, resolver)
	assert.Equal(t, []types.VersionedPolicy{{ID: "opa:a", Version: "h1"}}, covered(got))
	assert.Empty(t, backfilling(got))
}

// The server does not start without knowing what it enforces; later, a
// failed recheck keeps what is in force.
func TestResolver_StoreErrors(t *testing.T) {
	t.Parallel()

	store := newVersions()
	store.fail(errors.New("database unavailable"))

	_, err := NewResolver(testInterval, store, nil, []string{"opa:a"})
	require.ErrorContains(t, err, "database unavailable")

	c := newClock()
	store = newVersions()
	store.register("opa:a", "h1")

	resolver := newResolver(t, store, c, "opa:a")
	inForce := []types.VersionedPolicy{{ID: "opa:a", Version: "h1"}}

	store.fail(errors.New("database unavailable"))
	require.Error(t, resolver.Refresh())
	assert.Equal(t, inForce, covered(resolver.Current()))
}

// Current rechecks in the background once the interval has passed, and not
// before.
func TestResolver_CurrentRefreshesAfterTheInterval(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("opa:a", "h1")

	resolver := newResolver(t, store, c, "opa:a")

	store.register("opa:a", "h2")

	assert.Equal(t, []types.VersionedPolicy{{ID: "opa:a", Version: "h1"}}, covered(resolver.Current()), "within the interval")

	c.advance(testInterval)

	assert.Eventually(t, func() bool {
		policies := covered(resolver.Current())

		return len(policies) == 1 && policies[0].Version == "h2"
	}, 5*time.Second, 10*time.Millisecond)
}

// A failed recheck in the background keeps what is in force.
func TestResolver_FailedBackgroundRecheckKeepsTheLast(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("opa:a", "h1")

	resolver := newResolver(t, store, c, "opa:a")
	inForce := []types.VersionedPolicy{{ID: "opa:a", Version: "h1"}}

	store.fail(errors.New("database unavailable"))
	c.advance(testInterval)

	assert.Equal(t, inForce, covered(resolver.Current()))

	assert.Eventually(t, func() bool {
		return !resolver.refreshing.Load() && resolver.refreshedAt.Load() == c.now().UnixNano()
	}, 5*time.Second, 10*time.Millisecond)

	assert.Equal(t, inForce, covered(resolver.Current()))
}

// Turning a policy on does not empty search while the records are evaluated:
// it is not enforced until its verdicts cover them, and then it is.
func TestResolver_FirstRolloutWaitsForTheRecordsToBeEvaluated(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("opa:a", "h1")
	store.lack(3)

	resolver := newResolver(t, store, c, "opa:a")

	got := resolver.Current()
	assert.Empty(t, covered(got))
	assert.Equal(t, []types.VersionedPolicy{{ID: "opa:a", Version: "h1"}}, backfilling(got))

	store.lack(0)

	got = refresh(t, resolver)
	assert.Equal(t, []types.VersionedPolicy{{ID: "opa:a", Version: "h1"}}, covered(got))
	assert.Empty(t, backfilling(got))
}

// Records indexed after the policy covered the records have no verdict until
// evaluated; the policy stays enforced, and they stay hidden.
func TestResolver_ACoveredPolicyStaysEnforcedWhenRecordsArrive(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("opa:a", "h1")

	resolver := newResolver(t, store, c, "opa:a")

	store.lack(4)

	got := refresh(t, resolver)
	assert.Equal(t, []types.VersionedPolicy{{ID: "opa:a", Version: "h1"}}, covered(got))
}

// An edit to a covered policy is enforced at once, however many records have
// not been evaluated under it yet: it is not a first rollout.
func TestResolver_AnEditToACoveredPolicyIsEnforcedAtOnce(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("opa:a", "h1")

	resolver := newResolver(t, store, c, "opa:a")

	store.register("opa:a", "h2")
	store.lack(1000)

	got := refresh(t, resolver)
	assert.Equal(t, []types.VersionedPolicy{{ID: "opa:a", Version: "h2"}}, covered(got))
	assert.Empty(t, backfilling(got))
}

// That a policy was covered survives a restart.
func TestResolver_ACoveredPolicyStaysEnforcedAfterARestart(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("opa:a", "h1")

	_ = newResolver(t, store, c, "opa:a")

	store.lack(4)

	restarted := newResolver(t, store, c, "opa:a")
	assert.Equal(t, []types.VersionedPolicy{{ID: "opa:a", Version: "h1"}}, covered(restarted.Current()))
}

// Every call the resolver makes can fail the start, so the server never runs
// without knowing what it enforces.
func TestResolver_EveryStoreCallCanFailTheStart(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		method string
		lack   int64
	}{
		{"PolicyBackfilled", 0},
		{"CountRecordsNeedingPolicyEvaluation", 1},
		{"MarkPolicyBackfilled", 0},
	} {
		t.Run(tt.method, func(t *testing.T) {
			t.Parallel()

			store := newVersions()
			store.register("opa:a", "h1")
			store.lack(tt.lack)
			store.failMethod(tt.method, errors.New("database unavailable"))

			_, err := NewResolver(testInterval, store, nil, []string{"opa:a"})
			require.ErrorContains(t, err, "database unavailable")
		})
	}
}
