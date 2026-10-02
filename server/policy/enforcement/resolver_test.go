// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package enforcement

import (
	"errors"
	"sync"
	"testing"
	"time"

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
// last.
type versions struct {
	mu      sync.Mutex
	current map[string]string
	err     error
}

func newVersions() *versions { return &versions{current: map[string]string{}} }

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

func enforcing(ids ...string) policyconfig.EnforcementConfig {
	return policyconfig.EnforcementConfig{
		Search:          policyconfig.ModeEnforce,
		Fetch:           policyconfig.ModeShadow,
		Policies:        ids,
		RefreshInterval: testInterval,
	}
}

// newResolver resolves once, then hands the resolver the test's clock.
func newResolver(t *testing.T, store *versions, c *clock, ids ...string) *Resolver {
	t.Helper()

	resolver, err := NewResolver(enforcing(ids...), store)
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

// The current version of each registered policy is enforced; the modes are
// the configured ones.
func TestResolver_EnforcesTheRegisteredVersion(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("opa:a", "h1")
	store.register("opa:b", "b7")

	got := newResolver(t, store, c, "opa:a", "opa:b").Current()

	assert.Equal(t, []types.EnforcedPolicy{{ID: "opa:a", Version: "h1"}, {ID: "opa:b", Version: "b7"}}, got.Policies)
	assert.Empty(t, got.Pending)
	assert.Equal(t, policyconfig.ModeEnforce, got.Search)
	assert.Equal(t, policyconfig.ModeShadow, got.Fetch)
}

// A policy no evaluator has registered is not enforced: there is no version
// whose verdicts could be asked for.
func TestResolver_UnregisteredPolicyIsPending(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()
	store.register("opa:a", "h1")

	got := newResolver(t, store, c, "opa:a", "opa:later").Current()

	assert.Equal(t, []types.EnforcedPolicy{{ID: "opa:a", Version: "h1"}}, got.Policies)
	assert.Equal(t, []types.EnforcedPolicy{{ID: "opa:later"}}, got.Pending)
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
	assert.Equal(t, []types.EnforcedPolicy{{ID: "opa:a", Version: "h2"}}, got.Policies)
}

// A policy registered after the server started is picked up.
func TestResolver_ALaterRegistrationIsPickedUp(t *testing.T) {
	t.Parallel()

	c := newClock()
	store := newVersions()

	resolver := newResolver(t, store, c, "opa:a")
	require.Equal(t, []types.EnforcedPolicy{{ID: "opa:a"}}, resolver.Current().Pending)

	store.register("opa:a", "h1")

	got := refresh(t, resolver)
	assert.Equal(t, []types.EnforcedPolicy{{ID: "opa:a", Version: "h1"}}, got.Policies)
	assert.Empty(t, got.Pending)
}

// The server does not start without knowing what it enforces; later, a
// failed recheck keeps what is in force.
func TestResolver_StoreErrors(t *testing.T) {
	t.Parallel()

	store := newVersions()
	store.fail(errors.New("database unavailable"))

	_, err := NewResolver(enforcing("opa:a"), store)
	require.ErrorContains(t, err, "database unavailable")

	c := newClock()
	store = newVersions()
	store.register("opa:a", "h1")

	resolver := newResolver(t, store, c, "opa:a")
	inForce := []types.EnforcedPolicy{{ID: "opa:a", Version: "h1"}}

	store.fail(errors.New("database unavailable"))
	require.Error(t, resolver.Refresh())
	assert.Equal(t, inForce, resolver.Current().Policies)
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

	assert.Equal(t, []types.EnforcedPolicy{{ID: "opa:a", Version: "h1"}}, resolver.Current().Policies, "within the interval")

	c.advance(testInterval)

	assert.Eventually(t, func() bool {
		policies := resolver.Current().Policies

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
	inForce := []types.EnforcedPolicy{{ID: "opa:a", Version: "h1"}}

	store.fail(errors.New("database unavailable"))
	c.advance(testInterval)

	assert.Equal(t, inForce, resolver.Current().Policies)

	assert.Eventually(t, func() bool {
		return !resolver.refreshing.Load() && resolver.refreshedAt.Load() == c.now().UnixNano()
	}, 5*time.Second, 10*time.Millisecond)

	assert.Equal(t, inForce, resolver.Current().Policies)
}
